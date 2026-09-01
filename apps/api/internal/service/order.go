package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/crypto"
	"github.com/speedplus/api/internal/dto"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/observability"
	"github.com/speedplus/api/internal/ports"
	"github.com/speedplus/api/internal/repo"
	"github.com/speedplus/api/internal/whatsapp"
	"gorm.io/gorm"
)

var (
	ErrIllegalTransition    = errors.New("illegal order state transition")
	ErrQuoteInvalid         = errors.New("quote invalid or expired")
	ErrMerchantClosed       = errors.New("merchant is currently closed")
	ErrRxRequired           = errors.New("prescription required for this order")
	ErrRxNotApproved        = errors.New("prescription is not yet approved by the pharmacy")
	ErrOrderNotFound        = errors.New("order not found")
	ErrGasValidation        = errors.New("gas order validation failed")
	ErrInsufficientBalance  = errors.New("insufficient balance")
	ErrOrderForbidden       = errors.New("forbidden")
)

type CreateOrderInput struct {
	CustomerID        uuid.UUID
	MerchantID        uuid.UUID
	QuoteID           uuid.UUID
	Vertical          string
	Items             []OrderItemInput
	DeliveryAddrID    uuid.UUID
	RecipientName     *string
	RecipientPhone    *string
	PrescriptionID    *uuid.UUID
	TipKobo           int64
	ScheduledFor      *time.Time
	PaymentMethod     string
	DeclaredValueKobo *int64
	IdempotencyKey    string
	Stops             []OrderStopInput
	// Gas-specific (Phase 1)
	GasMode    *string    // swap|refill|new_cylinder
	CylinderID *uuid.UUID // refill mode: customer's registered cylinder
}

type OrderStopInput struct {
	Sequence       int
	AddressID      uuid.UUID
	RecipientName  *string
	RecipientPhone *string
	Notes          *string
}

type OrderItemInput struct {
	ProductID        uuid.UUID
	Name             string
	Quantity         int
	UnitPriceKobo    int64
	WeightKg         float64
	SizeCategory     string
	Customizations   *string
	SubstitutionPref *string
}

type OrderService struct {
	orders        repo.OrderRepo
	pricing       *PricingService
	ledger        *LedgerService
	tier          *TierService
	dispatch      *DispatchService
	hub           wsPublisher
	deliveryCodes *DeliveryCodeService
	recipients    *crypto.Cipher // encrypts/decrypts recipient name+phone at rest
	email         orderEmailSender
	users         repo.UserRepo
	wa            ports.WhatsAppNotifier
	// enqueueReview schedules post-review rating/badge recomputation. Injected
	// after construction to avoid a service→worker import cycle.
	enqueueReview func(revieweeID, revieweeType string) error
}

// InjectReviewQueue wires the asynq enqueue function for post-review
// aggregation. Until called, reviews persist but aggregation is skipped.
func (s *OrderService) InjectReviewQueue(fn func(revieweeID, revieweeType string) error) {
	s.enqueueReview = fn
}

// EnqueueBadgeCheck schedules a driver badge re-evaluation — e.g. once a
// delivery completes. Call it *after* the enclosing transaction commits: the
// worker counts delivered orders, so running it against an uncommitted tx
// undercounts and would silently skip a driver's first_delivery badge.
// Enqueue failure is logged, not returned: the delivery itself already
// succeeded and must not be rolled back over a badge.
func (s *OrderService) EnqueueBadgeCheck(ctx context.Context, driverID uuid.UUID) {
	if s.enqueueReview == nil {
		return
	}
	if err := s.enqueueReview(driverID.String(), "driver"); err != nil {
		observability.CaptureError(ctx, err, "EnqueueBadgeCheck: enqueue failed",
			"driver_id", driverID.String())
	}
}

// orderEmailSender is the subset of email.Client used by OrderService.
type orderEmailSender interface {
	SendDeliveryCode(ctx context.Context, toEmail, firstName, code, orderID string)
}

// wsPublisher is the subset of ws.Hub used by OrderService.
type wsPublisher interface {
	Publish(ctx context.Context, channel, event string, data interface{}) error
}

func NewOrderService(orders repo.OrderRepo, pricing *PricingService, ledger *LedgerService, tier *TierService) *OrderService {
	return &OrderService{orders: orders, pricing: pricing, ledger: ledger, tier: tier}
}

func (s *OrderService) InjectDispatch(d *DispatchService, hub wsPublisher) {
	s.dispatch = d
	s.hub = hub
}

func (s *OrderService) InjectDeliveryCodes(dc *DeliveryCodeService) {
	s.deliveryCodes = dc
}

// InjectEmail wires the email client and user repo so Transition can send
// the delivery code to the customer when the order goes in_transit.
func (s *OrderService) InjectEmail(e orderEmailSender, u repo.UserRepo) {
	s.email = e
	s.users = u
}

// InjectWhatsApp wires the WhatsApp notifier for order lifecycle notifications.
// Optional — if not called all WA sends are silent no-ops.
func (s *OrderService) InjectWhatsApp(wa ports.WhatsAppNotifier) {
	s.wa = wa
}

// InjectRecipientCipher wires the AES-GCM cipher used to encrypt/decrypt
// recipient name+phone. Required before any order carrying recipient data is
// created — see encryptRecipient.
func (s *OrderService) InjectRecipientCipher(c *crypto.Cipher) {
	s.recipients = c
}

// encryptRecipient encrypts a recipient PII field for storage. nil in, nil
// out. If the field is set but no cipher is configured, this fails loudly
// rather than ever writing plaintext under an "_enc" column.
func (s *OrderService) encryptRecipient(v *string) (*string, error) {
	if v == nil {
		return nil, nil
	}
	if s.recipients == nil {
		return nil, fmt.Errorf("recipient encryption not configured")
	}
	return s.recipients.EncryptPtr(v)
}

// generateTrackingRef produces a short human-readable tracking reference.
// Format: SPX-XXXXX where X is uppercase alphanumeric (no ambiguous chars).
// Collision probability at 1M orders: ~0.5% — DB unique index is the hard guard.
func generateTrackingRef() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 5)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			b[i] = chars[0]
			continue
		}
		b[i] = chars[n.Int64()]
	}
	return "SPX-" + string(b)
}

