package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

// AdminService handles all admin-only business logic.
// Every money-moving operation routes through LedgerService — never direct DB writes.
type AdminService struct {
	repo   repo.AdminRepo
	ledger *LedgerService
}

func NewAdminService(r repo.AdminRepo, ledger *LedgerService) *AdminService {
	return &AdminService{repo: r, ledger: ledger}
}

// ── Merchants ─────────────────────────────────────────────────────────────────

type MerchantRow struct {
	ID           uuid.UUID            `json:"id"`
	UserID       uuid.UUID            `json:"userId"`
	BusinessName string               `json:"businessName"`
	Vertical     model.MerchantVertical `json:"vertical"`
	Status       model.MerchantStatus `json:"status"`
	Rating       float64              `json:"rating"`
	CreatedAt    time.Time            `json:"createdAt"`
}

func (s *AdminService) ListMerchants(ctx context.Context, status string, cursor *uuid.UUID, limit int) ([]MerchantRow, error) {
	profiles, err := s.repo.ListMerchantProfiles(ctx, status, cursor, limit)
	if err != nil {
		return nil, err
	}
	rows := make([]MerchantRow, len(profiles))
	for i, p := range profiles {
		rows[i] = MerchantRow{
			ID:           p.ID,
			UserID:       p.UserID,
			BusinessName: p.BusinessName,
			Vertical:     p.Vertical,
			Status:       p.Status,
			Rating:       p.Rating,
			CreatedAt:    p.CreatedAt,
		}
	}
	return rows, nil
}

func (s *AdminService) SetMerchantStatus(ctx context.Context, merchantID, adminID uuid.UUID, status, reason string) error {
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		mp, err := s.repo.LockMerchantProfileTx(ctx, tx, merchantID)
		if err != nil {
			return fmt.Errorf("merchant not found")
		}
		mp.Status = model.MerchantStatus(status)
		if err := s.repo.SaveMerchantProfileTx(ctx, tx, mp); err != nil {
			return err
		}
		return s.repo.CreateAuditLogTx(ctx, tx, &model.AdminAuditLog{
			ID:         uuid.New(),
			AdminID:    adminID,
			Action:     "merchant_status_change",
			TargetType: "merchant_profile",
			TargetID:   merchantID,
			Reason:     reason,
			CreatedAt:  time.Now(),
		})
	})
}

// ── Drivers ───────────────────────────────────────────────────────────────────

type DriverRow struct {
	ID              uuid.UUID          `json:"id"`
	UserID          uuid.UUID          `json:"userId"`
	Status          model.DriverStatus `json:"status"`
	VehicleType     model.VehicleType  `json:"vehicleType"`
	VehiclePlate    string             `json:"vehiclePlate"`
	Rating          float64            `json:"rating"`
	TotalDeliveries int                `json:"totalDeliveries"`
	CreatedAt       time.Time          `json:"createdAt"`
}

func (s *AdminService) ListDrivers(ctx context.Context, status string, cursor *uuid.UUID, limit int) ([]DriverRow, error) {
	profiles, err := s.repo.ListDriverProfiles(ctx, status, cursor, limit)
	if err != nil {
		return nil, err
	}
	rows := make([]DriverRow, len(profiles))
	for i, p := range profiles {
		rows[i] = DriverRow{
			ID:              p.ID,
			UserID:          p.UserID,
			Status:          p.Status,
			VehicleType:     p.VehicleType,
			VehiclePlate:    p.VehiclePlate,
			Rating:          p.Rating,
			TotalDeliveries: p.TotalDeliveries,
			CreatedAt:       p.CreatedAt,
		}
	}
	return rows, nil
}

func (s *AdminService) SetDriverStatus(ctx context.Context, driverID, adminID uuid.UUID, status, reason string) error {
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		dp, err := s.repo.LockDriverProfileTx(ctx, tx, driverID)
		if err != nil {
			return fmt.Errorf("driver not found")
		}
		dp.Status = model.DriverStatus(status)
		if err := s.repo.SaveDriverProfileTx(ctx, tx, dp); err != nil {
			return err
		}
		return s.repo.CreateAuditLogTx(ctx, tx, &model.AdminAuditLog{
			ID:         uuid.New(),
			AdminID:    adminID,
			Action:     "driver_status_change",
			TargetType: "driver_profile",
			TargetID:   driverID,
			Reason:     reason,
			CreatedAt:  time.Now(),
		})
	})
}

