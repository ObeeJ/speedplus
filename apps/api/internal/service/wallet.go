package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	enqueue  func(cashoutID string) error // injected after construction
}

type walletEmailSender interface {
	SendWalletFunded(ctx context.Context, toEmail, firstName string, amountKobo, newBalanceKobo int64)
	SendTransferReceived(ctx context.Context, toEmail, firstName string, amountKobo int64, senderName string)
}

func NewWalletService(db *gorm.DB, ledger *LedgerService, pins ports.PINVerifier, provider payment.Provider, email walletEmailSender, users repo.UserRepo) *WalletService {
	return &WalletService{db: db, ledger: ledger, pins: pins, provider: provider, email: email, users: users}
}

// InjectQueue wires the asynq enqueue function after construction to avoid a
// circular dependency. Must be called before any cashout is initiated.
func (s *WalletService) InjectQueue(fn func(cashoutID string) error) {
	s.enqueue = fn
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
	var notify *fundedNotice
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		// Defer the funding email until after commit. Querying from a
		// goroutine here would share the connection the open transaction is
		// still using; concurrent use of one connection panics inside pgx,
		// and a panic in a detached goroutine takes down the whole process.
		notify = &fundedNotice{
			userID:  intent.UserID,
			amount:  verified.AmountKobo,
			balance: newBal,
		}

		return nil
	})
	if err != nil {
		return err
	}
	if notify != nil {
		s.sendFundedEmail(*notify)
	}
	return nil
}

// fundedNotice carries the data needed to send a wallet-funded email once the
// crediting transaction has committed.
type fundedNotice struct {
	userID  uuid.UUID
	amount  int64
	balance int64
}

// sendFundedEmail delivers the wallet-funded notification best-effort, on its
// own connection, with panic recovery so a notification failure can never take
// down the API process or affect the already-committed credit.
func (s *WalletService) sendFundedEmail(n fundedNotice) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("wallet funded email panicked", "user_id", n.userID.String(), "panic", r)
			}
		}()
		u, err := s.users.FindByID(context.Background(), n.userID)
		if err != nil || u.Email == nil {
			return
		}
		s.email.SendWalletFunded(context.Background(), *u.Email, u.FirstName, n.amount, n.balance)
	}()
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
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		if err := s.ledger.adjustBalance(ctx, tx, driverWallet.ID, -(amountKobo + EWACashoutFeeKobo)); err != nil {
			return fmt.Errorf("EWACashout: adjust driver balance: %w", err)
		}

		cashout := model.CashoutRequest{
			ID:             uuid.New(),
			DriverID:       driverID,
			AmountKobo:     amountKobo,
			FeeKobo:        EWACashoutFeeKobo,
			Status:         "pending",
			IdempotencyKey: idempotencyKey,
		}
		return tx.Create(&cashout).Error
	})
	if err != nil {
		return err
	}
	// Enqueue outside the transaction — only fires after DB commit is durable.
	if s.enqueue != nil {
		var created model.CashoutRequest
		s.db.WithContext(ctx).Where("idempotency_key = ?", idempotencyKey).First(&created)
		if enqErr := s.enqueue(created.ID.String()); enqErr != nil {
			observability.CaptureError(ctx, enqErr, "EWACashout: enqueue failed", "idempotency_key", idempotencyKey)
		}
	}
	return nil
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

// ── Merchant withdrawal ───────────────────────────────────────────────────────

const (
	// MerchantWithdrawMinKobo is the minimum withdrawal amount (₦1,000).
	MerchantWithdrawMinKobo = 100_000
	// MerchantInstantFeePct is the fee rate for instant withdrawals (1%).
	MerchantInstantFeePct = 0.01
	// MerchantInstantFeeMinKobo is the minimum instant fee (₦10 — covers provider cost).
	MerchantInstantFeeMinKobo = 1_000
	// MerchantInstantFeeMaxKobo caps the instant fee at ₦500.
	MerchantInstantFeeMaxKobo = 50_000
)

