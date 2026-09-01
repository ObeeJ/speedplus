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
	repo     repo.WalletRepo
	ledger   *LedgerService
	pins     ports.PINVerifier
	provider payment.Provider
	email    walletEmailSender
	users    repo.UserRepo
	enqueue  func(cashoutID string) error
	velocity *VelocityService // nil = no velocity checks (dev/test)
	tiers    *TierService     // nil = no tier lookup (dev/test)
}

type walletEmailSender interface {
	SendWalletFunded(ctx context.Context, toEmail, firstName string, amountKobo, newBalanceKobo int64)
	SendTransferReceived(ctx context.Context, toEmail, firstName string, amountKobo int64, senderName string)
}

func NewWalletService(r repo.WalletRepo, ledger *LedgerService, pins ports.PINVerifier, provider payment.Provider, email walletEmailSender, users repo.UserRepo) *WalletService {
	return &WalletService{repo: r, ledger: ledger, pins: pins, provider: provider, email: email, users: users}
}

// InjectQueue wires the asynq enqueue function after construction to avoid a
// circular dependency. Must be called before any cashout is initiated.
func (s *WalletService) InjectQueue(fn func(cashoutID string) error) {
	s.enqueue = fn
}

// InjectVelocity wires velocity checking after construction.
func (s *WalletService) InjectVelocity(v *VelocityService, t *TierService) {
	s.velocity = v
	s.tiers = t
}

// ── Fund wallet (pay-in) ──────────────────────────────────────────────────────

