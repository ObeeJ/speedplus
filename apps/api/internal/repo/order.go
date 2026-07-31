package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FillAccuracyRow is the per-merchant result of the fill-accuracy aggregation query.
type FillAccuracyRow struct {
	MerchantID  string
	AvgAccuracy float64
	SampleCount int
}

// CylinderRecertRow is one cylinder approaching recertification expiry.
type CylinderRecertRow struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Serial       string
	LastRecertAt *time.Time
}

type OrderRepo interface {
	// ── single-row reads ────────────────────────────────────────────────────
	FindByID(ctx context.Context, id uuid.UUID) (*model.Order, error)
	FindByIDWithItems(ctx context.Context, id uuid.UUID) (*model.Order, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*model.Order, error)
	FindPrescription(ctx context.Context, id uuid.UUID) (*model.Prescription, error)
	// ConsumePrescriptionTx atomically flips an approved, unexpired,
	// merchant-matched, customer-owned prescription to 'consumed' and binds it
	// to orderID — the DB row lock (not a Go-level check) is what prevents two
	// concurrent order creations from both succeeding off the same Rx. Returns
	// rows affected: 0 means the precondition failed (already used, wrong
	// merchant, wrong customer, expired, or not approved).
	ConsumePrescriptionTx(ctx context.Context, tx *gorm.DB, prescriptionID, customerID, merchantID, orderID uuid.UUID) (int64, error)
	FindMerchant(ctx context.Context, id uuid.UUID) (*model.Merchant, error)
	FindAddress(ctx context.Context, id uuid.UUID) (*model.Address, error)
	FindDriverProfile(ctx context.Context, driverID uuid.UUID) (*model.DriverProfile, error)
	FindQuote(ctx context.Context, id uuid.UUID) (*model.PricingQuote, error)

	// ── list reads ──────────────────────────────────────────────────────────
	ListByCustomer(ctx context.Context, customerID uuid.UUID, cursor *uuid.UUID, limit int) ([]model.Order, error)
	ListByMerchant(ctx context.Context, merchantID uuid.UUID, cursor *uuid.UUID, limit int) ([]model.Order, error)
	ListByDriver(ctx context.Context, driverID uuid.UUID, cursor *uuid.UUID, limit int) ([]model.Order, error)
	ListByCustomerFiltered(ctx context.Context, customerID uuid.UUID, vertical, status string, cursor *uuid.UUID, limit int) ([]model.Order, error)
	ListByMerchantFiltered(ctx context.Context, merchantID uuid.UUID, status string, cursor *uuid.UUID, limit int) ([]model.Order, error)
	ListEvents(ctx context.Context, orderID uuid.UUID) ([]model.OrderEvent, error)
	ListStops(ctx context.Context, orderID uuid.UUID) ([]model.OrderStop, error)
	FindDriverBadges(ctx context.Context, driverID uuid.UUID) ([]model.DriverBadge, error)
	FindReviewByOrderAndReviewer(ctx context.Context, orderID, reviewerID uuid.UUID) (*model.OrderReview, error)
	FindStaleRecipientPIIOrders(ctx context.Context, cutoff interface{}, frozenStatus model.EscrowStatus) ([]uuid.UUID, error)

	// ── writes (outside tx) ─────────────────────────────────────────────────
	Save(ctx context.Context, o *model.Order) error
	CreateEvent(ctx context.Context, e *model.OrderEvent) error
	CreateQuote(ctx context.Context, q *model.PricingQuote) error
	MarkQuoteUsed(ctx context.Context, id uuid.UUID) error
	CreateReview(ctx context.Context, r *model.OrderReview) error
	UpsertBadge(ctx context.Context, badge *model.DriverBadge) error

	// ── aggregate reads ─────────────────────────────────────────────────────
	CountDeliveredByDriver(ctx context.Context, driverID uuid.UUID) (int64, error)
	AverageRating(ctx context.Context, revieweeID uuid.UUID, revieweeType string) (float64, error)
	CountReviews(ctx context.Context, revieweeID uuid.UUID, revieweeType string) (int64, error)

	// ── aggregate writes ────────────────────────────────────────────────────
	UpdateDriverRating(ctx context.Context, driverID uuid.UUID, avg float64) error
	UpdateMerchantRating(ctx context.Context, merchantID uuid.UUID, avg float64) error

	// ── transactional writes (caller supplies tx) ───────────────────────────
	// LockForUpdate returns the order with SELECT FOR UPDATE — use inside a tx.
	LockForUpdate(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.Order, error)
	CreateTx(ctx context.Context, tx *gorm.DB, o *model.Order) error
	CreateStopsTx(ctx context.Context, tx *gorm.DB, stops []model.OrderStop) error
	SaveTx(ctx context.Context, tx *gorm.DB, o *model.Order) error
	SaveStopTx(ctx context.Context, tx *gorm.DB, stop *model.OrderStop) error
	CreateEventTx(ctx context.Context, tx *gorm.DB, e *model.OrderEvent) error
	CreateCustodyEventTx(ctx context.Context, tx *gorm.DB, e *model.CylinderCustodyEvent) error
	PurgeRecipientPIITx(ctx context.Context, tx *gorm.DB, orderIDs []uuid.UUID) error

	// ── gas fill accuracy ───────────────────────────────────────────────────
	GasFillAccuracyStats(ctx context.Context) ([]FillAccuracyRow, error)
	UpdateMerchantFillAccuracy(ctx context.Context, merchantID string, avgAccuracy float64, sampleCount int, fillStatus string) error

	// ── zone launch status ───────────────────────────────────────────────────
	FindZoneLaunchStatus(ctx context.Context, lat, lng float64) (string, error)

	// ── cylinder recertification ────────────────────────────────────────────
	FindCylindersNearRecert(ctx context.Context, cutoff time.Time, periodDays int) ([]CylinderRecertRow, error)

	// ── cylinder registry ────────────────────────────────────────────────────
	FindCylinder(ctx context.Context, id uuid.UUID) (*model.CustomerCylinder, error)
	ListCylinders(ctx context.Context, userID uuid.UUID) ([]model.CustomerCylinder, error)
	CreateCylinder(ctx context.Context, c *model.CustomerCylinder) error
	UpdateCylinderStatus(ctx context.Context, id uuid.UUID, status string) error
	ListCylinderSpecs(ctx context.Context) ([]model.CylinderSpec, error)

	// ── transaction runner ──────────────────────────────────────────────────
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
}

