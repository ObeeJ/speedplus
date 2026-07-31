package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

type ProofMediaRepo interface {
	Create(ctx context.Context, m *model.ProofMedia) error
	ListByOrder(ctx context.Context, orderID uuid.UUID) ([]model.ProofMedia, error)
}

type proofMediaRepo struct{ db *gorm.DB }

func NewProofMediaRepo(db *gorm.DB) ProofMediaRepo { return &proofMediaRepo{db: db} }

func (r *proofMediaRepo) Create(ctx context.Context, m *model.ProofMedia) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *proofMediaRepo) ListByOrder(ctx context.Context, orderID uuid.UUID) ([]model.ProofMedia, error) {
	var rows []model.ProofMedia
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).Order("captured_at ASC").Find(&rows).Error
	return rows, err
}
