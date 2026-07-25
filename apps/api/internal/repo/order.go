package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OrderRepo interface {
	Create(ctx context.Context, o *model.Order) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Order, error)
	FindByIDWithItems(ctx context.Context, id uuid.UUID) (*model.Order, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*model.Order, error)
	Save(ctx context.Context, o *model.Order) error
	// LockForUpdate returns the order with SELECT FOR UPDATE — use inside a tx.
	LockForUpdate(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.Order, error)

	ListByCustomer(ctx context.Context, customerID uuid.UUID, cursor *uuid.UUID, limit int) ([]model.Order, error)
	ListByMerchant(ctx context.Context, merchantID uuid.UUID, cursor *uuid.UUID, limit int) ([]model.Order, error)
	ListByDriver(ctx context.Context, driverID uuid.UUID, cursor *uuid.UUID, limit int) ([]model.Order, error)

	CreateEvent(ctx context.Context, e *model.OrderEvent) error
	ListEvents(ctx context.Context, orderID uuid.UUID) ([]model.OrderEvent, error)

	FindMerchant(ctx context.Context, id uuid.UUID) (*model.Merchant, error)
	FindQuote(ctx context.Context, id uuid.UUID) (*model.PricingQuote, error)
	MarkQuoteUsed(ctx context.Context, id uuid.UUID) error
	CreateQuote(ctx context.Context, q *model.PricingQuote) error

	CreatePrescription(ctx context.Context, p *model.Prescription) error
	FindPrescription(ctx context.Context, id uuid.UUID) (*model.Prescription, error)
	UpdatePrescription(ctx context.Context, p *model.Prescription) error
}

type orderRepo struct{ db *gorm.DB }

func NewOrderRepo(db *gorm.DB) OrderRepo { return &orderRepo{db: db} }

func (r *orderRepo) Create(ctx context.Context, o *model.Order) error {
	return r.db.WithContext(ctx).Create(o).Error
}

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

func (r *orderRepo) Save(ctx context.Context, o *model.Order) error {
	return r.db.WithContext(ctx).Save(o).Error
}

func (r *orderRepo) LockForUpdate(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.Order, error) {
	var o model.Order
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&o, id).Error
	return &o, err
}

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
	q := r.db.WithContext(ctx).Where(where, arg).Order("created_at DESC").Limit(limit)
	if cursor != nil {
		var pivot model.Order
		if err := r.db.WithContext(ctx).First(&pivot, cursor).Error; err == nil {
			q = q.Where("created_at < ?", pivot.CreatedAt)
		}
	}
	var orders []model.Order
	return orders, q.Find(&orders).Error
}

func (r *orderRepo) CreateEvent(ctx context.Context, e *model.OrderEvent) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *orderRepo) ListEvents(ctx context.Context, orderID uuid.UUID) ([]model.OrderEvent, error) {
	var events []model.OrderEvent
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).Order("created_at ASC").Find(&events).Error
	return events, err
}

func (r *orderRepo) FindMerchant(ctx context.Context, id uuid.UUID) (*model.Merchant, error) {
	var m model.Merchant
	err := r.db.WithContext(ctx).First(&m, id).Error
	return &m, err
}

func (r *orderRepo) FindQuote(ctx context.Context, id uuid.UUID) (*model.PricingQuote, error) {
	var q model.PricingQuote
	err := r.db.WithContext(ctx).First(&q, id).Error
	return &q, err
}

func (r *orderRepo) MarkQuoteUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.PricingQuote{}).
		Where("id = ?", id).
		Update("used_at", "NOW()").Error
}

func (r *orderRepo) CreateQuote(ctx context.Context, q *model.PricingQuote) error {
	return r.db.WithContext(ctx).Create(q).Error
}

func (r *orderRepo) CreatePrescription(ctx context.Context, p *model.Prescription) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *orderRepo) FindPrescription(ctx context.Context, id uuid.UUID) (*model.Prescription, error) {
	var p model.Prescription
	err := r.db.WithContext(ctx).First(&p, id).Error
	return &p, err
}

func (r *orderRepo) UpdatePrescription(ctx context.Context, p *model.Prescription) error {
	return r.db.WithContext(ctx).Save(p).Error
}
