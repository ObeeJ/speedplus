package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

var (
	ErrIllegalTransition = errors.New("illegal order state transition")
	ErrQuoteInvalid      = errors.New("quote invalid or expired")
	ErrMerchantClosed    = errors.New("merchant is currently closed")
	ErrRxRequired        = errors.New("prescription required for this order")
	// POD checkout is gated by the trust ladder, but server-side POD
	// settlement (wallet debit at the door) is not yet implemented — orders
	// must be wallet-funded so escrow can settle. Remove once POD settlement
	// lands in LedgerService.Settle.
	ErrPODNotYetEnabled = errors.New("pay on arrival is not yet available — pay from wallet")
)

type CreateOrderInput struct {
	CustomerID     uuid.UUID
	MerchantID     uuid.UUID
	QuoteID        uuid.UUID
	Vertical       string
	Items          []OrderItemInput
	DeliveryAddrID uuid.UUID
	RecipientName  *string
	RecipientPhone *string
	PrescriptionID *uuid.UUID
	TipKobo        int64
	ScheduledFor   *time.Time
	PaymentMethod  string // ""|"wallet"|"pay_on_arrival"
	IdempotencyKey string
	// Multi-drop stops (package vertical only). When present, DeliveryAddrID
	// is the first stop's address and Stops[0] is the first drop-off.
	Stops []OrderStopInput
}

type OrderStopInput struct {
	Sequence       int
	AddressID      uuid.UUID
	RecipientName  *string
	RecipientPhone *string
	Notes          *string
}

type OrderItemInput struct {
	ProductID        uuid.UUID
	Name             string
	Quantity         int
	UnitPriceKobo    int64
	WeightKg         float64
	SizeCategory     string
	Customizations   *string
	SubstitutionPref *string
}

type OrderService struct {
	db             *gorm.DB
	pricing        *PricingService
	ledger         *LedgerService
	tier           *TierService
	dispatch       *DispatchService
	hub            wsPublisher
	deliveryCodes  *DeliveryCodeService
}

// wsPublisher is the subset of ws.Hub used by OrderService.
type wsPublisher interface {
	Publish(ctx context.Context, channel, event string, data interface{}) error
}

func NewOrderService(db *gorm.DB, pricing *PricingService, ledger *LedgerService, tier *TierService) *OrderService {
	return &OrderService{db: db, pricing: pricing, ledger: ledger, tier: tier}
}

func (s *OrderService) InjectDispatch(d *DispatchService, hub wsPublisher) {
	s.dispatch = d
	s.hub = hub
}

func (s *OrderService) InjectDeliveryCodes(dc *DeliveryCodeService) {
	s.deliveryCodes = dc
}

