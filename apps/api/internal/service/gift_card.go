package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

type GiftCardService struct {
	repo   repo.GiftCardRepo
	ledger *LedgerService
}

func NewGiftCardService(r repo.GiftCardRepo, ledger *LedgerService) *GiftCardService {
	return &GiftCardService{repo: r, ledger: ledger}
}

func randomGiftCode() (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, 16)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		code[i] = chars[n.Int64()]
	}
	return fmt.Sprintf("%s-%s-%s-%s", code[:4], code[4:8], code[8:12], code[12:]), nil
}

func hashCode(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}

func (s *GiftCardService) Issue(ctx context.Context, issuerID uuid.UUID, amountKobo int64, expiryDays int) (string, *model.GiftCard, error) {
	code, err := randomGiftCode()
	if err != nil {
		return "", nil, err
	}

	var expiresAt *time.Time
	if expiryDays > 0 {
		t := time.Now().AddDate(0, 0, expiryDays)
		expiresAt = &t
	}

	gc := &model.GiftCard{
		ID:         uuid.New(),
		CodeHash:   hashCode(code),
		AmountKobo: amountKobo,
		IssuerID:   issuerID,
		ExpiresAt:  expiresAt,
	}
	if err := s.repo.Create(ctx, gc); err != nil {
		return "", nil, err
	}
	return code, gc, nil
}

func (s *GiftCardService) Redeem(ctx context.Context, redeemerID uuid.UUID, code string) error {
	codeHash := hashCode(code)
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		gc, err := s.repo.LockByCodeHash(ctx, tx, codeHash)
		if err != nil {
			return fmt.Errorf("gift card not found or already redeemed")
		}
		if gc.ExpiresAt != nil && time.Now().After(*gc.ExpiresAt) {
			return fmt.Errorf("gift card expired")
		}

		if err := s.ledger.CreditWallet(ctx, tx, redeemerID, gc.AmountKobo, "gift_card", &gc.ID); err != nil {
			return err
		}

		now := time.Now()
		gc.RedeemedBy = &redeemerID
		gc.RedeemedAt = &now
		return s.repo.SaveTx(ctx, tx, gc)
	})
}
