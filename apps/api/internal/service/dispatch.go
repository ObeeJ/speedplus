package service

import (
	"container/heap"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	offerRadiusKm  = 5.0
	offerTTL       = 15 * time.Second
	maxOfferCascade = 10
)

// vehicleClassRules maps vertical → minimum vehicle type for non-gas verticals.
var vehicleClassRules = map[string]model.VehicleType{
	"grocery":  model.VehicleMotorcycle,
	"food":     model.VehicleMotorcycle,
	"pharmacy": model.VehicleMotorcycle,
	"package":  model.VehicleMotorcycle,
}

// vehicleClassFor returns the minimum vehicle class for an order.
// Gas derives from total cylinder weight; all other verticals use the static map.
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
	db *gorm.DB
}

func NewDispatchService(db *gorm.DB) *DispatchService {
	return &DispatchService{db: db}
}

type driverCandidate struct {
	DriverID    uuid.UUID
	DistanceKm  float64
	Rating      float64
	index       int
}

// candidateHeap is a min-heap sorted by distance, then rating descending.
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

// DispatchCandidate is one driver targeted by a cascaded offer.
type DispatchCandidate struct {
	DriverID   uuid.UUID
	OfferID    uuid.UUID
	DistanceKm float64
}

// Dispatch finds nearby eligible drivers and creates offer records, each bound
// to the specific driver it was cascaded to (DriverID is set at creation, not
// left null — an offer only that driver may accept).
// Returns the ordered list of candidates to notify (via WS).
func (s *DispatchService) Dispatch(ctx context.Context, order *model.Order, merchantLat, merchantLng float64) ([]DispatchCandidate, error) {
	minVehicle := vehicleClassFor(order.Vertical, order.WeightKg)

	// PostGIS KNN: find online approved drivers within 5km, ordered by distance
	type row struct {
		DriverID   uuid.UUID
		DistanceKm float64
		Rating     float64
	}
	var rows []row

	vehicleFilter := vehicleFilter(minVehicle)

	err := s.db.WithContext(ctx).Raw(`
		SELECT dp.user_id AS driver_id,
		       ST_Distance(dl.location::geography, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography) / 1000.0 AS distance_km,
		       dp.rating
		FROM driver_locations dl
		JOIN driver_profiles dp ON dp.user_id = dl.driver_id
		WHERE dp.is_online = true
		  AND dp.status = 'approved'
		  AND `+vehicleFilter+`
		  AND ST_DWithin(dl.location::geography, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography, ?)
		  AND dl.updated_at > NOW() - INTERVAL '30 seconds'
		ORDER BY distance_km ASC
		LIMIT ?
	`, merchantLng, merchantLat, merchantLng, merchantLat, offerRadiusKm*1000, maxOfferCascade*3).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("dispatch query: %w", err)
	}

	// Build min-heap
	h := &candidateHeap{}
	heap.Init(h)
	for _, r := range rows {
		heap.Push(h, &driverCandidate{
			DriverID:   r.DriverID,
			DistanceKm: r.DistanceKm,
			Rating:     r.Rating,
		})
	}

	// Create offer records for top N candidates, each bound to its driver.
	var notifyOrder []DispatchCandidate
	exp := time.Now().Add(offerTTL)
	for h.Len() > 0 && len(notifyOrder) < maxOfferCascade {
		c := heap.Pop(h).(*driverCandidate)
		driverID := c.DriverID
		offer := model.DeliveryOffer{
			ID:        uuid.New(),
			OrderID:   order.ID,
			DriverID:  &driverID,
			Status:    "pending",
			ExpiresAt: exp,
		}
		if err := s.db.WithContext(ctx).Create(&offer).Error; err != nil {
			continue
		}
		notifyOrder = append(notifyOrder, DispatchCandidate{
			DriverID:   c.DriverID,
			OfferID:    offer.ID,
			DistanceKm: c.DistanceKm,
		})
		exp = exp.Add(offerTTL) // cascade: each driver gets 15s window
	}

	return notifyOrder, nil
}

// AcceptOffer atomically assigns the driver to the order.
// Offers are bound to their target driver at creation (Dispatch sets DriverID),
// so this requires driver_id = the calling driver — a driver can only accept
// an offer actually cascaded to them, not any pending offer by ID.
func (s *DispatchService) AcceptOffer(ctx context.Context, offerID, driverID uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.DeliveryOffer{}).
			Where("id = ? AND driver_id = ? AND status = 'pending' AND expires_at > ?", offerID, driverID, time.Now()).
			Update("status", "accepted")
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("offer not found, not yours, already taken, or expired")
		}

		// Assign driver to order
		var offer model.DeliveryOffer
		if err := tx.First(&offer, offerID).Error; err != nil {
			return err
		}

		return tx.Model(&model.Order{}).
			Where("id = ? AND driver_id IS NULL", offer.OrderID).
			Updates(map[string]interface{}{
				"driver_id": driverID,
				"status":    string(model.OrderDriverAssigned),
			}).Error
	})
}

// UpdateLocation upserts a driver's PostGIS location.
func (s *DispatchService) UpdateLocation(ctx context.Context, driverID uuid.UUID, lat, lng, heading float64) error {
	return s.db.WithContext(ctx).Exec(`
		INSERT INTO driver_locations (id, driver_id, location, heading, updated_at)
		VALUES (gen_random_uuid(), ?, ST_SetSRID(ST_MakePoint(?, ?), 4326), ?, NOW())
		ON CONFLICT (driver_id) DO UPDATE
		SET location = EXCLUDED.location,
		    heading = EXCLUDED.heading,
		    updated_at = EXCLUDED.updated_at
	`, driverID, lng, lat, heading).Error
}

func vehicleFilter(minType model.VehicleType) string {
	switch minType {
	case model.VehicleVan:
		return "dp.vehicle_type = 'van'"
	case model.VehicleCar:
		return "dp.vehicle_type IN ('car', 'van')"
	default:
		return "dp.vehicle_type IN ('motorcycle', 'car', 'van')"
	}
}

// ExpireOffers is called by a cron to mark stale offers expired.
func (s *DispatchService) ExpireOffers(ctx context.Context) error {
	return s.db.WithContext(ctx).
		Model(&model.DeliveryOffer{}).
		Where("status = 'pending' AND expires_at < ?", time.Now()).
		Update("status", "expired").Error
}

// RejectOffer marks an offer rejected by the driver.
func (s *DispatchService) RejectOffer(ctx context.Context, offerID uuid.UUID) error {
	return s.db.WithContext(ctx).
		Model(&model.DeliveryOffer{}).
		Where("id = ? AND status = 'pending'", offerID).
		Update("status", "rejected").Error
}

// ManualAssign allows admin to assign a driver directly.
func (s *DispatchService) ManualAssign(ctx context.Context, orderID, driverID uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, orderID).Error; err != nil {
			return err
		}
		if order.DriverID != nil {
			return fmt.Errorf("driver already assigned")
		}
		order.DriverID = &driverID
		order.Status = model.OrderDriverAssigned
		return tx.Save(&order).Error
	})
}
