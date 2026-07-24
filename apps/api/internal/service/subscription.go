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

func (s *SubscriptionService) chargeOne(ctx context.Context, sub model.Subscription) error {
	wallet, err := s.ledger.EnsureWallet(ctx, nil, sub.CustomerID)
	if err != nil {
		return err
	}
	bal, err := s.ledger.GetBalance(ctx, sub.CustomerID)
	if err != nil {
		return err
	}

	// Estimate cost — use last order price as proxy; real impl queries pricing service
	const estimatedKobo int64 = 500000 // ₦5,000 placeholder
	if bal < estimatedKobo {
		return fmt.Errorf("insufficient wallet balance for subscription %s", sub.ID)
	}
	_ = wallet

	// TODO: create order via OrderService using sub.MerchantID + sub.AddressID
	return nil
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