func (s *WalletService) InitiateFund(ctx context.Context, userID uuid.UUID, amountKobo int64, email, idempotencyKey, callbackURL string) (*payment.ChargeResponse, error) {
	if existing, err := s.repo.FindPaymentIntentByKey(ctx, idempotencyKey); err == nil {
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
	if err := s.repo.CreatePaymentIntent(ctx, &intent); err != nil {
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

// ── Stablecoin fund via Bridge.xyz ───────────────────────────────────────────

// InitiateCryptoFund creates a Bridge Orchestration payment session so the user
// can fund their wallet with USDC/USDT. bridge must be a *payment.BridgeProvider.
func (s *WalletService) InitiateCryptoFund(
	ctx context.Context,
	userID uuid.UUID,
	amountKobo int64,
	email, fullName, idempotencyKey, callbackURL string,
	bridge *payment.BridgeProvider,
) (*payment.ChargeResponse, error) {
	if existing, err := s.repo.FindPaymentIntentByKey(ctx, idempotencyKey); err == nil {
		return &payment.ChargeResponse{Reference: *existing.ProviderRef}, nil
	}

	walletID, err := bridge.EnsureWallet(ctx, userID.String(), email, fullName)
	if err != nil {
		return nil, fmt.Errorf("bridge wallet: %w", err)
	}

	ref := uuid.NewString()
	intent := model.PaymentIntent{
		ID:             uuid.New(),
		UserID:         userID,
		AmountKobo:     amountKobo,
		Provider:       "bridge",
		Status:         "pending",
		IdempotencyKey: idempotencyKey,
		ProviderRef:    &ref,
	}
	if err := s.repo.CreatePaymentIntent(ctx, &intent); err != nil {
		return nil, err
	}

	resp, err := bridge.InitiateCharge(ctx, payment.ChargeRequest{
		AmountKobo:  amountKobo,
		Email:       email,
		Reference:   ref,
		CallbackURL: callbackURL,
		Metadata:    map[string]string{"bridge_wallet_id": walletID},
	})
	if err != nil {
		return nil, fmt.Errorf("bridge charge: %w", err)
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
	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		if _, err := s.repo.FindWebhookEventTx(ctx, tx, p.Provider, p.EventID); err == nil {
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
		if err := s.repo.CreateWebhookEventTx(ctx, tx, &event); err != nil {
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

		intent, err := s.repo.LockPaymentIntentByRefTx(ctx, tx, p.Reference)
		if err != nil {
			// The provider confirmed a payment we hold no intent for. Almost
			// always a race: the webhook beat POST /wallet/fund's commit.
			// Acking here would strand the customer's money permanently — the
			// webhook_events row written above suppresses every future
			// redelivery of this event ID. Returning retryable rolls the whole
			// transaction back (that row included) so the provider redelivers.
			observability.CaptureError(ctx, err, "webhook: no payment intent for reference — forcing provider retry",
				"provider", p.Provider, "reference", p.Reference)
			return ErrWebhookRetryable
		}

		// Already settled by an earlier delivery of this same event: ack without
		// re-crediting. Committing here persists the webhook_events row, so
		// later redeliveries short-circuit at the replay guard above.
		if intent.Status == "success" {
			return nil
		}

		if err := s.ledger.CreditWallet(ctx, tx, intent.UserID, verified.AmountKobo, "payment_intent", &intent.ID); err != nil {
			// Ledger write failed — transient, must retry
			return ErrWebhookRetryable
		}

		// Velocity check for fund — runs after credit so the counter only
		// increments on confirmed payments, not on failed/pending intents.
		// Non-blocking: a velocity breach on a confirmed payment is logged
		// but does not roll back the credit (money already moved at provider).
		if s.velocity != nil && s.tiers != nil {
			tier := s.tiers.GetTier(ctx, intent.UserID)
			if err := s.velocity.Check(ctx, tx, intent.UserID, verified.AmountKobo, tier, "fund"); err != nil {
				slog.WarnContext(ctx, "velocity: fund limit exceeded after credit — flagged but not reversed",
					"user_id", intent.UserID, "amount_kobo", verified.AmountKobo, "error", err)
			}
		}

		intent.Status = "success"
		if err := s.repo.SavePaymentIntentTx(ctx, tx, intent); err != nil {
			return ErrWebhookRetryable
		}

		event.ProcessedAt = &now
		if err := s.repo.SaveWebhookEventTx(ctx, tx, &event); err != nil {
			return ErrWebhookRetryable
		}

		newBal, _ := s.ledger.GetBalance(ctx, intent.UserID)
		// Resolve the recipient here, on the transaction's own connection, and
		// hand the notifier plain values. The email goroutine must never touch
		// the database: it would share the connection this transaction is
		// using, and concurrent use of a single connection panics inside pgx —
		// a panic in a detached goroutine takes down the whole process.
		if u, uErr := s.users.FindByID(ctx, intent.UserID); uErr == nil && u.Email != nil {
			notify = &fundedNotice{
				email:     *u.Email,
				firstName: u.FirstName,
				userID:    intent.UserID,
				amount:    verified.AmountKobo,
				balance:   newBal,
			}
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

// fundedNotice carries everything the wallet-funded email needs, resolved
// inside the crediting transaction. It holds plain values rather than IDs
// precisely so the sender never has to query.
type fundedNotice struct {
	email     string
	firstName string
	userID    uuid.UUID // for log correlation only
	amount    int64
	balance   int64
}

// sendFundedEmail delivers the wallet-funded notification best-effort. It
// performs no database access — see fundedNotice — and recovers from panics so
// a notification failure can never take down the API process or affect the
// already-committed credit.
func (s *WalletService) sendFundedEmail(n fundedNotice) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("wallet funded email panicked", "user_id", n.userID.String(), "panic", r)
			}
		}()
		s.email.SendWalletFunded(context.Background(), n.email, n.firstName, n.amount, n.balance)
	}()
}

// ── Wallet-to-wallet transfer ─────────────────────────────────────────────────

func (s *WalletService) Transfer(ctx context.Context, senderID, recipientID uuid.UUID, amountKobo int64, pin, idempotencyKey string) error {
	// PIN verification
	if err := s.pins.VerifyPIN(ctx, senderID, pin); err != nil {
		return fmt.Errorf("pin verification failed")
	}

	// FIX #2: capture notification data outside the goroutine so the email
	// fires after the transaction commits, never inside it.
	var notifyEmail *string
	var notifyFirst, notifySender string

	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		// Velocity check — inside the transaction so a limit breach rolls back atomically.
		if s.velocity != nil && s.tiers != nil {
			tier := s.tiers.GetTier(ctx, senderID)
			if err := s.velocity.Check(ctx, tx, senderID, amountKobo, tier, "transfer"); err != nil {
				return err
			}
		}
		// idempotency_keys.key is a global PRIMARY KEY, not scoped per user. A
		// key value already claimed by a DIFFERENT account would otherwise read
		// as "this transfer already happened" and return success without moving
		// money. Verify ownership before treating it as a replay.
		if existing, err := s.repo.FindIdempotencyKeyTx(ctx, tx, idempotencyKey); err == nil {
			if existing.UserID != senderID {
				return errors.New("idempotency key is already in use; retry with a fresh key")
			}
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

		// Deadlock avoidance. Two concurrent transfers in opposite directions
		// (A→B and B→A) would each grab one wallet's row lock and then block on
		// the other, and Postgres resolves that by aborting one with 40P01 —
		// surfacing to the user as a spurious failure on a money operation.
		// Acquire BOTH row locks up front in a total order derived from the
		// account UUID, so every concurrent transfer requests them in the same
		// sequence and a cycle can never form. Row locks are re-entrant within
		// a transaction, so re-locking the sender below is free.
		lockFirst, lockSecond := senderWallet.ID, recipientWallet.ID
		if lockFirst.String() > lockSecond.String() {
			lockFirst, lockSecond = lockSecond, lockFirst
		}
		if _, err := s.repo.LockWalletBalanceTx(ctx, tx, lockFirst); err != nil {
			return err
		}
		if _, err := s.repo.LockWalletBalanceTx(ctx, tx, lockSecond); err != nil {
			return err
		}

		senderBal, err := s.repo.LockWalletBalanceTx(ctx, tx, senderWallet.ID)
		if err != nil {
			return err
		}
		if senderBal.BalanceKobo < amountKobo {
			return fmt.Errorf("%w", ErrInsufficientBalance)
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

		if err := s.repo.CreateIdempotencyKeyTx(ctx, tx, &model.IdempotencyKey{
			Key:       idempotencyKey,
			UserID:    senderID,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}); err != nil {
			return err
		}

		// FIX #2: resolve user details INSIDE the transaction (same connection,
		// no goroutine) so we hold plain values. Email fires after commit.
		if sender, sErr := s.users.FindByID(ctx, senderID); sErr == nil {
			if recipient, rErr := s.users.FindByID(ctx, recipientID); rErr == nil && recipient.Email != nil {
				notifyEmail = recipient.Email
				notifyFirst = recipient.FirstName
				notifySender = sender.FirstName + " " + sender.LastName
			}
		}

		return nil
	})
	if err != nil {
		return err
	}
	// Post-commit: send notification with pre-resolved plain values — no DB access.
	if notifyEmail != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("transfer email panicked", "panic", r)
				}
			}()
			s.email.SendTransferReceived(context.Background(), *notifyEmail, notifyFirst, amountKobo, notifySender)
		}()
	}
	return nil
}