func (s *OrderService) Create(ctx context.Context, in CreateOrderInput) (*model.Order, error) {
	// Idempotency check
	var existing model.Order
	if err := s.db.WithContext(ctx).Where("idempotency_key = ?", in.IdempotencyKey).First(&existing).Error; err == nil {
		return &existing, nil
	}

	// Validate quote
	var subtotal int64
	for _, item := range in.Items {
		subtotal += item.UnitPriceKobo * int64(item.Quantity)
	}
	quote, err := s.pricing.ValidateQuote(ctx, in.QuoteID, subtotal)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrQuoteInvalid, err)
	}

	// Pharmacy vertical requires approved prescription
	if in.Vertical == "pharmacy" && in.PrescriptionID == nil {
		return nil, ErrRxRequired
	}

	// Pay-on-arrival gate: trust-ladder checks run first so the customer gets
	// the precise reason (tier locked / cap / active POD). Even when eligible,
	// POD checkout is declined until POD settlement is implemented — creating
	// an escrow-less order today would leave it unsettleable at the door.
	if in.PaymentMethod == "pay_on_arrival" {
		if s.tier == nil {
			return nil, ErrPODNotYetEnabled
		}
		totalKobo := quote.TotalKobo + in.TipKobo
		if err := s.tier.CanUsePayOnArrival(ctx, in.CustomerID, totalKobo); err != nil {
			return nil, err
		}
		return nil, ErrPODNotYetEnabled
	}

	var order *model.Order
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check merchant is open + KYC approved
		var merchant model.Merchant
		if err := tx.First(&merchant, in.MerchantID).Error; err != nil {
			return fmt.Errorf("merchant not found")
		}
		if !merchant.IsOpen {
			return ErrMerchantClosed
		}

		order = &model.Order{
			ID:                uuid.New(),
			CustomerID:        in.CustomerID,
			MerchantID:        in.MerchantID,
			QuoteID:           in.QuoteID,
			Vertical:          in.Vertical,
			Status:            model.OrderPending,
			SubtotalKobo:      quote.SubtotalKobo,
			DeliveryKobo:      quote.DeliveryKobo,
			ServiceKobo:       quote.ServiceKobo,
			TipKobo:           in.TipKobo,
			TotalKobo:         quote.TotalKobo + in.TipKobo,
			DeliveryAddressID: in.DeliveryAddrID,
			RecipientName:     in.RecipientName,
			RecipientPhone:    in.RecipientPhone,
			PrescriptionID:    in.PrescriptionID,
			ScheduledFor:      in.ScheduledFor,
			PaymentMethod:     "wallet",
			IdempotencyKey:    in.IdempotencyKey,
		}
		if in.PaymentMethod == "pay_on_arrival" {
			order.PaymentMethod = "pay_on_arrival"
		}

		for _, item := range in.Items {
			order.Items = append(order.Items, model.OrderItem{
				ID:               uuid.New(),
				OrderID:          order.ID,
				ProductID:        item.ProductID,
				Name:             item.Name,
				Quantity:         item.Quantity,
				UnitPriceKobo:    item.UnitPriceKobo,
				TotalKobo:        item.UnitPriceKobo * int64(item.Quantity),
				WeightKg:         item.WeightKg,
				SizeCategory:     item.SizeCategory,
				Customizations:   item.Customizations,
				SubstitutionPref: item.SubstitutionPref,
			})
		}

		if err := tx.Create(order).Error; err != nil {
			return err
		}

		// Mark quote used
		if err := s.pricing.MarkQuoteUsed(ctx, in.QuoteID); err != nil {
			return err
		}

		// Multi-drop stops (package vertical)
		for _, stop := range in.Stops {
			order.Stops = append(order.Stops, model.OrderStop{
				ID:             uuid.New(),
				OrderID:        order.ID,
				Sequence:       stop.Sequence,
				AddressID:      stop.AddressID,
				RecipientName:  stop.RecipientName,
				RecipientPhone: stop.RecipientPhone,
				Notes:          stop.Notes,
				QRCode:         "", // generated when order transitions to in_transit
				Status:         "pending",
			})
		}

		// Escrow hold — debit customer wallet
		if err := s.ledger.HoldEscrow(ctx, tx, order.ID, in.CustomerID, order.TotalKobo); err != nil {
			return fmt.Errorf("escrow hold: %w", err)
		}

		// Audit event
		return tx.Create(&model.OrderEvent{
			ID:        uuid.New(),
			OrderID:   order.ID,
			ToStatus:  model.OrderPending,
			ActorID:   in.CustomerID,
			ActorRole: "customer",
		}).Error
	})
	if err != nil {
		return nil, err
	}

	// Trigger dispatch asynchronously after the transaction commits.
	// Merchant lat/lng is used as the pickup point for KNN driver search.
	if s.dispatch != nil {
		go func(o *model.Order) {
			var merchant model.Merchant
			if dbErr := s.db.First(&merchant, o.MerchantID).Error; dbErr != nil {
				return
			}
			var dropoff model.Address
			_ = s.db.First(&dropoff, o.DeliveryAddressID).Error // best-effort label; ok if empty

			candidates, dispErr := s.dispatch.Dispatch(context.Background(), o, merchant.Lat, merchant.Lng)
			if dispErr != nil || len(candidates) == 0 {
				return
			}
			// Notify each targeted driver via WS on their personal channel.
			// Payload matches what the driver app's offer card expects:
			// offerId (so accept/reject hits the right row), addresses, distance.
			if s.hub != nil {
				for _, cand := range candidates {
					_ = s.hub.Publish(context.Background(),
						"driver:"+cand.DriverID.String(),
						"new_offer",
						map[string]interface{}{
							"offerId":         cand.OfferID,
							"orderId":         o.ID,
							"vertical":        o.Vertical,
							"totalKobo":       o.TotalKobo,
							"distanceKm":      cand.DistanceKm,
							"pickupAddress":   merchant.BusinessName,
							"dropoffAddress":  fmt.Sprintf("%s, %s", dropoff.Street, dropoff.City),
						},
					)
				}
			}
			// Notify customer that we're searching.
			if s.hub != nil {
				_ = s.hub.Publish(context.Background(),
					"order:"+o.ID.String(),
					"searching_driver",
					map[string]interface{}{"orderId": o.ID},
				)
			}
		}(order)
	}

	return order, nil
}

// Transition moves an order through the state machine.
// Returns 409-equivalent error on illegal transition.
func (s *OrderService) Transition(ctx context.Context, orderID, actorID uuid.UUID, actorRole string, to model.OrderStatus, note *string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Clauses(/* FOR UPDATE */ ).First(&order, orderID).Error; err != nil {
			return err
		}

		allowed := model.ValidTransitions[order.Status]
		valid := false
		for _, s := range allowed {
			if s == to {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("%w: %s → %s", ErrIllegalTransition, order.Status, to)
		}

		from := order.Status
		order.Status = to
		if to == model.OrderDelivered {
			now := time.Now()
			order.DeliveredAt = &now
		}

		if err := tx.Save(&order).Error; err != nil {
			return err
		}

		return tx.Create(&model.OrderEvent{
			ID:         uuid.New(),
			OrderID:    orderID,
			FromStatus: from,
			ToStatus:   to,
			ActorID:    actorID,
			ActorRole:  actorRole,
			Note:       note,
		}).Error
	})
}