func (s *OrderService) Create(ctx context.Context, in CreateOrderInput) (*model.Order, error) {
	// Idempotency check
	if existing, err := s.orders.FindByIdempotencyKey(ctx, in.IdempotencyKey); err == nil {
		return existing, nil
	}

	// Validate quote — fast-fail before opening a transaction for the common
	// cases (expired, tampered, wrong subtotal). The real race-safe check
	// happens inside the transaction via LockQuoteTx below.
	var subtotal int64
	for _, item := range in.Items {
		subtotal += item.UnitPriceKobo * int64(item.Quantity)
	}
	quote, err := s.pricing.ValidateQuote(ctx, in.QuoteID, subtotal)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrQuoteInvalid, err)
	}

	// Pharmacy vertical requires a prescription that has actually been
	// approved by the merchant this order is being placed against, is not
	// expired, and has not already been used on another order. This
	// pre-transaction check is a fast-fail for the common case (bad ID,
	// wrong customer, not yet approved) with a clear error; the real
	// single-use guarantee comes from ConsumePrescriptionTx's conditional
	// UPDATE inside the transaction below, which is what actually prevents
	// two concurrent order-creations from both succeeding off the same Rx —
	// a Go-level check here alone would race.
	if in.Vertical == "pharmacy" {
		if in.PrescriptionID == nil {
			return nil, ErrRxRequired
		}
		rx, err := s.orders.FindPrescription(ctx, *in.PrescriptionID)
		if err != nil {
			return nil, ErrRxRequired
		}
		if rx.CustomerID != in.CustomerID {
			return nil, ErrRxRequired
		}
		if rx.MerchantID == nil || *rx.MerchantID != in.MerchantID {
			return nil, ErrRxRequired
		}
		if rx.Status != "approved" {
			return nil, ErrRxNotApproved
		}
		if rx.ExpiresAt != nil && rx.ExpiresAt.Before(time.Now()) {
			return nil, ErrRxNotApproved
		}
	}

	// Gas vertical: validate gas_mode; require registered cylinder for refill;
	// set DeclaredValueKobo to the cylinder's custody value.
	if in.Vertical == "gas" {
		validModes := map[string]bool{"swap": true, "refill": true, "new_cylinder": true}
		if in.GasMode == nil || !validModes[*in.GasMode] {
			return nil, fmt.Errorf("%w: gas_mode must be swap, refill, or new_cylinder", ErrGasValidation)
		}
		if *in.GasMode == "refill" {
			if in.CylinderID == nil {
				return nil, fmt.Errorf("%w: refill orders require a registered cylinder (cylinder_id)", ErrGasValidation)
			}
			// Verify the cylinder belongs to this customer and is active.
			cyl, err := s.orders.FindCylinder(ctx, *in.CylinderID)
			if err != nil {
				return nil, fmt.Errorf("%w: cylinder not found", ErrGasValidation)
			}
			if cyl.UserID != in.CustomerID {
				return nil, fmt.Errorf("%w: cylinder does not belong to this customer", ErrGasValidation)
			}
			if cyl.Status != "active" {
				return nil, fmt.Errorf("%w: cylinder is not available (status: %s)", ErrGasValidation, cyl.Status)
			}
			// Set declared value to the cylinder's replacement cost (₦50,000 default).
			if in.DeclaredValueKobo == nil {
				v := int64(5_000_000) // ₦50,000 in kobo
				in.DeclaredValueKobo = &v
			}
		}
	}

	// Pay-on-arrival gate: trust-ladder checks only.
	// Settlement path (card scan → PIN → wallet debit) is implemented in
	// PaycodeService.ConfirmByCard — POD orders are fully settleable.
	if in.PaymentMethod == "pay_on_arrival" {
		if s.tier == nil {
			return nil, ErrPODTierLocked
		}
		totalKobo := quote.TotalKobo + in.TipKobo
		if err := s.tier.CanUsePayOnArrival(ctx, in.CustomerID, totalKobo); err != nil {
			return nil, err
		}
		// Eligible — fall through to order creation below (no escrow hold for POD).
	}

	recipientNameEnc, err := s.encryptRecipient(in.RecipientName)
	if err != nil {
		return nil, fmt.Errorf("encrypt recipient name: %w", err)
	}
	recipientPhoneEnc, err := s.encryptRecipient(in.RecipientPhone)
	if err != nil {
		return nil, fmt.Errorf("encrypt recipient phone: %w", err)
	}

	var order *model.Order
	err = s.orders.Transaction(ctx, func(tx *gorm.DB) error {
		// Check merchant is open + not suspended + KYC approved
		merchant, err := s.orders.FindMerchant(ctx, in.MerchantID)
		if err != nil {
			return fmt.Errorf("merchant not found")
		}
		if merchant.Status != model.MerchantActive {
			return fmt.Errorf("merchant is not active")
		}
		if !merchant.IsOpen {
			return ErrMerchantClosed
		}
		if in.Vertical == "gas" && merchant.FillStatus == "delisted" {
			return fmt.Errorf("%w: this merchant is temporarily unavailable for gas orders", ErrGasValidation)
		}

		order = &model.Order{
			ID:                uuid.New(),
			CustomerID:        in.CustomerID,
			MerchantID:        in.MerchantID,
			QuoteID:           in.QuoteID,
			Vertical:          in.Vertical,
			Status:            model.OrderPending,
			SubtotalKobo:      quote.SubtotalKobo,
			DeliveryKobo:      quote.DeliveryKobo,
			ServiceKobo:       quote.ServiceKobo,
			TipKobo:           in.TipKobo,
			TotalKobo:         quote.TotalKobo + in.TipKobo,
			DeliveryAddressID: in.DeliveryAddrID,
			RecipientNameEnc:  recipientNameEnc,
			RecipientPhoneEnc: recipientPhoneEnc,
			PrescriptionID:    in.PrescriptionID,
			ScheduledFor:      in.ScheduledFor,
			PaymentMethod:     "wallet",
			DeclaredValueKobo: in.DeclaredValueKobo,
			GasMode:           in.GasMode,
			CylinderID:        in.CylinderID,
			IdempotencyKey:    in.IdempotencyKey,
		}
		if in.PaymentMethod == "pay_on_arrival" {
			order.PaymentMethod = "pay_on_arrival"
		}
		// Generate a short human-readable tracking reference for package orders.
		// Retry once on the rare collision — DB unique index is the final guard.
		if in.Vertical == "package" {
			ref := generateTrackingRef()
			order.TrackingRef = &ref
		}

		for _, item := range in.Items {
			order.Items = append(order.Items, model.OrderItem{
				ID:               uuid.New(),
				OrderID:          order.ID,
				ProductID:        item.ProductID,
				Name:             item.Name,
				Quantity:         item.Quantity,
				UnitPriceKobo:    item.UnitPriceKobo,
				TotalKobo:        item.UnitPriceKobo * int64(item.Quantity),
				WeightKg:         item.WeightKg,
				SizeCategory:     item.SizeCategory,
				Customizations:   item.Customizations,
				SubstitutionPref: item.SubstitutionPref,
			})
		}

		if err := s.orders.CreateTx(ctx, tx, order); err != nil {
			return err
		}

		// Atomically consume the Rx now that we're holding the transaction —
		// this is the actual single-use guarantee (see comment above). A race
		// where two requests both passed the pre-check above will have exactly
		// one succeed here; the loser gets ErrRxNotApproved even though its
		// earlier read said "approved".
		if in.Vertical == "pharmacy" {
			rows, err := s.orders.ConsumePrescriptionTx(ctx, tx, *in.PrescriptionID, in.CustomerID, in.MerchantID, order.ID)
			if err != nil {
				return fmt.Errorf("consume prescription: %w", err)
			}
			if rows == 0 {
				return ErrRxNotApproved
			}
		}

		// Mark quote used — lock first so two concurrent order-creations
		// cannot both pass the UsedAt == nil check on the same quote.
		quote, err := s.orders.LockQuoteTx(ctx, tx, in.QuoteID)
		if err != nil {
			return fmt.Errorf("%w: quote lock failed", ErrQuoteInvalid)
		}
		if quote.UsedAt != nil {
			return fmt.Errorf("%w: quote already used", ErrQuoteInvalid)
		}
		if err := s.pricing.MarkQuoteUsed(ctx, in.QuoteID); err != nil {
			return err
		}

		// Multi-drop stops (package vertical)
		for _, stop := range in.Stops {
			stopNameEnc, err := s.encryptRecipient(stop.RecipientName)
			if err != nil {
				return fmt.Errorf("encrypt stop %d recipient name: %w", stop.Sequence, err)
			}
			stopPhoneEnc, err := s.encryptRecipient(stop.RecipientPhone)
			if err != nil {
				return fmt.Errorf("encrypt stop %d recipient phone: %w", stop.Sequence, err)
			}
			order.Stops = append(order.Stops, model.OrderStop{
				ID:                uuid.New(),
				OrderID:           order.ID,
				Sequence:          stop.Sequence,
				AddressID:         stop.AddressID,
				RecipientNameEnc:  stopNameEnc,
				RecipientPhoneEnc: stopPhoneEnc,
				Notes:             stop.Notes,
				QRCode:            "", // generated when order transitions to in_transit
				Status:            "pending",
			})
		}
		if len(in.Stops) > 0 {
			if err := s.orders.CreateStopsTx(ctx, tx, order.Stops); err != nil {
				return fmt.Errorf("create stops: %w", err)
			}
		}

		// Escrow hold — debit customer wallet for wallet-payment orders.
		// Pay-on-arrival orders skip this: the wallet is debited at the door
		// when the rider scans the customer's Fourdat card + PIN.
		if order.PaymentMethod != "pay_on_arrival" {
			if err := s.ledger.HoldEscrow(ctx, tx, order.ID, in.CustomerID, order.TotalKobo); err != nil {
				return fmt.Errorf("escrow hold: %w", err)
			}
		}

		// Audit event
		return s.orders.CreateEventTx(ctx, tx, &model.OrderEvent{
			ID:        uuid.New(),
			OrderID:   order.ID,
			ToStatus:  model.OrderPending,
			ActorID:   in.CustomerID,
			ActorRole: "customer",
		})
	})
	if err != nil {
		return nil, err
	}

	// Trigger dispatch asynchronously after the transaction commits.
	// Merchant lat/lng is used as the pickup point for KNN driver search.
	if s.dispatch != nil {
		go func(o *model.Order) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("dispatch goroutine panic", "panic", r, "order_id", o.ID)
				}
			}()
			merchant, dbErr := s.orders.FindMerchant(context.Background(), o.MerchantID)
			if dbErr != nil {
				return
			}
			dropoff, _ := s.orders.FindAddress(context.Background(), o.DeliveryAddressID) // best-effort

			candidates, dispErr := s.dispatch.Dispatch(context.Background(), o, merchant.Lat, merchant.Lng)
			if dispErr != nil || len(candidates) == 0 {
				return
			}
			// Notify each targeted driver via WS on their personal channel.
			// Payload matches what the driver app's offer card expects:
			// offerId (so accept/reject hits the right row), addresses, distance.
			if s.hub != nil {
				dropoffLabel := ""
				if dropoff != nil {
					dropoffLabel = fmt.Sprintf("%s, %s", dropoff.Street, dropoff.City)
				}
				for _, cand := range candidates {
					_ = s.hub.Publish(context.Background(),
						"driver:"+cand.DriverID.String(),
						"new_offer",
						map[string]interface{}{
							"offerId":        cand.OfferID,
							"orderId":        o.ID,
							"vertical":       o.Vertical,
							"totalKobo":      o.TotalKobo,
							"distanceKm":     cand.DistanceKm,
							"pickupAddress":  merchant.BusinessName,
							"dropoffAddress": dropoffLabel,
						},
					)
				}
			}
			// Notify customer that we're searching.
			if s.hub != nil {
				_ = s.hub.Publish(context.Background(),
					"order:"+o.ID.String(),
					"searching_driver",
					map[string]interface{}{"orderId": o.ID},
				)
			}
		}(order)
	}

	// Notify customer on WhatsApp — best-effort, never blocks order creation.
	if s.wa != nil && s.users != nil {
		go func(o *model.Order) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("whatsapp order_confirmed goroutine panic", "panic", r, "order_id", o.ID)
				}
			}()
			u, err := s.users.FindByID(context.Background(), o.CustomerID)
			if err != nil || u.Phone == "" {
				return
			}
			merchant, err := s.orders.FindMerchant(context.Background(), o.MerchantID)
			if err != nil {
				return
			}
			total := fmt.Sprintf("₦%.0f", float64(o.TotalKobo)/100)
			s.wa.OrderConfirmed(whatsapp.NormalisePhone(u.Phone), o.ID.String(), merchant.BusinessName, total)
		}(order)
	}

	return order, nil
}