// ── EWA cashout ───────────────────────────────────────────────────────────────

const EWACashoutFeeKobo = 10000 // ₦100 instant cashout fee

var ErrBankAccountRequired = errors.New("bank account required — add a verified bank account before withdrawing")

func (s *WalletService) EWACashout(ctx context.Context, driverID uuid.UUID, amountKobo int64, idempotencyKey string) error {
	// Verify bank account exists before opening a transaction or debiting anything.
	// Frontend gating is not a security control.
	if _, err := s.repo.FindDriverBankAccountTx(ctx, nil, driverID); err != nil {
		return ErrBankAccountRequired
	}

	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		// Velocity check for cashout.
		if s.velocity != nil && s.tiers != nil {
			tier := s.tiers.GetTier(ctx, driverID)
			if err := s.velocity.Check(ctx, tx, driverID, amountKobo, tier, "cashout"); err != nil {
				return err
			}
		}

		// See Transfer: the key is globally unique, so confirm this driver owns
		// it before treating it as a replay — otherwise a cashout can return
		// success while paying nothing out.
		if existing, err := s.repo.FindIdempotencyKeyTx(ctx, tx, idempotencyKey); err == nil {
			if existing.UserID != driverID {
				return errors.New("idempotency key is already in use; retry with a fresh key")
			}
			return nil
		}

		totalEarned, err := s.repo.SumUnpaidEarnings(ctx, tx, driverID)
		if err != nil {
			return fmt.Errorf("EWACashout: sum earnings: %w", err)
		}
		if totalEarned < amountKobo+EWACashoutFeeKobo {
			return fmt.Errorf("insufficient earned balance")
		}

		// FIX #1: propagate EnsureWallet / platformAccount errors — discarding
		// them with _ returns a zero-value struct whose ID is uuid.Nil, causing
		// every subsequent ledger write to silently corrupt the wrong account.
		driverWallet, err := s.ledger.EnsureWallet(ctx, tx, driverID)
		if err != nil {
			return fmt.Errorf("EWACashout: driver wallet: %w", err)
		}
		revenueAcct, err := s.ledger.platformAccount(ctx, tx, model.AccountRevenue)
		if err != nil {
			return fmt.Errorf("EWACashout: revenue account: %w", err)
		}
		// FIX #5: use an earnings/liability account as the payout pass-through,
		// not revenue. Revenue should only be recognised as the fee portion.
		// We reuse AccountEarnings (driver-side liability) as the pass-through
		// so the net revenue impact is exactly EWACashoutFeeKobo.
		earningsAcct, err := s.ledger.platformAccount(ctx, tx, model.AccountEarnings)
		if err != nil {
			return fmt.Errorf("EWACashout: earnings account: %w", err)
		}

		journalID := uuid.New()
		entries := []model.LedgerEntry{
			// Debit driver wallet for full amount + fee
			{ID: uuid.New(), JournalID: journalID, AccountID: driverWallet.ID, AmountKobo: -(amountKobo + EWACashoutFeeKobo), Description: "EWA cashout debit", RefType: "cashout"},
			// Fee goes to platform revenue
			{ID: uuid.New(), JournalID: journalID, AccountID: revenueAcct.ID, AmountKobo: EWACashoutFeeKobo, Description: "EWA cashout fee", RefType: "cashout"},
			// Net payout is a pass-through via earnings until provider confirms
			{ID: uuid.New(), JournalID: journalID, AccountID: earningsAcct.ID, AmountKobo: amountKobo, Description: "EWA payout staging", RefType: "cashout"},
		}

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
		return s.repo.CreateCashoutTx(ctx, tx, &cashout)
	})
	if err != nil {
		return err
	}
	if s.enqueue != nil {
		if created, err := s.repo.FindCashoutByKey(ctx, idempotencyKey); err == nil {
			if enqErr := s.enqueue(created.ID.String()); enqErr != nil {
				observability.CaptureError(ctx, enqErr, "EWACashout: enqueue failed", "idempotency_key", idempotencyKey)
			}
		}
	}
	return nil
}

