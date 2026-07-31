package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PaymentLinkRepo interface {
	Create(ctx context.Context, pl *model.PaymentLink) error
	FindPendingBySlug(ctx context.Context, slug string) (*model.PaymentLink, error)
	ExpireLink(ctx context.Context, id uuid.UUID) error
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	FindIdempotencyKeyTx(ctx context.Context, tx *gorm.DB, key string) (*model.IdempotencyKey, error)
	LockPendingBySlugTx(ctx context.Context, tx *gorm.DB, slug string) (*model.PaymentLink, error)
	ExpireLinkTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) error
	SaveLinkTx(ctx context.Context, tx *gorm.DB, pl *model.PaymentLink) error
	CreateIdempotencyKeyTx(ctx context.Context, tx *gorm.DB, k *model.IdempotencyKey) error
	CreatePaymentIntent(ctx context.Context, intent *model.PaymentIntent) error
	UpdateLinkProviderRef(ctx context.Context, id uuid.UUID, ref, email string) error
}

type paymentLinkRepo struct{ db *gorm.DB }

func NewPaymentLinkRepo(db *gorm.DB) PaymentLinkRepo { return &paymentLinkRepo{db: db} }

func (r *paymentLinkRepo) Create(ctx context.Context, pl *model.PaymentLink) error {
	return r.db.WithContext(ctx).Create(pl).Error
}

func (r *paymentLinkRepo) FindPendingBySlug(ctx context.Context, slug string) (*model.PaymentLink, error) {
	var pl model.PaymentLink
	err := r.db.WithContext(ctx).Where("slug = ? AND status = 'pending'", slug).First(&pl).Error
	return &pl, err
}

func (r *paymentLinkRepo) ExpireLink(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.PaymentLink{}).Where("id = ?", id).Update("status", "expired").Error
}

func (r *paymentLinkRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *paymentLinkRepo) FindIdempotencyKeyTx(ctx context.Context, tx *gorm.DB, key string) (*model.IdempotencyKey, error) {
	var k model.IdempotencyKey
	err := tx.WithContext(ctx).Where("key = ?", key).First(&k).Error
	return &k, err
}

func (r *paymentLinkRepo) LockPendingBySlugTx(ctx context.Context, tx *gorm.DB, slug string) (*model.PaymentLink, error) {
	var pl model.PaymentLink
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("slug = ? AND status = 'pending'", slug).First(&pl).Error
	return &pl, err
}

func (r *paymentLinkRepo) ExpireLinkTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) error {
	return tx.WithContext(ctx).Model(&model.PaymentLink{}).Where("id = ?", id).Update("status", "expired").Error
}

func (r *paymentLinkRepo) SaveLinkTx(ctx context.Context, tx *gorm.DB, pl *model.PaymentLink) error {
	return tx.WithContext(ctx).Save(pl).Error
}

func (r *paymentLinkRepo) CreateIdempotencyKeyTx(ctx context.Context, tx *gorm.DB, k *model.IdempotencyKey) error {
	return tx.WithContext(ctx).Create(k).Error
}

func (r *paymentLinkRepo) CreatePaymentIntent(ctx context.Context, intent *model.PaymentIntent) error {
	return r.db.WithContext(ctx).Create(intent).Error
}

func (r *paymentLinkRepo) UpdateLinkProviderRef(ctx context.Context, id uuid.UUID, ref, email string) error {
	return r.db.WithContext(ctx).Model(&model.PaymentLink{}).Where("id = ?", id).
		Updates(map[string]interface{}{"provider_ref": ref, "paid_by_email": email}).Error
}

// WalletBalanceLockTx locks a wallet_balance row for update inside a tx.
func (r *paymentLinkRepo) WalletBalanceLockTx(ctx context.Context, tx *gorm.DB, accountID uuid.UUID) (*model.WalletBalance, error) {
	var b model.WalletBalance
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("account_id = ?", accountID).First(&b).Error
	return &b, err
}

// CreateIdempotencyKey is the non-tx version used outside transactions.
func (r *paymentLinkRepo) CreateIdempotencyKey(ctx context.Context, k *model.IdempotencyKey) error {
	return r.db.WithContext(ctx).Create(k).Error
}
