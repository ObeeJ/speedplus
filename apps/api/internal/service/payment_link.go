package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/observability"
	"github.com/speedplus/api/internal/payment"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type linkEmailSender interface {
	SendPaymentLinkPaid(ctx context.Context, toEmail, firstName string, amountKobo int64, payerName, linkSlug string)
}

type PaymentLinkService struct {
	repo     repo.PaymentLinkRepo
	ledger   *LedgerService
	provider payment.Provider
	email    linkEmailSender
	users    repo.UserRepo
}

func NewPaymentLinkService(r repo.PaymentLinkRepo, ledger *LedgerService, provider payment.Provider, email linkEmailSender, users repo.UserRepo) *PaymentLinkService {
	return &PaymentLinkService{repo: r, ledger: ledger, provider: provider, email: email, users: users}
}

func randomSlug() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil // 12 hex chars
}

func (s *PaymentLinkService) Create(ctx context.Context, creatorID uuid.UUID, amountKobo int64, note *string) (*model.PaymentLink, error) {
	slug, err := randomSlug()
	if err != nil {
		return nil, err
	}
	pl := &model.PaymentLink{
		ID:         uuid.New(),
		Slug:       slug,
		CreatorID:  creatorID,
		AmountKobo: amountKobo,
		Note:       note,
		Status:     "pending",
		ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
	}
	if err := s.repo.Create(ctx, pl); err != nil {
		return nil, err
	}
	return pl, nil
}

func (s *PaymentLinkService) Get(ctx context.Context, slug string) (*model.PaymentLink, error) {
	pl, err := s.repo.FindPendingBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if time.Now().After(pl.ExpiresAt) {
		_ = s.repo.ExpireLink(ctx, pl.ID) //nolint:errcheck — best-effort status update; expiry already enforced by the time check above
		return nil, fmt.Errorf("payment link expired")
	}
	return pl, nil
}

func (s *PaymentLinkService) PayByWallet(ctx context.Context, slug string, payerID uuid.UUID, idempotencyKey string) error {
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		if _, err := s.repo.FindIdempotencyKeyTx(ctx, tx, idempotencyKey); err == nil {
			return nil
		}

		pl, err := s.repo.LockPendingBySlugTx(ctx, tx, slug)
		if err != nil {
			return fmt.Errorf("link not found or inactive")
		}
		if time.Now().After(pl.ExpiresAt) {
			_ = s.repo.ExpireLinkTx(ctx, tx, pl.ID) //nolint:errcheck — best-effort; expiry enforced by time check
			return fmt.Errorf("payment link expired")
		}
		if pl.CreatorID == payerID {
			return fmt.Errorf("cannot pay your own link")
		}

		payerWallet, err := s.ledger.EnsureWallet(ctx, tx, payerID)
		if err != nil {
			return err
		}
		creatorWallet, err := s.ledger.EnsureWallet(ctx, tx, pl.CreatorID)
		if err != nil {
			return err
		}

		var payerBal model.WalletBalance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("account_id = ?", payerWallet.ID).First(&payerBal).Error; err != nil {
			return err
		}
		if payerBal.BalanceKobo < pl.AmountKobo {
			return fmt.Errorf("%w", ErrInsufficientBalance)
		}

		journalID := uuid.New()
		entries := []model.LedgerEntry{
			{ID: uuid.New(), JournalID: journalID, AccountID: payerWallet.ID, AmountKobo: -pl.AmountKobo, Description: "payment link debit", RefType: "payment_link"},
			{ID: uuid.New(), JournalID: journalID, AccountID: creatorWallet.ID, AmountKobo: pl.AmountKobo, Description: "payment link credit", RefType: "payment_link"},
		}
		if err := s.ledger.journal(ctx, tx, entries); err != nil {
			return err
		}
		if err := s.ledger.adjustBalance(ctx, tx, payerWallet.ID, -pl.AmountKobo); err != nil {
			return err
		}
		if err := s.ledger.adjustBalance(ctx, tx, creatorWallet.ID, pl.AmountKobo); err != nil {
			return err
		}

		now := time.Now()
		pl.Status = "paid"
		pl.PaidByID = &payerID
		pl.PaidAt = &now
		if err := s.repo.SaveLinkTx(ctx, tx, pl); err != nil {
			return err
		}

		return s.repo.CreateIdempotencyKeyTx(ctx, tx, &model.IdempotencyKey{
			Key:       idempotencyKey,
			UserID:    payerID,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
	})
}

func (s *PaymentLinkService) InitiateGuestPayment(ctx context.Context, slug, email, callbackURL string) (*payment.ChargeResponse, error) {
	pl, err := s.Get(ctx, slug)
	if err != nil {
		return nil, err
	}

	ref := uuid.NewString()
	intent := model.PaymentIntent{
		ID:          uuid.New(),
		UserID:      pl.CreatorID,
		AmountKobo:  pl.AmountKobo,
		Provider:    s.provider.Name(),
		Status:      "pending",
		ProviderRef: &ref,
	}
	if err := s.repo.CreatePaymentIntent(ctx, &intent); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateLinkProviderRef(ctx, pl.ID, ref, email); err != nil {
		observability.CaptureError(ctx, err, "payment link: UpdateLinkProviderRef failed", "link_id", pl.ID.String())
		// non-fatal: the charge is already initiated; the link row missing the ref
		// is recoverable via the webhook reference match
	}

	return s.provider.InitiateCharge(ctx, payment.ChargeRequest{
		AmountKobo:  pl.AmountKobo,
		Email:       email,
		Reference:   ref,
		CallbackURL: callbackURL,
	})
}
