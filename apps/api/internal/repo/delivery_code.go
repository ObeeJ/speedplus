package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DeliveryCodeRepo handles persistence for delivery OTP codes.
type DeliveryCodeRepo interface {
	// Upsert creates or replaces the delivery code for an order.
	// Called when the order transitions to in_transit.
	Upsert(ctx context.Context, code *model.DeliveryCode) error

	// LockByOrder returns the active (unused, unexpired) code for an order
	// with SELECT FOR UPDATE. Returns error if not found or already used.
	LockByOrder(ctx context.Context, tx *gorm.DB, orderID uuid.UUID) (*model.DeliveryCode, error)

	// IncrementAttempts increments the failed attempt counter.
	IncrementAttempts(ctx context.Context, tx *gorm.DB, id uuid.UUID) error

	// MarkUsed marks the code as used at the given time.
	MarkUsed(ctx context.Context, tx *gorm.DB, id uuid.UUID, at time.Time) error

	// RecordConfirmLocation stores the GPS evidence captured when the code was
	// successfully entered (keyed by order — one code per order). Flag-only —
	// never blocks settlement.
	RecordConfirmLocation(ctx context.Context, tx *gorm.DB, orderID uuid.UUID, lat, lng, distanceM *float64, flagged bool) error
}

type deliveryCodeRepo struct{ db *gorm.DB }

func NewDeliveryCodeRepo(db *gorm.DB) DeliveryCodeRepo { return &deliveryCodeRepo{db: db} }

func (r *deliveryCodeRepo) Upsert(ctx context.Context, code *model.DeliveryCode) error {
	// ON CONFLICT on order_id: replace the existing code (e.g. rider re-dispatched)
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "order_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"code_hash", "attempts", "expires_at", "used_at", "created_at"}),
		}).
		Create(code).Error
}

func (r *deliveryCodeRepo) LockByOrder(ctx context.Context, tx *gorm.DB, orderID uuid.UUID) (*model.DeliveryCode, error) {
	var code model.DeliveryCode
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("order_id = ? AND used_at IS NULL AND expires_at > NOW()", orderID).
		First(&code).Error
	return &code, err
}

func (r *deliveryCodeRepo) IncrementAttempts(ctx context.Context, tx *gorm.DB, id uuid.UUID) error {
	return tx.WithContext(ctx).
		Model(&model.DeliveryCode{}).
		Where("id = ?", id).
		UpdateColumn("attempts", gorm.Expr("attempts + 1")).Error
}

func (r *deliveryCodeRepo) RecordConfirmLocation(ctx context.Context, tx *gorm.DB, orderID uuid.UUID, lat, lng, distanceM *float64, flagged bool) error {
	return tx.WithContext(ctx).
		Model(&model.DeliveryCode{}).
		Where("order_id = ?", orderID).
		Updates(map[string]any{
			"confirm_lat":        lat,
			"confirm_lng":        lng,
			"confirm_distance_m": distanceM,
			"location_flagged":   flagged,
		}).Error
}

func (r *deliveryCodeRepo) MarkUsed(ctx context.Context, tx *gorm.DB, id uuid.UUID, at time.Time) error {
	return tx.WithContext(ctx).
		Model(&model.DeliveryCode{}).
		Where("id = ?", id).
		Update("used_at", at).Error
}
