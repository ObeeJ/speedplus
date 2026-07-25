package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AdminService handles all admin-only business logic.
// Every money-moving operation routes through LedgerService — never direct DB writes.
type AdminService struct {
	db     *gorm.DB
	ledger *LedgerService
}

func NewAdminService(db *gorm.DB, ledger *LedgerService) *AdminService {
	return &AdminService{db: db, ledger: ledger}
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

func (s *AdminService) ListMerchants(ctx context.Context, status string, page, limit int) ([]MerchantRow, error) {
	q := s.db.WithContext(ctx).Model(&model.MerchantProfile{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var rows []MerchantRow
	err := q.
		Select("id, user_id, business_name, vertical, status, rating, created_at").
		Order("created_at DESC").
		Offset(page * limit).
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (s *AdminService) SetMerchantStatus(ctx context.Context, merchantID, adminID uuid.UUID, status, reason string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var mp model.MerchantProfile
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&mp, merchantID).Error; err != nil {
			return fmt.Errorf("merchant not found")
		}
		mp.Status = model.MerchantStatus(status)
		if err := tx.Save(&mp).Error; err != nil {
			return err
		}
		return tx.Create(&model.AdminAuditLog{
			ID:         uuid.New(),
			AdminID:    adminID,
			Action:     "merchant_status_change",
			TargetType: "merchant_profile",
			TargetID:   merchantID,
			Reason:     reason,
			CreatedAt:  time.Now(),
		}).Error
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

func (s *AdminService) ListDrivers(ctx context.Context, status string, page, limit int) ([]DriverRow, error) {
	q := s.db.WithContext(ctx).Model(&model.DriverProfile{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var rows []DriverRow
	err := q.
		Select("id, user_id, status, vehicle_type, vehicle_plate, rating, total_deliveries, created_at").
		Order("created_at DESC").
		Offset(page * limit).
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (s *AdminService) SetDriverStatus(ctx context.Context, driverID, adminID uuid.UUID, status, reason string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var dp model.DriverProfile
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&dp, driverID).Error; err != nil {
			return fmt.Errorf("driver not found")
		}
		dp.Status = model.DriverStatus(status)
		if err := tx.Save(&dp).Error; err != nil {
			return err
		}
		return tx.Create(&model.AdminAuditLog{
			ID:         uuid.New(),
			AdminID:    adminID,
			Action:     "driver_status_change",
			TargetType: "driver_profile",
			TargetID:   driverID,
			Reason:     reason,
			CreatedAt:  time.Now(),
		}).Error
	})
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

func (s *AdminService) SearchOrders(ctx context.Context, q, status string, page, limit int) ([]OrderSummary, error) {
	db := s.db.WithContext(ctx).Model(&model.Order{})
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if q != "" {
		// Support order ID prefix or customer ID exact match
		db = db.Where("CAST(id AS TEXT) ILIKE ? OR CAST(customer_id AS TEXT) = ?", q+"%", q)
	}
	var rows []OrderSummary
	err := db.
		Select("id, customer_id, merchant_id, driver_id, vertical, status, total_kobo, created_at").
		Order("created_at DESC").
		Offset(page * limit).
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (s *AdminService) GetOrderDetail(ctx context.Context, orderID uuid.UUID) (*OrderDetail, error) {
	var order model.Order
	if err := s.db.WithContext(ctx).
		Preload("Items").
		Preload("Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		First(&order, orderID).Error; err != nil {
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

// FreezeEscrow transitions EscrowHeld → EscrowFrozen.
// No money moves. Records the admin actor and reason.
func (s *AdminService) FreezeEscrow(ctx context.Context, orderID, adminID uuid.UUID, reason string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		hold, err := s.ledger.repo.LockEscrowHold(ctx, tx, orderID, model.EscrowHeld)
		if err != nil {
			return fmt.Errorf("no held escrow found for order")
		}
		now := time.Now()
		hold.Status = model.EscrowFrozen
		hold.FrozenAt = &now
		hold.FrozenReason = &reason
		hold.ApprovalOne = &adminID
		if err := s.ledger.repo.SaveEscrowHold(ctx, tx, hold); err != nil {
			return err
		}
		return tx.Create(&model.AdminAuditLog{
			ID:         uuid.New(),
			AdminID:    adminID,
			Action:     "escrow_freeze",
			TargetType: "escrow_hold",
			TargetID:   hold.ID,
			Reason:     reason,
			CreatedAt:  time.Now(),
		}).Error
	})
}

// ReleaseEscrow resolves a frozen escrow dispute.
// Dual-admin approval: first call records ApprovalOne (if not already set) or ApprovalTwo.
// Funds only move when both approvals are present and they are different admins.
// recipient: "customer" = full refund to customer wallet; "merchant" = settle to merchant wallet.
func (s *AdminService) ReleaseEscrow(ctx context.Context, orderID, adminID uuid.UUID, recipient, reason string) (string, error) {
	var result string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		hold, err := s.ledger.repo.LockEscrowHold(ctx, tx, orderID, model.EscrowFrozen)
		if err != nil {
			return fmt.Errorf("no frozen escrow found for order")
		}

		// Record first approval
		if hold.ApprovalOne == nil {
			hold.ApprovalOne = &adminID
			if err := s.ledger.repo.SaveEscrowHold(ctx, tx, hold); err != nil {
				return err
			}
			result = "approval recorded — awaiting second admin approval"
			return nil
		}

		// Prevent same admin approving twice
		if *hold.ApprovalOne == adminID {
			return fmt.Errorf("same admin cannot provide both approvals")
		}

		// Second approval — execute the release
		hold.ApprovalTwo = &adminID

		var order model.Order
		if err := tx.First(&order, orderID).Error; err != nil {
			return fmt.Errorf("order not found")
		}

		escrowAcct, err := s.ledger.platformAccount(ctx, tx, model.AccountEscrow)
		if err != nil {
			return fmt.Errorf("escrow account: %w", err)
		}

		journalID := uuid.New()
		now := time.Now()

		switch recipient {
		case "customer":
			customerWallet, err := s.ledger.EnsureWallet(ctx, tx, order.CustomerID)
			if err != nil {
				return fmt.Errorf("customer wallet: %w", err)
			}
			entries := []model.LedgerEntry{
				{ID: uuid.New(), JournalID: journalID, AccountID: escrowAcct.ID, AmountKobo: -hold.AmountKobo, Description: "admin dispute refund debit escrow", RefType: "dispute", RefID: &order.ID},
				{ID: uuid.New(), JournalID: journalID, AccountID: customerWallet.ID, AmountKobo: hold.AmountKobo, Description: "admin dispute refund to customer", RefType: "dispute", RefID: &order.ID},
			}
			if err := s.ledger.journal(ctx, tx, entries); err != nil {
				return err
			}
			if err := s.ledger.adjustBalance(ctx, tx, escrowAcct.ID, -hold.AmountKobo); err != nil {
				return fmt.Errorf("adjust escrow: %w", err)
			}
			if err := s.ledger.adjustBalance(ctx, tx, customerWallet.ID, hold.AmountKobo); err != nil {
				return fmt.Errorf("adjust customer: %w", err)
			}

		case "merchant":
			merchantWallet, err := s.ledger.EnsureWallet(ctx, tx, order.MerchantID)
			if err != nil {
				return fmt.Errorf("merchant wallet: %w", err)
			}
			entries := []model.LedgerEntry{
				{ID: uuid.New(), JournalID: journalID, AccountID: escrowAcct.ID, AmountKobo: -hold.AmountKobo, Description: "admin dispute settle to merchant", RefType: "dispute", RefID: &order.ID},
				{ID: uuid.New(), JournalID: journalID, AccountID: merchantWallet.ID, AmountKobo: hold.AmountKobo, Description: "admin dispute settle merchant credit", RefType: "dispute", RefID: &order.ID},
			}
			if err := s.ledger.journal(ctx, tx, entries); err != nil {
				return err
			}
			if err := s.ledger.adjustBalance(ctx, tx, escrowAcct.ID, -hold.AmountKobo); err != nil {
				return fmt.Errorf("adjust escrow: %w", err)
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

		if err := tx.Create(&model.AdminAuditLog{
			ID:         uuid.New(),
			AdminID:    adminID,
			Action:     "escrow_release",
			TargetType: "escrow_hold",
			TargetID:   hold.ID,
			Reason:     reason,
			CreatedAt:  now,
		}).Error; err != nil {
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
	var rules []model.CancellationRule
	err := s.db.WithContext(ctx).
		Order("vertical ASC, order_status_at_cancel ASC").
		Find(&rules).Error
	return rules, err
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
	// Upsert on (vertical, order_status_at_cancel) uniqueness
	err := s.db.WithContext(ctx).
		Where("vertical = ? AND order_status_at_cancel = ?", in.Vertical, in.OrderStatusAtCancel).
		Assign(rule).
		FirstOrCreate(&rule).Error
	return &rule, err
}

func (s *AdminService) DeleteCancellationRule(ctx context.Context, ruleID uuid.UUID) error {
	return s.db.WithContext(ctx).Delete(&model.CancellationRule{}, ruleID).Error
}