func (s *OrderService) GetByID(ctx context.Context, orderID, requesterID uuid.UUID, requesterRole string) (*model.Order, error) {
	var order model.Order
	if err := s.db.WithContext(ctx).Preload("Items").Preload("Events").First(&order, orderID).Error; err != nil {
		return nil, err
	}
	// Row-level ownership: customer sees own orders, driver sees assigned, merchant sees theirs, admin sees all
	switch requesterRole {
	case "customer":
		if order.CustomerID != requesterID {
			return nil, errors.New("forbidden")
		}
	case "driver":
		if order.DriverID == nil || *order.DriverID != requesterID {
			return nil, errors.New("forbidden")
		}
	case "merchant":
		if order.MerchantID != requesterID {
			return nil, errors.New("forbidden")
		}
	}
	return &order, nil
}

// Cancel handles cancellation with the refund engine.
func (s *OrderService) Cancel(ctx context.Context, orderID, actorID uuid.UUID, actorRole, reason string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.First(&order, orderID).Error; err != nil {
			return err
		}

		// Validate transition
		allowed := model.ValidTransitions[order.Status]
		canCancel := false
		for _, s := range allowed {
			if s == model.OrderCancelled {
				canCancel = true
				break
			}
		}
		if !canCancel {
			return fmt.Errorf("%w: cannot cancel from %s", ErrIllegalTransition, order.Status)
		}

		order.Status = model.OrderCancelled
		order.CancelReason = &reason
		if err := tx.Save(&order).Error; err != nil {
			return err
		}

		// Audit
		actorUUID := actorID
		if err := tx.Create(&model.OrderEvent{
			ID:         uuid.New(),
			OrderID:    orderID,
			FromStatus: order.Status,
			ToStatus:   model.OrderCancelled,
			ActorID:    actorUUID,
			ActorRole:  actorRole,
			Note:       &reason,
		}).Error; err != nil {
			return err
		}

		// Refund engine
		return s.ledger.ProcessCancellationRefund(ctx, tx, &order)
	})
}

// GetStops returns all stops for a multi-drop package order, ordered by sequence.
func (s *OrderService) GetStops(ctx context.Context, orderID uuid.UUID) ([]model.OrderStop, error) {
	var stops []model.OrderStop
	err := s.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("sequence ASC").
		Find(&stops).Error
	return stops, err
}

// ConfirmStop marks a single stop as confirmed using the per-stop delivery code.
// When all stops are confirmed the order transitions to delivered and escrow settles.
func (s *OrderService) ConfirmStop(ctx context.Context, orderID, driverID uuid.UUID, sequence int, code string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Preload("Stops").First(&order, orderID).Error; err != nil {
			return fmt.Errorf("order not found")
		}
		if order.DriverID == nil || *order.DriverID != driverID {
			return fmt.Errorf("not the assigned driver")
		}
		if order.Status != model.OrderInTransit {
			return fmt.Errorf("order not in transit")
		}

		var stop *model.OrderStop
		for i := range order.Stops {
			if order.Stops[i].Sequence == sequence {
				stop = &order.Stops[i]
				break
			}
		}
		if stop == nil {
			return fmt.Errorf("stop %d not found", sequence)
		}
		if stop.Status == "confirmed" {
			return fmt.Errorf("stop already confirmed")
		}

		// Verify delivery code — uses the shared DeliveryCodeService
		if s.deliveryCodes != nil {
			if err := s.deliveryCodes.Verify(ctx, tx, orderID, code); err != nil {
				return err
			}
		}

		now := time.Now()
		stop.Status = "confirmed"
		stop.ConfirmedAt = &now
		if err := tx.Save(stop).Error; err != nil {
			return err
		}

		// Check if all stops are confirmed → settle and deliver
		allDone := true
		for _, st := range order.Stops {
			if st.ID == stop.ID {
				continue
			}
			if st.Status != "confirmed" {
				allDone = false
				break
			}
		}

		if allDone {
			if err := s.ledger.Settle(ctx, tx, &order, uuid.New()); err != nil {
				return fmt.Errorf("settlement: %w", err)
			}
			order.Status = model.OrderDelivered
			order.DeliveredAt = &now
			if err := tx.Save(&order).Error; err != nil {
				return err
			}
			return tx.Create(&model.OrderEvent{
				ID:         uuid.New(),
				OrderID:    orderID,
				FromStatus: model.OrderInTransit,
				ToStatus:   model.OrderDelivered,
				ActorID:    driverID,
				ActorRole:  "driver",
			}).Error
		}
		return nil
	})
}