// ── Weekly auto-payout (called by asynq cron) ─────────────────────────────────

func (s *WalletService) WeeklyAutoPayout(ctx context.Context) error {
	drivers, err := s.repo.ListDriversWithUnpaidEarnings(ctx)
	if err != nil {
		return fmt.Errorf("weekly payout: list drivers: %w", err)
	}

	for _, driverID := range drivers {
		total, err := s.repo.SumUnpaidEarnings(ctx, nil, driverID)
		if err != nil {
			observability.CaptureError(ctx, err, "weekly payout: sum earnings", "driver_id", driverID.String())
			continue
		}
		if total <= 0 {
			continue
		}

		isoYear, isoWeek := time.Now().ISOWeek()
		key := fmt.Sprintf("weekly-payout:%s:%d-W%02d", driverID, isoYear, isoWeek)
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

	err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		// Velocity check for merchant withdrawal.
		if s.velocity != nil && s.tiers != nil {
			tier := s.tiers.GetTier(ctx, userID)
			if err := s.velocity.Check(ctx, tx, userID, amountKobo, tier, "withdraw"); err != nil {
				return err
			}
		}

		// See Transfer: the key is globally unique. userID (not merchantID) owns
		// the ledger account, so that is the identity the key must match.
		if existing, err := s.repo.FindIdempotencyKeyTx(ctx, tx, idempotencyKey); err == nil {
			if existing.UserID != userID {
				return errors.New("idempotency key is already in use; retry with a fresh key")
			}
			return nil
		}

		bankAcct, err := s.repo.FindMerchantBankAccountTx(ctx, tx, merchantID)
		if err != nil {
			return fmt.Errorf("no verified bank account on file — add one before withdrawing")
		}
		_ = bankAcct

		var feeKobo int64
		if withdrawalType == "instant" {
			feeKobo = merchantInstantFee(amountKobo)
		}
		totalDebit := amountKobo + feeKobo

		// FIX #4: use EnsureMerchantWallet which resolves merchants.id → user_id
		// before touching ledger_accounts. Passing merchantID directly violates
		// the FK on ledger_accounts.owner_id → users(id).
		merchantWallet, err := s.ledger.EnsureMerchantWallet(ctx, tx, merchantID)
		if err != nil {
			return err
		}
		var bal model.WalletBalance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("account_id = ?", merchantWallet.ID).First(&bal).Error; err != nil {
			return err
		}
		if bal.BalanceKobo < totalDebit {
			return fmt.Errorf("%w: have %s, need %s",
				ErrInsufficientBalance, formatKobo(bal.BalanceKobo), formatKobo(totalDebit))
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
				AmountKobo:  -totalDebit,
				Description: fmt.Sprintf("merchant %s withdrawal debit", withdrawalType), RefType: "cashout"},
			{ID: uuid.New(), JournalID: journalID, AccountID: revenueAcct.ID,
				AmountKobo:  feeKobo,
				Description: fmt.Sprintf("merchant %s withdrawal fee", withdrawalType), RefType: "cashout"},
			// Net payout: revenue is pass-through, balanced when provider confirms transfer.
			{ID: uuid.New(), JournalID: journalID, AccountID: revenueAcct.ID,
				AmountKobo:  -amountKobo,
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
		return s.repo.CreateCashoutTx(ctx, tx, &cashout)
	})
	if err != nil {
		return err
	}
	if s.enqueue != nil {
		if created, err := s.repo.FindCashoutByKey(ctx, idempotencyKey); err == nil && withdrawalType == "instant" {
			if enqErr := s.enqueue(created.ID.String()); enqErr != nil {
				observability.CaptureError(ctx, enqErr, "MerchantWithdraw: enqueue failed", "idempotency_key", idempotencyKey)
			}
		}
	}
	return nil
}