// Transition moves an order through the state machine.
// Returns 409-equivalent error on illegal transition.
// Transition moves an order through the state machine. actorID must be the
// *business-entity* ID for the actor's role, not their login user ID — i.e.
// for a merchant, the resolved model.Merchant.ID (see LedgerService.ResolveWalletOwner
// for the same wrinkle on the wallet side); for a driver, order.DriverID
// already IS the login user ID. Callers must resolve that before invoking.
func (s *OrderService) Transition(ctx context.Context, orderID, actorID uuid.UUID, actorRole string, to model.OrderStatus, note *string) error {
	return s.orders.Transaction(ctx, func(tx *gorm.DB) error {
		return s.transitionTx(ctx, tx, orderID, actorID, actorRole, to, note)
	})
}

// transitionTx runs the state machine check and all side-effects inside an
// already-open transaction. Use this when the caller owns the transaction
// (e.g. paycode confirmation, dispatch assignment) to avoid nested tx issues.
func (s *OrderService) transitionTx(ctx context.Context, tx *gorm.DB, orderID, actorID uuid.UUID, actorRole string, to model.OrderStatus, note *string) error {
	order, err := s.orders.LockForUpdate(ctx, tx, orderID)
	if err != nil {
		return err
	}

	// Row-level ownership — fail closed.
	switch actorRole {
	case "merchant":
		if order.MerchantID != actorID {
			return ErrOrderForbidden
		}
	case "driver":
		if order.DriverID == nil || *order.DriverID != actorID {
			return ErrOrderForbidden
		}
	case "customer":
		if order.CustomerID != actorID {
			return ErrOrderForbidden
		}
	case "admin":
		// admin may transition any order
	default:
		return ErrOrderForbidden
	}

	allowed := model.ValidTransitions[order.Status]
	valid := false
	for _, st := range allowed {
		if st == to {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("%w: %s → %s", ErrIllegalTransition, order.Status, to)
	}

	from := order.Status
	order.Status = to
	if to == model.OrderDelivered {
		now := time.Now()
		order.DeliveredAt = &now
	}

	if err := s.orders.SaveTx(ctx, tx, order); err != nil {
		return err
	}

	if err := s.orders.CreateEventTx(ctx, tx, &model.OrderEvent{
		ID:         uuid.New(),
		OrderID:    orderID,
		FromStatus: from,
		ToStatus:   to,
		ActorID:    actorID,
		ActorRole:  actorRole,
		Note:       note,
	}); err != nil {
		return err
	}

	if to == model.OrderInTransit && s.deliveryCodes != nil {
		code, err := s.deliveryCodes.Generate(ctx, orderID)
		if err != nil {
			return fmt.Errorf("delivery code generate: %w", err)
		}
		if s.email != nil && s.users != nil {
			go func(customerID uuid.UUID, c, oid string) {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("delivery code email goroutine panic", "panic", r)
					}
				}()
				u, err := s.users.FindByID(context.Background(), customerID)
				if err != nil || u.Email == nil {
					return
				}
				s.email.SendDeliveryCode(context.Background(), *u.Email, u.FirstName, c, oid)
			}(order.CustomerID, code, orderID.String())
		}
		// WhatsApp delivery code — free (inside CSW opened by order_confirmed reply)
		if s.wa != nil && s.users != nil {
			go func(customerID uuid.UUID, c string) {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("whatsapp delivery_code goroutine panic", "panic", r)
					}
				}()
				u, err := s.users.FindByID(context.Background(), customerID)
				if err != nil || u.Phone == "" {
					return
				}
				s.wa.DeliveryCode(whatsapp.NormalisePhone(u.Phone), c)
			}(order.CustomerID, code)
		}
		if s.hub != nil {
			_ = s.hub.Publish(ctx,
				"order:"+orderID.String(),
				"delivery_code",
				map[string]interface{}{"code": code, "orderId": orderID},
			)
		}
	}

	if to == model.OrderDriverAssigned && s.hub != nil {
		_ = s.hub.Publish(ctx,
			"order:"+orderID.String(),
			"driver_assigned",
			map[string]interface{}{"orderId": orderID, "driverId": order.DriverID},
		)
	}

	// WhatsApp: rider assigned notification
	if to == model.OrderDriverAssigned && s.wa != nil && s.users != nil {
		go func(o *model.Order) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("whatsapp rider_assigned goroutine panic", "panic", r, "order_id", o.ID)
				}
			}()
			u, err := s.users.FindByID(context.Background(), o.CustomerID)
			if err != nil || u.Phone == "" {
				return
			}
			riderName := "Your rider"
			if o.DriverID != nil {
				if driver, err := s.users.FindByID(context.Background(), *o.DriverID); err == nil {
					riderName = driver.FirstName
				}
			}
			// ETA is not stored on the order at this stage; omit rather than lie.
			// Wire a real value here once OSRM duration is persisted on the offer.
			s.wa.RiderAssigned(whatsapp.NormalisePhone(u.Phone), riderName, "")
		}(order)
	}

	// WhatsApp: order delivered notification
	if to == model.OrderDelivered && s.wa != nil && s.users != nil {
		go func(o *model.Order) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("whatsapp order_delivered goroutine panic", "panic", r, "order_id", o.ID)
				}
			}()
			u, err := s.users.FindByID(context.Background(), o.CustomerID)
			if err != nil || u.Phone == "" {
				return
			}
			s.wa.OrderDelivered(whatsapp.NormalisePhone(u.Phone), o.ID.String())
		}(order)
	}

	if s.hub != nil {
		_ = s.hub.Publish(ctx,
			"order:"+orderID.String(),
			"order_status_changed",
			map[string]interface{}{"orderId": orderID, "status": to},
		)
	}

	if to == model.OrderDelivered && s.hub != nil {
		_ = s.hub.Publish(ctx,
			"order:"+orderID.String(),
			"order_delivered",
			map[string]interface{}{"orderId": orderID},
		)
	}

	return nil
}