// SetUserActive sets users.is_active and immediately revokes all refresh tokens
// so no new access tokens can be issued. Existing access tokens expire naturally
// within JWTAccessTTLMin; the Auth middleware enforces is_active on every request.
func (s *AdminService) SetUserActive(ctx context.Context, userID, adminID uuid.UUID, active bool, reason string) error {
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		u, err := s.repo.LockUserTx(ctx, tx, userID)
		if err != nil {
			return fmt.Errorf("user not found")
		}
		u.IsActive = active
		if err := s.repo.SaveUserTx(ctx, tx, u); err != nil {
			return err
		}
		if !active {
			if err := s.repo.RevokeAllUserRefreshTokensTx(ctx, tx, userID); err != nil {
				return err
			}
		}
		action := "user_activated"
		if !active {
			action = "user_deactivated"
		}
		return s.repo.CreateAuditLogTx(ctx, tx, &model.AdminAuditLog{
			ID:         uuid.New(),
			AdminID:    adminID,
			Action:     action,
			TargetType: "user",
			TargetID:   userID,
			Reason:     reason,
			CreatedAt:  time.Now(),
		})
	})
}

// ── Admin console listings ────────────────────────────────────────────────────

type UserRow struct {
	ID         uuid.UUID `json:"id"`
	Role       string    `json:"role"`
	FirstName  string    `json:"firstName"`
	LastName   string    `json:"lastName"`
	Phone      string    `json:"phone"`
	Email      *string   `json:"email,omitempty"`
	IsVerified bool      `json:"isVerified"`
	IsActive   bool      `json:"isActive"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (s *AdminService) ListUsers(ctx context.Context, role, q string, cursor *uuid.UUID, limit int) ([]UserRow, error) {
	users, err := s.repo.ListUsers(ctx, role, q, cursor, limit)
	if err != nil {
		return nil, err
	}
	rows := make([]UserRow, len(users))
	for i, u := range users {
		rows[i] = UserRow{
			ID: u.ID, Role: string(u.Role), FirstName: u.FirstName, LastName: u.LastName,
			Phone: u.Phone, Email: u.Email, IsVerified: u.IsVerified, IsActive: u.IsActive,
			CreatedAt: u.CreatedAt,
		}
	}
	return rows, nil
}

type RunRow struct {
	ID              uuid.UUID  `json:"id"`
	ZoneID          uuid.UUID  `json:"zoneId"`
	DriverID        *uuid.UUID `json:"driverId,omitempty"`
	WindowStart     time.Time  `json:"windowStart"`
	WindowEnd       time.Time  `json:"windowEnd"`
	Status          string     `json:"status"`
	TotalDistanceKm float64    `json:"totalDistanceKm"`
	OrderCount      int        `json:"orderCount"`
}

func (s *AdminService) ListRuns(ctx context.Context, status string, cursor *uuid.UUID, limit int) ([]RunRow, error) {
	runs, err := s.repo.ListRuns(ctx, status, cursor, limit)
	if err != nil {
		return nil, err
	}
	rows := make([]RunRow, len(runs))
	for i, r := range runs {
		rows[i] = RunRow{
			ID: r.ID, ZoneID: r.ZoneID, DriverID: r.DriverID,
			WindowStart: r.WindowStart, WindowEnd: r.WindowEnd, Status: r.Status,
			TotalDistanceKm: r.TotalDistanceKm, OrderCount: len(r.Orders),
		}
	}
	return rows, nil
}

type SubscriptionRow struct {
	ID         uuid.UUID  `json:"id"`
	CustomerID uuid.UUID  `json:"customerId"`
	MerchantID uuid.UUID  `json:"merchantId"`
	Vertical   string     `json:"vertical"`
	Frequency  string     `json:"frequency"` // model field is Cadence
	Status     string     `json:"status"`
	NextRunAt  *time.Time `json:"nextRunAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func (s *AdminService) ListSubscriptions(ctx context.Context, status string, cursor *uuid.UUID, limit int) ([]SubscriptionRow, error) {
	subs, err := s.repo.ListSubscriptions(ctx, status, cursor, limit)
	if err != nil {
		return nil, err
	}
	rows := make([]SubscriptionRow, len(subs))
	for i, sub := range subs {
		// NextChargeAt is a non-pointer zero value when never scheduled. Surface
		// that as null rather than year 1, which the console would otherwise
		// render as a real — and alarming — overdue date.
		var nextRun *time.Time
		if !sub.NextChargeAt.IsZero() {
			t := sub.NextChargeAt
			nextRun = &t
		}
		rows[i] = SubscriptionRow{
			ID: sub.ID, CustomerID: sub.CustomerID, MerchantID: sub.MerchantID,
			Vertical: sub.Vertical, Frequency: sub.Cadence, Status: sub.Status,
			NextRunAt: nextRun, CreatedAt: sub.CreatedAt,
		}
	}
	return rows, nil
}

