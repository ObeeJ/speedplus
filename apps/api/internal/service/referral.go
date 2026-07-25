package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

const ReferralRewardKobo = 50000 // ₦500 to referrer wallet after referee's first order

type ReferralService struct {
	db      *gorm.DB
	ledger  *LedgerService
	loyalty *LoyaltyService
}

func NewReferralService(db *gorm.DB, ledger *LedgerService, loyalty *LoyaltyService) *ReferralService {
	return &ReferralService{db: db, ledger: ledger, loyalty: loyalty}
}

// Record links referee to referrer at registration. Safe to call with empty code.
func (s *ReferralService) Record(ctx context.Context, refereeID uuid.UUID, referrerID uuid.UUID) error {
	if referrerID == refereeID {
		return nil
	}
	ref := model.Referral{ID: uuid.New(), ReferrerID: referrerID, RefereeID: refereeID}
	return s.db.WithContext(ctx).Create(&ref).Error
}

// SettleFirstOrder pays out the referral reward. Called after referee's first completed order.
func (s *ReferralService) SettleFirstOrder(ctx context.Context, refereeID uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ref model.Referral
		if err := tx.Where("referee_id = ? AND reward_paid_at IS NULL", refereeID).First(&ref).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil // no referral, nothing to do
			}
			return err
		}

		if err := s.ledger.CreditWallet(ctx, tx, ref.ReferrerID, ReferralRewardKobo, "referral", &ref.ID); err != nil {
			return err
		}

		refID := ref.ID
		if err := s.loyalty.Award(ctx, tx, ref.ReferrerID, "referral", PointsReferral, &refID); err != nil {
			return err
		}

		return tx.Model(&ref).Update("reward_paid_at", gorm.Expr("NOW()")).Error
	})
}