func (s *OrderService) GetByID(ctx context.Context, orderID, requesterID uuid.UUID, requesterRole string) (*model.Order, error) {
	order, err := s.orders.FindByIDWithItems(ctx, orderID)
	if err != nil {
		return nil, err
	}
	// Row-level ownership. Fail closed: only the four known roles are allowed,
	// and each (except admin) must own the row. An empty or unrecognized role
	// — e.g. a middleware slip — must NOT fall through to full access (BOLA).
	switch requesterRole {
	case "customer":
		if order.CustomerID != requesterID {
			return nil, ErrOrderForbidden
		}
	case "driver":
		if order.DriverID == nil || *order.DriverID != requesterID {
			return nil, ErrOrderForbidden
		}
	case "merchant":
		if order.MerchantID != requesterID {
			return nil, ErrOrderForbidden
		}
	case "admin":
		// admin sees all
	default:
		return nil, ErrOrderForbidden
	}
	return order, nil
}

// ToResponse converts an Order to the API response shape, enriching with
// driver profile data when a driver is assigned.
func (s *OrderService) ToResponse(ctx context.Context, o *model.Order) dto.OrderResponse {
	resp := dto.OrderFromModel(o)
	if o.DriverID == nil {
		return resp
	}
	// Best-effort driver enrichment — never blocks the response.
	dp, err := s.orders.FindDriverProfile(ctx, *o.DriverID)
	if err != nil {
		return resp
	}
	u, err := s.users.FindByID(ctx, *o.DriverID)
	if err != nil {
		return resp
	}
	name := u.FirstName + " " + u.LastName
	vehicle := string(dp.VehicleType)
	resp.DriverName = &name
	resp.DriverRating = &dp.Rating
	resp.DriverVehicle = &vehicle
	if u.Phone != "" {
		resp.DriverPhone = &u.Phone
	}
	return resp
}

