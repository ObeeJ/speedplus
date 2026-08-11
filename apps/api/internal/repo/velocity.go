package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type VelocityRepo interface {
	FindVelocityLimit(ctx context.Context, tier int, operation string) (*model.VelocityLimit, error)
	ListVelocityLimits(ctx context.Context) ([]model.VelocityLimit, error)
	UpsertVelocityLimit(ctx context.Context, limit *model.VelocityLimit) error
	UpsertVelocityCounter(ctx context.Context, tx *gorm.DB, counter *model.VelocityCounter) error
	// SumVelocityWindow returns both the summed amount and the transaction
	// count in the window. Structuring detection needs the count: many small
	// transfers and one large one can sum identically but mean very different
	// things.
	SumVelocityWindow(ctx context.Context, tx *gorm.DB, userID uuid.UUID, operation string, since time.Time) (sumKobo int64, txnCount int, err error)
	CreateSuspiciousActivity(ctx context.Context, tx *gorm.DB, sa *model.SuspiciousActivity) error
	ListSuspiciousActivity(ctx context.Context, cursor *uuid.UUID, limit int) ([]model.SuspiciousActivity, error)
}

type velocityRepo struct{ db *gorm.DB }

func NewVelocityRepo(db *gorm.DB) VelocityRepo { return &velocityRepo{db: db} }

func (r *velocityRepo) FindVelocityLimit(ctx context.Context, tier int, operation string) (*model.VelocityLimit, error) {
	var limit model.VelocityLimit
	err := r.db.WithContext(ctx).
		Where("tier = ? AND operation = ?", tier, operation).
		First(&limit).Error
	return &limit, err
}

func (r *velocityRepo) ListVelocityLimits(ctx context.Context) ([]model.VelocityLimit, error) {
	var limits []model.VelocityLimit
	err := r.db.WithContext(ctx).Order("tier, operation").Find(&limits).Error
	return limits, err
}

func (r *velocityRepo) UpsertVelocityLimit(ctx context.Context, limit *model.VelocityLimit) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "tier"}, {Name: "operation"}},
			DoUpdates: clause.AssignmentColumns([]string{"per_txn_kobo", "daily_kobo", "monthly_kobo"}),
		}).
		Create(limit).Error
}

func (r *velocityRepo) UpsertVelocityCounter(ctx context.Context, tx *gorm.DB, counter *model.VelocityCounter) error {
	// ON CONFLICT: increment the existing counter for this (user, operation, window_start, window_size).
	return tx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "user_id"}, {Name: "operation"},
				{Name: "window_start"}, {Name: "window_size"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"amount_kobo": gorm.Expr("velocity_counters.amount_kobo + EXCLUDED.amount_kobo"),
				"txn_count":   gorm.Expr("velocity_counters.txn_count + 1"),
				"updated_at":  gorm.Expr("NOW()"),
			}),
		}).
		Create(counter).Error
}

func (r *velocityRepo) SumVelocityWindow(ctx context.Context, tx *gorm.DB, userID uuid.UUID, operation string, since time.Time) (int64, int, error) {
	var agg struct {
		SumKobo  int64
		TxnCount int
	}
	db := r.db
	if tx != nil {
		db = tx
	}
	err := db.WithContext(ctx).
		Model(&model.VelocityCounter{}).
		Where("user_id = ? AND operation = ? AND window_start >= ?", userID, operation, since).
		Select("COALESCE(SUM(amount_kobo), 0) AS sum_kobo, COALESCE(SUM(txn_count), 0) AS txn_count").
		Scan(&agg).Error
	return agg.SumKobo, agg.TxnCount, err
}

func (r *velocityRepo) CreateSuspiciousActivity(ctx context.Context, tx *gorm.DB, sa *model.SuspiciousActivity) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Create(sa).Error
}

func (r *velocityRepo) ListSuspiciousActivity(ctx context.Context, cursor *uuid.UUID, limit int) ([]model.SuspiciousActivity, error) {
	q := r.db.WithContext(ctx).Order("created_at DESC, id DESC").Limit(limit)
	if cursor != nil {
		var pivot model.SuspiciousActivity
		if err := r.db.WithContext(ctx).First(&pivot, cursor).Error; err == nil {
			q = q.Where("(created_at, id) < (?, ?)", pivot.CreatedAt, pivot.ID)
		}
	}
	var rows []model.SuspiciousActivity
	return rows, q.Find(&rows).Error
}
