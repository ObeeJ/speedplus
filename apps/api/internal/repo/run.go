package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

type RunOrderRow struct {
	OrderID    uuid.UUID
	MerchantID uuid.UUID
	Lat        float64
	Lng        float64
}

type RunRepo interface {
	FindZone(ctx context.Context, id uuid.UUID) (*model.ServiceZone, error)
	FindOrdersInZoneWindow(ctx context.Context, boundary string, windowStart, windowEnd time.Time, limit int) ([]RunOrderRow, error)
	FindGasMerchantLocation(ctx context.Context, merchantID uuid.UUID) (lat, lng float64, err error)
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	CreateRunTx(ctx context.Context, tx *gorm.DB, run *model.DeliveryRun) error
	CreateRunOrderTx(ctx context.Context, tx *gorm.DB, ro *model.RunOrder) error
	FindRunWithOrders(ctx context.Context, id uuid.UUID) (*model.DeliveryRun, error)
	UpdateRunStatus(ctx context.Context, runID uuid.UUID, driverID uuid.UUID, status string) error
	SumRunWeightKg(ctx context.Context, runID uuid.UUID) (float64, error)
	ListActiveZones(ctx context.Context) ([]model.ServiceZone, error)
	CountRunsForZoneWindow(ctx context.Context, zoneID uuid.UUID, windowStart time.Time) (int64, error)
}

type runRepo struct{ db *gorm.DB }

func NewRunRepo(db *gorm.DB) RunRepo { return &runRepo{db: db} }

func (r *runRepo) FindZone(ctx context.Context, id uuid.UUID) (*model.ServiceZone, error) {
	var z model.ServiceZone
	err := r.db.WithContext(ctx).First(&z, id).Error
	return &z, err
}

func (r *runRepo) FindOrdersInZoneWindow(ctx context.Context, boundary string, windowStart, windowEnd time.Time, limit int) ([]RunOrderRow, error) {
	var rows []RunOrderRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT o.id AS order_id, o.merchant_id, a.lat, a.lng
		FROM orders o
		JOIN addresses a ON a.id = o.delivery_address_id
		WHERE o.vertical = 'gas'
		  AND o.status   = 'pending'
		  AND o.scheduled_for >= ?
		  AND o.scheduled_for <  ?
		  AND ST_Contains(ST_GeomFromText(?, 4326), ST_SetSRID(ST_MakePoint(a.lng, a.lat), 4326))
		ORDER BY o.scheduled_for ASC
		LIMIT ?
	`, windowStart, windowEnd, boundary, limit).Scan(&rows).Error
	return rows, err
}

func (r *runRepo) FindGasMerchantLocation(ctx context.Context, merchantID uuid.UUID) (float64, float64, error) {
	var lat, lng float64
	row := r.db.WithContext(ctx).Raw(`SELECT lat, lng FROM merchants WHERE id = ? LIMIT 1`, merchantID).Row()
	err := row.Scan(&lat, &lng)
	return lat, lng, err
}

func (r *runRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *runRepo) CreateRunTx(ctx context.Context, tx *gorm.DB, run *model.DeliveryRun) error {
	return tx.WithContext(ctx).Create(run).Error
}

func (r *runRepo) CreateRunOrderTx(ctx context.Context, tx *gorm.DB, ro *model.RunOrder) error {
	return tx.WithContext(ctx).Create(ro).Error
}

func (r *runRepo) FindRunWithOrders(ctx context.Context, id uuid.UUID) (*model.DeliveryRun, error) {
	var run model.DeliveryRun
	err := r.db.WithContext(ctx).Preload("Orders").First(&run, id).Error
	return &run, err
}

func (r *runRepo) UpdateRunStatus(ctx context.Context, runID uuid.UUID, driverID uuid.UUID, status string) error {
	return r.db.WithContext(ctx).Model(&model.DeliveryRun{}).
		Where("id = ? AND status = 'assembling'", runID).
		Updates(map[string]interface{}{"driver_id": driverID, "status": status}).Error
}

func (r *runRepo) SumRunWeightKg(ctx context.Context, runID uuid.UUID) (float64, error) {
	var totalKg float64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(oi.weight_kg * oi.quantity), 0)
		FROM run_orders ro
		JOIN order_items oi ON oi.order_id = ro.order_id
		WHERE ro.run_id = ?
	`, runID).Scan(&totalKg).Error
	return totalKg, err
}

func (r *runRepo) ListActiveZones(ctx context.Context) ([]model.ServiceZone, error) {
	var zones []model.ServiceZone
	err := r.db.WithContext(ctx).Where("is_active = true").Find(&zones).Error
	return zones, err
}

func (r *runRepo) CountRunsForZoneWindow(ctx context.Context, zoneID uuid.UUID, windowStart time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.DeliveryRun{}).
		Where("zone_id = ? AND window_start = ? AND status != 'cancelled'", zoneID, windowStart).
		Count(&count).Error
	return count, err
}