// Cancel handles cancellation with the refund engine.
func (s *OrderService) Cancel(ctx context.Context, orderID, actorID uuid.UUID, actorRole, reason string) error {
	err := s.orders.Transaction(ctx, func(tx *gorm.DB) error {
		order, err := s.orders.LockForUpdate(ctx, tx, orderID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrderNotFound
			}
			return err
		}
		allowed := model.ValidTransitions[order.Status]
		canCancel := false
		for _, s := range allowed {
			if s == model.OrderCancelled {
				canCancel = true
				break
			}
		}
		if !canCancel {
			return fmt.Errorf("%w: cannot cancel from %s", ErrIllegalTransition, order.Status)
		}

		from := order.Status
		order.Status = model.OrderCancelled
		order.CancelReason = &reason
		if err := s.orders.SaveTx(ctx, tx, order); err != nil {
			return err
		}

		if err := s.orders.CreateEventTx(ctx, tx, &model.OrderEvent{
			ID:         uuid.New(),
			OrderID:    orderID,
			FromStatus: from,
			ToStatus:   model.OrderCancelled,
			ActorID:    actorID,
			ActorRole:  actorRole,
			Note:       &reason,
		}); err != nil {
			return err
		}

		// Refund engine
		if err := s.ledger.ProcessCancellationRefund(ctx, tx, order); err != nil {
			return err
		}

		// Notify all parties via WS — customer tracking page and driver app
		// both subscribe to order:{id} and must know immediately.
		if s.hub != nil {
			_ = s.hub.Publish(ctx,
				"order:"+orderID.String(),
				"order_cancelled",
				map[string]interface{}{
					"orderId": orderID,
					"reason":  reason,
				},
			)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// WhatsApp cancellation notification fires AFTER the transaction commits.
	// Spawning inside the tx closure would send the message even on rollback
	// (e.g. if ProcessCancellationRefund fails), producing a phantom cancel
	// notification for an order that is still active.
	if s.wa != nil && s.users != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("whatsapp order_cancelled goroutine panic", "panic", r, "order_id", orderID)
				}
			}()
			// Re-fetch the order post-commit to get the final state.
			o, err := s.orders.FindByID(context.Background(), orderID)
			if err != nil {
				return
			}
			u, err := s.users.FindByID(context.Background(), o.CustomerID)
			if err != nil || u.Phone == "" {
				return
			}
			refund := fmt.Sprintf("₦%.0f", float64(o.TotalKobo)/100)
			if o.PaymentMethod == "pay_on_arrival" {
				refund = "no charge"
			}
			s.wa.OrderCancelled(whatsapp.NormalisePhone(u.Phone), reason, refund)
		}()
	}
	return nil
}

// OrderStopOut is the API-facing view of a stop. Recipient fields are only
// ever populated by GetStops when the caller is authorized to see them.
type OrderStopOut struct {
	ID                  uuid.UUID  `json:"id"`
	OrderID             uuid.UUID  `json:"orderId"`
	Sequence            int        `json:"sequence"`
	AddressID           uuid.UUID  `json:"addressId"`
	RecipientName       *string    `json:"recipientName,omitempty"`
	RecipientPhone      *string    `json:"recipientPhone,omitempty"`
	Notes               *string    `json:"notes,omitempty"`
	Status              string     `json:"status"`
	ConfirmedAt         *time.Time `json:"confirmedAt,omitempty"`
	EmptyCollected      bool       `json:"emptyCollected"`
	EmptyCylinderSerial *string    `json:"emptyCylinderSerial,omitempty"`
}

// ListForCustomer returns the customer's order history with optional filtering.
// Cursor is stable: uses (created_at, id) composite to prevent duplicates
// when two orders share the same timestamp.
func (s *OrderService) ListForCustomer(ctx context.Context, customerID uuid.UUID, vertical, status string, cursor *uuid.UUID, limit int) ([]model.Order, error) {
	return s.orders.ListByCustomerFiltered(ctx, customerID, vertical, status, cursor, limit)
}

// ReceiptResponse is the full invoice view of a completed order.
type ReceiptResponse struct {
	OrderID       string              `json:"orderId"`
	Vertical      string              `json:"vertical"`
	Status        string              `json:"status"`
	PaymentMethod string              `json:"paymentMethod"`
	Items         []ReceiptItemView   `json:"items"`
	SubtotalKobo  int64               `json:"subtotalKobo"`
	DeliveryKobo  int64               `json:"deliveryKobo"`
	ServiceKobo   int64               `json:"serviceKobo"`
	TipKobo       int64               `json:"tipKobo"`
	TotalKobo     int64               `json:"totalKobo"`
	MerchantName  string              `json:"merchantName"`
	DriverName    string              `json:"driverName,omitempty"`
	CreatedAt     string              `json:"createdAt"`
	DeliveredAt   *string             `json:"deliveredAt,omitempty"`
	Review        *model.OrderReview  `json:"review,omitempty"`
}

// ReceiptItemView is the camelCase-tagged projection of model.OrderItem for
// the receipt endpoint. model.OrderItem has no json tags (GORM convention),
// so embedding it directly serialises as PascalCase — breaking the frontend.
type ReceiptItemView struct {
	Name          string `json:"name"`
	Quantity      int    `json:"quantity"`
	UnitPriceKobo int64  `json:"unitPriceKobo"`
	TotalKobo     int64  `json:"totalKobo"`
}

