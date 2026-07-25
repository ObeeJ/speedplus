package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
)

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
