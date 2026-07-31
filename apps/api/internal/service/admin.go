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

// FreezeEscrow transitions EscrowHeld → EscrowFrozen.
// No money moves. Records the admin actor and reason.
func (s *AdminService) FreezeEscrow(ctx context.Context, orderID, adminID uuid.UUID, reason string) error {
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
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
		return s.repo.CreateAuditLogTx(ctx, tx, &model.AdminAuditLog{
			ID:         uuid.New(),
			AdminID:    adminID,
			Action:     "escrow_freeze",
			TargetType: "escrow_hold",
			TargetID:   hold.ID,
			Reason:     reason,
			CreatedAt:  time.Now(),
		})
	})
}

// ReleaseEscrow resolves a frozen escrow dispute.
// Dual-admin approval: first call records ApprovalOne (if not already set) or ApprovalTwo.
// Funds only move when both approvals are present and they are different admins.
// recipient: "customer" = full refund to customer wallet; "merchant" = settle to merchant wallet.
func (s *AdminService) ReleaseEscrow(ctx context.Context, orderID, adminID uuid.UUID, recipient, reason string) (string, error) {
	var result string
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		hold, err := s.ledger.repo.LockEscrowHold(ctx, tx, orderID, model.EscrowFrozen)
		if err != nil {
			return fmt.Errorf("no frozen escrow found for order")
		}

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

		order, err := s.repo.FindOrderTx(ctx, tx, orderID)
		if err != nil {
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