type orderRepo struct{ db *gorm.DB }

func NewOrderRepo(db *gorm.DB) OrderRepo { return &orderRepo{db: db} }

func (r *orderRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

// ── single-row reads ─────────────────────────────────────────────────────────

func (r *orderRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	var o model.Order
	err := r.db.WithContext(ctx).First(&o, id).Error
	return &o, err
}

func (r *orderRepo) FindByIDWithItems(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	var o model.Order
	err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Events").
		Preload("Stops").
		First(&o, id).Error
	return &o, err
}

func (r *orderRepo) FindByIdempotencyKey(ctx context.Context, key string) (*model.Order, error) {
	var o model.Order
	err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&o).Error
	return &o, err
}

func (r *orderRepo) FindPrescription(ctx context.Context, id uuid.UUID) (*model.Prescription, error) {
	var p model.Prescription
	err := r.db.WithContext(ctx).First(&p, id).Error
	return &p, err
}

func (r *orderRepo) ConsumePrescriptionTx(ctx context.Context, tx *gorm.DB, prescriptionID, customerID, merchantID, orderID uuid.UUID) (int64, error) {
	res := tx.WithContext(ctx).Model(&model.Prescription{}).
		Where("id = ? AND customer_id = ? AND merchant_id = ? AND status = 'approved' AND (expires_at IS NULL OR expires_at > now())",
			prescriptionID, customerID, merchantID).
		Updates(map[string]interface{}{"status": "consumed", "consumed_order_id": orderID})
	return res.RowsAffected, res.Error
}

func (r *orderRepo) FindMerchant(ctx context.Context, id uuid.UUID) (*model.Merchant, error) {
	var m model.Merchant
	err := r.db.WithContext(ctx).First(&m, id).Error
	return &m, err
}

func (r *orderRepo) FindAddress(ctx context.Context, id uuid.UUID) (*model.Address, error) {
	var a model.Address
	err := r.db.WithContext(ctx).First(&a, id).Error
	return &a, err
}

func (r *orderRepo) FindDriverProfile(ctx context.Context, driverID uuid.UUID) (*model.DriverProfile, error) {
	var dp model.DriverProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", driverID).First(&dp).Error
	return &dp, err
}

func (r *orderRepo) FindQuote(ctx context.Context, id uuid.UUID) (*model.PricingQuote, error) {
	var q model.PricingQuote
	err := r.db.WithContext(ctx).First(&q, id).Error
	return &q, err
}

// ── list reads ───────────────────────────────────────────────────────────────

