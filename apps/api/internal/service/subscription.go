package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/observability"
	"github.com/speedplus/api/internal/repo"
)

type SubscriptionService struct {
	repo     repo.SubscriptionRepo
	orders   repo.OrderRepo
	orderSvc *OrderService
	ledger   *LedgerService
}

func NewSubscriptionService(r repo.SubscriptionRepo, orders repo.OrderRepo, orderSvc *OrderService, ledger *LedgerService) *SubscriptionService {
	return &SubscriptionService{repo: r, orders: orders, orderSvc: orderSvc, ledger: ledger}
}

// ErrZoneNotLive means the customer's address falls outside a zone marked
// 'live' — gas auto-refill is the clearest "marketing" surface of the whole
// vertical, so it's the one gated per zone: don't let a customer subscribe
// to a promise (an automatic delivery every ~28 days) in an area where
// batching isn't actually running yet.
var ErrZoneNotLive = fmt.Errorf("gas auto-refill isn't available in this area yet")

func (s *SubscriptionService) Create(ctx context.Context, customerID, merchantID, addressID uuid.UUID, vertical, cadence, paymentMethod string, gasMode *string, cylinderSpecID *uuid.UUID) (*model.Subscription, error) {
	if vertical == "gas" {
		addr, err := s.orders.FindAddress(ctx, addressID)
		if err != nil {
			return nil, fmt.Errorf("create subscription: address %s: %w", addressID, err)
		}
		status, err := s.orders.FindZoneLaunchStatus(ctx, addr.Lat, addr.Lng)
		if err != nil {
			return nil, fmt.Errorf("create subscription: zone lookup: %w", err)
		}
		if status != "live" {
			return nil, ErrZoneNotLive
		}
	}

	nextCharge := nextChargeTime(cadence)
	sub := &model.Subscription{
		ID:             uuid.New(),
		CustomerID:     customerID,
		MerchantID:     merchantID,
		Vertical:       vertical,
		Cadence:        cadence,
		AddressID:      addressID,
		PaymentMethod:  paymentMethod,
		Status:         "active",
		NextChargeAt:   nextCharge,
		GasMode:        gasMode,
		CylinderSpecID: cylinderSpecID,
	}
	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

func (s *SubscriptionService) Pause(ctx context.Context, customerID, subID uuid.UUID) error {
	return s.repo.PauseByCustomer(ctx, subID, customerID)
}

func (s *SubscriptionService) Cancel(ctx context.Context, customerID, subID uuid.UUID) error {
	return s.repo.CancelByCustomer(ctx, subID, customerID)
}

// ProcessDue is called by the asynq cron job. It charges and re-schedules all due subscriptions.
func (s *SubscriptionService) ProcessDue(ctx context.Context) error {
	subs, err := s.repo.ListDue(ctx)
	if err != nil {
		return err
	}

	for _, sub := range subs {
		if err := s.chargeOne(ctx, sub); err != nil {
			observability.CaptureError(ctx, err, "subscription charge failed",
				"subscription_id", sub.ID.String(), "customer_id", sub.CustomerID.String())
			s.repo.UpdateDunning(ctx, sub.ID, sub.DunningCount+1, time.Now().Add(24*time.Hour))
			if sub.DunningCount+1 >= 3 {
				s.repo.PauseByID(ctx, sub.ID)
			}
			continue
		}
		s.repo.UpdateNextCharge(ctx, sub.ID, nextChargeFor(sub))
	}
	return nil
}

// chargeOne creates a gas order and holds escrow in a single transaction.
// A wallet debit with no order is a direct money-loss bug — both must succeed
// or both must roll back. This is the invariant the comment in the old stub
// described; this implementation honours it.
func (s *SubscriptionService) chargeOne(ctx context.Context, sub model.Subscription) error {
	if s.orderSvc == nil || s.ledger == nil {
		return fmt.Errorf("subscription service not fully wired (orders or ledger nil)")
	}

	// Resolve which product (and therefore which cylinder weight) this
	// subscription actually charges for. Getting either wrong is a real bug:
	// wrong product mischarges the customer, wrong weight both mispricing
	// (gas is priced per-kg) and misdispatching (vehicle class is weight-derived).
	var (
		productID uuid.UUID
		weightKg  float64
	)
	if sub.CylinderSpecID != nil {
		spec, err := s.repo.FindCylinderSpec(ctx, *sub.CylinderSpecID)
		if err != nil {
			return fmt.Errorf("chargeOne: cylinder spec %s: %w", *sub.CylinderSpecID, err)
		}
		prod, err := s.repo.FindProductBySpec(ctx, sub.MerchantID, *sub.CylinderSpecID)
		if err != nil {
			return fmt.Errorf("chargeOne: no product for spec %s at merchant %s: %w", *sub.CylinderSpecID, sub.MerchantID, err)
		}
		productID = prod.ID
		weightKg = spec.SizeKg
	} else {
		// No spec chosen at subscription time — fall back to the merchant's
		// cheapest cylinder product and read its actual weight rather than
		// guessing a fixed size.
		prod, err := s.repo.FindCheapestProduct(ctx, sub.MerchantID)
		if err != nil {
			return fmt.Errorf("chargeOne: cheapest product for merchant %s: %w", sub.MerchantID, err)
		}
		productID = prod.ID
		if prod.CylinderSpecID == nil {
			return fmt.Errorf("chargeOne: product %s has no cylinder_spec_id, cannot determine weight", prod.ID)
		}
		spec, err := s.repo.FindCylinderSpec(ctx, *prod.CylinderSpecID)
		if err != nil {
			return fmt.Errorf("chargeOne: cylinder spec for product %s: %w", prod.ID, err)
		}
		weightKg = spec.SizeKg
	}
	if weightKg <= 0 {
		return fmt.Errorf("chargeOne: resolved a non-positive cylinder weight for subscription %s", sub.ID)
	}

	product, err := s.repo.FindProduct(ctx, productID)
	if err != nil {
		return fmt.Errorf("chargeOne: product %s: %w", productID, err)
	}
	subtotalKobo := product.PriceKobo

	merchant, err := s.orders.FindMerchant(ctx, sub.MerchantID)
	if err != nil {
		return fmt.Errorf("chargeOne: merchant %s: %w", sub.MerchantID, err)
	}

	addr, err := s.orders.FindAddress(ctx, sub.AddressID)
	if err != nil {
		return fmt.Errorf("chargeOne: address %s: %w", sub.AddressID, err)
	}

	quote, err := s.orderSvc.pricing.Quote(ctx, QuoteRequest{
		CustomerID:   sub.CustomerID,
		MerchantID:   sub.MerchantID,
		Vertical:     "gas",
		SubtotalKobo: subtotalKobo,
		OriginLat:    merchant.Lat,
		OriginLng:    merchant.Lng,
		DestLat:      addr.Lat,
		DestLng:      addr.Lng,
		WeightKg:     weightKg,
	})
	if err != nil {
		return fmt.Errorf("chargeOne: quote: %w", err)
	}

	idempotencyKey := fmt.Sprintf("sub:%s:%s", sub.ID, sub.NextChargeAt.UTC().Format("2006-01-02"))

	gasMode := "swap"
	if sub.GasMode != nil {
		gasMode = *sub.GasMode
	}

	_, err = s.orderSvc.Create(ctx, CreateOrderInput{
		CustomerID:     sub.CustomerID,
		MerchantID:     sub.MerchantID,
		QuoteID:        quote.ID,
		Vertical:       "gas",
		DeliveryAddrID: sub.AddressID,
		PaymentMethod:  sub.PaymentMethod,
		IdempotencyKey: idempotencyKey,
		GasMode:        &gasMode,
		Items: []OrderItemInput{
			{
				ProductID:     productID,
				Name:          product.Name,
				Quantity:      1,
				UnitPriceKobo: subtotalKobo,
				WeightKg:      weightKg,
			},
		},
	})
	return err
}

// UpdateBurnRates recomputes avg_days_between_refills and predicted_runout_at
// for all active gas subscriptions from their delivered order history.
// Called nightly by the worker.
func (s *SubscriptionService) UpdateBurnRates(ctx context.Context) error {
	rows, err := s.repo.BurnRateStats(ctx)
	if err != nil {
		return fmt.Errorf("update burn rates: %w", err)
	}

	for _, r := range rows {
		if r.AvgDaysBetweenFills <= 0 {
			continue
		}
		predictedRunout := r.LastDeliveredAt.Add(
			time.Duration(r.AvgDaysBetweenFills*24) * time.Hour,
		)
		if err := s.repo.UpdateBurnRate(ctx, r.CustomerID, r.AvgDaysBetweenFills, predictedRunout); err != nil {
			observability.CaptureError(ctx, err, "update burn rates: update subscription",
				"customer_id", r.CustomerID)
		}
	}
	return nil
}

// ── LPG price index ───────────────────────────────────────────────────────────

// LPGIndexEntry is the API-facing view of an LPG price row.
type LPGIndexEntry struct {
	ID             uuid.UUID `json:"id"`
	Region         string    `json:"region"`
	PricePerKgKobo int64     `json:"pricePerKgKobo"`
	Source         string    `json:"source"`
	EffectiveAt    time.Time `json:"effectiveAt"`
}

// GetLiveLPGPrice returns the current price per kg for a region.
func (s *SubscriptionService) GetLiveLPGPrice(ctx context.Context, region string) (*LPGIndexEntry, error) {
	row, err := s.repo.GetLiveLPGPrice(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("lpg price: no entry for region %q: %w", region, err)
	}
	return &LPGIndexEntry{
		ID:             row.ID,
		Region:         row.Region,
		PricePerKgKobo: row.PricePerKgKobo,
		Source:         row.Source,
		EffectiveAt:    row.EffectiveAt,
	}, nil
}

func (s *SubscriptionService) RecordLPGPrice(ctx context.Context, adminID uuid.UUID, region string, pricePerKgKobo int64, source string) (*model.LPGPriceIndex, string, error) {
	prev, hasPrev := (*model.LPGPriceIndex)(nil), false
	if p, err := s.repo.GetPrevLPGPrice(ctx, region); err == nil {
		prev = p
		hasPrev = true
	}

	row := &model.LPGPriceIndex{
		ID:             uuid.New(),
		Region:         region,
		PricePerKgKobo: pricePerKgKobo,
		Source:         source,
		EffectiveAt:    time.Now(),
		UpdatedBy:      adminID,
	}
	if err := s.repo.CreateLPGPriceRow(ctx, row); err != nil {
		return nil, "", fmt.Errorf("record lpg price: %w", err)
	}

	var suggestion string
	if hasPrev && prev.PricePerKgKobo > 0 {
		delta := float64(pricePerKgKobo-prev.PricePerKgKobo) / float64(prev.PricePerKgKobo)
		if delta > 0.10 || delta < -0.10 {
			suggestion = fmt.Sprintf(
				"LPG price moved %.1f%% (₦%.2f/kg → ₦%.2f/kg). Consider updating product prices.",
				delta*100,
				float64(prev.PricePerKgKobo)/100,
				float64(pricePerKgKobo)/100,
			)
		}
	}
	return row, suggestion, nil
}

func nextChargeTime(cadence string) time.Time {
	switch cadence {
	case "weekly":
		return time.Now().AddDate(0, 0, 7)
	case "biweekly":
		return time.Now().AddDate(0, 0, 14)
	default: // monthly
		return time.Now().AddDate(0, 1, 0)
	}
}

// nextChargeFor returns the next charge time for a subscription.
// For gas subscriptions with a predicted runout, we use that date minus a
// 1-day lead time so the delivery arrives before the customer runs out.
// The cadence floor is still enforced: we never charge sooner than half the
// cadence interval to prevent runaway charging on a noisy burn-rate estimate.
func nextChargeFor(sub model.Subscription) time.Time {
	cadenceFloor := nextChargeTime(sub.Cadence)
	if sub.Vertical != "gas" || sub.PredictedRunoutAt == nil {
		return cadenceFloor
	}
	// Aim to deliver 1 day before predicted runout.
	predicted := sub.PredictedRunoutAt.Add(-24 * time.Hour)
	// Half-cadence floor: don't charge more than twice the normal rate.
	halfCadence := nextChargeTime(sub.Cadence)
	switch sub.Cadence {
	case "weekly":
		halfCadence = time.Now().AddDate(0, 0, 3)
	case "biweekly":
		halfCadence = time.Now().AddDate(0, 0, 7)
	default:
		halfCadence = time.Now().AddDate(0, 0, 14)
	}
	if predicted.Before(halfCadence) {
		return halfCadence
	}
	// Cap at 1.5× cadence so a long gap doesn't delay indefinitely.
	var cap time.Time
	switch sub.Cadence {
	case "weekly":
		cap = time.Now().AddDate(0, 0, 10)
	case "biweekly":
		cap = time.Now().AddDate(0, 0, 21)
	default:
		cap = time.Now().AddDate(0, 0, 45)
	}
	if predicted.After(cap) {
		return cap
	}
	return predicted
}
