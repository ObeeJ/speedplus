package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/card"
	"github.com/speedplus/api/internal/config"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/ports"
	"github.com/speedplus/api/internal/repo"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrPaycodeExpired   = errors.New("paycode expired")
	ErrPaycodeInvalid   = errors.New("paycode invalid")
	ErrNotAssigned      = errors.New("scanner is not the assigned driver")
	ErrAlreadyConfirmed = errors.New("paycode already confirmed")
	ErrNotOwner         = errors.New("caller is not the order's customer")
)

type PaycodeService struct {
	repo          repo.PaycodeRepo
	cfg           *config.Config
	ledger        *LedgerService
	orders        *OrderService
	tier          ports.TierRecorder
	email         paycodeEmailSender
	users         repo.UserRepo
	orderRepo     repo.OrderRepo
	deliveryCodes *DeliveryCodeService
	referrals     *ReferralService
}

type paycodeEmailSender interface {
	SendOrderDelivered(ctx context.Context, toEmail, firstName, orderID, merchantName string, totalKobo int64)
}

func NewPaycodeService(r repo.PaycodeRepo, cfg *config.Config, ledger *LedgerService, orders *OrderService, tier ports.TierRecorder, email paycodeEmailSender, users repo.UserRepo, orderRepo repo.OrderRepo, deliveryCodes *DeliveryCodeService, referrals *ReferralService) *PaycodeService {
	return &PaycodeService{repo: r, cfg: cfg, ledger: ledger, orders: orders, tier: tier, email: email, users: users, orderRepo: orderRepo, deliveryCodes: deliveryCodes, referrals: referrals}
}

// settleReferral fires the referral payout check after a completed order.
// Non-blocking, nil-safe — order settlement must never depend on it.
func (s *PaycodeService) settleReferral(customerID uuid.UUID, subtotalKobo int64) {
	if s.referrals == nil {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("settleReferral goroutine panic", "panic", r)
			}
		}()
		if err := s.referrals.SettleCompletedOrder(context.Background(), customerID, subtotalKobo); err != nil {
			slog.Error("referral settlement failed", "customer_id", customerID.String(), "error", err)
		}
	}()
}

// Generate creates a signed paycode for the order.
// Format: spq.v1.{orderId}.{nonce}.{exp}.{HMAC-SHA256}
// Only the order's customer (the receiver) may generate the QR — possession of
// the rendered code is the receiver-identity proof the settlement flow relies on.
func (s *PaycodeService) Generate(ctx context.Context, orderID, callerID uuid.UUID) (*model.Paycode, error) {
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, ErrPaycodeInvalid
	}
	if order.CustomerID != callerID {
		return nil, ErrNotOwner
	}
	switch order.Status {
	case model.OrderDelivered, model.OrderCancelled, model.OrderRefunded:
		return nil, fmt.Errorf("cannot generate paycode for %s order", order.Status)
	}

	nonce := uuid.NewString()
	exp := time.Now().Add(10 * time.Minute)
	expUnix := fmt.Sprintf("%d", exp.Unix())

	payload := strings.Join([]string{"spq.v1", orderID.String(), nonce, expUnix}, ".")
	sig := s.sign(payload)
	fullPayload := payload + "." + sig

	pc := model.Paycode{
		ID:        uuid.New(),
		OrderID:   orderID,
		Nonce:     nonce,
		Payload:   fullPayload,
		ExpiresAt: exp,
	}
	if err := s.repo.Create(ctx, &pc); err != nil {
		return nil, err
	}
	return &pc, nil
}

// Resolve is called when a driver scans the QR.
// Verifies signature, expiry, order state, and that scanner is the assigned driver.
func (s *PaycodeService) Resolve(ctx context.Context, payload string, scannerID uuid.UUID) (*model.Order, *model.Paycode, error) {
	parts := strings.Split(payload, ".")
	// spq.v1.{orderId}.{nonce}.{exp}.{sig}
	if len(parts) != 6 || parts[0] != "spq" || parts[1] != "v1" {
		return nil, nil, ErrPaycodeInvalid
	}

	orderIDStr, nonce, expStr, sig := parts[2], parts[3], parts[4], parts[5]
	unsigned := strings.Join(parts[:5], ".")
	if !hmac.Equal([]byte(s.sign(unsigned)), []byte(sig)) {
		return nil, nil, ErrPaycodeInvalid
	}

	var expUnix int64
	fmt.Sscanf(expStr, "%d", &expUnix)
	if time.Now().Unix() > expUnix {
		return nil, nil, ErrPaycodeExpired
	}

	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return nil, nil, ErrPaycodeInvalid
	}

	pc, err := s.repo.FindByNonce(ctx, orderID, nonce)
	if err != nil {
		return nil, nil, ErrPaycodeInvalid
	}

	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, nil, fmt.Errorf("order not found")
	}

	// Scanner must be the assigned driver
	if order.DriverID == nil || *order.DriverID != scannerID {
		return nil, nil, ErrNotAssigned
	}

	// Order must be in_transit
	if order.Status != model.OrderInTransit {
		return nil, nil, fmt.Errorf("order not in transit (status: %s)", order.Status)
	}

	return order, pc, nil
}