// ReverseCashout credits the wallet back when a provider transfer fails permanently.
// Called by the worker after all retries are exhausted.
func (s *WalletService) ReverseCashout(ctx context.Context, cashout *model.CashoutRequest) error {
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		// Idempotency guard. This runs from an asynq worker, which retries on
		// failure — and a retry after the credit committed but before the caller
		// observed success would credit the wallet a SECOND time. Re-read the row
		// under a lock and bail if the reversal has already been applied.
		//
		// LedgerService.Reverse is deliberately NOT used here: the cashout
		// journal carries offsetting +amount/-amount legs on the same earnings
		// account, and Reverse adjusts every leg individually, so the negative
		// leg would be applied first and rejected by the balance floor. An
		// explicit guard is the correct tool for this journal shape.
		locked, err := s.repo.LockCashoutTx(ctx, tx, cashout.ID)
		if err != nil {
			return fmt.Errorf("reverse cashout: lock cashout %s: %w", cashout.ID, err)
		}
		if locked.Status == "failed" {
			return nil // already reversed — crediting again would duplicate the refund
		}

		// Determine which wallet to credit back
		var wallet *model.LedgerAccount
		if cashout.ActorType == "merchant" && cashout.MerchantID != nil {
			// EnsureMerchantWallet resolves merchants.id → user_id before
			// touching ledger_accounts. Passing merchantID directly to
			// EnsureWallet violates the FK on ledger_accounts.owner_id → users(id).
			var walletErr error
			wallet, walletErr = s.ledger.EnsureMerchantWallet(ctx, tx, *cashout.MerchantID)
			if walletErr != nil {
				return fmt.Errorf("reverse cashout: merchant wallet: %w", walletErr)
			}
		} else {
			var walletErr error
			wallet, walletErr = s.ledger.EnsureWallet(ctx, tx, cashout.DriverID)
			if walletErr != nil {
				return fmt.Errorf("reverse cashout: wallet: %w", walletErr)
			}
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
				AmountKobo:  totalCredit,
				Description: "cashout reversal — transfer failed", RefType: "cashout", RefID: &cashout.ID},
			{ID: uuid.New(), JournalID: journalID, AccountID: revenueAcct.ID,
				AmountKobo:  -totalCredit,
				Description: "cashout reversal — unwind revenue", RefType: "cashout", RefID: &cashout.ID},
		}
		if err := s.ledger.journal(ctx, tx, entries); err != nil {
			return err
		}
		if err := s.ledger.adjustBalance(ctx, tx, wallet.ID, totalCredit); err != nil {
			return fmt.Errorf("reverse cashout: adjust balance: %w", err)
		}

		cashout.Status = "failed"
		return s.repo.SaveCashoutTx(ctx, tx, cashout)
	})
}