// GetReceipt returns the full invoice for an order the caller owns.
func (s *OrderService) GetReceipt(ctx context.Context, orderID, customerID uuid.UUID) (*ReceiptResponse, error) {
	order, err := s.orders.FindByIDWithItems(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.CustomerID != customerID {
		return nil, ErrOrderForbidden
	}

	merchant, _ := s.orders.FindMerchant(ctx, order.MerchantID)
	merchantName := ""
	if merchant != nil {
		merchantName = merchant.BusinessName
	}

	driverName := ""
	if order.DriverID != nil {
		if u, err := s.users.FindByID(ctx, *order.DriverID); err == nil {
			driverName = u.FirstName + " " + u.LastName
		}
	}

	var reviewPtr *model.OrderReview
	if rev, err := s.orders.FindReviewByOrderAndReviewer(ctx, orderID, customerID); err == nil {
		reviewPtr = rev
	}

	var deliveredAt *string
	if order.DeliveredAt != nil {
		s := order.DeliveredAt.Format("2006-01-02T15:04:05Z07:00")
		deliveredAt = &s
	}

	return &ReceiptResponse{
		OrderID:       order.ID.String(),
		Vertical:      order.Vertical,
		Status:        string(order.Status),
		PaymentMethod: order.PaymentMethod,
		Items: func() []ReceiptItemView {
			views := make([]ReceiptItemView, len(order.Items))
			for i, it := range order.Items {
				views[i] = ReceiptItemView{
					Name:          it.Name,
					Quantity:      it.Quantity,
					UnitPriceKobo: it.UnitPriceKobo,
					TotalKobo:     it.TotalKobo,
				}
			}
			return views
		}(),
		SubtotalKobo:  order.SubtotalKobo,
		DeliveryKobo:  order.DeliveryKobo,
		ServiceKobo:   order.ServiceKobo,
		TipKobo:       order.TipKobo,
		TotalKobo:     order.TotalKobo,
		MerchantName:  merchantName,
		DriverName:    driverName,
		CreatedAt:     order.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		DeliveredAt:   deliveredAt,
		Review:        reviewPtr,
	}, nil
}

// SubmitReview saves a post-delivery rating + comment.
// Only the order's customer may review, only after delivery, and only about
// the driver/merchant who actually fulfilled that order — revieweeID is
// derived from the order, never trusted from the request.
func (s *OrderService) SubmitReview(ctx context.Context, orderID, customerID uuid.UUID, revieweeType string, rating int, comment *string) error {
	if rating < 1 || rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}
	if revieweeType != "driver" && revieweeType != "merchant" {
		return fmt.Errorf("revieweeType must be driver or merchant")
	}

	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("order not found")
	}
	if order.CustomerID != customerID {
		return ErrOrderForbidden
	}
	if order.Status != model.OrderDelivered {
		return fmt.Errorf("can only review a delivered order")
	}

	var revieweeID uuid.UUID
	if revieweeType == "driver" {
		if order.DriverID == nil {
			return fmt.Errorf("order has no assigned driver")
		}
		revieweeID = *order.DriverID
	} else {
		if order.MerchantID == uuid.Nil {
			return fmt.Errorf("order has no merchant")
		}
		revieweeID = order.MerchantID
	}

	review := &model.OrderReview{
		ID:           uuid.New(),
		OrderID:      orderID,
		ReviewerID:   customerID,
		RevieweeID:   revieweeID,
		RevieweeType: revieweeType,
		Rating:       rating,
		Comment:      comment,
	}
	if err := s.orders.CreateReview(ctx, review); err != nil {
		return fmt.Errorf("review already submitted")
	}

	// Rating recomputation + badge award run in the worker: retried on failure,
	// observable, and a panic there cannot take down the API process.
	// The review itself is already durable, so an enqueue failure is logged,
	// not returned — the customer's review must not appear to have failed.
	if s.enqueueReview != nil {
		if err := s.enqueueReview(revieweeID.String(), revieweeType); err != nil {
			observability.CaptureError(ctx, err, "SubmitReview: enqueue aggregate failed",
				"reviewee_id", revieweeID.String(), "reviewee_type", revieweeType)
		}
	}
	return nil
}

// UpdateAggregateRating recalculates and persists the reviewee's average
// rating. Called from the worker so failures retry and surface.
func (s *OrderService) UpdateAggregateRating(ctx context.Context, revieweeID uuid.UUID, revieweeType string) error {
	avg, err := s.orders.AverageRating(ctx, revieweeID, revieweeType)
	if err != nil {
		return fmt.Errorf("aggregate rating: %w", err)
	}
	if revieweeType == "driver" {
		return s.orders.UpdateDriverRating(ctx, revieweeID, avg)
	}
	return s.orders.UpdateMerchantRating(ctx, revieweeID, avg)
}

// Thresholds for the top_rated badge.
const (
	topRatedMinReviews = 20
	topRatedMinAvg     = 4.8
)

// AwardBadgeIfEligible checks delivery milestones and awards badges.
var badgeMilestones = []struct {
	Count int
	Type  string
}{
	{1, "first_delivery"},
	{10, "10_deliveries"},
	{50, "50_deliveries"},
	{100, "100_deliveries"},
}

func (s *OrderService) AwardBadgeIfEligible(ctx context.Context, driverID uuid.UUID) error {
	count, err := s.orders.CountDeliveredByDriver(ctx, driverID)
	if err != nil {
		return fmt.Errorf("badge: count deliveries: %w", err)
	}

	for _, m := range badgeMilestones {
		if int(count) >= m.Count {
			badge := &model.DriverBadge{
				ID:                uuid.New(),
				DriverID:          driverID,
				BadgeType:         m.Type,
				OrderCountAtAward: int(count),
				AwardedAt:         time.Now(),
			}
			if err := s.orders.UpsertBadge(ctx, badge); err != nil {
				return fmt.Errorf("badge %s: %w", m.Type, err)
			}
		}
	}

	// top_rated: avg rating >= 4.8 with >= 20 reviews
	reviewCount, err := s.orders.CountReviews(ctx, driverID, "driver")
	if err != nil {
		return fmt.Errorf("badge: count reviews: %w", err)
	}
	avgRating, err := s.orders.AverageRating(ctx, driverID, "driver")
	if err != nil {
		return fmt.Errorf("badge: avg rating: %w", err)
	}
	if reviewCount >= topRatedMinReviews && avgRating >= topRatedMinAvg {
		badge := &model.DriverBadge{ID: uuid.New(), DriverID: driverID, BadgeType: "top_rated", OrderCountAtAward: int(count), AwardedAt: time.Now()}
		if err := s.orders.UpsertBadge(ctx, badge); err != nil {
			return fmt.Errorf("badge top_rated: %w", err)
		}
	}
	return nil
}

