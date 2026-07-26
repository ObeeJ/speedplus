package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/observability"
	"github.com/speedplus/api/internal/payment"
	"github.com/speedplus/api/internal/service"
)

const (
	TaskWeeklyPayout        = "driver:weekly_payout"
	TaskExpireOffers        = "dispatch:expire_offers"
	TaskSubscriptionRun     = "subscription:run"
	TaskCashoutProcess      = "cashout:process"
	TaskMerchantBatchPayout = "cashout:merchant_batch" // daily 18:00 WAT standard withdrawals
	TaskEscrowReconcile     = "ledger:escrow_reconcile"
	TaskPlatformSnapshot    = "ledger:platform_snapshot"
	TaskOnboardUser         = "user:onboard"
	TaskPurgeRecipientPII   = "order:purge_recipient_pii"
)

// ── Scheduler (cron) ──────────────────────────────────────────────────────────

func NewScheduler(redisURL string) *asynq.Scheduler {
	opt, _ := asynq.ParseRedisURI(redisURL)
	s := asynq.NewScheduler(opt, nil)

	// Weekly payout — every Monday 02:00 WAT (UTC+1)
	s.Register("0 1 * * 1", asynq.NewTask(TaskWeeklyPayout, nil))
	// Expire stale offers — every minute
	s.Register("*/1 * * * *", asynq.NewTask(TaskExpireOffers, nil))
	// Subscription charge check — daily 06:00 UTC
	s.Register("0 6 * * *", asynq.NewTask(TaskSubscriptionRun, nil))
	// Merchant standard withdrawal batch — daily 17:00 UTC (18:00 WAT)
	s.Register("0 17 * * *", asynq.NewTask(TaskMerchantBatchPayout, nil))
	// Escrow reconciliation — daily 02:30 WAT (01:30 UTC)
	s.Register("30 1 * * *", asynq.NewTask(TaskEscrowReconcile, nil))
	// Platform balance snapshot — daily 03:00 WAT (02:00 UTC)
	s.Register("0 2 * * *", asynq.NewTask(TaskPlatformSnapshot, nil))
	// NDPR recipient PII purge — daily 04:00 WAT (03:00 UTC)
	s.Register("0 3 * * *", asynq.NewTask(TaskPurgeRecipientPII, nil))

	return s
}

// ── Server ────────────────────────────────────────────────────────────────────

func NewServer(redisURL string) *asynq.Server {
	opt, _ := asynq.ParseRedisURI(redisURL)
	return asynq.NewServer(opt, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			"critical": 6,
			"default":  3,
			"low":      1,
		},
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			observability.CaptureError(ctx, err, "worker task failed", "task_type", task.Type())
		}),
	})
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// Handlers holds all worker dependencies.
type Handlers struct {
	subscriptions *service.SubscriptionService
	wallet        *service.WalletService
	dispatch      *service.DispatchService
	ledger        *service.LedgerService
	onboarding    onboardingRunner
	orders        *service.OrderService
	asynqClient   *asynq.Client
}

// onboardingRunner is the subset of ports.OnboardingRunner used by the worker.
type onboardingRunner interface {
	RunByID(ctx context.Context, userID string) error
}

func NewHandlers(wallet *service.WalletService, dispatch *service.DispatchService, ledger *service.LedgerService, subscriptions *service.SubscriptionService, onboarding onboardingRunner, asynqClient *asynq.Client) *Handlers {
	return &Handlers{wallet: wallet, dispatch: dispatch, ledger: ledger, subscriptions: subscriptions, onboarding: onboarding, asynqClient: asynqClient}
}

// InjectOrders wires OrderService after construction (avoids widening the
// NewHandlers constructor for a single cron dependency).
func (h *Handlers) InjectOrders(o *service.OrderService) {
	h.orders = o
}

