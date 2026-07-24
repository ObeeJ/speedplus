package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/observability"
	"github.com/speedplus/api/internal/payment"
	"github.com/speedplus/api/internal/ports"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WalletService struct {
	db       *gorm.DB
	ledger   *LedgerService
	pins     ports.PINVerifier
	provider payment.Provider
	email    walletEmailSender
	users    repo.UserRepo
}

type walletEmailSender interface {
	SendWalletFunded(ctx context.Context, toEmail, firstName string, amountKobo, newBalanceKobo int64)
	SendTransferReceived(ctx context.Context, toEmail, firstName string, amountKobo int64, senderName string)
}

func NewWalletService(db *gorm.DB, ledger *LedgerService, pins ports.PINVerifier, provider payment.Provider, email walletEmailSender, users repo.UserRepo) *WalletService {
	return &WalletService{db: db, ledger: ledger, pins: pins, provider: provider, email: email, users: users}
}

// ── Fund wallet (pay-in) ──────────────────────────────────────────────────────

func (s *WalletService) InitiateFund(ctx context.Context, userID uuid.UUID, amountKobo int64, email, idempotencyKey, callbackURL string) (*payment.ChargeResponse, error) {
	// Idempotency check
	var existing model.PaymentIntent
	if err := s.db.WithContext(ctx).Where("idempotency_key = ?", idempotencyKey).First(&existing).Error; err == nil {
		return &payment.ChargeResponse{Reference: *existing.ProviderRef}, nil
	}

	ref := uuid.NewString()
	intent := model.PaymentIntent{
		ID:             uuid.New(),
		UserID:         userID,
		AmountKobo:     amountKobo,
		Provider:       s.provider.Name(),
		Status:         "pending",
		IdempotencyKey: idempotencyKey,
		ProviderRef:    &ref,
	}
	if err := s.db.WithContext(ctx).Create(&intent).Error; err != nil {
		return nil, err
	}

	resp, err := s.provider.InitiateCharge(ctx, payment.ChargeRequest{
		AmountKobo:  amountKobo,
		Email:       email,
		Reference:   ref,
		CallbackURL: callbackURL,
	})
	if err != nil {
		return nil, fmt.Errorf("initiate charge: %w", err)
	}
	return resp, nil
}

// ── Webhook processing (dedupe + verify-API call before crediting) ────────────

type WebhookPayload struct {
	Provider  string
	EventID   string
	EventType string
	Reference string
	RawBody   []byte
}

// ErrWebhookRetryable signals a transient failure — the handler returns 5xx
// so the provider retries. Non-retryable outcomes (already processed, unknown
// reference) return nil so the handler returns 200 and stops retries.
var ErrWebhookRetryable = errors.New("transient webhook error — retry")