// GetDriverBadges returns all badges for a driver.
func (s *OrderService) GetDriverBadges(ctx context.Context, driverID uuid.UUID) ([]model.DriverBadge, error) {
	return s.orders.FindDriverBadges(ctx, driverID)
}

// ListForMerchant returns the merchant's order queue, newest first, optionally
// filtered by status (e.g. "pending" for the incoming-orders tab). merchantID
// is the resolved model.Merchant.ID (business entity), not the login user ID.
// Uses (created_at, id) composite cursor — consistent with ListForCustomer.
func (s *OrderService) ListForMerchant(ctx context.Context, merchantID uuid.UUID, status string, cursor *uuid.UUID, limit int) ([]model.Order, error) {
	return s.orders.ListByMerchantFiltered(ctx, merchantID, status, cursor, limit)
}

// GetStops returns all stops for a multi-drop package order, ordered by
// sequence, with recipient PII decrypted only for an authorized caller:
//   - the order's own customer, or an admin: every stop's recipient is visible
//     (it's the sender's own data / needed for dispute resolution)
//   - the assigned driver, while the order is in_transit: only the next
//     unconfirmed stop's recipient is visible — never past or future stops,
//     never after delivery. This is the "only for the active stop" rule.
//   - anyone else: recipient fields are omitted entirely.
func (s *OrderService) GetStops(ctx context.Context, orderID, requesterID uuid.UUID, requesterRole string) ([]OrderStopOut, error) {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	stops, err := s.orders.ListStops(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// The active stop for a driver: lowest-sequence stop not yet confirmed/skipped.
	var activeStopID uuid.UUID
	for _, st := range stops {
		if st.Status != "confirmed" && st.Status != "skipped" {
			activeStopID = st.ID
			break
		}
	}

	isOwnerOrAdmin := requesterRole == "admin" || (requesterRole == "customer" && order.CustomerID == requesterID)
	isAssignedDriver := requesterRole == "driver" && order.DriverID != nil && *order.DriverID == requesterID

	out := make([]OrderStopOut, 0, len(stops))
	for _, st := range stops {
		view := OrderStopOut{
			ID:                  st.ID,
			OrderID:             st.OrderID,
			Sequence:            st.Sequence,
			AddressID:           st.AddressID,
			Notes:               st.Notes,
			Status:              st.Status,
			ConfirmedAt:         st.ConfirmedAt,
			EmptyCollected:      st.EmptyCollected,
			EmptyCylinderSerial: st.EmptyCylinderSerial,
		}

		canSeeRecipient := isOwnerOrAdmin ||
			(isAssignedDriver && order.Status == model.OrderInTransit && st.ID == activeStopID)

		if canSeeRecipient && s.recipients != nil {
			if name, err := s.recipients.DecryptPtr(st.RecipientNameEnc); err == nil {
				view.RecipientName = name
			}
			if phone, err := s.recipients.DecryptPtr(st.RecipientPhoneEnc); err == nil {
				view.RecipientPhone = phone
			}
		}

		out = append(out, view)
	}
	return out, nil
}

// ConfirmStopInput carries the delivery code plus optional empty-cylinder
// collection data for gas refill orders.
type ConfirmStopInput struct {
	Code                string
	EmptyCollected      bool
	EmptyCylinderSerial *string
	CapturedLat         *float64
	CapturedLng         *float64
}

// ConfirmStop marks a single stop as confirmed using the per-stop delivery code.
// When all stops are confirmed the order transitions to delivered and escrow settles.
func (s *OrderService) ConfirmStop(ctx context.Context, orderID, driverID uuid.UUID, sequence int, in ConfirmStopInput) error {
	return s.orders.Transaction(ctx, func(tx *gorm.DB) error {
		order, err := s.orders.LockForUpdate(ctx, tx, orderID)
		if err != nil {
			return fmt.Errorf("order not found")
		}
		// Preload stops via tx so they're in the same transaction snapshot.
		stops, err := s.orders.ListStops(ctx, orderID)
		if err != nil {
			return err
		}
		order.Stops = stops

		if order.DriverID == nil || *order.DriverID != driverID {
			return fmt.Errorf("not the assigned driver")
		}
		if order.Status != model.OrderInTransit {
			return fmt.Errorf("order not in transit")
		}

		var stop *model.OrderStop
		for i := range order.Stops {
			if order.Stops[i].Sequence == sequence {
				stop = &order.Stops[i]
				break
			}
		}
		if stop == nil {
			return fmt.Errorf("stop %d not found", sequence)
		}
		if stop.Status == "confirmed" {
			return fmt.Errorf("stop already confirmed")
		}

		// Verify delivery code
		if s.deliveryCodes != nil {
			if err := s.deliveryCodes.Verify(ctx, tx, orderID, in.Code); err != nil {
				return err
			}
		}

		now := time.Now()
		stop.Status = "confirmed"
		stop.ConfirmedAt = &now

		// Gas reverse logistics: record empty cylinder collection.
		if order.Vertical == "gas" && in.EmptyCollected {
			stop.EmptyCollected = true
			stop.EmptyCylinderSerial = in.EmptyCylinderSerial
			stopID := stop.ID
			custodyEvent := &model.CylinderCustodyEvent{
				ID:          uuid.New(),
				OrderID:     orderID,
				StopID:      &stopID,
				Serial:      in.EmptyCylinderSerial,
				EventType:   "collected",
				ActorID:     driverID,
				CapturedLat: in.CapturedLat,
				CapturedLng: in.CapturedLng,
				OccurredAt:  now,
			}
			if err := s.orders.CreateCustodyEventTx(ctx, tx, custodyEvent); err != nil {
				return fmt.Errorf("custody event: %w", err)
			}
		}

		if err := s.orders.SaveStopTx(ctx, tx, stop); err != nil {
			return err
		}

		// Check if all stops are confirmed → settle and deliver
		allDone := true
		for _, st := range order.Stops {
			if st.ID == stop.ID {
				continue
			}
			if st.Status != "confirmed" {
				allDone = false
				break
			}
		}

		if allDone {
			if err := s.ledger.Settle(ctx, tx, order, uuid.New()); err != nil {
				return fmt.Errorf("settlement: %w", err)
			}
			order.Status = model.OrderDelivered
			order.DeliveredAt = &now
			if err := s.orders.SaveTx(ctx, tx, order); err != nil {
				return err
			}
			return s.orders.CreateEventTx(ctx, tx, &model.OrderEvent{
				ID:         uuid.New(),
				OrderID:    orderID,
				FromStatus: model.OrderInTransit,
				ToStatus:   model.OrderDelivered,
				ActorID:    driverID,
				ActorRole:  "driver",
			})
		}
		return nil
	})
}

// RaiseDispute is the customer-facing entry point for a post-delivery complaint.
// It freezes the escrow hold and sets the 72h SLA deadline the worker already
// monitors. The admin team resolves via FreezeEscrow/ReleaseEscrow.
// Only the order's customer may raise a dispute, and only on a delivered order
// whose escrow has not already been frozen or released.
func (s *OrderService) RaiseDispute(ctx context.Context, orderID, customerID uuid.UUID, reason string) error {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return ErrOrderNotFound
	}
	if order.CustomerID != customerID {
		return ErrOrderForbidden
	}
	if order.Status != model.OrderDelivered {
		return fmt.Errorf("disputes can only be raised on delivered orders")
	}
	return s.orders.Transaction(ctx, func(tx *gorm.DB) error {
		return s.ledger.FreezeEscrowForCustomer(ctx, tx, order, reason)
	})
}

