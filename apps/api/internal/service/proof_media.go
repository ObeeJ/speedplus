package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/observability"
	"github.com/speedplus/api/internal/repo"
	"github.com/speedplus/api/internal/storage"
)

const (
	proofUploadTTL = 5 * time.Minute
	proofViewTTL   = 15 * time.Minute
)

var validProofKinds = map[string]bool{
	"pickup_photo": true, "pickup_video": true,
	"dropoff_photo": true, "dropoff_video": true,
	"weight_photo": true,
}

type ProofMediaService struct {
	repo   repo.ProofMediaRepo
	orders repo.OrderRepo
	r2     *storage.R2Client
}

func NewProofMediaService(r repo.ProofMediaRepo, orders repo.OrderRepo, r2 *storage.R2Client) *ProofMediaService {
	return &ProofMediaService{repo: r, orders: orders, r2: r2}
}

func (s *ProofMediaService) mustBeAssignedDriver(ctx context.Context, orderID, driverID uuid.UUID) (*model.Order, error) {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found")
	}
	if order.DriverID == nil || *order.DriverID != driverID {
		return nil, fmt.Errorf("not the assigned driver")
	}
	return order, nil
}

// mustBeAssignedDriverWithItems is mustBeAssignedDriver but preloads Items —
// used only where the caller needs order weight (weight_photo validation),
// so the common PresignUpload/ConfirmUpload paths don't pay for a join they
// don't need.
func (s *ProofMediaService) mustBeAssignedDriverWithItems(ctx context.Context, orderID, driverID uuid.UUID) (*model.Order, error) {
	order, err := s.orders.FindByIDWithItems(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found")
	}
	if order.DriverID == nil || *order.DriverID != driverID {
		return nil, fmt.Errorf("not the assigned driver")
	}
	return order, nil
}

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

func buildProofKey(orderID uuid.UUID, stopID *uuid.UUID, kind string) string {
	scope := orderID.String()
	if stopID != nil {
		scope = fmt.Sprintf("%s/%s", orderID, stopID)
	}
	return fmt.Sprintf("proof/%s/%s-%s", scope, kind, uuid.New())
}

type ConfirmUploadInput struct {
	OrderID     uuid.UUID
	StopID      *uuid.UUID
	Kind        string
	Key         string
	SHA256      string
	SealSerial  *string
	MeasuredKg  *float64
	CapturedLat *float64
	CapturedLng *float64
}

// Weight-photo plausibility band: a reading outside [minWeightPlausibilityFactor,
// maxWeightPlausibilityFactor] × ordered weight is rejected so the driver
// re-weighs, rather than silently triggering a near-total automatic merchant
// clawback via the ledger's shortfall refund on a fat-fingered scale entry
// (e.g. "1.25" typed for "12.5"). The two bounds are independent constants —
// deriving one from the other by arithmetic is easy to get subtly wrong when
// either is tuned later.
const (
	minWeightPlausibilityFactor = 0.5
	maxWeightPlausibilityFactor = 1.5
)

// overfillWarnFactor flags (but does not reject or refund) a measured weight
// well above what was ordered — a possible sign of scale tampering or a swap
// with a larger cylinder, worth surfacing for manual review even though the
// shortfall path only ever acts on underfill.
const overfillWarnFactor = 1.1

