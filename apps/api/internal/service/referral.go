package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

const ReferralRewardKobo = 50000
const ReferralMinOrderKobo = 200000

type ReferralService struct {
	repo    repo.ReferralRepo
	users   repo.UserRepo
	ledger  *LedgerService
	loyalty *LoyaltyService
}

func NewReferralService(r repo.ReferralRepo, users repo.UserRepo, ledger *LedgerService, loyalty *LoyaltyService) *ReferralService {
	return &ReferralService{repo: r, users: users, ledger: ledger, loyalty: loyalty}
}

func (s *ReferralService) Record(ctx context.Context, refereeID uuid.UUID, referrerID uuid.UUID) error {
	if referrerID == refereeID {
		return nil
	}

	referrer, err := s.users.FindByID(ctx, referrerID)
	if err != nil {
		return nil // unknown referrer — ignore
	}
	referee, err := s.users.FindByID(ctx, refereeID)
	if err != nil {
		return err
	}
	if normalizePhone(referrer.Phone) != "" && normalizePhone(referrer.Phone) == normalizePhone(referee.Phone) {
		return nil
	}

	tier, err := s.repo.FindTrustTier(ctx, referrerID)
	if err == nil && tier.Frozen {
		return nil
	}

	ref := &model.Referral{ID: uuid.New(), ReferrerID: referrerID, RefereeID: refereeID}
	return s.repo.CreateReferral(ctx, ref)
}

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

func (s *ReferralService) SettleCompletedOrder(ctx context.Context, refereeID uuid.UUID, subtotalKobo int64) error {
	if subtotalKobo < ReferralMinOrderKobo {
		return nil
	}
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		ref, err := s.repo.FindUnpaidReferral(ctx, tx, refereeID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
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

		return s.repo.MarkReferralPaidTx(ctx, tx, ref)
	})
}