func (r *orderRepo) ListByCustomer(ctx context.Context, customerID uuid.UUID, cursor *uuid.UUID, limit int) ([]model.Order, error) {
	return r.listWithCursor(ctx, "customer_id = ?", customerID, cursor, limit)
}

func (r *orderRepo) ListByMerchant(ctx context.Context, merchantID uuid.UUID, cursor *uuid.UUID, limit int) ([]model.Order, error) {
	return r.listWithCursor(ctx, "merchant_id = ?", merchantID, cursor, limit)
}

func (r *orderRepo) ListByDriver(ctx context.Context, driverID uuid.UUID, cursor *uuid.UUID, limit int) ([]model.Order, error) {
	return r.listWithCursor(ctx, "driver_id = ?", driverID, cursor, limit)
}

func (r *orderRepo) listWithCursor(ctx context.Context, where string, arg interface{}, cursor *uuid.UUID, limit int) ([]model.Order, error) {
	q := r.db.WithContext(ctx).Where(where, arg).Order("created_at DESC, id DESC").Limit(limit)
	if cursor != nil {
		var pivot model.Order
		if err := r.db.WithContext(ctx).First(&pivot, cursor).Error; err == nil {
			q = q.Where("(created_at, id) < (?, ?)", pivot.CreatedAt, pivot.ID)
		}
	}
	var orders []model.Order
	return orders, q.Find(&orders).Error
}

func (r *orderRepo) ListByCustomerFiltered(ctx context.Context, customerID uuid.UUID, vertical, status string, cursor *uuid.UUID, limit int) ([]model.Order, error) {
	q := r.db.WithContext(ctx).
		Preload("Items").
		Where("customer_id = ?", customerID).
		Order("created_at DESC, id DESC").
		Limit(limit)
	if vertical != "" {
		q = q.Where("vertical = ?", vertical)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if cursor != nil {
		var pivot model.Order
		if err := r.db.WithContext(ctx).First(&pivot, cursor).Error; err == nil {
			q = q.Where("(created_at, id) < (?, ?)", pivot.CreatedAt, pivot.ID)
		}
	}
	var orders []model.Order
	return orders, q.Find(&orders).Error
}

func (r *orderRepo) ListByMerchantFiltered(ctx context.Context, merchantID uuid.UUID, status string, cursor *uuid.UUID, limit int) ([]model.Order, error) {
	q := r.db.WithContext(ctx).
		Preload("Items").
		Where("merchant_id = ?", merchantID).
		Order("created_at DESC, id DESC").
		Limit(limit)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if cursor != nil {
		var pivot model.Order
		if err := r.db.WithContext(ctx).First(&pivot, cursor).Error; err == nil {
			q = q.Where("(created_at, id) < (?, ?)", pivot.CreatedAt, pivot.ID)
		}
	}
	var orders []model.Order
	return orders, q.Find(&orders).Error
}

func (r *orderRepo) ListEvents(ctx context.Context, orderID uuid.UUID) ([]model.OrderEvent, error) {
	var events []model.OrderEvent
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).Order("created_at ASC").Find(&events).Error
	return events, err
}

func (r *orderRepo) ListStops(ctx context.Context, orderID uuid.UUID) ([]model.OrderStop, error) {
	var stops []model.OrderStop
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).Order("sequence ASC").Find(&stops).Error
	return stops, err
}

func (r *orderRepo) FindDriverBadges(ctx context.Context, driverID uuid.UUID) ([]model.DriverBadge, error) {
	var badges []model.DriverBadge
	err := r.db.WithContext(ctx).Where("driver_id = ?", driverID).Order("awarded_at ASC").Find(&badges).Error
	return badges, err
}

func (r *orderRepo) FindReviewByOrderAndReviewer(ctx context.Context, orderID, reviewerID uuid.UUID) (*model.OrderReview, error) {
	var rev model.OrderReview
	err := r.db.WithContext(ctx).Where("order_id = ? AND reviewer_id = ?", orderID, reviewerID).First(&rev).Error
	return &rev, err
}

func (r *orderRepo) FindStaleRecipientPIIOrders(ctx context.Context, cutoff interface{}, frozenStatus model.EscrowStatus) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("delivered_at IS NOT NULL AND delivered_at < ?", cutoff).
		Where("recipient_name_enc IS NOT NULL OR recipient_phone_enc IS NOT NULL").
		Where(`NOT EXISTS (
			SELECT 1 FROM escrow_holds eh
			WHERE eh.order_id = orders.id AND eh.status = ?
		)`, frozenStatus).
		Pluck("id", &ids).Error
	return ids, err
}

