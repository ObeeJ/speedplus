package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

type ReferralRepo interface {
	FindTrustTier(ctx context.Context, userID uuid.UUID) (*model.UserTrustTier, error)
	CreateReferral(ctx context.Context, r *model.Referral) error
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	FindUnpaidReferral(ctx context.Context, tx *gorm.DB, refereeID uuid.UUID) (*model.Referral, error)
	MarkReferralPaidTx(ctx context.Context, tx *gorm.DB, ref *model.Referral) error
}

type referralRepo struct{ db *gorm.DB }

func NewReferralRepo(db *gorm.DB) ReferralRepo { return &referralRepo{db: db} }

func (r *referralRepo) FindTrustTier(ctx context.Context, userID uuid.UUID) (*model.UserTrustTier, error) {
	var tier model.UserTrustTier
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&tier).Error
	return &tier, err
}

func (r *referralRepo) CreateReferral(ctx context.Context, ref *model.Referral) error {
	return r.db.WithContext(ctx).Create(ref).Error
}

func (r *referralRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *referralRepo) FindUnpaidReferral(ctx context.Context, tx *gorm.DB, refereeID uuid.UUID) (*model.Referral, error) {
	var ref model.Referral
	err := tx.WithContext(ctx).Where("referee_id = ? AND reward_paid_at IS NULL", refereeID).First(&ref).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &ref, err
}

func (r *referralRepo) MarkReferralPaidTx(ctx context.Context, tx *gorm.DB, ref *model.Referral) error {
	return tx.WithContext(ctx).Model(ref).Update("reward_paid_at", gorm.Expr("NOW()")).Error
}
