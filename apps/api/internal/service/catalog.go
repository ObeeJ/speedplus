package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"github.com/speedplus/api/internal/storage"
)

var (
	errForbidden        = errors.New("forbidden")
	ErrPrescriptionUsed = errors.New("prescription already reviewed")
)

const prescriptionViewTTL = 15 * time.Minute

type CatalogService struct {
	repo repo.CatalogRepo
	r2   *storage.R2Client // nil-safe: presigned Rx image views fail closed without it
}

func NewCatalogService(r repo.CatalogRepo, r2 *storage.R2Client) *CatalogService {
	return &CatalogService{repo: r, r2: r2}
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

// ── Merchant prescription review ────────────────────────────────────────────

// PrescriptionView is the API-facing shape: a presigned, time-boxed image URL
// instead of the raw R2 key.
type PrescriptionView struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customerId"`
	ViewURL    string    `json:"viewUrl"`
	Status     string    `json:"status"`
	ReviewNote *string   `json:"reviewNote,omitempty"`
	CreatedAt  string    `json:"createdAt"`
}

// ListPrescriptionsForMerchant returns the merchant's review queue with
// presigned image URLs. status filters (e.g. "pending"); empty returns all.
func (s *CatalogService) ListPrescriptionsForMerchant(ctx context.Context, merchantID uuid.UUID, status string) ([]PrescriptionView, error) {
	rows, err := s.repo.ListPrescriptionsForMerchant(ctx, merchantID, status)
	if err != nil {
		return nil, err
	}
	out := make([]PrescriptionView, 0, len(rows))
	for _, p := range rows {
		view := PrescriptionView{
			ID: p.ID, CustomerID: p.CustomerID, Status: p.Status,
			ReviewNote: p.ReviewNote, CreatedAt: p.CreatedAt.Format(time.RFC3339),
		}
		if s.r2 != nil {
			if url, err := s.r2.PresignGet(ctx, p.R2Key, prescriptionViewTTL); err == nil {
				view.ViewURL = url
			}
		}
		out = append(out, view)
	}
	return out, nil
}

// ReviewPrescription approves or rejects a pending prescription. Ownership
// (the prescription's merchant matches the caller) and idempotency (already
// reviewed can't be re-reviewed) are enforced here — this is the gate
// OrderService.Create relies on before letting a pharmacy order through.
func (s *CatalogService) ReviewPrescription(ctx context.Context, reviewerUserID, merchantID, prescriptionID uuid.UUID, approve bool, note *string) (*model.Prescription, error) {
	p, err := s.repo.GetPrescriptionByID(ctx, prescriptionID)
	if err != nil {
		return nil, err
	}
	if p.MerchantID == nil || *p.MerchantID != merchantID {
		return nil, errForbidden
	}
	if p.Status != "pending" {
		return nil, ErrPrescriptionUsed
	}
	if approve {
		p.Status = "approved"
	} else {
		p.Status = "rejected"
	}
	p.ReviewerID = &reviewerUserID
	p.ReviewNote = note
	return p, s.repo.UpdatePrescription(ctx, p)
}
