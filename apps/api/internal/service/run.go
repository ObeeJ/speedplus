package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/observability"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

// maxRunOrders caps how many orders can be batched into one run.
// Matches the plan's "8–20 cylinders" upper bound; OSRM trip payload size
// and rider load are the practical constraints.
const maxRunOrders = 20

// RunService assembles and dispatches batched gas delivery runs.
type RunService struct {
	repo     repo.RunRepo
	orders   repo.OrderRepo
	pricing  *PricingService
	dispatch *DispatchService
}

func NewRunService(r repo.RunRepo, orders repo.OrderRepo, pricing *PricingService, dispatch *DispatchService) *RunService {
	return &RunService{repo: r, orders: orders, pricing: pricing, dispatch: dispatch}
}

func (s *RunService) GetRun(ctx context.Context, id, requesterID uuid.UUID, requesterRole string) (*model.DeliveryRun, error) {
	run, err := s.repo.FindRunWithOrders(ctx, id)
	if err != nil {
		return nil, err
	}
	// Admins see all runs. Drivers only see runs assigned to them.
	if requesterRole == "admin" {
		return run, nil
	}
	if run.DriverID == nil || *run.DriverID != requesterID {
		return nil, errors.New("forbidden")
	}
	return run, nil
}

// AssembleRun finds all pending scheduled gas orders whose delivery address
// falls within the given zone and whose scheduled_for falls within the window,
// sequences them with osrmTrip (reusing the existing implementation), and
// persists a DeliveryRun + RunOrder rows. Returns nil, nil when there are no
// eligible orders — not an error.
func (s *RunService) AssembleRun(ctx context.Context, zoneID uuid.UUID, windowStart, windowEnd time.Time) (*model.DeliveryRun, error) {
	zone, err := s.repo.FindZone(ctx, zoneID)
	if err != nil {
		return nil, fmt.Errorf("assemble run: zone %s: %w", zoneID, err)
	}
	if !zone.IsActive {
		return nil, nil
	}

	rows, err := s.repo.FindOrdersInZoneWindow(ctx, zone.Boundary, windowStart, windowEnd, maxRunOrders)
	if err != nil {
		return nil, fmt.Errorf("assemble run: query orders: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}

	merchantLat, merchantLng, err := s.repo.FindGasMerchantLocation(ctx, rows[0].MerchantID)
	if err != nil {
		return nil, fmt.Errorf("assemble run: merchant location for %s: %w", rows[0].MerchantID, err)
	}

	waypoints := make([]LatLng, 0, len(rows)+1)
	waypoints = append(waypoints, LatLng{Lat: merchantLat, Lng: merchantLng})
	for _, r := range rows {
		waypoints = append(waypoints, LatLng{Lat: r.Lat, Lng: r.Lng})
	}

	distKm, _, err := s.pricing.osrmTrip(ctx, waypoints)
	if err != nil {
		return nil, fmt.Errorf("assemble run: osrm trip: %w", err)
	}

	sequence := make([]string, len(rows))
	for i, r := range rows {
		sequence[i] = r.OrderID.String()
	}
	seqJSON, _ := json.Marshal(sequence)

	run := &model.DeliveryRun{
		ID:                uuid.New(),
		ZoneID:            zoneID,
		WindowStart:       windowStart,
		WindowEnd:         windowEnd,
		Status:            "assembling",
		OptimizedSequence: string(seqJSON),
		TotalDistanceKm:   distKm,
	}

	err = s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.repo.CreateRunTx(ctx, tx, run); err != nil {
			return err
		}
		for i, r := range rows {
			if err := s.repo.CreateRunOrderTx(ctx, tx, &model.RunOrder{
				ID:       uuid.New(),
				RunID:    run.ID,
				OrderID:  r.OrderID,
				Sequence: i + 1,
			}); err != nil {
				return fmt.Errorf("run order %d: %w", i+1, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("assemble run: persist: %w", err)
	}
	return run, nil
}

// DispatchRun offers an assembled run to the nearest eligible van driver,
// reusing the existing Dispatch KNN machinery. The run must be in status
// "assembling"; on success it transitions to "dispatched".
func (s *RunService) DispatchRun(ctx context.Context, runID uuid.UUID) error {
	run, err := s.repo.FindRunWithOrders(ctx, runID)
	if err != nil {
		return fmt.Errorf("dispatch run: load: %w", err)
	}
	if run.Status != "assembling" {
		return fmt.Errorf("dispatch run: run is %s, not assembling", run.Status)
	}
	if len(run.Orders) == 0 {
		return fmt.Errorf("dispatch run: no orders in run")
	}

	firstOrder, err := s.orders.FindByID(ctx, run.Orders[0].OrderID)
	if err != nil {
		return fmt.Errorf("dispatch run: first order: %w", err)
	}
	addr, err := s.orders.FindAddress(ctx, firstOrder.DeliveryAddressID)
	if err != nil {
		return fmt.Errorf("dispatch run: address: %w", err)
	}

	totalKg, err := s.repo.SumRunWeightKg(ctx, runID)
	if err != nil {
		return fmt.Errorf("dispatch run: sum weight: %w", err)
	}
	if totalKg <= 0 {
		totalKg = 25 // fallback: treat as van-class if items have no weight recorded
	}

	synthetic := &model.Order{
		ID:       run.ID,
		Vertical: "gas",
		Items:    []model.OrderItem{{WeightKg: totalKg, Quantity: 1}},
	}
	candidates, err := s.dispatch.Dispatch(ctx, synthetic, addr.Lat, addr.Lng)
	if err != nil {
		return fmt.Errorf("dispatch run: knn: %w", err)
	}
	if len(candidates) == 0 {
		return fmt.Errorf("dispatch run: no eligible drivers found")
	}

	return s.repo.UpdateRunStatus(ctx, runID, candidates[0].DriverID, "dispatched")
}

// AssembleAllDueRuns iterates every active zone and assembles a run for the
// current window if one doesn't already exist. Called by the cron worker.
func (s *RunService) AssembleAllDueRuns(ctx context.Context) error {
	zones, err := s.repo.ListActiveZones(ctx)
	if err != nil {
		return fmt.Errorf("assemble all runs: zones: %w", err)
	}

	now := time.Now().UTC()
	dayBit := int16(1 << now.Weekday())

	for _, zone := range zones {
		if zone.ActiveDays&dayBit == 0 {
			continue
		}
		midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		wStart := midnight.Add(time.Duration(zone.WindowStart) * time.Minute)
		wEnd := midnight.Add(time.Duration(zone.WindowEnd) * time.Minute)
		if now.Before(wStart) || now.After(wEnd) {
			continue
		}

		count, err := s.repo.CountRunsForZoneWindow(ctx, zone.ID, wStart)
		if err != nil {
			observability.CaptureError(ctx, err, "assemble runs: count check failed", "zone_id", zone.ID.String())
			continue
		}
		if count > 0 {
			continue
		}

		run, err := s.AssembleRun(ctx, zone.ID, wStart, wEnd)
		if err != nil {
			observability.CaptureError(ctx, err, "assemble runs: AssembleRun failed", "zone_id", zone.ID.String())
			continue
		}
		if run == nil {
			continue
		}
		if err := s.DispatchRun(ctx, run.ID); err != nil {
			observability.CaptureError(ctx, err, "assemble runs: DispatchRun failed", "run_id", run.ID.String())
		}
	}
	return nil
}
