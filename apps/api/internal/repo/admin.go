package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AdminRepo interface {
	ListMerchantProfiles(ctx context.Context, status string, cursor *uuid.UUID, limit int) ([]model.MerchantProfile, error)
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	LockMerchantProfileTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.MerchantProfile, error)
	SaveMerchantProfileTx(ctx context.Context, tx *gorm.DB, mp *model.MerchantProfile) error
	CreateAuditLogTx(ctx context.Context, tx *gorm.DB, log *model.AdminAuditLog) error
	ListDriverProfiles(ctx context.Context, status string, cursor *uuid.UUID, limit int) ([]model.DriverProfile, error)
	LockDriverProfileTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.DriverProfile, error)
	SaveDriverProfileTx(ctx context.Context, tx *gorm.DB, dp *model.DriverProfile) error
	SearchOrders(ctx context.Context, q, status string, cursor *uuid.UUID, limit int) ([]model.Order, error)
	FindOrderWithEvents(ctx context.Context, id uuid.UUID) (*model.Order, error)
	FindOrderTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.Order, error)
	ListCancellationRules(ctx context.Context) ([]model.CancellationRule, error)
	UpsertCancellationRule(ctx context.Context, rule model.CancellationRule) (*model.CancellationRule, error)
	DeleteCancellationRule(ctx context.Context, id uuid.UUID) error
	// Gas: fill_status
	ListGasMerchants(ctx context.Context, fillStatus string, cursor *uuid.UUID, limit int) ([]model.Merchant, error)
	LockMerchantTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.Merchant, error)
	SaveMerchantTx(ctx context.Context, tx *gorm.DB, m *model.Merchant) error
	// Gas: zones / launch_status
	ListZones(ctx context.Context, launchStatus string, cursor *uuid.UUID, limit int) ([]model.ServiceZone, error)
	LockZoneTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.ServiceZone, error)
	SaveZoneTx(ctx context.Context, tx *gorm.DB, z *model.ServiceZone) error
}

type adminRepo struct{ db *gorm.DB }

func NewAdminRepo(db *gorm.DB) AdminRepo { return &adminRepo{db: db} }

func (r *adminRepo) ListMerchantProfiles(ctx context.Context, status string, cursor *uuid.UUID, limit int) ([]model.MerchantProfile, error) {
	q := r.db.WithContext(ctx).Model(&model.MerchantProfile{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if cursor != nil {
		var pivot model.MerchantProfile
		if err := r.db.WithContext(ctx).First(&pivot, cursor).Error; err == nil {
			q = q.Where("(created_at, id) < (?, ?)", pivot.CreatedAt, pivot.ID)
		}
	}
	var rows []model.MerchantProfile
	err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *adminRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *adminRepo) LockMerchantProfileTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.MerchantProfile, error) {
	var mp model.MerchantProfile
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&mp, id).Error
	return &mp, err
}

func (r *adminRepo) SaveMerchantProfileTx(ctx context.Context, tx *gorm.DB, mp *model.MerchantProfile) error {
	return tx.WithContext(ctx).Save(mp).Error
}

func (r *adminRepo) CreateAuditLogTx(ctx context.Context, tx *gorm.DB, log *model.AdminAuditLog) error {
	return tx.WithContext(ctx).Create(log).Error
}

func (r *adminRepo) ListDriverProfiles(ctx context.Context, status string, cursor *uuid.UUID, limit int) ([]model.DriverProfile, error) {
	q := r.db.WithContext(ctx).Model(&model.DriverProfile{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if cursor != nil {
		var pivot model.DriverProfile
		if err := r.db.WithContext(ctx).First(&pivot, cursor).Error; err == nil {
			q = q.Where("(created_at, id) < (?, ?)", pivot.CreatedAt, pivot.ID)
		}
	}
	var rows []model.DriverProfile
	err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *adminRepo) LockDriverProfileTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.DriverProfile, error) {
	var dp model.DriverProfile
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&dp, id).Error
	return &dp, err
}