// GetDisputeStatus returns the current escrow hold status for an order the
// caller owns, so the customer can see whether their dispute is pending review.
func (s *OrderService) GetDisputeStatus(ctx context.Context, orderID, customerID uuid.UUID) (string, error) {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return "", ErrOrderNotFound
	}
	if order.CustomerID != customerID {
		return "", ErrOrderForbidden
	}
	status, err := s.ledger.GetEscrowStatus(ctx, orderID)
	if err != nil {
		return "", fmt.Errorf("no escrow record found for order")
	}
	return string(status), nil
}
type RecertReminderResult struct {
	CylinderID      uuid.UUID
	UserID          uuid.UUID
	Serial          string
	LastRecert      time.Time
	DaysUntilExpiry int
}

// recertPeriodDays is the standard LPG cylinder recertification interval.
// Nigeria's DPR standard is 5 years (1825 days); we warn at 60 days out.
const (
	recertPeriodDays  = 1825 // 5 years
	recertWarningDays = 60
)

// RecertReminders returns cylinders whose recertification is due within
// recertWarningDays. Called nightly by the worker to drive push notifications.
func (s *OrderService) RecertReminders(ctx context.Context) ([]RecertReminderResult, error) {
	cutoff := time.Now().AddDate(0, 0, recertWarningDays)
	rows, err := s.orders.FindCylindersNearRecert(ctx, cutoff, recertPeriodDays)
	if err != nil {
		return nil, fmt.Errorf("recert reminders: %w", err)
	}
	out := make([]RecertReminderResult, 0, len(rows))
	for _, r := range rows {
		if r.LastRecertAt == nil {
			continue
		}
		expiryDate := r.LastRecertAt.AddDate(0, 0, recertPeriodDays)
		daysLeft := int(time.Until(expiryDate).Hours() / 24)
		out = append(out, RecertReminderResult{
			CylinderID:      r.ID,
			UserID:          r.UserID,
			Serial:          r.Serial,
			LastRecert:      *r.LastRecertAt,
			DaysUntilExpiry: daysLeft,
		})
	}
	return out, nil
}

// minFillSamplesForJudgment is the minimum number of recent verified fills
// before a merchant's fill_status can move off 'good'. A merchant with 1-2
// fills shouldn't be flagged on noise — this is a floor on sample size, not
// a grace period.
const minFillSamplesForJudgment = 5

// fillStatusFor derives a merchant's remediation state from their rolling
// fill-accuracy average. Thresholds are deliberately staged so a merchant
// gets warned before they're hidden, and hidden before they're blocked —
// nobody goes from "good" to "delisted" on one nightly run.
func fillStatusFor(avgAccuracy float64, sampleCount int) string {
	if sampleCount < minFillSamplesForJudgment {
		return "good"
	}
	switch {
	case avgAccuracy >= 0.98:
		return "good"
	case avgAccuracy >= 0.95:
		return "warned"
	case avgAccuracy >= 0.90:
		return "probation"
	default:
		return "delisted"
	}
}

// RecomputeFillAccuracy recalculates fill_accuracy_pct, fill_sample_count and
// fill_status for all gas merchants from a rolling window of their most
// recent weight_photo proof rows. Called nightly.
func (s *OrderService) RecomputeFillAccuracy(ctx context.Context) error {
	rows, err := s.orders.GasFillAccuracyStats(ctx)
	if err != nil {
		return fmt.Errorf("recompute fill accuracy: %w", err)
	}
	for _, r := range rows {
		status := fillStatusFor(r.AvgAccuracy, r.SampleCount)
		if err := s.orders.UpdateMerchantFillAccuracy(ctx, r.MerchantID, r.AvgAccuracy, r.SampleCount, status); err != nil {
			return fmt.Errorf("recompute fill accuracy: update merchant %s: %w", r.MerchantID, err)
		}
	}
	return nil
}

// recipientPIIRetentionDays: recipient name/phone are third-party PII the
// recipient never consented to us storing — the sender attests consent
// per-order, not the recipient. NDPR data-minimization: purge once the
// order is settled and no dispute is open.
const recipientPIIRetentionDays = 30

// PurgeStaleRecipientPII nulls recipient_*_enc on delivered orders older than
// the retention window, skipping any order whose escrow hold is frozen (an
// open dispute may still need the recipient's identity). Also purges the
// same fields on that order's stops. Returns the number of orders purged.
func (s *OrderService) PurgeStaleRecipientPII(ctx context.Context) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -recipientPIIRetentionDays)

	orderIDs, err := s.orders.FindStaleRecipientPIIOrders(ctx, cutoff, model.EscrowFrozen)
	if err != nil {
		return 0, fmt.Errorf("purge: find stale orders: %w", err)
	}
	if len(orderIDs) == 0 {
		return 0, nil
	}

	err = s.orders.Transaction(ctx, func(tx *gorm.DB) error {
		return s.orders.PurgeRecipientPIITx(ctx, tx, orderIDs)
	})
	if err != nil {
		return 0, err
	}
	return int64(len(orderIDs)), nil
}

// IsOrderParticipant returns true when userID is the customer, assigned driver,
// or merchant of the given order. Used by the WS hub to gate order channel subscriptions.
// Order.MerchantID carries the merchant's User.ID (same key space as CustomerID/DriverID).
func (s *OrderService) IsOrderParticipant(ctx context.Context, orderID uuid.UUID, userID string) bool {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return false
	}
	if order.CustomerID.String() == userID {
		return true
	}
	if order.DriverID != nil && order.DriverID.String() == userID {
		return true
	}
	if order.MerchantID.String() == userID {
		return true
	}
	return false
}
