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
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	CountActivePODOrders(ctx context.Context, userID uuid.UUID) (int64, error)
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

func (r *tierRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *tierRepo) CountActivePODOrders(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Order{}).
		Where("customer_id = ? AND payment_method = 'pay_on_arrival' AND status NOT IN ('delivered','cancelled','refunded')", userID).
		Count(&count).Error
	return count, err
}
