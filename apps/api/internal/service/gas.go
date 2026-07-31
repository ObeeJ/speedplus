package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
)

var (
	ErrCylinderNotFound  = errors.New("cylinder not found")
	ErrCylinderForbidden = errors.New("forbidden")
)

// GasService manages the customer cylinder registry and cylinder spec catalog.
type GasService struct {
	orders repo.OrderRepo
}

func NewGasService(orders repo.OrderRepo) *GasService {
	return &GasService{orders: orders}
}

// ── Cylinder specs ────────────────────────────────────────────────────────────

// ListSpecs returns all active cylinder specs ordered by size.
func (s *GasService) ListSpecs(ctx context.Context) ([]model.CylinderSpec, error) {
	return s.orders.ListCylinderSpecs(ctx)
}

// ── Customer cylinder registry ────────────────────────────────────────────────

// RegisterCylinderInput is the customer-supplied data for a new cylinder.
type RegisterCylinderInput struct {
	SpecID          *uuid.UUID
	Serial          string
	ManufactureYear *int16
	ValveType       *string
	TareKg          *float64
	LastRecertAt    *time.Time
	Notes           *string
}

// RegisterCylinder adds a cylinder to the customer's registry.
func (s *GasService) RegisterCylinder(ctx context.Context, customerID uuid.UUID, in RegisterCylinderInput) (*model.CustomerCylinder, error) {
	if in.Serial == "" {
		return nil, fmt.Errorf("serial number is required")
	}
	c := &model.CustomerCylinder{
		ID:              uuid.New(),
		UserID:          customerID,
		SpecID:          in.SpecID,
		Serial:          in.Serial,
		ManufactureYear: in.ManufactureYear,
		ValveType:       in.ValveType,
		TareKg:          in.TareKg,
		LastRecertAt:    in.LastRecertAt,
		Status:          "active",
		Notes:           in.Notes,
	}
	if err := s.orders.CreateCylinder(ctx, c); err != nil {
		return nil, fmt.Errorf("register cylinder: %w", err)
	}
	return c, nil
}

// ListCylinders returns the customer's registered cylinders (excluding retired).
func (s *GasService) ListCylinders(ctx context.Context, customerID uuid.UUID) ([]model.CustomerCylinder, error) {
	return s.orders.ListCylinders(ctx, customerID)
}

// RetireCylinder marks a cylinder as retired (no longer in service).
// Only the owning customer may retire their own cylinder.
func (s *GasService) RetireCylinder(ctx context.Context, customerID, cylinderID uuid.UUID) error {
	cyl, err := s.orders.FindCylinder(ctx, cylinderID)
	if err != nil {
		return ErrCylinderNotFound
	}
	if cyl.UserID != customerID {
		return ErrCylinderForbidden
	}
	if cyl.Status == "in_custody" {
		return fmt.Errorf("cannot retire a cylinder that is currently in custody")
	}
	return s.orders.UpdateCylinderStatus(ctx, cylinderID, "retired")
}
