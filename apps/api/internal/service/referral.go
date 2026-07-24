package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

const ReferralRewardKobo = 50000 // ₦500 to referrer wallet after referee's first qualifying order

// ReferralMinOrderKobo: orders below this subtotal don't qualify (and don't
// consume the reward — a later qualifying order still pays it). Blocks
// bonus-farming via ₦100 token orders.
const ReferralMinOrderKobo = 200000 // ₦2,000

type ReferralService struct {
	db      *gorm.DB
	ledger  *LedgerService
	loyalty *LoyaltyService
}

func NewReferralService(db *gorm.DB, ledger *LedgerService, loyalty *LoyaltyService) *ReferralService {
	return &ReferralService{db: db, ledger: ledger, loyalty: loyalty}
}

// Record links referee to referrer at registration. Safe to call with empty code.
// Rejects self-referral (same user or same phone across accounts) and referrers
// whose trust tier is frozen.
func (s *ReferralService) Record(ctx context.Context, refereeID uuid.UUID, referrerID uuid.UUID) error {
	if referrerID == refereeID {
		return nil
	}

	var referrer, referee model.User
	if err := s.db.WithContext(ctx).First(&referrer, referrerID).Error; err != nil {
		return nil // unknown referrer code — ignore silently, registration proceeds
	}
	if err := s.db.WithContext(ctx).First(&referee, refereeID).Error; err != nil {
		return err
	}
	if normalizePhone(referrer.Phone) != "" && normalizePhone(referrer.Phone) == normalizePhone(referee.Phone) {
		return nil // same human, different account — no referral link
	}

	var tier model.UserTrustTier
	if err := s.db.WithContext(ctx).Where("user_id = ?", referrerID).First(&tier).Error; err == nil && tier.Frozen {
		return nil // frozen accounts earn nothing
	}

	ref := model.Referral{ID: uuid.New(), ReferrerID: referrerID, RefereeID: refereeID}
	return s.db.WithContext(ctx).Create(&ref).Error
}

// normalizePhone strips non-digits and maps a leading 0 to +234 so the same
// SIM registered two ways still matches.
func normalizePhone(p string) string {
	var b []rune
	for _, r := range p {
		if r >= '0' && r <= '9' || r == '+' {
			b = append(b, r)
		}
	}
	n := string(b)
	if len(n) > 1 && n[0] == '0' {
		n = "+234" + n[1:]
	}
	return n
}

// SettleCompletedOrder pays out the referral reward after a referee's
// completed order, provided the order meets the minimum subtotal. Idempotent —
// reward_paid_at guards double payment.
func (s *ReferralService) SettleCompletedOrder(ctx context.Context, refereeID uuid.UUID, subtotalKobo int64) error {
	if subtotalKobo < ReferralMinOrderKobo {
		return nil // below threshold: reward stays pending for a later qualifying order
	}
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
