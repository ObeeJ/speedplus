package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

// OnboardingRepo handles persistence for the post-registration onboarding flow.
type OnboardingRepo interface {
	FindVirtualAccount(ctx context.Context, userID uuid.UUID) (*model.VirtualAccount, error)
	CreateVirtualAccount(ctx context.Context, va *model.VirtualAccount) error
	FindUserCard(ctx context.Context, userID uuid.UUID) (*model.UserCard, error)
	CreateUserCard(ctx context.Context, card *model.UserCard) error
	FindOrCreateTrustTier(ctx context.Context, userID uuid.UUID) (*model.UserTrustTier, error)
}

type onboardingRepo struct{ db *gorm.DB }

func NewOnboardingRepo(db *gorm.DB) OnboardingRepo { return &onboardingRepo{db: db} }

func (r *onboardingRepo) FindVirtualAccount(ctx context.Context, userID uuid.UUID) (*model.VirtualAccount, error) {
	var va model.VirtualAccount
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&va).Error
	return &va, err
}

func (r *onboardingRepo) CreateVirtualAccount(ctx context.Context, va *model.VirtualAccount) error {
	return r.db.WithContext(ctx).Create(va).Error
}

func (r *onboardingRepo) FindUserCard(ctx context.Context, userID uuid.UUID) (*model.UserCard, error) {
	var card model.UserCard
	err := r.db.WithContext(ctx).Where("user_id = ? AND is_active = true", userID).First(&card).Error
	return &card, err
}

func (r *onboardingRepo) CreateUserCard(ctx context.Context, card *model.UserCard) error {
	return r.db.WithContext(ctx).Create(card).Error
}

func (r *onboardingRepo) FindOrCreateTrustTier(ctx context.Context, userID uuid.UUID) (*model.UserTrustTier, error) {
	tier := model.UserTrustTier{UserID: userID}
	err := r.db.WithContext(ctx).
		Where(model.UserTrustTier{UserID: userID}).
		FirstOrCreate(&tier).Error
	return &tier, err
}
