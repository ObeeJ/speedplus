package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/speedplus/api/internal/observability"
	"github.com/speedplus/api/internal/service"
)

const (
	TaskWeeklyPayout      = "driver:weekly_payout"
	TaskExpireOffers      = "dispatch:expire_offers"
	TaskSubscriptionRun   = "subscription:run"
	TaskCashoutProcess    = "cashout:process"
	TaskEscrowReconcile   = "ledger:escrow_reconcile"
	TaskPlatformSnapshot  = "ledger:platform_snapshot"
	TaskOnboardUser       = "user:onboard" // DVA + card + trust tier after registration
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
	// Escrow reconciliation — daily 02:30 WAT (01:30 UTC)
	s.Register("30 1 * * *", asynq.NewTask(TaskEscrowReconcile, nil))
	// Platform balance snapshot — daily 03:00 WAT (02:00 UTC)
	s.Register("0 2 * * *", asynq.NewTask(TaskPlatformSnapshot, nil))

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
}

// onboardingRunner is the subset of ports.OnboardingRunner used by the worker.
type onboardingRunner interface {
	RunByID(ctx context.Context, userID string) error
}

func NewHandlers(wallet *service.WalletService, dispatch *service.DispatchService, ledger *service.LedgerService, subscriptions *service.SubscriptionService, onboarding onboardingRunner) *Handlers {
	return &Handlers{wallet: wallet, dispatch: dispatch, ledger: ledger, subscriptions: subscriptions, onboarding: onboarding}
}

func (h *Handlers) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskWeeklyPayout, h.handleWeeklyPayout)
	mux.HandleFunc(TaskExpireOffers, h.handleExpireOffers)
	mux.HandleFunc(TaskSubscriptionRun, h.handleSubscriptionRun)
	mux.HandleFunc(TaskCashoutProcess, h.handleCashoutProcess)
	mux.HandleFunc(TaskEscrowReconcile, h.handleEscrowReconcile)
	mux.HandleFunc(TaskPlatformSnapshot, h.handlePlatformSnapshot)
	mux.HandleFunc(TaskOnboardUser, h.handleOnboardUser)
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

// CashoutPayload is the typed payload for individual cashout tasks.
type CashoutPayload struct {
	CashoutID string `json:"cashout_id"`
}

func (h *Handlers) handleCashoutProcess(ctx context.Context, t *asynq.Task) error {
	var p CashoutPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("cashout payload: %w", err)
	}
	slog.Info("worker: processing cashout", "id", p.CashoutID)
	// TODO: call provider transfer API, update cashout status
	return nil
}

// EnqueueCashout enqueues a cashout processing task with a 5-minute delay.
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
