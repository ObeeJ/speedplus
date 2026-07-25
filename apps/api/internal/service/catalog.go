package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
)

var errForbidden = errors.New("forbidden")

type CatalogService struct {
	repo repo.CatalogRepo
}

func NewCatalogService(r repo.CatalogRepo) *CatalogService {
	return &CatalogService{repo: r}
}

func (s *CatalogService) ListProducts(ctx context.Context, merchantID uuid.UUID, category string, page, limit int) ([]model.Product, error) {
	return s.repo.ListProducts(ctx, merchantID, category, page, limit)
}

func (s *CatalogService) GetProduct(ctx context.Context, id uuid.UUID) (*model.Product, error) {
	return s.repo.GetProduct(ctx, id)
}

func (s *CatalogService) SearchProducts(ctx context.Context, query, vertical string) ([]model.Product, error) {
	return s.repo.SearchProducts(ctx, query, vertical, 40)
}

func (s *CatalogService) ListMerchants(ctx context.Context, vertical string, page, limit int) ([]model.Merchant, error) {
	return s.repo.ListMerchants(ctx, vertical, page, limit)
}

func (s *CatalogService) GetMerchant(ctx context.Context, id uuid.UUID) (*model.Merchant, error) {
	return s.repo.GetMerchant(ctx, id)
}

func (s *CatalogService) CreatePrescription(ctx context.Context, customerID uuid.UUID, r2Key string, merchantID *uuid.UUID) (*model.Prescription, error) {
	p := &model.Prescription{
		ID:         uuid.New(),
		CustomerID: customerID,
		MerchantID: merchantID,
		R2Key:      r2Key,
		Status:     "pending",
	}
	return p, s.repo.CreatePrescription(ctx, p)
}

func (s *CatalogService) GetPrescription(ctx context.Context, id, customerID uuid.UUID) (*model.Prescription, error) {
	return s.repo.GetPrescription(ctx, id, customerID)
}

func (s *CatalogService) ListPrescriptions(ctx context.Context, customerID uuid.UUID) ([]model.Prescription, error) {
	return s.repo.ListPrescriptions(ctx, customerID)
}

// ── Merchant catalog management ─────────────────────────────────────────────

type ProductInput struct {
	Name        string
	Description *string
	PriceKobo   int64
	Category    string
	IsAvailable bool
}

func (s *CatalogService) CreateProduct(ctx context.Context, merchantID uuid.UUID, in ProductInput) (*model.Product, error) {
	p := &model.Product{
		ID:          uuid.New(),
		MerchantID:  merchantID,
		Name:        in.Name,
		Description: in.Description,
		PriceKobo:   in.PriceKobo,
		Category:    in.Category,
		IsAvailable: in.IsAvailable,
	}
	return p, s.repo.CreateProduct(ctx, p)
}

// UpdateProduct edits a product, enforcing that the caller's merchant owns it
// — without this check any merchant could edit any other merchant's listing.
func (s *CatalogService) UpdateProduct(ctx context.Context, merchantID, productID uuid.UUID, in ProductInput) (*model.Product, error) {
	p, err := s.repo.GetProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	if p.MerchantID != merchantID {
		return nil, errForbidden
	}
	p.Name = in.Name
	p.Description = in.Description
	p.PriceKobo = in.PriceKobo
	p.Category = in.Category
	p.IsAvailable = in.IsAvailable
	return p, s.repo.UpdateProduct(ctx, p)
}

// SetProductAvailability is the merchant "delete" path — soft-disable rather
// than a hard delete, since historical OrderItems reference the product ID.
func (s *CatalogService) SetProductAvailability(ctx context.Context, merchantID, productID uuid.UUID, available bool) error {
	p, err := s.repo.GetProduct(ctx, productID)
	if err != nil {
		return err
	}
	if p.MerchantID != merchantID {
		return errForbidden
	}
	p.IsAvailable = available
	return s.repo.UpdateProduct(ctx, p)
}

func (s *CatalogService) ListProductsForMerchant(ctx context.Context, merchantID uuid.UUID) ([]model.Product, error) {
	return s.repo.ListProductsForMerchant(ctx, merchantID)
}
