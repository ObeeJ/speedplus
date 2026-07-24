package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

type DispatchRepo interface {
	UpsertDriverLocation(ctx context.Context, driverID uuid.UUID, lat, lng, heading float64) error
	CreateOffer(ctx context.Context, offer *model.DeliveryOffer) error
	// AtomicAcceptOffer sets driver_id WHERE driver_id IS NULL — race-safe.
	AtomicAcceptOffer(ctx context.Context, offerID, driverID uuid.UUID) (bool, error)
	ExpireStaleOffers(ctx context.Context) error
	AssignDriverToOrder(ctx context.Context, tx *gorm.DB, orderID, driverID uuid.UUID) error

	// NearbyDrivers returns candidates within radiusMetres ordered by distance.
	NearbyDrivers(ctx context.Context, lat, lng, radiusMetres float64, vehicleFilter string, limit int) ([]NearbyDriver, error)

	// KYC admin queue
	CreateKYCCheck(ctx context.Context, check *model.KYCCheck) error
	FindKYCCheck(ctx context.Context, id uuid.UUID) (*model.KYCCheck, error)
	SaveKYCCheck(ctx context.Context, check *model.KYCCheck) error
	ListPendingKYCChecks(ctx context.Context, offset, limit int) ([]model.KYCCheck, error)
	CreateKYCDocument(ctx context.Context, doc *model.KYCDocument) error
}

type NearbyDriver struct {
	DriverID   uuid.UUID
	DistanceKm float64
	Rating     float64
}

type dispatchRepo struct{ db *gorm.DB }

func NewDispatchRepo(db *gorm.DB) DispatchRepo { return &dispatchRepo{db: db} }

func (r *dispatchRepo) UpsertDriverLocation(ctx context.Context, driverID uuid.UUID, lat, lng, heading float64) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO driver_locations (id, driver_id, location, heading, updated_at)
		VALUES (gen_random_uuid(), ?, ST_SetSRID(ST_MakePoint(?, ?), 4326), ?, NOW())
		ON CONFLICT (driver_id) DO UPDATE
		SET location = EXCLUDED.location,
		    heading  = EXCLUDED.heading,
		    updated_at = EXCLUDED.updated_at
	`, driverID, lng, lat, heading).Error
}

func (r *dispatchRepo) CreateOffer(ctx context.Context, offer *model.DeliveryOffer) error {
	return r.db.WithContext(ctx).Create(offer).Error
}

func (r *dispatchRepo) AtomicAcceptOffer(ctx context.Context, offerID, driverID uuid.UUID) (bool, error) {
	result := r.db.WithContext(ctx).
		Model(&model.DeliveryOffer{}).
		Where("id = ? AND driver_id IS NULL AND status = 'pending' AND expires_at > ?", offerID, time.Now()).
		Updates(map[string]interface{}{"driver_id": driverID, "status": "accepted"})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *dispatchRepo) ExpireStaleOffers(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Model(&model.DeliveryOffer{}).
		Where("status = 'pending' AND expires_at < ?", time.Now()).
		Update("status", "expired").Error
}

func (r *dispatchRepo) AssignDriverToOrder(ctx context.Context, tx *gorm.DB, orderID, driverID uuid.UUID) error {
	return tx.WithContext(ctx).
		Model(&model.Order{}).
		Where("id = ? AND driver_id IS NULL", orderID).
		Updates(map[string]interface{}{
			"driver_id": driverID,
			"status":    string(model.OrderDriverAssigned),
		}).Error
}

func (r *dispatchRepo) NearbyDrivers(ctx context.Context, lat, lng, radiusMetres float64, vehicleFilter string, limit int) ([]NearbyDriver, error) {
	type row struct {
		DriverID   uuid.UUID `gorm:"column:driver_id"`
		DistanceKm float64   `gorm:"column:distance_km"`
		Rating     float64   `gorm:"column:rating"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT dp.user_id AS driver_id,
		       ST_Distance(dl.location::geography, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography) / 1000.0 AS distance_km,
		       dp.rating
		FROM driver_locations dl
		JOIN driver_profiles dp ON dp.user_id = dl.driver_id
		WHERE dp.is_online = true
		  AND dp.status = 'approved'
		  AND `+vehicleFilter+`
		  AND ST_DWithin(dl.location::geography, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography, ?)
		  AND dl.updated_at > NOW() - INTERVAL '30 seconds'
		ORDER BY distance_km ASC
		LIMIT ?
	`, lng, lat, lng, lat, radiusMetres, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]NearbyDriver, len(rows))
	for i, r := range rows {
		out[i] = NearbyDriver{DriverID: r.DriverID, DistanceKm: r.DistanceKm, Rating: r.Rating}
	}
	return out, nil
}

func (r *dispatchRepo) CreateKYCCheck(ctx context.Context, check *model.KYCCheck) error {
	return r.db.WithContext(ctx).Create(check).Error
}

func (r *dispatchRepo) FindKYCCheck(ctx context.Context, id uuid.UUID) (*model.KYCCheck, error) {
	var c model.KYCCheck
	err := r.db.WithContext(ctx).First(&c, id).Error
	return &c, err
}

func (r *dispatchRepo) SaveKYCCheck(ctx context.Context, check *model.KYCCheck) error {
	return r.db.WithContext(ctx).Save(check).Error
}

func (r *dispatchRepo) ListPendingKYCChecks(ctx context.Context, offset, limit int) ([]model.KYCCheck, error) {
	var checks []model.KYCCheck
	err := r.db.WithContext(ctx).
		Where("status IN ?", []model.KYCStatus{model.KYCPending, model.KYCReview}).
		Order("created_at ASC").
		Offset(offset).Limit(limit).
		Find(&checks).Error
	return checks, err
}

func (r *dispatchRepo) CreateKYCDocument(ctx context.Context, doc *model.KYCDocument) error {
	return r.db.WithContext(ctx).Create(doc).Error
}