func (s *WalletService) DB() *gorm.DB               { return nil }
func (s *WalletService) Provider() payment.Provider { return s.provider }

// ── Driver bank account ───────────────────────────────────────────────────────

// ResolveDriverBankAccount calls the payment provider to verify an account
// number and return the authoritative account name. Never trusts the caller.
func (s *WalletService) ResolveDriverBankAccount(ctx context.Context, bankCode, accountNumber string) (string, error) {
	ps, ok := s.provider.(*payment.PaystackProvider)
	if !ok {
		return "", fmt.Errorf("bank account resolution requires Paystack provider")
	}
	name, err := ps.ResolveAccount(ctx, bankCode, accountNumber)
	if err != nil {
		return "", fmt.Errorf("resolve account: %w", err)
	}
	return name, nil
}

// SaveDriverBankAccount persists a verified bank account for a driver.
// accountName must come from ResolveDriverBankAccount, never from the client.
func (s *WalletService) SaveDriverBankAccount(ctx context.Context, driverID uuid.UUID, bankCode, bankName, accountNumber, accountName string) (*model.DriverBankAccount, error) {
	acct := &model.DriverBankAccount{
		DriverID:      driverID,
		BankCode:      bankCode,
		BankName:      bankName,
		AccountNumber: accountNumber,
		AccountName:   accountName,
		IsVerified:    true,
	}
	if err := s.users.UpsertDriverBankAccount(ctx, acct); err != nil {
		return nil, err
	}
	return acct, nil
}

// GetDriverBankAccount returns the driver's saved bank account, or nil.
func (s *WalletService) GetDriverBankAccount(ctx context.Context, driverID uuid.UUID) (*model.DriverBankAccount, error) {
	acct, err := s.repo.FindDriverBankAccountTx(ctx, nil, driverID)
	if err != nil {
		return nil, nil // not found is not an error on the read path
	}
	return acct, nil
}

func (s *WalletService) ResolveCashoutRecipient(ctx context.Context, cashout *model.CashoutRequest) (recipientCode, accountName, reason string, err error) {
	if cashout.ActorType == "merchant" && cashout.MerchantID != nil {
		bankAcct, err := s.repo.FindMerchantBankAccountTx(ctx, nil, *cashout.MerchantID)
		if err != nil {
			return "", "", "", fmt.Errorf("merchant bank account: %w", err)
		}
		if s.provider.Name() == "monnify" {
			return bankAcct.BankCode + ":" + bankAcct.AccountNumber, bankAcct.AccountName, "Fourdat merchant payout", nil
		}
		return bankAcct.AccountNumber, bankAcct.AccountName, "Fourdat merchant payout", nil
	}
	// Driver payout
	bankAcct, err := s.repo.FindDriverBankAccountTx(ctx, nil, cashout.DriverID)
	if err != nil {
		return "", "", "", fmt.Errorf("driver bank account not found — driver must add a bank account before withdrawing: %w", err)
	}
	if s.provider.Name() == "monnify" {
		return bankAcct.BankCode + ":" + bankAcct.AccountNumber, bankAcct.AccountName, "Fourdat driver payout", nil
	}
	return bankAcct.AccountNumber, bankAcct.AccountName, "Fourdat driver payout", nil
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