type PrescriptionRow struct {
	ID         uuid.UUID  `json:"id"`
	CustomerID uuid.UUID  `json:"customerId"`
	MerchantID uuid.UUID  `json:"merchantId"`
	Status     string     `json:"status"`
	ReviewNote *string    `json:"reviewNote,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func (s *AdminService) ListPrescriptions(ctx context.Context, status string, cursor *uuid.UUID, limit int) ([]PrescriptionRow, error) {
	rx, err := s.repo.ListPrescriptions(ctx, status, cursor, limit)
	if err != nil {
		return nil, err
	}
	rows := make([]PrescriptionRow, len(rx))
	for i, p := range rx {
		var merchantID uuid.UUID
		if p.MerchantID != nil {
			merchantID = *p.MerchantID
		}
		rows[i] = PrescriptionRow{
			ID: p.ID, CustomerID: p.CustomerID, MerchantID: merchantID,
			Status: p.Status, ReviewNote: p.ReviewNote, ExpiresAt: p.ExpiresAt,
			CreatedAt: p.CreatedAt,
		}
	}
	return rows, nil
}

// ── Orders ────────────────────────────────────────────────────────────────────

type OrderSummary struct {
	ID         uuid.UUID          `json:"id"`
	CustomerID uuid.UUID          `json:"customerId"`
	MerchantID uuid.UUID          `json:"merchantId"`
	DriverID   *uuid.UUID         `json:"driverId,omitempty"`
	Vertical   string             `json:"vertical"`
	Status     model.OrderStatus  `json:"status"`
	TotalKobo  int64              `json:"totalKobo"`
	CreatedAt  time.Time          `json:"createdAt"`
}

type OrderDetail struct {
	OrderSummary
	Items  []model.OrderItem  `json:"items"`
	Events []model.OrderEvent `json:"events"`
}

func (s *AdminService) SearchOrders(ctx context.Context, q, status string, cursor *uuid.UUID, limit int) ([]OrderSummary, error) {
	orders, err := s.repo.SearchOrders(ctx, q, status, cursor, limit)
	if err != nil {
		return nil, err
	}
	rows := make([]OrderSummary, len(orders))
	for i, o := range orders {
		rows[i] = OrderSummary{
			ID:         o.ID,
			CustomerID: o.CustomerID,
			MerchantID: o.MerchantID,
			DriverID:   o.DriverID,
			Vertical:   o.Vertical,
			Status:     o.Status,
			TotalKobo:  o.TotalKobo,
			CreatedAt:  o.CreatedAt,
		}
	}
	return rows, nil
}

func (s *AdminService) GetOrderDetail(ctx context.Context, orderID uuid.UUID) (*OrderDetail, error) {
	order, err := s.repo.FindOrderWithEvents(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return &OrderDetail{
		OrderSummary: OrderSummary{
			ID:         order.ID,
			CustomerID: order.CustomerID,
			MerchantID: order.MerchantID,
			DriverID:   order.DriverID,
			Vertical:   order.Vertical,
			Status:     order.Status,
			TotalKobo:  order.TotalKobo,
			CreatedAt:  order.CreatedAt,
		},
		Items:  order.Items,
		Events: order.Events,
	}, nil
}

// ── Escrow dispute: freeze ────────────────────────────────────────────────────

const (
	// disputeSLAHours: frozen escrows older than this trigger an alert.
	disputeSLAHours = 72
	// disputeSingleAdminThresholdKobo: below this, one admin can release.
	// Above this, dual-admin approval is required (Tier-3).
	disputeSingleAdminThresholdKobo int64 = 5_000_000 // ₦50,000
)

// FreezeEscrow transitions EscrowHeld → EscrowFrozen.
// Runs tier-1 auto-adjudication first:
//   - Order delivered with no proof media → auto full-refund (no manual review needed)
//   - Gas order with weight shortfall beyond tolerance → auto partial-refund
//   - Otherwise → freeze for manual review, set 72h SLA deadline
//
// When freezing for manual review, funds move escrow → disputed_escrow so the
// dispute is visible as a distinct line in the ledger.
func (s *AdminService) FreezeEscrow(ctx context.Context, orderID, adminID uuid.UUID, reason string) error {
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		hold, err := s.ledger.repo.LockEscrowHold(ctx, tx, orderID, model.EscrowHeld)
		if err != nil {
			return fmt.Errorf("no held escrow found for order")
		}

		order, err := s.repo.FindOrderTx(ctx, tx, orderID)
		if err != nil {
			return fmt.Errorf("order not found")
		}

		// ── Tier-1 auto-adjudication ──────────────────────────────────────────

		// Check proof media existence.
		//
		// The error MUST be checked. A failed count leaves proofCount at 0,
		// which is indistinguishable from "this delivery has no proof at all" —
		// and that verdict triggers an automatic FULL REFUND below. A transient
		// database blip would silently refund a legitimately delivered order,
		// taking money from the merchant with no human in the loop.
		// Fail closed: no auto-adjudication unless we can actually read the
		// evidence. (Same reasoning as LedgerService.weightProof.)
		var proofCount int64
		if err := tx.WithContext(ctx).Model(&model.ProofMedia{}).
			Where("order_id = ?", orderID).Count(&proofCount).Error; err != nil {
			return fmt.Errorf("auto-adjudication: count proof media for order %s: %w", orderID, err)
		}

		if order.Status == model.OrderDelivered && proofCount == 0 {
			// No proof of delivery at all → auto full-refund.
			if err := s.ledger.fullRefundFrozen(ctx, tx, order, hold); err != nil {
				return fmt.Errorf("auto-adjudication full refund: %w", err)
			}
			hold.AutoAdjudicated = true
			if err := s.ledger.repo.SaveEscrowHold(ctx, tx, hold); err != nil {
				return err
			}
			return s.repo.CreateAuditLogTx(ctx, tx, &model.AdminAuditLog{
				ID:         uuid.New(),
				AdminID:    adminID,
				Action:     "escrow_auto_refund",
				TargetType: "escrow_hold",
				TargetID:   hold.ID,
				Reason:     "auto-adjudicated: no proof media — " + reason,
				CreatedAt:  time.Now(),
			})
		}

		if order.Vertical == "gas" && order.Status == model.OrderDelivered {
			measuredKg, _ := s.ledger.weightProof(ctx, tx, orderID)
			orderedKg, _ := s.ledger.orderedWeightKg(ctx, tx, orderID)
			if measuredKg > 0 && orderedKg > 0 {
				shortfall := orderedKg - measuredKg
				if shortfall/orderedKg > shortfallTolerance {
					// Partial refund for shortfall — reuse existing Settle shortfall math.
					refundKobo, calcErr := s.ledger.shortfallRefundKobo(ctx, tx, order, measuredKg, orderedKg)
					if calcErr == nil && refundKobo > 0 {
						if err := s.ledger.partialRefundFrozen(ctx, tx, order, hold, refundKobo); err != nil {
							return fmt.Errorf("auto-adjudication partial refund: %w", err)
						}
						hold.AutoAdjudicated = true
						hold.PartialRefundKobo = &refundKobo
						if err := s.ledger.repo.SaveEscrowHold(ctx, tx, hold); err != nil {
							return err
						}
						return s.repo.CreateAuditLogTx(ctx, tx, &model.AdminAuditLog{
							ID:         uuid.New(),
							AdminID:    adminID,
							Action:     "escrow_auto_partial_refund",
							TargetType: "escrow_hold",
							TargetID:   hold.ID,
							Reason:     fmt.Sprintf("auto-adjudicated: gas shortfall %.2fkg of %.2fkg — %s", shortfall, orderedKg, reason),
							CreatedAt:  time.Now(),
						})
					}
				}
			}
		}

		// ── Manual review path ────────────────────────────────────────────────

		// Move funds escrow → disputed_escrow so the dispute is a distinct
		// ledger line. The escrow account balance decreases; disputed_escrow increases.
		escrowAcct, err := s.ledger.platformAccount(ctx, tx, model.AccountEscrow)
		if err != nil {
			return fmt.Errorf("escrow account: %w", err)
		}
		disputedAcct, err := s.ledger.platformAccount(ctx, tx, model.AccountDisputedEscrow)
		if err != nil {
			return fmt.Errorf("disputed escrow account: %w", err)
		}
		journalID := uuid.New()
		entries := []model.LedgerEntry{
			{ID: uuid.New(), JournalID: journalID, AccountID: escrowAcct.ID, AmountKobo: -hold.AmountKobo, Description: "dispute freeze: debit escrow", RefType: "dispute", RefID: &orderID},
			{ID: uuid.New(), JournalID: journalID, AccountID: disputedAcct.ID, AmountKobo: hold.AmountKobo, Description: "dispute freeze: credit disputed_escrow", RefType: "dispute", RefID: &orderID},
		}
		if err := s.ledger.journal(ctx, tx, entries); err != nil {
			return err
		}
		if err := s.ledger.adjustBalance(ctx, tx, escrowAcct.ID, -hold.AmountKobo); err != nil {
			return err
		}
		if err := s.ledger.adjustBalance(ctx, tx, disputedAcct.ID, hold.AmountKobo); err != nil {
			return err
		}

		now := time.Now()
		slaDeadline := now.Add(disputeSLAHours * time.Hour)
		hold.Status = model.EscrowFrozen
		hold.FrozenAt = &now
		hold.FrozenReason = &reason
		hold.FrozenSLADeadline = &slaDeadline
		// AccountID now points to disputed_escrow so ReleaseEscrow debits the right account.
		hold.AccountID = disputedAcct.ID
		if err := s.ledger.repo.SaveEscrowHold(ctx, tx, hold); err != nil {
			return err
		}
		return s.repo.CreateAuditLogTx(ctx, tx, &model.AdminAuditLog{
			ID:         uuid.New(),
			AdminID:    adminID,
			Action:     "escrow_freeze",
			TargetType: "escrow_hold",
			TargetID:   hold.ID,
			Reason:     reason,
			CreatedAt:  now,
		})
	})
}

// ReleaseEscrow resolves a frozen escrow dispute.
// Tier-2 (below disputeSingleAdminThresholdKobo): single admin can release.
// Tier-3 (above threshold): dual-admin approval required.
// recipient: "customer" = full refund; "merchant" = settle to merchant.
func (s *AdminService) ReleaseEscrow(ctx context.Context, orderID, adminID uuid.UUID, recipient, reason string) (string, error) {
	var result string
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		hold, err := s.ledger.repo.LockEscrowHold(ctx, tx, orderID, model.EscrowFrozen)
		if err != nil {
			return fmt.Errorf("no frozen escrow found for order")
		}

		// Tier-2: single admin sufficient for low-value disputes.
		singleAdminSufficient := hold.AmountKobo <= disputeSingleAdminThresholdKobo

		if !singleAdminSufficient {
			// Tier-3: dual-admin required.
			if hold.ApprovalOne == nil {
				hold.ApprovalOne = &adminID
				if err := s.ledger.repo.SaveEscrowHold(ctx, tx, hold); err != nil {
					return err
				}
				result = "approval recorded — awaiting second admin approval"
				return nil
			}
			if *hold.ApprovalOne == adminID {
				return fmt.Errorf("same admin cannot provide both approvals")
			}
			hold.ApprovalTwo = &adminID
		}

		order, err := s.repo.FindOrderTx(ctx, tx, orderID)
		if err != nil {
			return fmt.Errorf("order not found")
		}

		// hold.AccountID points to disputed_escrow (set during FreezeEscrow).
		// Debit it directly — no need to look up the account type again.
		disputedAccountID := hold.AccountID

		journalID := uuid.New()
		now := time.Now()

		switch recipient {
		case "customer":
			customerWallet, err := s.ledger.EnsureWallet(ctx, tx, order.CustomerID)
			if err != nil {
				return fmt.Errorf("customer wallet: %w", err)
			}
			entries := []model.LedgerEntry{
				{ID: uuid.New(), JournalID: journalID, AccountID: disputedAccountID, AmountKobo: -hold.AmountKobo, Description: "admin dispute refund debit disputed_escrow", RefType: "dispute", RefID: &order.ID},
				{ID: uuid.New(), JournalID: journalID, AccountID: customerWallet.ID, AmountKobo: hold.AmountKobo, Description: "admin dispute refund to customer", RefType: "dispute", RefID: &order.ID},
			}
			if err := s.ledger.journal(ctx, tx, entries); err != nil {
				return err
			}
			if err := s.ledger.adjustBalance(ctx, tx, disputedAccountID, -hold.AmountKobo); err != nil {
				return fmt.Errorf("adjust disputed_escrow: %w", err)
			}
			if err := s.ledger.adjustBalance(ctx, tx, customerWallet.ID, hold.AmountKobo); err != nil {
				return fmt.Errorf("adjust customer: %w", err)
			}

		case "merchant":
			merchantWallet, err := s.ledger.EnsureMerchantWallet(ctx, tx, order.MerchantID)
			if err != nil {
				return fmt.Errorf("merchant wallet: %w", err)
			}
			entries := []model.LedgerEntry{
				{ID: uuid.New(), JournalID: journalID, AccountID: disputedAccountID, AmountKobo: -hold.AmountKobo, Description: "admin dispute settle debit disputed_escrow", RefType: "dispute", RefID: &order.ID},
				{ID: uuid.New(), JournalID: journalID, AccountID: merchantWallet.ID, AmountKobo: hold.AmountKobo, Description: "admin dispute settle merchant credit", RefType: "dispute", RefID: &order.ID},
			}
			if err := s.ledger.journal(ctx, tx, entries); err != nil {
				return err
			}
			if err := s.ledger.adjustBalance(ctx, tx, disputedAccountID, -hold.AmountKobo); err != nil {
				return fmt.Errorf("adjust disputed_escrow: %w", err)
			}
			if err := s.ledger.adjustBalance(ctx, tx, merchantWallet.ID, hold.AmountKobo); err != nil {
				return fmt.Errorf("adjust merchant: %w", err)
			}

		default:
			return fmt.Errorf("invalid recipient: %s", recipient)
		}

		hold.Status = model.EscrowReleased
		hold.ReleasedAt = &now
		hold.ReleasedBy = &adminID
		if err := s.ledger.repo.SaveEscrowHold(ctx, tx, hold); err != nil {
			return err
		}

		if err := s.repo.CreateAuditLogTx(ctx, tx, &model.AdminAuditLog{
			ID:         uuid.New(),
			AdminID:    adminID,
			Action:     "escrow_release",
			TargetType: "escrow_hold",
			TargetID:   hold.ID,
			Reason:     reason,
			CreatedAt:  now,
		}); err != nil {
			return err
		}

		result = "escrow released"
		return nil
	})
	return result, err
}

// ── Cancellation rules ────────────────────────────────────────────────────────

type CancellationRuleInput struct {
	Vertical               string
	OrderStatusAtCancel    string
	MerchantCompKobo       int64
	MerchantCompPct        float64
	RiderCompPctOfDelivery float64
	FullRefund             bool
}

func (s *AdminService) ListCancellationRules(ctx context.Context) ([]model.CancellationRule, error) {
	return s.repo.ListCancellationRules(ctx)
}

func (s *AdminService) UpsertCancellationRule(ctx context.Context, in CancellationRuleInput) (*model.CancellationRule, error) {
	rule := model.CancellationRule{
		Vertical:               in.Vertical,
		OrderStatusAtCancel:    in.OrderStatusAtCancel,
		MerchantCompKobo:       in.MerchantCompKobo,
		MerchantCompPct:        in.MerchantCompPct,
		RiderCompPctOfDelivery: in.RiderCompPctOfDelivery,
		FullRefund:             in.FullRefund,
	}
	return s.repo.UpsertCancellationRule(ctx, rule)
}

func (s *AdminService) DeleteCancellationRule(ctx context.Context, ruleID uuid.UUID) error {
	return s.repo.DeleteCancellationRule(ctx, ruleID)
}

// ── Gas: fill_status ──────────────────────────────────────────────────────────

type GasMerchantRow struct {
	ID              uuid.UUID `json:"id"`
	BusinessName    string    `json:"businessName"`
	FillAccuracyPct *float64  `json:"fillAccuracyPct"`
	FillSampleCount int       `json:"fillSampleCount"`
	FillStatus      string    `json:"fillStatus"`
}

func (s *AdminService) ListGasMerchants(ctx context.Context, fillStatus string, cursor *uuid.UUID, limit int) ([]GasMerchantRow, error) {
	merchants, err := s.repo.ListGasMerchants(ctx, fillStatus, cursor, limit)
	if err != nil {
		return nil, err
	}
	rows := make([]GasMerchantRow, len(merchants))
	for i, m := range merchants {
		rows[i] = GasMerchantRow{
			ID:              m.ID,
			BusinessName:    m.BusinessName,
			FillAccuracyPct: m.FillAccuracyPct,
			FillSampleCount: m.FillSampleCount,
			FillStatus:      m.FillStatus,
		}
	}
	return rows, nil
}

func (s *AdminService) SetMerchantFillStatus(ctx context.Context, merchantID, adminID uuid.UUID, status, reason string) error {
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		m, err := s.repo.LockMerchantTx(ctx, tx, merchantID)
		if err != nil {
			return fmt.Errorf("merchant not found")
		}
		m.FillStatus = status
		if err := s.repo.SaveMerchantTx(ctx, tx, m); err != nil {
			return err
		}
		return s.repo.CreateAuditLogTx(ctx, tx, &model.AdminAuditLog{
			ID:         uuid.New(),
			AdminID:    adminID,
			Action:     "merchant_fill_status_override",
			TargetType: "merchant",
			TargetID:   merchantID,
			Reason:     reason,
			CreatedAt:  time.Now(),
		})
	})
}

// ── Gas: zones / launch_status ────────────────────────────────────────────────

type ZoneRow struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	LaunchStatus string    `json:"launchStatus"`
	IsActive     bool      `json:"isActive"`
	WindowStart  int16     `json:"windowStart"`
	WindowEnd    int16     `json:"windowEnd"`
}

func (s *AdminService) ListZones(ctx context.Context, launchStatus string, cursor *uuid.UUID, limit int) ([]ZoneRow, error) {
	zones, err := s.repo.ListZones(ctx, launchStatus, cursor, limit)
	if err != nil {
		return nil, err
	}
	rows := make([]ZoneRow, len(zones))
	for i, z := range zones {
		rows[i] = ZoneRow{
			ID:           z.ID,
			Name:         z.Name,
			LaunchStatus: z.LaunchStatus,
			IsActive:     z.IsActive,
			WindowStart:  z.WindowStart,
			WindowEnd:    z.WindowEnd,
		}
	}
	return rows, nil
}

func (s *AdminService) SetZoneLaunchStatus(ctx context.Context, zoneID, adminID uuid.UUID, status, reason string) error {
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		z, err := s.repo.LockZoneTx(ctx, tx, zoneID)
		if err != nil {
			return fmt.Errorf("zone not found")
		}
		z.LaunchStatus = status
		if err := s.repo.SaveZoneTx(ctx, tx, z); err != nil {
			return err
		}
		return s.repo.CreateAuditLogTx(ctx, tx, &model.AdminAuditLog{
			ID:         uuid.New(),
			AdminID:    adminID,
			Action:     "zone_launch_status_change",
			TargetType: "service_zone",
			TargetID:   zoneID,
			Reason:     reason,
			CreatedAt:  time.Now(),
		})
	})
}

// ── Operational metrics ───────────────────────────────────────────────────────

func (s *AdminService) GetMetrics(ctx context.Context) (*repo.OperationalMetrics, error) {
	return s.repo.GetMetrics(ctx)
}
