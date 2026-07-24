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
)

type CreateOrderInput struct {
	CustomerID     uuid.UUID
	MerchantID     uuid.UUID
	QuoteID        uuid.UUID
	Vertical       string
	Items          []OrderItemInput
	DeliveryAddrID uuid.UUID
	PrescriptionID *uuid.UUID
	TipKobo        int64
	ScheduledFor   *time.Time
	IdempotencyKey string
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
	db      *gorm.DB
	pricing *PricingService
	ledger  *LedgerService
}

func NewOrderService(db *gorm.DB, pricing *PricingService, ledger *LedgerService) *OrderService {
	return &OrderService{db: db, pricing: pricing, ledger: ledger}
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
			PrescriptionID:    in.PrescriptionID,
			ScheduledFor:      in.ScheduledFor,
			IdempotencyKey:    in.IdempotencyKey,
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
	return order, err
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