func (s *ProofMediaService) ConfirmUpload(ctx context.Context, driverID uuid.UUID, in ConfirmUploadInput) (*model.ProofMedia, error) {
	if !validProofKinds[in.Kind] {
		return nil, fmt.Errorf("invalid proof kind %q", in.Kind)
	}
	if in.Key == "" || in.SHA256 == "" {
		return nil, fmt.Errorf("key and sha256 are required")
	}
	if in.Kind == "weight_photo" && (in.MeasuredKg == nil || *in.MeasuredKg <= 0) {
		return nil, fmt.Errorf("weight_photo requires a positive measured_kg")
	}

	var order *model.Order
	var err error
	if in.Kind == "weight_photo" {
		order, err = s.mustBeAssignedDriverWithItems(ctx, in.OrderID, driverID)
	} else {
		order, err = s.mustBeAssignedDriver(ctx, in.OrderID, driverID)
	}
	if err != nil {
		return nil, err
	}

	if in.Kind == "weight_photo" {
		var orderedKg float64
		for _, item := range order.Items {
			orderedKg += item.WeightKg * float64(item.Quantity)
		}
		if orderedKg > 0 {
			minKg := orderedKg * minWeightPlausibilityFactor
			maxKg := orderedKg * maxWeightPlausibilityFactor
			if *in.MeasuredKg < minKg || *in.MeasuredKg > maxKg {
				return nil, fmt.Errorf("measured_kg %.2f is implausible for a %.2fkg order (expected %.2f–%.2fkg) — re-weigh and retry", *in.MeasuredKg, orderedKg, minKg, maxKg)
			}
			if *in.MeasuredKg > orderedKg*overfillWarnFactor {
				observability.CaptureMessage(ctx, "gas weight_photo measured significantly over ordered weight — possible scale tampering or wrong cylinder",
					"order_id", in.OrderID.String(), "driver_id", driverID.String(),
					"ordered_kg", fmt.Sprintf("%.2f", orderedKg), "measured_kg", fmt.Sprintf("%.2f", *in.MeasuredKg))
			}
		} else {
			// No item weight recorded — the plausibility check above can't
			// run. This shouldn't happen for a gas order (items always carry
			// WeightKg by the time they reach delivery), so surface it rather
			// than silently accepting an unchecked reading.
			observability.CaptureMessage(ctx, "weight_photo confirmed for an order with zero ordered weight — plausibility check skipped",
				"order_id", in.OrderID.String(), "driver_id", driverID.String())
		}
	}

	media := &model.ProofMedia{
		ID:          uuid.New(),
		OrderID:     in.OrderID,
		StopID:      in.StopID,
		Kind:        in.Kind,
		R2Key:       in.Key,
		SHA256:      in.SHA256,
		SealSerial:  in.SealSerial,
		MeasuredKg:  in.MeasuredKg,
		CapturedLat: in.CapturedLat,
		CapturedLng: in.CapturedLng,
		CapturedAt:  time.Now(),
		CapturedBy:  driverID,
	}
	if err := s.repo.Create(ctx, media); err != nil {
		return nil, err
	}
	return media, nil
}

type ProofMediaView struct {
	ID          uuid.UUID  `json:"id"`
	StopID      *uuid.UUID `json:"stopId,omitempty"`
	Kind        string     `json:"kind"`
	ViewURL     string     `json:"viewUrl"`
	SHA256      string     `json:"sha256"`
	SealSerial  *string    `json:"sealSerial,omitempty"`
	MeasuredKg  *float64   `json:"measuredKg,omitempty"`
	CapturedLat *float64   `json:"capturedLat,omitempty"`
	CapturedLng *float64   `json:"capturedLng,omitempty"`
	CapturedAt  time.Time  `json:"capturedAt"`
}

func (s *ProofMediaService) GetMediaForOrder(ctx context.Context, orderID, requesterID uuid.UUID, requesterRole string) ([]ProofMediaView, error) {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	allowed := requesterRole == "admin" || (requesterRole == "customer" && order.CustomerID == requesterID)
	if !allowed {
		return nil, fmt.Errorf("forbidden")
	}

	rows, err := s.repo.ListByOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	out := make([]ProofMediaView, 0, len(rows))
	for _, r := range rows {
		view := ProofMediaView{
			ID: r.ID, StopID: r.StopID, Kind: r.Kind, SHA256: r.SHA256,
			SealSerial: r.SealSerial, MeasuredKg: r.MeasuredKg,
			CapturedLat: r.CapturedLat, CapturedLng: r.CapturedLng,
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
