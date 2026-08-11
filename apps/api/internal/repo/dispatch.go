package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DispatchRepo interface {
	UpsertDriverLocation(ctx context.Context, driverID uuid.UUID, lat, lng, heading float64) error
	CreateOffer(ctx context.Context, offer *model.DeliveryOffer) error
	AtomicAcceptOffer(ctx context.Context, offerID, driverID uuid.UUID) (bool, error)
	ExpireStaleOffers(ctx context.Context) error
	AssignDriverToOrder(ctx context.Context, tx *gorm.DB, orderID, driverID uuid.UUID) error
	NearbyDrivers(ctx context.Context, lat, lng, radiusMetres float64, minVehicle model.VehicleType, requireHazmat bool, limit int) ([]NearbyDriver, error)

	// KYC
	CreateKYCCheck(ctx context.Context, check *model.KYCCheck) error
	FindKYCCheck(ctx context.Context, id uuid.UUID) (*model.KYCCheck, error)
	SaveKYCCheck(ctx context.Context, check *model.KYCCheck) error
	ListPendingKYCChecks(ctx context.Context, offset, limit int) ([]model.KYCCheck, error)
	ListKYCChecksByUser(ctx context.Context, userID uuid.UUID) ([]model.KYCCheck, error)
	CreateKYCDocument(ctx context.Context, doc *model.KYCDocument) error

	// KYC tx helpers (used inside AdminApprove transaction)
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	FindKYCCheckTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.KYCCheck, error)
	SaveKYCCheckTx(ctx context.Context, tx *gorm.DB, check *model.KYCCheck) error
	FindUserTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*model.User, error)
	FindDriverProfileTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*model.DriverProfile, error)
	SaveDriverProfileTx(ctx context.Context, tx *gorm.DB, dp *model.DriverProfile) error
	FindMerchantProfileTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*model.MerchantProfile, error)
	SaveMerchantProfileTx(ctx context.Context, tx *gorm.DB, mp *model.MerchantProfile) error

	// Dispatch tx helpers
	AcceptOfferTx(ctx context.Context, tx *gorm.DB, offerID, driverID uuid.UUID) (int64, error)
	FindOfferTx(ctx context.Context, tx *gorm.DB, offerID uuid.UUID) (*model.DeliveryOffer, error)
	SetDriverID(ctx context.Context, tx *gorm.DB, orderID, driverID uuid.UUID) error
	UpdateOfferStatus(ctx context.Context, offerID, driverID uuid.UUID, status string) error
	LockOrderTx(ctx context.Context, tx *gorm.DB, orderID uuid.UUID) (*model.Order, error)
	SaveOrderTx(ctx context.Context, tx *gorm.DB, o *model.Order) error
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
	result := tx.WithContext(ctx).
		Model(&model.Order{}).
		Where("id = ? AND driver_id IS NULL", orderID).
		Updates(map[string]interface{}{
			"driver_id": driverID,
			"status":    string(model.OrderDriverAssigned),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("order already assigned or not found")
	}
	return nil
}