// merchantInstantFee computes the 1% instant withdrawal fee, clamped to [min, max].
func merchantInstantFee(amountKobo int64) int64 {
	fee := int64(float64(amountKobo) * MerchantInstantFeePct)
	if fee < MerchantInstantFeeMinKobo {
		return MerchantInstantFeeMinKobo
	}
	if fee > MerchantInstantFeeMaxKobo {
		return MerchantInstantFeeMaxKobo
	}
	return fee
}

// MerchantWithdraw debits the merchant's wallet and enqueues a bank transfer.
// withdrawalType must be "instant" or "standard".
// - standard: free, batched daily at 18:00 WAT.
// - instant: 1% fee (min ₦10, max ₦500), processed within minutes.
// PIN is verified before any money moves. The actual provider transfer is
// dispatched by the asynq worker (TaskCashoutProcess).
func (s *WalletService) MerchantWithdraw(ctx context.Context, merchantID uuid.UUID, userID uuid.UUID, amountKobo int64, pin, idempotencyKey, withdrawalType string) error {
	if amountKobo < MerchantWithdrawMinKobo {
		return fmt.Errorf("minimum withdrawal is %s", formatKobo(MerchantWithdrawMinKobo))
	}
	if withdrawalType != "instant" && withdrawalType != "standard" {
		withdrawalType = "standard"
	}

	// PIN verification — uses the merchant's login User.ID (PIN is on the user row)
	if err := s.pins.VerifyPIN(ctx, userID, pin); err != nil {
		return fmt.Errorf("pin verification failed")
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Idempotency
		var existing model.CashoutRequest
		if err := tx.Where("idempotency_key = ?", idempotencyKey).First(&existing).Error; err == nil {
			return nil
		}

		// Verify bank account is saved
		var bankAcct model.MerchantBankAccount
		if err := tx.Where("merchant_id = ? AND is_verified = true", merchantID).First(&bankAcct).Error; err != nil {
			return fmt.Errorf("no verified bank account on file — add one before withdrawing")
		}

		var feeKobo int64
		if withdrawalType == "instant" {
			feeKobo = merchantInstantFee(amountKobo)
		}
		totalDebit := amountKobo + feeKobo

		// Check balance
		merchantWallet, err := s.ledger.EnsureWallet(ctx, tx, merchantID)
		if err != nil {
			return err
		}
		var bal model.WalletBalance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("account_id = ?", merchantWallet.ID).First(&bal).Error; err != nil {
			return err
		}
		if bal.BalanceKobo < totalDebit {
			return fmt.Errorf("insufficient balance: have %s, need %s",
				formatKobo(bal.BalanceKobo), formatKobo(totalDebit))
		}

		// Debit merchant wallet; fee (if any) goes to platform revenue.
		// Net payout amount is held in revenue as a pass-through until the
		// worker confirms the provider transfer.
		revenueAcct, err := s.ledger.platformAccount(ctx, tx, model.AccountRevenue)
		if err != nil {
			return fmt.Errorf("revenue account: %w", err)
		}
		journalID := uuid.New()
		entries := []model.LedgerEntry{
			{ID: uuid.New(), JournalID: journalID, AccountID: merchantWallet.ID,
				AmountKobo: -totalDebit,
				Description: fmt.Sprintf("merchant %s withdrawal debit", withdrawalType), RefType: "cashout"},
			{ID: uuid.New(), JournalID: journalID, AccountID: revenueAcct.ID,
				AmountKobo: feeKobo,
				Description: fmt.Sprintf("merchant %s withdrawal fee", withdrawalType), RefType: "cashout"},
			// Net payout: revenue is pass-through, balanced when provider confirms transfer.
			{ID: uuid.New(), JournalID: journalID, AccountID: revenueAcct.ID,
				AmountKobo: -amountKobo,
				Description: "merchant payout to bank", RefType: "cashout"},
		}
		if err := s.ledger.journal(ctx, tx, entries); err != nil {
			return err
		}
		if err := s.ledger.adjustBalance(ctx, tx, merchantWallet.ID, -totalDebit); err != nil {
			return fmt.Errorf("adjust merchant balance: %w", err)
		}

		cashout := model.CashoutRequest{
			ID:             uuid.New(),
			DriverID:       uuid.Nil,
			MerchantID:     &merchantID,
			ActorType:      "merchant",
			AmountKobo:     amountKobo,
			FeeKobo:        feeKobo,
			Status:         "pending",
			IdempotencyKey: idempotencyKey,
		}
		return tx.Create(&cashout).Error
	})
	if err != nil {
		return err
	}
	// Enqueue outside the transaction — only fires after DB commit is durable.
	// instant → immediate processing; standard → picked up by 18:00 WAT batch.
	if s.enqueue != nil {
		var created model.CashoutRequest
		s.db.WithContext(ctx).Where("idempotency_key = ?", idempotencyKey).First(&created)
		if withdrawalType == "instant" {
			if enqErr := s.enqueue(created.ID.String()); enqErr != nil {
				observability.CaptureError(ctx, enqErr, "MerchantWithdraw: enqueue failed", "idempotency_key", idempotencyKey)
			}
		}
		// standard withdrawals are picked up by the daily batch cron — no immediate enqueue
	}
	return nil
}

