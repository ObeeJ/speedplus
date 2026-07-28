package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/storage"
	"gorm.io/gorm"
)

const (
	proofUploadTTL = 5 * time.Minute  // window to actually PUT the file after presigning
	proofViewTTL   = 15 * time.Minute // window a viewer's presigned GET stays valid
)

var validProofKinds = map[string]bool{
	"pickup_photo": true, "pickup_video": true,
	"dropoff_photo": true, "dropoff_video": true,
}

// ProofMediaService issues presigned upload/view URLs for chain-of-custody
// media and records the append-only evidence row once a capture is confirmed.
// r2 may be nil (R2 not configured) — presign calls fail closed in that case
// rather than silently skipping the evidence requirement.
type ProofMediaService struct {
	db *gorm.DB
	r2 *storage.R2Client
}

func NewProofMediaService(db *gorm.DB, r2 *storage.R2Client) *ProofMediaService {
	return &ProofMediaService{db: db, r2: r2}
}

// mustBeAssignedDriver loads the order and returns an error unless driverID
// is the currently assigned driver — the only actor allowed to capture proof.
func (s *ProofMediaService) mustBeAssignedDriver(ctx context.Context, orderID, driverID uuid.UUID) (*model.Order, error) {
	var order model.Order
	if err := s.db.WithContext(ctx).First(&order, orderID).Error; err != nil {
		return nil, fmt.Errorf("order not found")
	}
	if order.DriverID == nil || *order.DriverID != driverID {
		return nil, fmt.Errorf("not the assigned driver")
	}
	return &order, nil
}

// PresignUpload returns a short-lived URL the driver's device uploads the
// captured file to directly (no bytes pass through our API server).
func (s *ProofMediaService) PresignUpload(ctx context.Context, orderID uuid.UUID, stopID *uuid.UUID, kind string, driverID uuid.UUID, contentType string) (uploadURL, key string, err error) {
	if !validProofKinds[kind] {
		return "", "", fmt.Errorf("invalid proof kind %q", kind)
	}
	if s.r2 == nil {
		return "", "", fmt.Errorf("media storage not configured")
	}
	if _, err := s.mustBeAssignedDriver(ctx, orderID, driverID); err != nil {
		return "", "", err
	}

	key = buildProofKey(orderID, stopID, kind)
	url, err := s.r2.PresignPut(ctx, key, contentType, proofUploadTTL)
	if err != nil {
		return "", "", err
	}
	return url, key, nil
}

// buildProofKey is a pure function (no I/O) so key layout is unit-testable
// without a live R2 connection.
func buildProofKey(orderID uuid.UUID, stopID *uuid.UUID, kind string) string {
	scope := orderID.String()
	if stopID != nil {
		scope = fmt.Sprintf("%s/%s", orderID, stopID)
	}
	return fmt.Sprintf("proof/%s/%s-%s", scope, kind, uuid.New())
}

// ConfirmUploadInput carries the evidence captured client-side.
type ConfirmUploadInput struct {
	OrderID     uuid.UUID
	StopID      *uuid.UUID
	Kind        string
	Key         string
	SHA256      string // hex sha256 of the uploaded file, computed client-side
	SealSerial  *string
	CapturedLat *float64
	CapturedLng *float64
}

// ConfirmUpload records the append-only evidence row after the driver's
// device has finished uploading. The sha256 the client reports is what
// downstream disputes trust — object existence is not re-verified here
// (would require a HEAD call to R2); the append-only table plus the driver's
// own account attribution is the audit trail.
func (s *ProofMediaService) ConfirmUpload(ctx context.Context, driverID uuid.UUID, in ConfirmUploadInput) (*model.ProofMedia, error) {
	if !validProofKinds[in.Kind] {
		return nil, fmt.Errorf("invalid proof kind %q", in.Kind)
	}
	if in.Key == "" || in.SHA256 == "" {
		return nil, fmt.Errorf("key and sha256 are required")
	}
	if _, err := s.mustBeAssignedDriver(ctx, in.OrderID, driverID); err != nil {
		return nil, err
	}

	media := &model.ProofMedia{
		ID:          uuid.New(),
		OrderID:     in.OrderID,
		StopID:      in.StopID,
		Kind:        in.Kind,
		R2Key:       in.Key,
		SHA256:      in.SHA256,
		SealSerial:  in.SealSerial,
		CapturedLat: in.CapturedLat,
		CapturedLng: in.CapturedLng,
		CapturedAt:  time.Now(),
		CapturedBy:  driverID,
	}
	if err := s.db.WithContext(ctx).Create(media).Error; err != nil {
		return nil, err
	}
	return media, nil
}

// ProofMediaView is the API-facing view: a presigned, time-boxed GET URL
// instead of the raw R2 key.
type ProofMediaView struct {
	ID          uuid.UUID  `json:"id"`
	StopID      *uuid.UUID `json:"stopId,omitempty"`
	Kind        string     `json:"kind"`
	ViewURL     string     `json:"viewUrl"`
	SHA256      string     `json:"sha256"`
	SealSerial  *string    `json:"sealSerial,omitempty"`
	CapturedLat *float64   `json:"capturedLat,omitempty"`
	CapturedLng *float64   `json:"capturedLng,omitempty"`
	CapturedAt  time.Time  `json:"capturedAt"`
}

// GetMediaForOrder returns proof media with presigned view URLs. The order's
// customer and admins may view everything (sender's own evidence / dispute
// resolution); anyone else gets nothing.
func (s *ProofMediaService) GetMediaForOrder(ctx context.Context, orderID, requesterID uuid.UUID, requesterRole string) ([]ProofMediaView, error) {
	var order model.Order
	if err := s.db.WithContext(ctx).First(&order, orderID).Error; err != nil {
		return nil, err
	}
	allowed := requesterRole == "admin" || (requesterRole == "customer" && order.CustomerID == requesterID)
	if !allowed {
		return nil, fmt.Errorf("forbidden")
	}

	var rows []model.ProofMedia
	if err := s.db.WithContext(ctx).Where("order_id = ?", orderID).Order("captured_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]ProofMediaView, 0, len(rows))
	for _, r := range rows {
		view := ProofMediaView{
			ID: r.ID, StopID: r.StopID, Kind: r.Kind, SHA256: r.SHA256,
			SealSerial: r.SealSerial, CapturedLat: r.CapturedLat, CapturedLng: r.CapturedLng,
			CapturedAt: r.CapturedAt,
		}
		if s.r2 != nil {
			if url, err := s.r2.PresignGet(ctx, r.R2Key, proofViewTTL); err == nil {
				view.ViewURL = url
			}
		}
		out = append(out, view)
	}
	return out, nil
}