func (s *WalletService) ProcessWebhook(ctx context.Context, p WebhookPayload) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Dedupe: skip if already processed — non-retryable, return nil → 200
		var existing model.WebhookEvent
		if err := tx.Where("provider = ? AND event_id = ?", p.Provider, p.EventID).First(&existing).Error; err == nil {
			return nil
		}

		now := time.Now()
		event := model.WebhookEvent{
			ID:        uuid.New(),
			Provider:  p.Provider,
			EventID:   p.EventID,
			EventType: p.EventType,
			Payload:   string(p.RawBody),
		}
		if err := tx.Create(&event).Error; err != nil {
			// DB write failed — transient, provider must retry
			return ErrWebhookRetryable
		}

		if p.EventType != "charge.success" && p.EventType != "transfer.success" {
			return nil // non-credit event — 200, no retry needed
		}

		// Provider verify-API call — NEVER trust webhook payload alone
		verified, err := s.provider.VerifyTransaction(ctx, p.Reference)
		if err != nil {
			observability.CaptureError(ctx, err, "webhook: provider verify failed",
				"provider", p.Provider, "reference", p.Reference)
			// Provider API unreachable — transient, must retry
			return ErrWebhookRetryable
		}
		if verified.Status != "success" {
			// Payment genuinely not successful — non-retryable, 200
			observability.CaptureMessage(ctx, "webhook: transaction not successful",
				"provider", p.Provider, "reference", p.Reference, "status", verified.Status)
			return nil
		}

		var intent model.PaymentIntent
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("provider_ref = ? AND status = 'pending'", p.Reference).
			First(&intent).Error; err != nil {
			// Already credited or reference unknown — non-retryable, 200
			return nil
		}

		if err := s.ledger.CreditWallet(ctx, tx, intent.UserID, verified.AmountKobo, "payment_intent", &intent.ID); err != nil {
			// Ledger write failed — transient, must retry
			return ErrWebhookRetryable
		}

		intent.Status = "success"
		if err := tx.Save(&intent).Error; err != nil {
			return ErrWebhookRetryable
		}

		event.ProcessedAt = &now
		if err := tx.Save(&event).Error; err != nil {
			return ErrWebhookRetryable
		}

		newBal, _ := s.ledger.GetBalance(ctx, intent.UserID)
		go func(userID uuid.UUID, amount, balance int64) {
			u, err := s.users.FindByID(context.Background(), userID)
			if err == nil && u.Email != nil {
				s.email.SendWalletFunded(context.Background(), *u.Email, u.FirstName, amount, balance)
			}
		}(intent.UserID, verified.AmountKobo, newBal)

		return nil
	})
}

// ── Wallet-to-wallet transfer ─────────────────────────────────────────────────

func (s *WalletService) Transfer(ctx context.Context, senderID, recipientID uuid.UUID, amountKobo int64, pin, idempotencyKey string) error {
	// PIN verification
	if err := s.pins.VerifyPIN(ctx, senderID, pin); err != nil {
		return fmt.Errorf("pin verification failed")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Idempotency
		var existing model.IdempotencyKey
		if err := tx.Where("key = ?", idempotencyKey).First(&existing).Error; err == nil {
			return nil
		}

		senderWallet, err := s.ledger.EnsureWallet(ctx, tx, senderID)
		if err != nil {
			return err
		}
		recipientWallet, err := s.ledger.EnsureWallet(ctx, tx, recipientID)
		if err != nil {
			return err
		}

		// Check balance (FOR UPDATE already in adjustBalance)
		var senderBal model.WalletBalance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("account_id = ?", senderWallet.ID).First(&senderBal).Error; err != nil {
			return err
		}
		if senderBal.BalanceKobo < amountKobo {
			return fmt.Errorf("insufficient balance")
		}

		journalID := uuid.New()
		entries := []model.LedgerEntry{
			{ID: uuid.New(), JournalID: journalID, AccountID: senderWallet.ID, AmountKobo: -amountKobo, Description: "wallet transfer out", RefType: "transfer"},
			{ID: uuid.New(), JournalID: journalID, AccountID: recipientWallet.ID, AmountKobo: amountKobo, Description: "wallet transfer in", RefType: "transfer"},
		}
		if err := s.ledger.journal(ctx, tx, entries); err != nil {
			return err
		}
		// FIX #6: propagate adjustBalance errors — discarding them lets the ledger
		// and wallet_balances table diverge silently.
		if err := s.ledger.adjustBalance(ctx, tx, senderWallet.ID, -amountKobo); err != nil {
			return fmt.Errorf("transfer: adjust sender balance: %w", err)
		}
		if err := s.ledger.adjustBalance(ctx, tx, recipientWallet.ID, amountKobo); err != nil {
			return fmt.Errorf("transfer: adjust recipient balance: %w", err)
		}

		if err := tx.Create(&model.IdempotencyKey{
			Key:       idempotencyKey,
			UserID:    senderID,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}).Error; err != nil {
			return err
		}

		// Transfer received email — best-effort, after transaction commits.
		go func(sid, rid uuid.UUID, amount int64) {
			sender, err := s.users.FindByID(context.Background(), sid)
			if err != nil {
				return
			}
			recipient, err := s.users.FindByID(context.Background(), rid)
			if err != nil {
				return
			}
			if recipient.Email != nil {
				s.email.SendTransferReceived(context.Background(), *recipient.Email, recipient.FirstName, amount,
					sender.FirstName+" "+sender.LastName)
			}
		}(senderID, recipientID, amountKobo)

		return nil
	})
}

