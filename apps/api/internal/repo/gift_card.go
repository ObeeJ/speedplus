package repo

import (
	"context"

	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GiftCardRepo interface {
	Create(ctx context.Context, gc *model.GiftCard) error
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	LockByCodeHash(ctx context.Context, tx *gorm.DB, codeHash string) (*model.GiftCard, error)
	SaveTx(ctx context.Context, tx *gorm.DB, gc *model.GiftCard) error
}

type giftCardRepo struct{ db *gorm.DB }

func NewGiftCardRepo(db *gorm.DB) GiftCardRepo { return &giftCardRepo{db: db} }

func (r *giftCardRepo) Create(ctx context.Context, gc *model.GiftCard) error {
	return r.db.WithContext(ctx).Create(gc).Error
}

func (r *giftCardRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *giftCardRepo) LockByCodeHash(ctx context.Context, tx *gorm.DB, codeHash string) (*model.GiftCard, error) {
	var gc model.GiftCard
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("code_hash = ? AND redeemed_by IS NULL", codeHash).First(&gc).Error
	return &gc, err
}

func (r *giftCardRepo) SaveTx(ctx context.Context, tx *gorm.DB, gc *model.GiftCard) error {
	return tx.WithContext(ctx).Save(gc).Error
}
