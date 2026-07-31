package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

type USSDRepo interface {
	CreatePaymentIntent(ctx context.Context, intent *model.PaymentIntent) error
	CreateIntent(ctx context.Context, intent *model.USSDIntent) error
	FindIntent(ctx context.Context, id, userID uuid.UUID) (*model.USSDIntent, error)
}

type ussdRepo struct{ db *gorm.DB }

func NewUSSDRepo(db *gorm.DB) USSDRepo { return &ussdRepo{db: db} }

func (r *ussdRepo) CreatePaymentIntent(ctx context.Context, intent *model.PaymentIntent) error {
	return r.db.WithContext(ctx).Create(intent).Error
}

func (r *ussdRepo) CreateIntent(ctx context.Context, intent *model.USSDIntent) error {
	return r.db.WithContext(ctx).Create(intent).Error
}

func (r *ussdRepo) FindIntent(ctx context.Context, id, userID uuid.UUID) (*model.USSDIntent, error) {
	var ui model.USSDIntent
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&ui).Error
	return &ui, err
}