func (h *Handlers) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskWeeklyPayout, h.handleWeeklyPayout)
	mux.HandleFunc(TaskExpireOffers, h.handleExpireOffers)
	mux.HandleFunc(TaskSubscriptionRun, h.handleSubscriptionRun)
	mux.HandleFunc(TaskCashoutProcess, h.handleCashoutProcess)
	mux.HandleFunc(TaskMerchantBatchPayout, h.handleMerchantBatchPayout)
	mux.HandleFunc(TaskEscrowReconcile, h.handleEscrowReconcile)
	mux.HandleFunc(TaskPlatformSnapshot, h.handlePlatformSnapshot)
	mux.HandleFunc(TaskOnboardUser, h.handleOnboardUser)
	mux.HandleFunc(TaskPurgeRecipientPII, h.handlePurgeRecipientPII)
}

func (h *Handlers) handleWeeklyPayout(ctx context.Context, _ *asynq.Task) error {
	slog.Info("worker: weekly payout starting")
	return h.wallet.WeeklyAutoPayout(ctx)
}

func (h *Handlers) handleExpireOffers(ctx context.Context, _ *asynq.Task) error {
	return h.dispatch.ExpireOffers(ctx)
}

func (h *Handlers) handleSubscriptionRun(ctx context.Context, _ *asynq.Task) error {
	slog.Info("worker: subscription run starting")
	return h.subscriptions.ProcessDue(ctx)
}

func (h *Handlers) handleEscrowReconcile(ctx context.Context, _ *asynq.Task) error {
	slog.Info("worker: escrow reconciliation starting")
	drift, err := h.ledger.ReconcileEscrow(ctx)
	if err != nil {
		return err
	}
	if drift != 0 {
		observability.CaptureError(ctx,
			fmt.Errorf("escrow drift detected: %d kobo", drift),
			"escrow reconciliation failed",
			"drift_kobo", fmt.Sprintf("%d", drift),
		)
		slog.Error("escrow drift detected", "drift_kobo", drift)
		return nil // don't retry — human must investigate
	}
	slog.Info("worker: escrow reconciliation clean", "drift_kobo", 0)
	return nil
}

func (h *Handlers) handlePlatformSnapshot(ctx context.Context, _ *asynq.Task) error {
	slog.Info("worker: platform balance snapshot starting")
	return h.ledger.SnapshotPlatformBalances(ctx)
}

func (h *Handlers) handlePurgeRecipientPII(ctx context.Context, _ *asynq.Task) error {
	if h.orders == nil {
		return nil // not wired (e.g. tests) — skip rather than fail the whole worker
	}
	n, err := h.orders.PurgeStaleRecipientPII(ctx)
	if err != nil {
		return err
	}
	slog.Info("worker: purged stale recipient PII", "orders_purged", n)
	return nil
}

// CashoutPayload is the typed payload for individual cashout tasks.
type CashoutPayload struct {
	CashoutID string `json:"cashout_id"`
}