// Confirm atomically: wallet debit (if unpaid) + escrow release + settlement + order → delivered.
// This is the ONLY path that releases escrow — satisfies the no-bypass rule.
func (s *PaycodeService) Confirm(ctx context.Context, paycodeID, driverID uuid.UUID, pin *string) error {
	var badgeDriverID *uuid.UUID
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		pc, err := s.repo.FindByIDTx(ctx, tx, paycodeID)
		if err != nil {
			return ErrPaycodeInvalid
		}
		if pc.ConfirmedAt != nil {
			return ErrAlreadyConfirmed
		}
		if time.Now().After(pc.ExpiresAt) {
			return ErrPaycodeExpired
		}

		order, err := s.repo.LockOrderTx(ctx, tx, pc.OrderID)
		if err != nil {
			return fmt.Errorf("order not found")
		}
		if order.DriverID == nil || *order.DriverID != driverID {
			return ErrNotAssigned
		}
		if order.Status != model.OrderInTransit {
			return fmt.Errorf("order not in transit (status: %s)", order.Status)
		}

		now := time.Now()
		pc.ConfirmedAt = &now
		pc.ScannedBy = &driverID
		if err := s.repo.SaveTx(ctx, tx, pc); err != nil {
			return err
		}

		if err := s.ledger.Settle(ctx, tx, order, pc.ID); err != nil {
			return fmt.Errorf("settlement: %w", err)
		}

		if err := s.orders.transitionTx(ctx, tx, order.ID, driverID, "driver", model.OrderDelivered, nil); err != nil {
			return fmt.Errorf("transition delivered: %w", err)
		}

		// Record completed order for trust tier evaluation (non-blocking).
		go s.tier.RecordCompletion(context.Background(), order.CustomerID)
		s.settleReferral(order.CustomerID, order.SubtotalKobo)
		// Enqueued post-commit — the worker counts delivered orders and would
		// not yet see this one from inside the open transaction.
		if order.DriverID != nil {
			did := *order.DriverID
			badgeDriverID = &did
		}

		// Order delivered email — best-effort, non-blocking.
		go func(o *model.Order) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("order delivered email goroutine panic", "panic", r)
				}
			}()
			customer, err := s.users.FindByID(context.Background(), o.CustomerID)
			if err != nil || customer.Email == nil {
				return
			}
			merchant, _ := s.orderRepo.FindMerchant(context.Background(), o.MerchantID)
			merchantName := ""
			if merchant != nil {
				merchantName = merchant.BusinessName
			}
			s.email.SendOrderDelivered(context.Background(), *customer.Email, customer.FirstName,
				o.ID.String(), merchantName, o.TotalKobo)
		}(order)

		return nil
	})
	if err != nil {
		return err
	}
	if badgeDriverID != nil {
		s.orders.EnqueueBadgeCheck(ctx, *badgeDriverID)
	}
	return nil
}

func (s *PaycodeService) sign(payload string) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.PaycodeSecret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// GenerateDeliveryCode creates a one-time 6-digit code for the order and
// returns the plaintext code. Called when order transitions to in_transit.
// The caller sends the code to the customer via SMS/push.
func (s *PaycodeService) GenerateDeliveryCode(ctx context.Context, orderID uuid.UUID) (string, error) {
	return s.deliveryCodes.Generate(ctx, orderID)
}

