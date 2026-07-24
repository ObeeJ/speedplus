package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/observability"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
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