func (r *dispatchRepo) NearbyDrivers(ctx context.Context, lat, lng, radiusMetres float64, minVehicle model.VehicleType, requireHazmat bool, limit int) ([]NearbyDriver, error) {
	type row struct {
		DriverID   uuid.UUID `gorm:"column:driver_id"`
		DistanceKm float64   `gorm:"column:distance_km"`
		Rating     float64   `gorm:"column:rating"`
	}

	var vehicleTypes []string
	switch minVehicle {
	case model.VehicleVan:
		vehicleTypes = []string{"van"}
	case model.VehicleCar:
		vehicleTypes = []string{"car", "van"}
	default:
		vehicleTypes = []string{"motorcycle", "car", "van"}
	}

	hazmatClause := ""
	if requireHazmat {
		hazmatClause = "AND dp.hazmat_certified = true"
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
		  AND dp.vehicle_type = ANY(?)
		  `+hazmatClause+`
		  AND ST_DWithin(dl.location::geography, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography, ?)
		  AND dl.updated_at > NOW() - INTERVAL '30 seconds'
		ORDER BY distance_km ASC
		LIMIT ?
	`, lng, lat, pq.Array(vehicleTypes), lng, lat, radiusMetres, limit).Scan(&rows).Error
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

func (r *dispatchRepo) ListKYCChecksByUser(ctx context.Context, userID uuid.UUID) ([]model.KYCCheck, error) {
	var checks []model.KYCCheck
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&checks).Error
	return checks, err
}

func (r *dispatchRepo) CreateKYCDocument(ctx context.Context, doc *model.KYCDocument) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

func (r *dispatchRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *dispatchRepo) FindKYCCheckTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.KYCCheck, error) {
	var c model.KYCCheck
	err := tx.WithContext(ctx).First(&c, id).Error
	return &c, err
}

func (r *dispatchRepo) SaveKYCCheckTx(ctx context.Context, tx *gorm.DB, check *model.KYCCheck) error {
	return tx.WithContext(ctx).Save(check).Error
}

func (r *dispatchRepo) FindUserTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*model.User, error) {
	db := r.db
	if tx != nil {
		db = tx
	}
	var u model.User
	err := db.WithContext(ctx).First(&u, userID).Error
	return &u, err
}

func (r *dispatchRepo) FindDriverProfileTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*model.DriverProfile, error) {
	var dp model.DriverProfile
	err := tx.WithContext(ctx).Where("user_id = ?", userID).First(&dp).Error
	return &dp, err
}

func (r *dispatchRepo) SaveDriverProfileTx(ctx context.Context, tx *gorm.DB, dp *model.DriverProfile) error {
	return tx.WithContext(ctx).Save(dp).Error
}

func (r *dispatchRepo) FindMerchantProfileTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*model.MerchantProfile, error) {
	var mp model.MerchantProfile
	err := tx.WithContext(ctx).Where("user_id = ?", userID).First(&mp).Error
	return &mp, err
}

func (r *dispatchRepo) SaveMerchantProfileTx(ctx context.Context, tx *gorm.DB, mp *model.MerchantProfile) error {
	return tx.WithContext(ctx).Save(mp).Error
}

func (r *dispatchRepo) AcceptOfferTx(ctx context.Context, tx *gorm.DB, offerID, driverID uuid.UUID) (int64, error) {
	result := tx.WithContext(ctx).Model(&model.DeliveryOffer{}).
		Where("id = ? AND driver_id = ? AND status = 'pending' AND expires_at > NOW()", offerID, driverID).
		Update("status", "accepted")
	return result.RowsAffected, result.Error
}

func (r *dispatchRepo) FindOfferTx(ctx context.Context, tx *gorm.DB, offerID uuid.UUID) (*model.DeliveryOffer, error) {
	var o model.DeliveryOffer
	err := tx.WithContext(ctx).First(&o, offerID).Error
	return &o, err
}

// SetDriverID sets driver_id on an order without touching status.
// Used by AcceptOffer so the ownership check in transitionTx can pass.
func (r *dispatchRepo) SetDriverID(ctx context.Context, tx *gorm.DB, orderID, driverID uuid.UUID) error {
	return tx.WithContext(ctx).
		Model(&model.Order{}).
		Where("id = ? AND driver_id IS NULL", orderID).
		Update("driver_id", driverID).Error
}

func (r *dispatchRepo) UpdateOfferStatus(ctx context.Context, offerID, driverID uuid.UUID, status string) error {
	return r.db.WithContext(ctx).Model(&model.DeliveryOffer{}).
		Where("id = ? AND driver_id = ? AND status = 'pending'", offerID, driverID).
		Update("status", status).Error
}

func (r *dispatchRepo) LockOrderTx(ctx context.Context, tx *gorm.DB, orderID uuid.UUID) (*model.Order, error) {
	var o model.Order
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&o, orderID).Error
	return &o, err
}

func (r *dispatchRepo) SaveOrderTx(ctx context.Context, tx *gorm.DB, o *model.Order) error {
	return tx.WithContext(ctx).Save(o).Error
}
