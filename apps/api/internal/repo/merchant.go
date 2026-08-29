package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

type MerchantRepo interface {
	CreateMerchant(ctx context.Context, m *model.Merchant) error
	FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Merchant, error)
	FindProfileByUserID(ctx context.Context, userID uuid.UUID) (*model.MerchantProfile, error)
	SetOpen(ctx context.Context, merchantID uuid.UUID, isOpen bool) error
	FindBankAccount(ctx context.Context, merchantID uuid.UUID) (*model.MerchantBankAccount, error)
	UpsertBankAccount(ctx context.Context, acct *model.MerchantBankAccount) error
}

type merchantRepo struct{ db *gorm.DB }

func NewMerchantRepo(db *gorm.DB) MerchantRepo { return &merchantRepo{db: db} }

func (r *merchantRepo) CreateMerchant(ctx context.Context, m *model.Merchant) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *merchantRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Merchant, error) {
	var m model.Merchant
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error
	return &m, err
}

func (r *merchantRepo) FindProfileByUserID(ctx context.Context, userID uuid.UUID) (*model.MerchantProfile, error) {
	var p model.MerchantProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error
	return &p, err
}

func (r *merchantRepo) SetOpen(ctx context.Context, merchantID uuid.UUID, isOpen bool) error {
	return r.db.WithContext(ctx).Model(&model.Merchant{}).
		Where("id = ?", merchantID).
		Update("is_open", isOpen).Error
}

func (r *merchantRepo) FindBankAccount(ctx context.Context, merchantID uuid.UUID) (*model.MerchantBankAccount, error) {
	var acct model.MerchantBankAccount
	err := r.db.WithContext(ctx).Where("merchant_id = ?", merchantID).First(&acct).Error
	return &acct, err
}

func (r *merchantRepo) UpsertBankAccount(ctx context.Context, acct *model.MerchantBankAccount) error {
	return r.db.WithContext(ctx).
		Where("merchant_id = ?", acct.MerchantID).
		Assign(acct).
		FirstOrCreate(acct).Error
}