// ── writes (outside tx) ──────────────────────────────────────────────────────

func (r *orderRepo) Save(ctx context.Context, o *model.Order) error {
	return r.db.WithContext(ctx).Save(o).Error
}

func (r *orderRepo) CreateEvent(ctx context.Context, e *model.OrderEvent) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *orderRepo) CreateQuote(ctx context.Context, q *model.PricingQuote) error {
	return r.db.WithContext(ctx).Create(q).Error
}

func (r *orderRepo) MarkQuoteUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.PricingQuote{}).
		Where("id = ?", id).
		Update("used_at", "NOW()").Error
}

func (r *orderRepo) CreateReview(ctx context.Context, rev *model.OrderReview) error {
	return r.db.WithContext(ctx).Create(rev).Error
}

func (r *orderRepo) UpsertBadge(ctx context.Context, badge *model.DriverBadge) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "driver_id"}, {Name: "badge_type"}},
			DoNothing: true,
		}).
		Create(badge).Error
}

// ── aggregate reads ──────────────────────────────────────────────────────────

func (r *orderRepo) CountDeliveredByDriver(ctx context.Context, driverID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Order{}).
		Where("driver_id = ? AND status = ?", driverID, model.OrderDelivered).
		Count(&count).Error
	return count, err
}

func (r *orderRepo) AverageRating(ctx context.Context, revieweeID uuid.UUID, revieweeType string) (float64, error) {
	var avg float64
	err := r.db.WithContext(ctx).
		Model(&model.OrderReview{}).
		Where("reviewee_id = ? AND reviewee_type = ?", revieweeID, revieweeType).
		Select("COALESCE(AVG(rating), 5.0)").
		Scan(&avg).Error
	return avg, err
}

func (r *orderRepo) CountReviews(ctx context.Context, revieweeID uuid.UUID, revieweeType string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.OrderReview{}).
		Where("reviewee_id = ? AND reviewee_type = ?", revieweeID, revieweeType).
		Count(&count).Error
	return count, err
}

// ── aggregate writes ─────────────────────────────────────────────────────────

func (r *orderRepo) UpdateDriverRating(ctx context.Context, driverID uuid.UUID, avg float64) error {
	return r.db.WithContext(ctx).Model(&model.DriverProfile{}).
		Where("user_id = ?", driverID).
		Update("rating", avg).Error
}

func (r *orderRepo) UpdateMerchantRating(ctx context.Context, merchantID uuid.UUID, avg float64) error {
	return r.db.WithContext(ctx).Model(&model.Merchant{}).
		Where("id = ?", merchantID).
		Update("rating", avg).Error
}

// ── transactional writes ─────────────────────────────────────────────────────

func (r *orderRepo) LockForUpdate(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.Order, error) {
	var o model.Order
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&o, id).Error
	return &o, err
}

func (r *orderRepo) CreateTx(ctx context.Context, tx *gorm.DB, o *model.Order) error {
	return tx.WithContext(ctx).Create(o).Error
}

func (r *orderRepo) CreateStopsTx(ctx context.Context, tx *gorm.DB, stops []model.OrderStop) error {
	return tx.WithContext(ctx).Create(&stops).Error
}

func (r *orderRepo) SaveTx(ctx context.Context, tx *gorm.DB, o *model.Order) error {
	return tx.WithContext(ctx).Save(o).Error
}

func (r *orderRepo) SaveStopTx(ctx context.Context, tx *gorm.DB, stop *model.OrderStop) error {
	return tx.WithContext(ctx).Save(stop).Error
}

func (r *orderRepo) CreateEventTx(ctx context.Context, tx *gorm.DB, e *model.OrderEvent) error {
	return tx.WithContext(ctx).Create(e).Error
}

func (r *orderRepo) CreateCustodyEventTx(ctx context.Context, tx *gorm.DB, e *model.CylinderCustodyEvent) error {
	return tx.WithContext(ctx).Create(e).Error
}

func (r *orderRepo) PurgeRecipientPIITx(ctx context.Context, tx *gorm.DB, orderIDs []uuid.UUID) error {
	if err := tx.WithContext(ctx).Model(&model.Order{}).
		Where("id IN ?", orderIDs).
		Updates(map[string]interface{}{"recipient_name_enc": nil, "recipient_phone_enc": nil}).Error; err != nil {
		return err
	}
	return tx.WithContext(ctx).Model(&model.OrderStop{}).
		Where("order_id IN ?", orderIDs).
		Updates(map[string]interface{}{"recipient_name_enc": nil, "recipient_phone_enc": nil}).Error
}

