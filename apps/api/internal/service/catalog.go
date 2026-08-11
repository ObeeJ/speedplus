package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/ports"
	"github.com/speedplus/api/internal/repo"
	"github.com/speedplus/api/internal/storage"
	"github.com/speedplus/api/internal/whatsapp"
)

var (
	errForbidden = errors.New("forbidden")
	// ErrForbidden is the exported alias of errForbidden, for handlers that
	// need to distinguish "not your resource" (403) from other failures
	// (500) — see merchant.go's ReviewPrescription handler, which used to
	// collapse every error into 403 and mask real DB failures.
	ErrForbidden           = errForbidden
	ErrPrescriptionUsed    = errors.New("prescription already reviewed")
	ErrInvalidContentType  = errors.New("unsupported content type for prescription upload")
	ErrMerchantNotPharmacy = errors.New("merchant is not a pharmacy")
	ErrStorageUnavailable  = errors.New("media storage not configured")
)

const (
	prescriptionViewTTL   = 15 * time.Minute
	prescriptionUploadTTL = 5 * time.Minute
	// prescriptionValidityWindow is how long an approved Rx may be used to
	// place an order before it expires. 48h is a placeholder default — this
	// is a business/regulatory decision (PCN guidance, not engineering) that
	// should be confirmed and may need to become admin-configurable per
	// vertical rather than a compile-time constant.
	prescriptionValidityWindow = 48 * time.Hour
)

var validPrescriptionContentTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
	"application/pdf": true,
}

type CatalogService struct {
	repo  repo.CatalogRepo
	r2    *storage.R2Client // nil-safe: presigned Rx image views fail closed without it
	wa    ports.WhatsAppNotifier // nil-safe: prescription_ready WA notification
	users repo.UserRepo          // nil-safe: needed for WA prescription notification
}

func NewCatalogService(r repo.CatalogRepo, r2 *storage.R2Client) *CatalogService {
	return &CatalogService{repo: r, r2: r2}
}

// InjectWhatsApp wires the WhatsApp notifier and user repo for prescription
// approval notifications. Both must be set together to have any effect.
func (s *CatalogService) InjectWhatsApp(wa ports.WhatsAppNotifier, users repo.UserRepo) {
	s.wa = wa
	s.users = users
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

// PresignPrescriptionUpload issues a short-lived direct-upload URL and the
// server-derived object key the client must PUT its file to. The key is never
// accepted from the client (see CreatePrescription) — this is the only way a
// prescription's r2Key comes into existence, so it always points at a real
// uploaded object.
func (s *CatalogService) PresignPrescriptionUpload(ctx context.Context, customerID uuid.UUID, contentType string) (uploadURL, key string, err error) {
	if !validPrescriptionContentTypes[contentType] {
		return "", "", ErrInvalidContentType
	}
	if s.r2 == nil {
		return "", "", ErrStorageUnavailable
	}
	key = fmt.Sprintf("prescriptions/%s/%s", customerID, uuid.New())
	url, err := s.r2.PresignPut(ctx, key, contentType, prescriptionUploadTTL)
	if err != nil {
		return "", "", err
	}
	return url, key, nil
}

// CreatePrescription requires a merchantID resolving to a pharmacy-vertical
// merchant — a prescription with no target pharmacy could never be reviewed
// (ReviewPrescription requires ownership match), so this is enforced at
// creation rather than left to fail silently downstream.
func (s *CatalogService) CreatePrescription(ctx context.Context, customerID uuid.UUID, r2Key string, merchantID uuid.UUID) (*model.Prescription, error) {
	merchant, err := s.repo.GetMerchant(ctx, merchantID)
	if err != nil {
		return nil, fmt.Errorf("merchant not found: %w", err)
	}
	if merchant.Vertical != model.VerticalPharmacy {
		return nil, ErrMerchantNotPharmacy
	}
	p := &model.Prescription{
		ID:         uuid.New(),
		CustomerID: customerID,
		MerchantID: &merchantID,
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
		if s.r2 == nil {
			return nil, ErrStorageUnavailable
		}
		url, err := s.r2.PresignGet(ctx, p.R2Key, prescriptionViewTTL)
		if err != nil {
			// Previously swallowed: a presign failure silently produced an
			// empty viewUrl with HTTP 200, so the pharmacist saw a broken
			// image with no error to explain why. Fail the whole call —
			// better a visible 500 than a silently unreviewable queue.
			return nil, fmt.Errorf("presign prescription %s: %w", p.ID, err)
		}
		view.ViewURL = url
		out = append(out, view)
	}
	return out, nil
}

// ReviewPrescription approves or rejects a pending prescription in one
// conditional UPDATE (ReviewPrescriptionAtomic) — ownership and idempotency
// are enforced by the WHERE clause itself, not a Go-level read-then-write, so
// two concurrent reviews of the same Rx can't both succeed (P9). This is the
// gate OrderService.Create relies on before letting a pharmacy order through.
func (s *CatalogService) ReviewPrescription(ctx context.Context, reviewerUserID, merchantID, prescriptionID uuid.UUID, approve bool, note *string) (*model.Prescription, error) {
	newStatus := "rejected"
	var expiresAt *time.Time
	if approve {
		newStatus = "approved"
		t := time.Now().Add(prescriptionValidityWindow)
		expiresAt = &t
	}
	rows, err := s.repo.ReviewPrescriptionAtomic(ctx, prescriptionID, merchantID, newStatus, reviewerUserID, note, expiresAt)
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		// Ambiguous by design at this layer: not-found, not-owned, and
		// already-reviewed all fail the same WHERE clause. Disambiguate by a
		// cheap follow-up read so the handler can return the right status
		// code instead of collapsing everything to 403.
		p, gerr := s.repo.GetPrescriptionByID(ctx, prescriptionID)
		if gerr != nil {
			return nil, gerr
		}
		if p.MerchantID == nil || *p.MerchantID != merchantID {
			return nil, errForbidden
		}
		return nil, ErrPrescriptionUsed
	}
	p, err := s.repo.GetPrescriptionByID(ctx, prescriptionID)
	if err != nil {
		return nil, err
	}

	// Notify customer on WhatsApp when prescription is approved — best-effort.
	if approve && s.wa != nil && s.users != nil {
		go func(rx *model.Prescription) {
			defer func() {
				if r := recover(); r != nil {
					return
				}
			}()
			customer, err := s.users.FindByID(context.Background(), rx.CustomerID)
			if err != nil || customer.Phone == "" {
				return
			}
			merchant, err := s.repo.GetMerchant(context.Background(), merchantID)
			if err != nil {
				return
			}
			s.wa.PrescriptionReady(whatsapp.NormalisePhone(customer.Phone), merchant.BusinessName)
		}(p)
	}
	return p, nil
}
