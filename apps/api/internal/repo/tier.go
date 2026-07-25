package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TierRepo handles persistence for the user trust tier.
type TierRepo interface {
	LockTier(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*model.UserTrustTier, error)
	SaveTier(ctx context.Context, tx *gorm.DB, tier *model.UserTrustTier) error
	GetTier(ctx context.Context, userID uuid.UUID) (*model.UserTrustTier, error)
}

type tierRepo struct{ db *gorm.DB }

func NewTierRepo(db *gorm.DB) TierRepo { return &tierRepo{db: db} }

func (r *tierRepo) LockTier(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*model.UserTrustTier, error) {
	var tier model.UserTrustTier
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		FirstOrCreate(&tier, model.UserTrustTier{UserID: userID}).Error
	return &tier, err
}

func (r *tierRepo) SaveTier(ctx context.Context, tx *gorm.DB, tier *model.UserTrustTier) error {
	return tx.WithContext(ctx).Save(tier).Error
}

func (r *tierRepo) GetTier(ctx context.Context, userID uuid.UUID) (*model.UserTrustTier, error) {
	var tier model.UserTrustTier
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&tier).Error
	return &tier, err
}