// ── gas fill accuracy ────────────────────────────────────────────────────────

// GasFillAccuracyStats aggregates each merchant's most recent 30 verified
// fills (a rolling window, not all-time history) so a merchant that
// recalibrates a bad scale can earn back to 'good' as old fills age out of
// the window, rather than being permanently marked by one bad patch.
func (r *orderRepo) GasFillAccuracyStats(ctx context.Context) ([]FillAccuracyRow, error) {
	var rows []FillAccuracyRow
	err := r.db.WithContext(ctx).Raw(`
		WITH ranked AS (
			SELECT
				o.merchant_id,
				pm.measured_kg / NULLIF(oi_agg.ordered_kg, 0) AS accuracy,
				ROW_NUMBER() OVER (PARTITION BY o.merchant_id ORDER BY pm.captured_at DESC) AS rn
			FROM proof_media pm
			JOIN orders o ON o.id = pm.order_id
			JOIN (
				SELECT order_id, SUM(weight_kg * quantity) AS ordered_kg
				FROM order_items
				GROUP BY order_id
			) oi_agg ON oi_agg.order_id = pm.order_id
			WHERE pm.kind = 'weight_photo'
			  AND pm.measured_kg IS NOT NULL
			  AND o.vertical = 'gas'
			  AND o.status = 'delivered'
		)
		SELECT
			merchant_id::text AS merchant_id,
			AVG(accuracy)     AS avg_accuracy,
			COUNT(*)          AS sample_count
		FROM ranked
		WHERE rn <= 30
		GROUP BY merchant_id
	`).Scan(&rows).Error
	return rows, err
}

func (r *orderRepo) UpdateMerchantFillAccuracy(ctx context.Context, merchantID string, avgAccuracy float64, sampleCount int, fillStatus string) error {
	return r.db.WithContext(ctx).Model(&model.Merchant{}).
		Where("id = ?", merchantID).
		Updates(map[string]interface{}{
			"fill_accuracy_pct": avgAccuracy,
			"fill_sample_count": sampleCount,
			"fill_status":       fillStatus,
		}).Error
}

// FindZoneLaunchStatus returns the launch_status of the active zone whose
// boundary contains the given point, or "" with no error if no zone covers
// it — treated by callers as "not launched here", the safe default for a
// point with no zone data at all.
func (r *orderRepo) FindZoneLaunchStatus(ctx context.Context, lat, lng float64) (string, error) {
	var status string
	err := r.db.WithContext(ctx).Raw(`
		SELECT launch_status
		FROM service_zones
		WHERE is_active = true
		  AND ST_Contains(boundary, ST_SetSRID(ST_MakePoint(?, ?), 4326))
		LIMIT 1
	`, lng, lat).Scan(&status).Error
	return status, err
}

func (r *orderRepo) FindCylindersNearRecert(ctx context.Context, cutoff time.Time, periodDays int) ([]CylinderRecertRow, error) {
	var rows []CylinderRecertRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, user_id, serial, last_recert_at
		FROM customer_cylinders
		WHERE status = 'active'
		  AND last_recert_at IS NOT NULL
		  AND last_recert_at + (? * INTERVAL '1 day') <= ?
	`, periodDays, cutoff).Scan(&rows).Error
	return rows, err
}

func (r *orderRepo) FindCylinder(ctx context.Context, id uuid.UUID) (*model.CustomerCylinder, error) {
	var c model.CustomerCylinder
	return &c, r.db.WithContext(ctx).First(&c, id).Error
}

func (r *orderRepo) ListCylinders(ctx context.Context, userID uuid.UUID) ([]model.CustomerCylinder, error) {
	var rows []model.CustomerCylinder
	return rows, r.db.WithContext(ctx).
		Where("user_id = ? AND status != 'retired'", userID).
		Order("created_at DESC").
		Find(&rows).Error
}

func (r *orderRepo) CreateCylinder(ctx context.Context, c *model.CustomerCylinder) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *orderRepo) UpdateCylinderStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.db.WithContext(ctx).Model(&model.CustomerCylinder{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *orderRepo) ListCylinderSpecs(ctx context.Context) ([]model.CylinderSpec, error) {
	var rows []model.CylinderSpec
	return rows, r.db.WithContext(ctx).
		Where("is_active = true").
		Order("size_kg ASC").
		Find(&rows).Error
}