func (h *Handlers) handleCashoutProcess(ctx context.Context, t *asynq.Task) error {
	var p CashoutPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("cashout payload: %w", err)
	}

	cashoutID, err := uuid.Parse(p.CashoutID)
	if err != nil {
		return fmt.Errorf("cashout: invalid id: %w", err)
	}

	// Load cashout
	var cashout model.CashoutRequest
	if err := h.wallet.DB().WithContext(ctx).First(&cashout, "id = ?", cashoutID).Error; err != nil {
		return fmt.Errorf("cashout not found: %w", err)
	}
	if cashout.Status != "pending" {
		slog.Info("worker: cashout already processed", "id", cashoutID, "status", cashout.Status)
		return nil // idempotent
	}

	// Resolve bank account
	recipientCode, reason, err := h.wallet.ResolveCashoutRecipient(ctx, &cashout)
	if err != nil {
		return fmt.Errorf("cashout recipient: %w", err)
	}

	// Mark processing to prevent duplicate transfers on concurrent retries
	cashout.Status = "processing"
	if err := h.wallet.DB().WithContext(ctx).Save(&cashout).Error; err != nil {
		return fmt.Errorf("cashout: mark processing: %w", err)
	}

	// Initiate provider transfer
	resp, err := h.wallet.Provider().InitiateTransfer(ctx, payment.TransferRequest{
		AmountKobo:    cashout.AmountKobo,
		RecipientCode: recipientCode,
		Reference:     cashout.ID.String(),
		Reason:        reason,
	})

	if err != nil {
		// Check if this is the last retry — asynq sets retried count in context
		retried, _ := asynq.GetRetryCount(ctx)
		maxRetry, _ := asynq.GetMaxRetry(ctx)
		if retried >= maxRetry {
			// Final failure — reverse the ledger debit so merchant gets money back
			slog.Error("worker: cashout final failure, reversing", "id", cashoutID, "err", err)
			if reverseErr := h.wallet.ReverseCashout(ctx, &cashout); reverseErr != nil {
				observability.CaptureError(ctx, reverseErr, "cashout: reversal failed", "cashout_id", cashoutID.String())
			}
			return nil // don't retry after reversal
		}
		return fmt.Errorf("cashout: provider transfer: %w", err) // retryable
	}

	// Success — mark paid and store provider reference
	cashout.Status = "paid"
	cashout.ProviderRef = &resp.TransferCode
	if err := h.wallet.DB().WithContext(ctx).Save(&cashout).Error; err != nil {
		observability.CaptureError(ctx, err, "cashout: mark paid failed", "cashout_id", cashoutID.String())
		// Transfer already sent — don't retry or we'd double-send. Log and move on.
		return nil
	}
	slog.Info("worker: cashout paid", "id", cashoutID, "provider_ref", resp.TransferCode)
	return nil
}

// EnqueueCashout enqueues a cashout processing task for immediate execution.
func EnqueueCashout(client *asynq.Client, cashoutID string) error {
	payload, _ := json.Marshal(CashoutPayload{CashoutID: cashoutID})
	task := asynq.NewTask(TaskCashoutProcess, payload,
		asynq.Queue("critical"),
		asynq.ProcessIn(5*time.Second),
		asynq.MaxRetry(3),
	)
	_, err := client.Enqueue(task)
	return err
}

func (h *Handlers) handleMerchantBatchPayout(ctx context.Context, _ *asynq.Task) error {
	slog.Info("worker: merchant batch payout starting")
	// Find all pending standard merchant cashouts and enqueue them now.
	// "standard" cashouts were not enqueued at creation — this is their trigger.
	var pending []model.CashoutRequest
	if err := h.wallet.DB().WithContext(ctx).
		Where("actor_type = 'merchant' AND status = 'pending'").
		Find(&pending).Error; err != nil {
		return fmt.Errorf("merchant batch: query pending: %w", err)
	}
	slog.Info("worker: merchant batch payout", "count", len(pending))
	for _, c := range pending {
		payload, _ := json.Marshal(CashoutPayload{CashoutID: c.ID.String()})
		task := asynq.NewTask(TaskCashoutProcess, payload,
			asynq.Queue("default"),
			asynq.MaxRetry(3),
			asynq.TaskID("cashout:"+c.ID.String()), // deduplication key
		)
		if _, err := h.asynqClient.Enqueue(task); err != nil {
			observability.CaptureError(ctx, err, "merchant batch: enqueue failed", "cashout_id", c.ID.String())
		}
	}
	return nil
}

// OnboardPayload is the typed payload for user onboarding tasks.
type OnboardPayload struct {
	UserID string `json:"user_id"`
}

func (h *Handlers) handleOnboardUser(ctx context.Context, t *asynq.Task) error {
	var p OnboardPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("onboard payload: %w", err)
	}
	slog.Info("worker: onboarding user", "user_id", p.UserID)
	return h.onboarding.RunByID(ctx, p.UserID)
}

// EnqueueOnboarding enqueues DVA + card + trust-tier creation for a new user.
// Retried up to 5 times with asynq backoff — replaces the fire-and-forget goroutine.
func EnqueueOnboarding(client *asynq.Client, userID string) error {
	payload, _ := json.Marshal(OnboardPayload{UserID: userID})
	task := asynq.NewTask(TaskOnboardUser, payload,
		asynq.Queue("default"),
		asynq.MaxRetry(5),
	)
	_, err := client.Enqueue(task)
	return err
}