// ConfirmByCode is the PRIMARY delivery confirmation path.
// The rider enters the 6-digit code the customer shared with them.
// On success: escrow releases, order marked delivered.
//
// Fallback: ConfirmByCard — used when customer has no phone signal.
//
// lat/lng are the rider's GPS at code entry — optional evidence, flag-only:
// a far or missing location never blocks settlement.
func (s *PaycodeService) ConfirmByCode(ctx context.Context, orderID, driverID uuid.UUID, code string, lat, lng *float64) error {
	var badgeDriverID *uuid.UUID
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		order, err := s.repo.LockOrderTx(ctx, tx, orderID)
		if err != nil {
			return fmt.Errorf("order not found")
		}
		if order.DriverID == nil || *order.DriverID != driverID {
			return ErrNotAssigned
		}
		if order.Status != model.OrderInTransit {
			return fmt.Errorf("order not in transit (status: %s)", order.Status)
		}

		if err := s.deliveryCodes.Verify(ctx, tx, orderID, code); err != nil {
			return err
		}

		if order.Vertical == "gas" {
			count, err := s.repo.CountWeightProof(ctx, tx, orderID)
			if err != nil {
				return fmt.Errorf("weight proof check: %w", err)
			}
			if count == 0 {
				return fmt.Errorf("gas order requires a weight photo before confirmation")
			}
		}

		if addr, err := s.repo.FindAddress(ctx, tx, order.DeliveryAddressID); err == nil {
			flagged, locErr := s.deliveryCodes.RecordConfirmLocation(ctx, tx, orderID, lat, lng, addr.Lat, addr.Lng)
			if locErr != nil {
				slog.Warn("delivery code: record confirm location failed", "order_id", orderID.String(), "error", locErr)
			} else if flagged {
				slog.Warn("delivery code confirmed far from dropoff — flagged for review",
					"order_id", orderID.String(), "driver_id", driverID.String())
			}
		}

		if err := s.ledger.Settle(ctx, tx, order, uuid.New()); err != nil {
			return fmt.Errorf("settlement: %w", err)
		}

		if err := s.orders.transitionTx(ctx, tx, order.ID, driverID, "driver", model.OrderDelivered, nil); err != nil {
			return fmt.Errorf("transition delivered: %w", err)
		}

		go s.tier.RecordCompletion(context.Background(), order.CustomerID)
		s.settleReferral(order.CustomerID, order.SubtotalKobo)
		// Enqueued post-commit — see Confirm for why.
		if order.DriverID != nil {
			did := *order.DriverID
			badgeDriverID = &did
		}
		go func(o *model.Order) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("order delivered email goroutine panic", "panic", r)
				}
			}()
			customer, err := s.users.FindByID(context.Background(), o.CustomerID)
			if err != nil || customer.Email == nil {
				return
			}
			merchant, _ := s.orderRepo.FindMerchant(context.Background(), o.MerchantID)
			merchantName := ""
			if merchant != nil {
				merchantName = merchant.BusinessName
			}
			s.email.SendOrderDelivered(context.Background(), *customer.Email, customer.FirstName,
				o.ID.String(), merchantName, o.TotalKobo)
		}(order)

		return nil
	})
	if err != nil {
		return err
	}
	if badgeDriverID != nil {
		s.orders.EnqueueBadgeCheck(ctx, *badgeDriverID)
	}
	return nil
}

// ConfirmByCard is the offline delivery path.
// The rider scans the customer's static SpeedPlus card QR.
// The customer then enters their 4-digit wallet PIN on the rider's phone.
// This is the customer's explicit consent at the door.
func (s *PaycodeService) ConfirmByCard(ctx context.Context, cardPayload string, driverID uuid.UUID, pin string) error {
	// 1. Verify card signature → extract customerID
	customerID, err := card.VerifyPayload(cardPayload, s.cfg.PaycodeSecret)
	if err != nil {
		return ErrPaycodeInvalid
	}

	// 2. Verify customer PIN (with lockout — mirrors AuthService.VerifyPIN)
	pinModel, err := s.users.FindPIN(ctx, customerID)
	if err != nil {
		return fmt.Errorf("pin not set")
	}
	if pinModel.LockedUntil != nil && time.Now().Before(*pinModel.LockedUntil) {
		return errors.New("PIN temporarily locked. Try again later.")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(pinModel.PINHash), []byte(pin)); err != nil {
		var lockUntil *time.Time
		if pinModel.FailedAttempts+1 >= pinMaxAttempts {
			t := time.Now().Add(pinLockDuration)
			lockUntil = &t
		}
		_ = s.users.IncrementPINFailure(ctx, customerID, lockUntil)
		if lockUntil != nil {
			return errors.New("PIN temporarily locked. Try again later.")
		}
		return fmt.Errorf("invalid pin")
	}
	_ = s.users.ResetPINFailures(ctx, customerID)

	// 3. Find the active in_transit order for this customer
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		order, err := s.repo.FindActiveInTransitOrder(ctx, tx, customerID)
		if err != nil {
			return fmt.Errorf("no active in-transit order found for this customer")
		}

		if order.DriverID == nil || *order.DriverID != driverID {
			return ErrNotAssigned
		}

		if err := s.ledger.Settle(ctx, tx, order, uuid.New()); err != nil {
			return fmt.Errorf("card settlement: %w", err)
		}

		if err := s.orders.transitionTx(ctx, tx, order.ID, driverID, "driver", model.OrderDelivered, nil); err != nil {
			return fmt.Errorf("transition delivered: %w", err)
		}

		go s.tier.RecordCompletion(context.Background(), customerID)
		s.settleReferral(customerID, order.SubtotalKobo)
		return nil
	})
}