func (r *adminRepo) SaveDriverProfileTx(ctx context.Context, tx *gorm.DB, dp *model.DriverProfile) error {
	return tx.WithContext(ctx).Save(dp).Error
}

func (r *adminRepo) SearchOrders(ctx context.Context, q, status string, cursor *uuid.UUID, limit int) ([]model.Order, error) {
	db := r.db.WithContext(ctx).Model(&model.Order{})
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if q != "" {
		db = db.Where("CAST(id AS TEXT) ILIKE ? OR CAST(customer_id AS TEXT) = ?", q+"%", q)
	}
	if cursor != nil {
		var pivot model.Order
		if err := r.db.WithContext(ctx).First(&pivot, cursor).Error; err == nil {
			db = db.Where("(created_at, id) < (?, ?)", pivot.CreatedAt, pivot.ID)
		}
	}
	var rows []model.Order
	err := db.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *adminRepo) FindOrderWithEvents(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	var o model.Order
	err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Events", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		First(&o, id).Error
	return &o, err
}

func (r *adminRepo) FindOrderTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.Order, error) {
	var o model.Order
	err := tx.WithContext(ctx).First(&o, id).Error
	return &o, err
}

func (r *adminRepo) ListCancellationRules(ctx context.Context) ([]model.CancellationRule, error) {
	var rules []model.CancellationRule
	err := r.db.WithContext(ctx).Order("vertical ASC, order_status_at_cancel ASC").Find(&rules).Error
	return rules, err
}

func (r *adminRepo) UpsertCancellationRule(ctx context.Context, rule model.CancellationRule) (*model.CancellationRule, error) {
	err := r.db.WithContext(ctx).
		Where("vertical = ? AND order_status_at_cancel = ?", rule.Vertical, rule.OrderStatusAtCancel).
		Assign(rule).FirstOrCreate(&rule).Error
	return &rule, err
}

func (r *adminRepo) DeleteCancellationRule(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.CancellationRule{}, id).Error
}

func (r *adminRepo) ListGasMerchants(ctx context.Context, fillStatus string, cursor *uuid.UUID, limit int) ([]model.Merchant, error) {
	q := r.db.WithContext(ctx).Model(&model.Merchant{}).Where("vertical = 'gas'")
	if fillStatus != "" {
		q = q.Where("fill_status = ?", fillStatus)
	}
	if cursor != nil {
		var pivot model.Merchant
		if err := r.db.WithContext(ctx).First(&pivot, cursor).Error; err == nil {
			q = q.Where("(created_at, id) < (?, ?)", pivot.CreatedAt, pivot.ID)
		}
	}
	var rows []model.Merchant
	err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *adminRepo) LockMerchantTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.Merchant, error) {
	var m model.Merchant
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&m, id).Error
	return &m, err
}

func (r *adminRepo) SaveMerchantTx(ctx context.Context, tx *gorm.DB, m *model.Merchant) error {
	return tx.WithContext(ctx).Save(m).Error
}

func (r *adminRepo) ListZones(ctx context.Context, launchStatus string, cursor *uuid.UUID, limit int) ([]model.ServiceZone, error) {
	q := r.db.WithContext(ctx).Model(&model.ServiceZone{})
	if launchStatus != "" {
		q = q.Where("launch_status = ?", launchStatus)
	}
	if cursor != nil {
		var pivot model.ServiceZone
		if err := r.db.WithContext(ctx).First(&pivot, cursor).Error; err == nil {
			q = q.Where("name > ?", pivot.Name)
		}
	}
	var rows []model.ServiceZone
	err := q.Order("name ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *adminRepo) LockZoneTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.ServiceZone, error) {
	var z model.ServiceZone
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&z, id).Error
	return &z, err
}

func (r *adminRepo) SaveZoneTx(ctx context.Context, tx *gorm.DB, z *model.ServiceZone) error {
	return tx.WithContext(ctx).Save(z).Error
}
