package service

import (
	"fmt"
	"context"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	PointsOrderCompleted = 50
	PointsReferral       = 200
	PointsDailyLogin     = 5
)

type LoyaltyService struct {
	db *gorm.DB
}

func NewLoyaltyService(db *gorm.DB) *LoyaltyService {
	return &LoyaltyService{db: db}
}

func (s *LoyaltyService) Award(ctx context.Context, tx *gorm.DB, userID uuid.UUID, eventType string, points int, refID *uuid.UUID) error {
	if tx == nil {
		tx = s.db
	}
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
	var b model.LoyaltyBalance
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&b).Error; err != nil {
		return 0, nil
	}
	return b.Points, nil
}

func (s *LoyaltyService) Redeem(ctx context.Context, userID uuid.UUID, points int) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var b model.LoyaltyBalance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", userID).First(&b).Error; err != nil {
			return err
		}
		if b.Points < points {
			return fmt.Errorf("insufficient loyalty points")
		}
		return tx.Exec(
			`UPDATE loyalty_balances SET points = points - ?, updated_at = NOW() WHERE user_id = ?`,
			points, userID,
		).Error
	})
}

func (s *LoyaltyService) History(ctx context.Context, userID uuid.UUID, limit int) ([]model.LoyaltyEvent, error) {
	var events []model.LoyaltyEvent
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}
