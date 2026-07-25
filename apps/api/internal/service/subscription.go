package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/observability"
	"gorm.io/gorm"
)

type SubscriptionService struct {
	db     *gorm.DB
	orders *OrderService
	ledger *LedgerService
}

func NewSubscriptionService(db *gorm.DB, orders *OrderService, ledger *LedgerService) *SubscriptionService {
	return &SubscriptionService{db: db, orders: orders, ledger: ledger}
}

func (s *SubscriptionService) Create(ctx context.Context, customerID, merchantID, addressID uuid.UUID, vertical, cadence, paymentMethod string) (*model.Subscription, error) {
	nextCharge := nextChargeTime(cadence)
	sub := &model.Subscription{
		ID:            uuid.New(),
		CustomerID:    customerID,
		MerchantID:    merchantID,
		Vertical:      vertical,
		Cadence:       cadence,
		AddressID:     addressID,
		PaymentMethod: paymentMethod,
		Status:        "active",
		NextChargeAt:  nextCharge,
	}
	if err := s.db.WithContext(ctx).Create(sub).Error; err != nil {
		return nil, err
	}
	return sub, nil
}

func (s *SubscriptionService) Pause(ctx context.Context, customerID, subID uuid.UUID) error {
	return s.db.WithContext(ctx).
		Model(&model.Subscription{}).
		Where("id = ? AND customer_id = ? AND status = 'active'", subID, customerID).
		Update("status", "paused").Error
}

func (s *SubscriptionService) Cancel(ctx context.Context, customerID, subID uuid.UUID) error {
	return s.db.WithContext(ctx).
		Model(&model.Subscription{}).
		Where("id = ? AND customer_id = ?", subID, customerID).
		Update("status", "cancelled").Error
}

// ProcessDue is called by the asynq cron job. It charges and re-schedules all due subscriptions.
func (s *SubscriptionService) ProcessDue(ctx context.Context) error {
	var subs []model.Subscription
	if err := s.db.WithContext(ctx).
		Where("status = 'active' AND next_charge_at <= NOW()").
		Find(&subs).Error; err != nil {
		return err
	}

	for _, sub := range subs {
		if err := s.chargeOne(ctx, sub); err != nil {
			observability.CaptureError(ctx, err, "subscription charge failed",
				"subscription_id", sub.ID.String(), "customer_id", sub.CustomerID.String())
			s.db.WithContext(ctx).Model(&sub).Updates(map[string]interface{}{
				"dunning_count":  gorm.Expr("dunning_count + 1"),
				"next_charge_at": time.Now().Add(24 * time.Hour), // retry tomorrow
			})
			// Suspend after 3 failed dunning attempts
			if sub.DunningCount+1 >= 3 {
				s.db.WithContext(ctx).Model(&sub).Update("status", "paused")
			}
			continue
		}
		s.db.WithContext(ctx).Model(&sub).Updates(map[string]interface{}{
			"dunning_count":  0,
			"next_charge_at": nextChargeTime(sub.Cadence),
		})
	}
	return nil
}

// chargeOne is intentionally unimplemented — it does NOT debit the wallet or
// create an order. Order creation via OrderService (pricing quote, escrow
// fund, dispatch) is not wired yet. Returning an error here (rather than a
// silent nil "success") is deliberate: ProcessDue's existing dunning logic
// treats every call as a failed attempt, incrementing dunning_count and
// auto-pausing the subscription after 3 consecutive cron cycles — so every
// currently-active subscription self-pauses within a few days instead of
// cycling forever as a phantom "active" subscription that never charges and
// never delivers. Do not add a wallet debit here without also wiring the
// order-creation call in the same transaction — a debit with no order is a
// direct money-loss bug for the customer.
func (s *SubscriptionService) chargeOne(ctx context.Context, sub model.Subscription) error {
	return fmt.Errorf("subscription order creation not implemented (subscription %s auto-pausing via dunning)", sub.ID)
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
