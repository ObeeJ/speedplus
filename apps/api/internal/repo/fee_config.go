package repo

import (
	"context"
	"time"

	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

type FeeConfigRepo interface {
	FindLatestByVertical(ctx context.Context, vertical string, at time.Time) (*model.FeeConfig, error)
	ListLatestPerVertical(ctx context.Context) ([]model.FeeConfig, error)
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	CreateTx(ctx context.Context, tx *gorm.DB, row *model.FeeConfig) error
	CreateAuditLogTx(ctx context.Context, tx *gorm.DB, log *model.AdminAuditLog) error
}

type feeConfigRepo struct{ db *gorm.DB }

func NewFeeConfigRepo(db *gorm.DB) FeeConfigRepo { return &feeConfigRepo{db: db} }

func (r *feeConfigRepo) FindLatestByVertical(ctx context.Context, vertical string, at time.Time) (*model.FeeConfig, error) {
	var row model.FeeConfig
	err := r.db.WithContext(ctx).
		Where("vertical = ? AND effective_at <= ?", vertical, at).
		Order("effective_at DESC").
		First(&row).Error
	return &row, err
}

func (r *feeConfigRepo) ListLatestPerVertical(ctx context.Context) ([]model.FeeConfig, error) {
	var rows []model.FeeConfig
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT ON (vertical) * FROM fee_configs
		WHERE effective_at <= NOW()
		ORDER BY vertical, effective_at DESC
	`).Scan(&rows).Error
	return rows, err
}

func (r *feeConfigRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *feeConfigRepo) CreateTx(ctx context.Context, tx *gorm.DB, row *model.FeeConfig) error {
	return tx.WithContext(ctx).Create(row).Error
}

func (r *feeConfigRepo) CreateAuditLogTx(ctx context.Context, tx *gorm.DB, log *model.AdminAuditLog) error {
	return tx.WithContext(ctx).Create(log).Error
}