// ReverseCashout credits the wallet back when a provider transfer fails permanently.
// Called by the worker after all retries are exhausted.
func (s *WalletService) ReverseCashout(ctx context.Context, cashout *model.CashoutRequest) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Determine which wallet to credit back
		var ownerID uuid.UUID
		if cashout.ActorType == "merchant" && cashout.MerchantID != nil {
			ownerID = *cashout.MerchantID
		} else {
			ownerID = cashout.DriverID
		}

		wallet, err := s.ledger.EnsureWallet(ctx, tx, ownerID)
		if err != nil {
			return fmt.Errorf("reverse cashout: wallet: %w", err)
		}
		revenueAcct, err := s.ledger.platformAccount(ctx, tx, model.AccountRevenue)
		if err != nil {
			return fmt.Errorf("reverse cashout: revenue account: %w", err)
		}

		// Reverse the original debit: credit wallet back (amount + fee),
		// debit revenue to unwind both the fee credit and the payout pass-through.
		totalCredit := cashout.AmountKobo + cashout.FeeKobo
		journalID := uuid.New()
		entries := []model.LedgerEntry{
			{ID: uuid.New(), JournalID: journalID, AccountID: wallet.ID,
				AmountKobo: totalCredit,
				Description: "cashout reversal — transfer failed", RefType: "cashout", RefID: &cashout.ID},
			{ID: uuid.New(), JournalID: journalID, AccountID: revenueAcct.ID,
				AmountKobo: -totalCredit,
				Description: "cashout reversal — unwind revenue", RefType: "cashout", RefID: &cashout.ID},
		}
		if err := s.ledger.journal(ctx, tx, entries); err != nil {
			return err
		}
		if err := s.ledger.adjustBalance(ctx, tx, wallet.ID, totalCredit); err != nil {
			return fmt.Errorf("reverse cashout: adjust balance: %w", err)
		}

		cashout.Status = "failed"
		return tx.Save(cashout).Error
	})
}

func (s *WalletService) DB() *gorm.DB { return s.db }
func (s *WalletService) Provider() payment.Provider { return s.provider }

// ResolveCashoutRecipient returns the provider recipient code and narration for a cashout.
func (s *WalletService) ResolveCashoutRecipient(ctx context.Context, cashout *model.CashoutRequest) (recipientCode, reason string, err error) {
	if cashout.ActorType == "merchant" && cashout.MerchantID != nil {
		var bankAcct model.MerchantBankAccount
		if err := s.db.WithContext(ctx).Where("merchant_id = ?", cashout.MerchantID).First(&bankAcct).Error; err != nil {
			return "", "", fmt.Errorf("merchant bank account: %w", err)
		}
		if s.provider.Name() == "monnify" {
			return bankAcct.BankCode + ":" + bankAcct.AccountNumber, "SpeedPlus merchant payout", nil
		}
		return bankAcct.AccountNumber, "SpeedPlus merchant payout", nil
	}
	return "", "", fmt.Errorf("driver bank account resolution not yet implemented")
}

func formatKobo(k int64) string {
	s := fmt.Sprintf("%d", k/100)
	out := ""
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += ","
		}
		out += string(ch)
	}
	return "₦" + out
}