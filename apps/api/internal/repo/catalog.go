package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

type CatalogRepo interface {
	ListProducts(ctx context.Context, merchantID uuid.UUID, category string, page, limit int) ([]model.Product, error)
	GetProduct(ctx context.Context, id uuid.UUID) (*model.Product, error)
	SearchProducts(ctx context.Context, query, vertical string, limit int) ([]model.Product, error)
	ListMerchants(ctx context.Context, vertical string, page, limit int) ([]model.Merchant, error)
	GetMerchant(ctx context.Context, id uuid.UUID) (*model.Merchant, error)

	CreatePrescription(ctx context.Context, p *model.Prescription) error
	GetPrescription(ctx context.Context, id, customerID uuid.UUID) (*model.Prescription, error)
	ListPrescriptions(ctx context.Context, customerID uuid.UUID) ([]model.Prescription, error)

	// Merchant catalog management. ListProductsForMerchant includes
	// unavailable items (unlike ListProducts, which is the public/customer view).
	CreateProduct(ctx context.Context, p *model.Product) error
	UpdateProduct(ctx context.Context, p *model.Product) error
	ListProductsForMerchant(ctx context.Context, merchantID uuid.UUID) ([]model.Product, error)

	// Merchant prescription review queue.
	ListPrescriptionsForMerchant(ctx context.Context, merchantID uuid.UUID, status string) ([]model.Prescription, error)
	GetPrescriptionByID(ctx context.Context, id uuid.UUID) (*model.Prescription, error)
	UpdatePrescription(ctx context.Context, p *model.Prescription) error
}

type catalogRepo struct{ db *gorm.DB }

func NewCatalogRepo(db *gorm.DB) CatalogRepo { return &catalogRepo{db: db} }

func (r *catalogRepo) ListProducts(ctx context.Context, merchantID uuid.UUID, category string, page, limit int) ([]model.Product, error) {
	var products []model.Product
	q := r.db.WithContext(ctx).
		Where("merchant_id = ? AND is_available = true", merchantID).
		Order("name ASC").
		Offset(page * limit).
		Limit(limit)
	if category != "" {
		q = q.Where("category = ?", category)
	}
	return products, q.Find(&products).Error
}

func (r *catalogRepo) GetProduct(ctx context.Context, id uuid.UUID) (*model.Product, error) {
	var p model.Product
	err := r.db.WithContext(ctx).First(&p, id).Error
	return &p, err
}

// SearchProducts uses the GIN tsvector index created by the trigger in migration 002.
func (r *catalogRepo) SearchProducts(ctx context.Context, query, vertical string, limit int) ([]model.Product, error) {
	var products []model.Product
	q := r.db.WithContext(ctx).
		Where("search_vec @@ plainto_tsquery('english', ?) AND is_available = true", query).
		Order(gorm.Expr("ts_rank(search_vec, plainto_tsquery('english', ?)) DESC", query)).
		Limit(limit)
	if vertical != "" {
		q = q.Joins("JOIN merchants ON merchants.id = products.merchant_id").
			Where("merchants.vertical = ? AND merchants.status = 'active'", vertical)
	}
	return products, q.Find(&products).Error
}

func (r *catalogRepo) ListMerchants(ctx context.Context, vertical string, page, limit int) ([]model.Merchant, error) {
	var merchants []model.Merchant
	q := r.db.WithContext(ctx).
		Where("status = 'active'").
		Order("rating DESC").
		Offset(page * limit).
		Limit(limit)
	if vertical != "" {
		q = q.Where("vertical = ?", vertical)
	}
	return merchants, q.Find(&merchants).Error
}

func (r *catalogRepo) GetMerchant(ctx context.Context, id uuid.UUID) (*model.Merchant, error) {
	var m model.Merchant
	err := r.db.WithContext(ctx).First(&m, id).Error
	return &m, err
}

func (r *catalogRepo) CreatePrescription(ctx context.Context, p *model.Prescription) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *catalogRepo) GetPrescription(ctx context.Context, id, customerID uuid.UUID) (*model.Prescription, error) {
	var p model.Prescription
	err := r.db.WithContext(ctx).
		Where("id = ? AND customer_id = ?", id, customerID).
		First(&p).Error
	return &p, err
}

func (r *catalogRepo) ListPrescriptions(ctx context.Context, customerID uuid.UUID) ([]model.Prescription, error) {
	var prescriptions []model.Prescription
	err := r.db.WithContext(ctx).
		Where("customer_id = ?", customerID).
		Order("created_at DESC").
		Find(&prescriptions).Error
	return prescriptions, err
}

func (r *catalogRepo) CreateProduct(ctx context.Context, p *model.Product) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *catalogRepo) UpdateProduct(ctx context.Context, p *model.Product) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *catalogRepo) ListProductsForMerchant(ctx context.Context, merchantID uuid.UUID) ([]model.Product, error) {
	var products []model.Product
	err := r.db.WithContext(ctx).
		Where("merchant_id = ?", merchantID).
		Order("created_at DESC").
		Find(&products).Error
	return products, err
}

func (r *catalogRepo) ListPrescriptionsForMerchant(ctx context.Context, merchantID uuid.UUID, status string) ([]model.Prescription, error) {
	q := r.db.WithContext(ctx).
		Where("merchant_id = ?", merchantID).
		Order("created_at DESC")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var prescriptions []model.Prescription
	err := q.Find(&prescriptions).Error
	return prescriptions, err
}

func (r *catalogRepo) GetPrescriptionByID(ctx context.Context, id uuid.UUID) (*model.Prescription, error) {
	var p model.Prescription
	err := r.db.WithContext(ctx).First(&p, id).Error
	return &p, err
}

func (r *catalogRepo) UpdatePrescription(ctx context.Context, p *model.Prescription) error {
	return r.db.WithContext(ctx).Save(p).Error
}
