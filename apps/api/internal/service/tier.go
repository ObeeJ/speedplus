package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/observability"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

// Pay-on-arrival (POD) limits for Tier 1. Caps bound the maximum loss a single
// ghosted POD order can cause while the trust ladder is still shallow.
const (
	Tier1PODCapKobo = 1_000_000 // ₦10,000 max order total on POD
)

var (
	ErrPODTierLocked   = errors.New("pay on arrival unlocks after 3 completed orders")
	ErrPODCapExceeded  = errors.New("order exceeds the pay-on-arrival limit for your account")
	ErrPODActiveExists = errors.New("finish your active pay-on-arrival order first")
)

// TierService evaluates a user's trust tier after order events.
// Only Tier 0 and Tier 1 are active. Rules:
//
//	Tier 0 → Tier 1: completed_orders >= 3 AND fraud_flags == 0 AND NOT frozen
//	Tier 1 → Tier 0: fraud_flags > 0 OR pay-on-arrival payment failure
//	Frozen:          permanent Tier 0, no promotion ever
type TierService struct {
	db   *gorm.DB // transaction boundary only — no direct queries
	repo repo.TierRepo
}

func NewTierService(db *gorm.DB, r repo.TierRepo) *TierService {
	return &TierService{db: db, repo: r}
}

// RecordCompletion increments completed_orders and re-evaluates tier.
// Satisfies ports.TierRecorder.
func (s *TierService) RecordCompletion(ctx context.Context, userID uuid.UUID) {
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tier, err := s.repo.LockTier(ctx, tx, userID)
		if err != nil {
			return err
		}
		if tier.Frozen {
			return nil
		}
		tier.CompletedOrders++
		tier.Tier = evaluate(tier)
		return s.repo.SaveTier(ctx, tx, tier)
	}); err != nil {
		observability.CaptureError(ctx, err, "tier: record completion", "user_id", userID.String())
	}
}

// RecordFraudFlag increments fraud_flags, demotes to Tier 0, and optionally
// freezes the account permanently.
func (s *TierService) RecordFraudFlag(ctx context.Context, userID uuid.UUID, freeze bool) {
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tier, err := s.repo.LockTier(ctx, tx, userID)
		if err != nil {
			return err
		}
		tier.FraudFlags++
		tier.Tier = model.TierNew
		if freeze {
			tier.Frozen = true
		}
		return s.repo.SaveTier(ctx, tx, tier)
	}); err != nil {
		observability.CaptureError(ctx, err, "tier: record fraud flag", "user_id", userID.String())
	}
}

// RecordPayOnArrivalFailure resets the user to Tier 0 (not frozen).
func (s *TierService) RecordPayOnArrivalFailure(ctx context.Context, userID uuid.UUID) {
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tier, err := s.repo.LockTier(ctx, tx, userID)
		if err != nil {
			return err
		}
		if tier.Frozen {
			return nil
		}
		tier.Tier = model.TierNew
		return s.repo.SaveTier(ctx, tx, tier)
	}); err != nil {
		observability.CaptureError(ctx, err, "tier: record pay-on-arrival failure", "user_id", userID.String())
	}
}

// GetTier returns the current tier for a user. Returns TierNew if no row exists.
func (s *TierService) GetTier(ctx context.Context, userID uuid.UUID) model.TrustTier {
	tier, err := s.repo.GetTier(ctx, userID)
	if err != nil {
		return model.TierNew
	}
	return tier.Tier
}

// CanUsePayOnArrival gates the POD trust ladder: Tier 1+, order under the
// per-tier cap, and no other POD order still in flight.
func (s *TierService) CanUsePayOnArrival(ctx context.Context, userID uuid.UUID, orderTotalKobo int64) error {
	if s.GetTier(ctx, userID) < model.TierRegular {
		return ErrPODTierLocked
	}
	if orderTotalKobo > Tier1PODCapKobo {
		return ErrPODCapExceeded
	}
	var active int64
	if err := s.db.WithContext(ctx).Model(&model.Order{}).
		Where("customer_id = ? AND payment_method = 'pay_on_arrival' AND status NOT IN ('delivered','cancelled','refunded')", userID).
		Count(&active).Error; err != nil {
		return err
	}
	if active > 0 {
		return ErrPODActiveExists
	}
	return nil
}

// evaluate is a pure function — no DB, no side effects, fully testable.
func evaluate(t *model.UserTrustTier) model.TrustTier {
	if t.Frozen || t.FraudFlags > 0 {
		return model.TierNew
	}
	if t.CompletedOrders >= 3 {
		return model.TierRegular
	}
	return model.TierNew
}
