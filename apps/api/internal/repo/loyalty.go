package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LoyaltyRepo interface {
	FindBalance(ctx context.Context, userID uuid.UUID) (*model.LoyaltyBalance, error)
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	LockBalance(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*model.LoyaltyBalance, error)
	DeductBalanceTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID, points int) error
	ListEvents(ctx context.Context, userID uuid.UUID, limit int) ([]model.LoyaltyEvent, error)
}

type loyaltyRepo struct{ db *gorm.DB }

func NewLoyaltyRepo(db *gorm.DB) LoyaltyRepo { return &loyaltyRepo{db: db} }

func (r *loyaltyRepo) FindBalance(ctx context.Context, userID uuid.UUID) (*model.LoyaltyBalance, error) {
	var b model.LoyaltyBalance
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&b).Error
	return &b, err
}

func (r *loyaltyRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *loyaltyRepo) LockBalance(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*model.LoyaltyBalance, error) {
	var b model.LoyaltyBalance
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).First(&b).Error
	return &b, err
}

func (r *loyaltyRepo) DeductBalanceTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID, points int) error {
	return tx.WithContext(ctx).Exec(
		`UPDATE loyalty_balances SET points = points - ?, updated_at = NOW() WHERE user_id = ?`,
		points, userID,
	).Error
}

func (r *loyaltyRepo) ListEvents(ctx context.Context, userID uuid.UUID, limit int) ([]model.LoyaltyEvent, error) {
	var events []model.LoyaltyEvent
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&events).Error
	return events, err
}
