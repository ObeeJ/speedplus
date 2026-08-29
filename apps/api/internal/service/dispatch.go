package service

import (
	"container/heap"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

const (
	offerRadiusKm   = 5.0
	offerTTL        = 15 * time.Second
	maxOfferCascade = 10
)

var vehicleClassRules = map[string]model.VehicleType{
	"grocery":  model.VehicleMotorcycle,
	"food":     model.VehicleMotorcycle,
	"pharmacy": model.VehicleMotorcycle,
	"package":  model.VehicleMotorcycle,
}

func vehicleClassFor(vertical string, totalKg float64) model.VehicleType {
	if vertical == "gas" {
		switch {
		case totalKg <= 6:
			return model.VehicleMotorcycle
		case totalKg <= 12.5:
			return model.VehicleCar
		default:
			return model.VehicleVan
		}
	}
	if v, ok := vehicleClassRules[vertical]; ok {
		return v
	}
	return model.VehicleMotorcycle
}

type DispatchService struct {
	repo   repo.DispatchRepo
	orders *OrderService
}

func NewDispatchService(r repo.DispatchRepo) *DispatchService {
	return &DispatchService{repo: r}
}

// InjectOrders wires the OrderService after both services are constructed
// (breaks the circular dependency at construction time).
func (s *DispatchService) InjectOrders(orders *OrderService) {
	s.orders = orders
}

type driverCandidate struct {
	DriverID   uuid.UUID
	DistanceKm float64
	Rating     float64
	index      int
}

type candidateHeap []*driverCandidate

func (h candidateHeap) Len() int { return len(h) }
func (h candidateHeap) Less(i, j int) bool {
	if h[i].DistanceKm != h[j].DistanceKm {
		return h[i].DistanceKm < h[j].DistanceKm
	}
	return h[i].Rating > h[j].Rating
}
func (h candidateHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i]; h[i].index = i; h[j].index = j }
func (h *candidateHeap) Push(x interface{}) { *h = append(*h, x.(*driverCandidate)) }
func (h *candidateHeap) Pop() interface{}   { old := *h; n := len(old); x := old[n-1]; *h = old[:n-1]; return x }

type DispatchCandidate struct {
	DriverID   uuid.UUID
	OfferID    uuid.UUID
	DistanceKm float64
}

func (s *DispatchService) Dispatch(ctx context.Context, order *model.Order, merchantLat, merchantLng float64) ([]DispatchCandidate, error) {
	var totalKg float64
	for _, item := range order.Items {
		totalKg += item.WeightKg * float64(item.Quantity)
	}
	minVehicle := vehicleClassFor(order.Vertical, totalKg)

	requireHazmat := order.Vertical == "gas" && totalKg > 12.5

	nearby, err := s.repo.NearbyDrivers(ctx, merchantLat, merchantLng, offerRadiusKm*1000, minVehicle, requireHazmat, maxOfferCascade*3)
	if err != nil {
		return nil, fmt.Errorf("dispatch query: %w", err)
	}

	h := &candidateHeap{}
	heap.Init(h)
	for _, r := range nearby {
		heap.Push(h, &driverCandidate{DriverID: r.DriverID, DistanceKm: r.DistanceKm, Rating: r.Rating})
	}

	var notifyOrder []DispatchCandidate
	exp := time.Now().Add(offerTTL)
	for h.Len() > 0 && len(notifyOrder) < maxOfferCascade {
		c := heap.Pop(h).(*driverCandidate)
		driverID := c.DriverID
		offer := &model.DeliveryOffer{
			ID:        uuid.New(),
			OrderID:   order.ID,
			DriverID:  &driverID,
			Status:    "pending",
			ExpiresAt: exp,
		}
		if err := s.repo.CreateOffer(ctx, offer); err != nil {
			continue
		}
		notifyOrder = append(notifyOrder, DispatchCandidate{DriverID: c.DriverID, OfferID: offer.ID, DistanceKm: c.DistanceKm})
		exp = exp.Add(offerTTL)
	}
	return notifyOrder, nil
}

func (s *DispatchService) AcceptOffer(ctx context.Context, offerID, driverID uuid.UUID) error {
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		rows, err := s.repo.AcceptOfferTx(ctx, tx, offerID, driverID)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("offer not found, not yours, already taken, or expired")
		}
		offer, err := s.repo.FindOfferTx(ctx, tx, offerID)
		if err != nil {
			return err
		}
		return s.repo.AssignDriverToOrder(ctx, tx, offer.OrderID, driverID)
	})
}

func (s *DispatchService) UpdateLocation(ctx context.Context, driverID uuid.UUID, lat, lng, heading float64) error {
	return s.repo.UpsertDriverLocation(ctx, driverID, lat, lng, heading)
}

func (s *DispatchService) SetOnline(ctx context.Context, driverID uuid.UUID, online bool) error {
	return s.repo.SetDriverOnline(ctx, driverID, online)
}

func (s *DispatchService) ExpireOffers(ctx context.Context) error {
	return s.repo.ExpireStaleOffers(ctx)
}

func (s *DispatchService) RejectOffer(ctx context.Context, offerID, driverID uuid.UUID) error {
	return s.repo.UpdateOfferStatus(ctx, offerID, driverID, "rejected")
}

func (s *DispatchService) ManualAssign(ctx context.Context, orderID, driverID uuid.UUID) error {
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		// Advisory lock serializes concurrent ManualAssign calls for this driver
		// so the active-order check below can't race with another assignment
		// that hasn't committed yet (TOCTOU double-booking).
		if err := s.repo.LockDriverForAssignmentTx(ctx, tx, driverID); err != nil {
			return fmt.Errorf("double-booking check: %w", err)
		}
		hasActive, err := s.repo.HasActiveOrderTx(ctx, tx, driverID)
		if err != nil {
			return fmt.Errorf("double-booking check: %w", err)
		}
		if hasActive {
			return fmt.Errorf("driver already has an active delivery")
		}
		order, err := s.repo.LockOrderTx(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if order.DriverID != nil {
			return fmt.Errorf("driver already assigned")
		}
		order.DriverID = &driverID
		order.Status = model.OrderDriverAssigned
		return s.repo.SaveOrderTx(ctx, tx, order)
	})
}