// ── EWA cashout ───────────────────────────────────────────────────────────────

const EWACashoutFeeKobo = 10000 // ₦100 instant cashout fee

func (s *WalletService) EWACashout(ctx context.Context, driverID uuid.UUID, amountKobo int64, idempotencyKey string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Idempotency
		var existing model.CashoutRequest
		if err := tx.Where("idempotency_key = ?", idempotencyKey).First(&existing).Error; err == nil {
			return nil
		}

		// Check unpaid earnings
		var totalEarned int64
		tx.Model(&model.DriverEarning{}).
			Where("driver_id = ? AND paid_out_at IS NULL", driverID).
			Select("COALESCE(SUM(amount_kobo + tip_kobo), 0)").Scan(&totalEarned)

		if totalEarned < amountKobo+EWACashoutFeeKobo {
			return fmt.Errorf("insufficient earned balance")
		}

		// Debit earnings wallet, credit fee to platform
		driverWallet, _ := s.ledger.EnsureWallet(ctx, tx, driverID)
		revenueAcct, _ := s.ledger.platformAccount(ctx, tx, model.AccountRevenue)

		journalID := uuid.New()
		entries := []model.LedgerEntry{
			{ID: uuid.New(), JournalID: journalID, AccountID: driverWallet.ID, AmountKobo: -(amountKobo + EWACashoutFeeKobo), Description: "EWA cashout debit", RefType: "cashout"},
			{ID: uuid.New(), JournalID: journalID, AccountID: revenueAcct.ID, AmountKobo: EWACashoutFeeKobo, Description: "EWA cashout fee", RefType: "cashout"},
		}
		// Net payout to driver's bank is handled by provider transfer
		// We need a "payout" account to balance — use revenue as pass-through
		entries = append(entries, model.LedgerEntry{
			ID: uuid.New(), JournalID: journalID, AccountID: revenueAcct.ID,
			AmountKobo: -amountKobo, Description: "EWA payout to bank", RefType: "cashout",
		})

		if err := s.ledger.journal(ctx, tx, entries); err != nil {
			return err
		}
		s.ledger.adjustBalance(ctx, tx, driverWallet.ID, -(amountKobo + EWACashoutFeeKobo))

		cashout := model.CashoutRequest{
			ID:             uuid.New(),
			DriverID:       driverID,
			AmountKobo:     amountKobo,
			FeeKobo:        EWACashoutFeeKobo,
			Status:         "pending",
			IdempotencyKey: idempotencyKey,
		}
		return tx.Create(&cashout).Error
		// Actual provider transfer dispatched by asynq worker
	})
}

// ── Weekly auto-payout (called by asynq cron) ─────────────────────────────────

func (s *WalletService) WeeklyAutoPayout(ctx context.Context) error {
	var drivers []uuid.UUID
	s.db.WithContext(ctx).
		Model(&model.DriverEarning{}).
		Where("paid_out_at IS NULL").
		Distinct("driver_id").
		Pluck("driver_id", &drivers)

	for _, driverID := range drivers {
		var total int64
		s.db.WithContext(ctx).
			Model(&model.DriverEarning{}).
			Where("driver_id = ? AND paid_out_at IS NULL", driverID).
			Select("COALESCE(SUM(amount_kobo + tip_kobo), 0)").Scan(&total)

		if total <= 0 {
			continue
		}

		key := fmt.Sprintf("weekly-payout:%s:%s", driverID, time.Now().Format("2006-W01"))
		if err := s.EWACashout(ctx, driverID, total, key); err != nil {
			observability.CaptureError(ctx, err, "weekly payout failed", "driver_id", driverID.String())
			continue
		}
	}
	return nil
}
