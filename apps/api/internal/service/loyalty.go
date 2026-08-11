package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

const (
	PointsOrderCompleted = 50
	PointsReferral       = 200
	PointsDailyLogin     = 5
)

type LoyaltyService struct {
	repo   repo.LoyaltyRepo
	ledger *LedgerService
}

func NewLoyaltyService(r repo.LoyaltyRepo, ledger *LedgerService) *LoyaltyService {
	return &LoyaltyService{repo: r, ledger: ledger}
}

func (s *LoyaltyService) Award(ctx context.Context, tx *gorm.DB, userID uuid.UUID, eventType string, points int, refID *uuid.UUID) error {
	event := model.LoyaltyEvent{
		ID: uuid.New(), UserID: userID, EventType: eventType, Points: points, RefID: refID,
	}
	if err := tx.WithContext(ctx).Create(&event).Error; err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(
		`INSERT INTO loyalty_balances (user_id, points, updated_at)
		 VALUES (?, ?, NOW())
		 ON CONFLICT (user_id) DO UPDATE SET points = loyalty_balances.points + ?, updated_at = NOW()`,
		userID, points, points,
	).Error
}

func (s *LoyaltyService) GetBalance(ctx context.Context, userID uuid.UUID) (int, error) {
	b, err := s.repo.FindBalance(ctx, userID)
	if err != nil {
		return 0, nil
	}
	return b.Points, nil
}

func (s *LoyaltyService) Redeem(ctx context.Context, userID uuid.UUID, points int) error {
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		b, err := s.repo.LockBalance(ctx, tx, userID)
		if err != nil {
			return err
		}
		if b.Points < points {
			return fmt.Errorf("insufficient loyalty points")
		}
		if err := s.repo.DeductBalanceTx(ctx, tx, userID, points); err != nil {
			return err
		}
		amountKobo := int64(points) * 100
		return s.ledger.CreditWallet(ctx, tx, userID, amountKobo, "loyalty_redemption", nil)
	})
}

func (s *LoyaltyService) History(ctx context.Context, userID uuid.UUID, limit int) ([]model.LoyaltyEvent, error) {
	return s.repo.ListEvents(ctx, userID, limit)
}
