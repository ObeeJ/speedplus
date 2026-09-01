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
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	TaskWeeklyPayout        = "driver:weekly_payout"
	TaskExpireOffers        = "dispatch:expire_offers"
	TaskSubscriptionRun     = "subscription:run"
	TaskCashoutProcess      = "cashout:process"
	TaskMerchantBatchPayout = "cashout:merchant_batch"
	TaskEscrowReconcile     = "ledger:escrow_reconcile"
	TaskPlatformSnapshot    = "ledger:platform_snapshot"
	TaskOnboardUser         = "user:onboard"
	TaskPurgeRecipientPII   = "order:purge_recipient_pii"
	TaskReviewAggregate     = "review:aggregate"
	TaskAssembleRuns        = "run:assemble"
	TaskFillAccuracy        = "gas:fill_accuracy"
	TaskBurnRateUpdate      = "gas:burn_rate_update"
	TaskRecertReminders     = "gas:recert_reminders"
	TaskProviderReconcile   = "ledger:provider_reconcile"
	TaskDisputeSLACheck     = "ledger:dispute_sla_check"
	TaskSendSMS             = "notification:sms" // queued SMS delivery with retry
)

// ── Scheduler (cron) ──────────────────────────────────────────────────────────

func NewScheduler(redisURL string) *asynq.Scheduler {
	opt, _ := asynq.ParseRedisURI(redisURL)
	s := asynq.NewScheduler(opt, nil)

	s.Register("0 1 * * 1", asynq.NewTask(TaskWeeklyPayout, nil))
	s.Register("*/1 * * * *", asynq.NewTask(TaskExpireOffers, nil))
	s.Register("0 6 * * *", asynq.NewTask(TaskSubscriptionRun, nil))
	s.Register("0 17 * * *", asynq.NewTask(TaskMerchantBatchPayout, nil))
	s.Register("30 1 * * *", asynq.NewTask(TaskEscrowReconcile, nil))
	s.Register("0 2 * * *", asynq.NewTask(TaskPlatformSnapshot, nil))
	s.Register("0 3 * * *", asynq.NewTask(TaskPurgeRecipientPII, nil))
	s.Register("*/30 6-17 * * *", asynq.NewTask(TaskAssembleRuns, nil))
	s.Register("30 2 * * *", asynq.NewTask(TaskFillAccuracy, nil))
	s.Register("0 3 * * *", asynq.NewTask(TaskBurnRateUpdate, nil))
	s.Register("0 4 * * *", asynq.NewTask(TaskRecertReminders, nil))
	s.Register("0 2 * * *", asynq.NewTask(TaskProviderReconcile, nil)) // 02:00 WAT daily
	s.Register("0 * * * *", asynq.NewTask(TaskDisputeSLACheck, nil))   // every hour

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

// smsSender is the subset of sms.Client used by the worker.
// Narrow interface so the worker package does not import sms directly.
type smsSender interface {
	Send(ctx context.Context, to, message string) error
}

// phoneResolver resolves a user's E.164 phone number from their UUID.
// Satisfied by repo.UserRepo — injected via InjectPhoneResolver to avoid
// importing the repo package directly into the worker.
type phoneResolver interface {
	PhoneByID(ctx context.Context, userID uuid.UUID) (string, error)
}

// Handlers holds all worker dependencies.
type Handlers struct {
	subscriptions  *service.SubscriptionService
	wallet         *service.WalletService
	dispatch       *service.DispatchService
	ledger         *service.LedgerService
	onboarding     onboardingRunner
	orders         *service.OrderService
	runs           *service.RunService
	asynqClient    *asynq.Client
	reconciliation *service.ReconciliationService
	sms            smsSender    // nil = SMS disabled
	phones         phoneResolver // nil = recert SMS skipped
	// FIX #3: explicit *gorm.DB so the worker never calls wallet.DB() which
	// returns nil and causes a guaranteed nil-pointer panic on every cashout task.
	db *gorm.DB
}

// onboardingRunner is the subset of ports.OnboardingRunner used by the worker.
type onboardingRunner interface {
	RunByID(ctx context.Context, userID string) error
}

func NewHandlers(wallet *service.WalletService, dispatch *service.DispatchService, ledger *service.LedgerService, subscriptions *service.SubscriptionService, onboarding onboardingRunner, asynqClient *asynq.Client) *Handlers {
	return &Handlers{wallet: wallet, dispatch: dispatch, ledger: ledger, subscriptions: subscriptions, onboarding: onboarding, asynqClient: asynqClient}
}

// InjectDB wires the GORM DB handle needed by the cashout worker.
// Must be called before the worker server starts.
func (h *Handlers) InjectDB(db *gorm.DB) { h.db = db }

// InjectRuns wires RunService after construction.
func (h *Handlers) InjectRuns(r *service.RunService) { h.runs = r }

// InjectOrders wires OrderService after construction.
func (h *Handlers) InjectOrders(o *service.OrderService) { h.orders = o }

// InjectReconciliation wires ReconciliationService after construction.
func (h *Handlers) InjectReconciliation(r *service.ReconciliationService) { h.reconciliation = r }

// InjectSMS wires the SMS sender after construction.
// Until called, TaskSendSMS tasks are processed but sends are no-ops.
func (h *Handlers) InjectSMS(s smsSender) { h.sms = s }

// InjectPhoneResolver wires the phone lookup used by handleRecertReminders.
// Until called, recert SMS tasks are enqueued but skipped (logged as warnings).
func (h *Handlers) InjectPhoneResolver(r phoneResolver) { h.phones = r }

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
	mux.HandleFunc(TaskReviewAggregate, h.handleReviewAggregate)
	mux.HandleFunc(TaskAssembleRuns, h.handleAssembleRuns)
	mux.HandleFunc(TaskFillAccuracy, h.handleFillAccuracy)
	mux.HandleFunc(TaskBurnRateUpdate, h.handleBurnRateUpdate)
	mux.HandleFunc(TaskRecertReminders, h.handleRecertReminders)
	mux.HandleFunc(TaskProviderReconcile, h.handleProviderReconcile)
	mux.HandleFunc(TaskDisputeSLACheck, h.handleDisputeSLACheck)
	mux.HandleFunc(TaskSendSMS, h.handleSendSMS)
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
		return nil
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
	if h.db == nil {
		return fmt.Errorf("cashout worker: DB not injected — call InjectDB before starting the server")
	}

	var p CashoutPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("cashout payload: %w", err)
	}
	cashoutID, err := uuid.Parse(p.CashoutID)
	if err != nil {
		return fmt.Errorf("cashout: invalid id: %w", err)
	}

	// FIX #10: use SELECT FOR UPDATE inside a transaction to prevent two
	// concurrent retries from both reading status="pending" and double-paying.
	var cashout model.CashoutRequest
	err = h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&cashout, "id = ?", cashoutID).Error; err != nil {
			return fmt.Errorf("cashout not found: %w", err)
		}
		if cashout.Status != "pending" {
			return nil // already processed — idempotent exit
		}
		cashout.Status = "processing"
		return tx.Save(&cashout).Error
	})
	if err != nil {
		return fmt.Errorf("cashout: claim processing: %w", err)
	}
	if cashout.Status != "processing" {
		slog.Info("worker: cashout already processed", "id", cashoutID, "status", cashout.Status)
		return nil
	}

	recipientCode, _, reason, err := h.wallet.ResolveCashoutRecipient(ctx, &cashout)
	if err != nil {
		return fmt.Errorf("cashout recipient: %w", err)
	}

	resp, err := h.wallet.Provider().InitiateTransfer(ctx, payment.TransferRequest{
		AmountKobo:    cashout.AmountKobo,
		RecipientCode: recipientCode,
		Reference:     cashout.ID.String(),
		Reason:        reason,
	})
	if err != nil {
		retried, _ := asynq.GetRetryCount(ctx)
		maxRetry, _ := asynq.GetMaxRetry(ctx)
		if retried >= maxRetry {
			slog.Error("worker: cashout final failure, reversing", "id", cashoutID, "err", err)
			if reverseErr := h.wallet.ReverseCashout(ctx, &cashout); reverseErr != nil {
				observability.CaptureError(ctx, reverseErr, "cashout: reversal failed", "cashout_id", cashoutID.String())
			}
			return nil
		}
		// Reset to pending so the next retry can re-claim it.
		cashout.Status = "pending"
		h.db.WithContext(ctx).Save(&cashout) //nolint:errcheck — best-effort reset
		return fmt.Errorf("cashout: provider transfer: %w", err)
	}

	cashout.Status = "paid"
	cashout.ProviderRef = &resp.TransferCode
	if err := h.db.WithContext(ctx).Save(&cashout).Error; err != nil {
		observability.CaptureError(ctx, err, "cashout: mark paid failed", "cashout_id", cashoutID.String())
		return nil // transfer already sent — don't retry
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
	if h.db == nil {
		return fmt.Errorf("merchant batch: DB not injected")
	}
	var pending []model.CashoutRequest
	if err := h.db.WithContext(ctx).
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
			asynq.TaskID("cashout:"+c.ID.String()),
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

// ReviewAggregatePayload is the typed payload for post-review recomputation.
type ReviewAggregatePayload struct {
	RevieweeID   string `json:"reviewee_id"`
	RevieweeType string `json:"reviewee_type"`
}

func (h *Handlers) handleReviewAggregate(ctx context.Context, t *asynq.Task) error {
	var p ReviewAggregatePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("review aggregate payload: %w", err)
	}
	if h.orders == nil {
		return fmt.Errorf("review aggregate: OrderService not injected")
	}
	revieweeID, err := uuid.Parse(p.RevieweeID)
	if err != nil {
		return fmt.Errorf("review aggregate: bad reviewee_id %q: %w", p.RevieweeID, err)
	}
	if err := h.orders.UpdateAggregateRating(ctx, revieweeID, p.RevieweeType); err != nil {
		return fmt.Errorf("review aggregate: rating: %w", err)
	}
	if p.RevieweeType == "driver" {
		if err := h.orders.AwardBadgeIfEligible(ctx, revieweeID); err != nil {
			return fmt.Errorf("review aggregate: badges: %w", err)
		}
	}
	return nil
}

// EnqueueReviewAggregate schedules rating recomputation + badge award after a review.
func EnqueueReviewAggregate(client *asynq.Client, revieweeID, revieweeType string) error {
	payload, _ := json.Marshal(ReviewAggregatePayload{RevieweeID: revieweeID, RevieweeType: revieweeType})
	task := asynq.NewTask(TaskReviewAggregate, payload,
		asynq.Queue("low"),
		asynq.MaxRetry(3),
	)
	_, err := client.Enqueue(task)
	return err
}

// EnqueueOnboarding enqueues DVA + card + trust-tier creation for a new user.
func EnqueueOnboarding(client *asynq.Client, userID string) error {
	payload, _ := json.Marshal(OnboardPayload{UserID: userID})
	task := asynq.NewTask(TaskOnboardUser, payload,
		asynq.Queue("default"),
		asynq.MaxRetry(5),
	)
	_, err := client.Enqueue(task)
	return err
}

func (h *Handlers) handleAssembleRuns(ctx context.Context, _ *asynq.Task) error {
	if h.runs == nil {
		return nil
	}
	slog.Info("worker: assembling gas delivery runs")
	return h.runs.AssembleAllDueRuns(ctx)
}

func (h *Handlers) handleFillAccuracy(ctx context.Context, _ *asynq.Task) error {
	if h.orders == nil {
		return nil
	}
	slog.Info("worker: recomputing gas fill accuracy")
	return h.orders.RecomputeFillAccuracy(ctx)
}

func (h *Handlers) handleBurnRateUpdate(ctx context.Context, _ *asynq.Task) error {
	if h.subscriptions == nil {
		return nil
	}
	slog.Info("worker: updating gas burn rates")
	return h.subscriptions.UpdateBurnRates(ctx)
}

func (h *Handlers) handleRecertReminders(ctx context.Context, _ *asynq.Task) error {
	if h.orders == nil {
		return nil
	}
	results, err := h.orders.RecertReminders(ctx)
	if err != nil {
		return err
	}
	slog.Info("worker: cylinder recert reminders", "count", len(results))
	for _, r := range results {
		slog.Info("recert reminder",
			"cylinder_id", r.CylinderID,
			"user_id", r.UserID,
			"serial", r.Serial,
			"days_until_expiry", r.DaysUntilExpiry,
		)
		// Enqueue per-user SMS via the durable queue — not a bare goroutine.
		// Each reminder is independent; one failure must not block the rest.
		if h.asynqClient == nil {
			continue
		}
		if h.phones == nil {
			slog.Warn("worker: recert SMS skipped — phone resolver not wired",
				"user_id", r.UserID)
			continue
		}
		phone, err := h.phones.PhoneByID(ctx, r.UserID)
		if err != nil {
			observability.CaptureError(ctx, err, "recert: phone lookup failed",
				"user_id", r.UserID.String())
			continue
		}
		msg := fmt.Sprintf(
			"Fourdat: Your cylinder (SN: %s) is due for recertification in %d days.",
			r.Serial, r.DaysUntilExpiry,
		)
		if err := EnqueueSMS(h.asynqClient, phone, msg); err != nil {
			observability.CaptureError(ctx, err, "recert: enqueue SMS failed",
				"user_id", r.UserID.String())
		}
	}
	return nil
}

func (h *Handlers) handleProviderReconcile(ctx context.Context, _ *asynq.Task) error {
	if h.reconciliation == nil {
		slog.Warn("worker: provider reconciliation skipped — service not wired")
		return nil
	}
	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	slog.Info("worker: provider reconciliation starting", "date", yesterday.Format("2006-01-02"))
	if err := h.reconciliation.RunForDate(ctx, yesterday); err != nil {
		return fmt.Errorf("provider reconciliation: %w", err)
	}
	slog.Info("worker: provider reconciliation complete", "date", yesterday.Format("2006-01-02"))
	return nil
}

func (h *Handlers) handleDisputeSLACheck(ctx context.Context, _ *asynq.Task) error {
	return h.ledger.AlertOverdueDisputes(ctx)
}

// ── SMS task ──────────────────────────────────────────────────────────────────

// SMSPayload is the typed payload for queued SMS delivery.
type SMSPayload struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

// EnqueueSMS enqueues a single SMS for durable delivery with retry.
// Queue: "default", MaxRetry: 5, backoff handled by asynq.
// Callers must normalise phone to E.164 without '+' before calling.
func EnqueueSMS(client *asynq.Client, phone, message string) error {
	payload, err := json.Marshal(SMSPayload{Phone: phone, Message: message})
	if err != nil {
		return fmt.Errorf("enqueue sms: marshal: %w", err)
	}
	task := asynq.NewTask(TaskSendSMS, payload,
		asynq.Queue("default"),
		asynq.MaxRetry(5),
		asynq.Timeout(15*time.Second),
	)
	_, err = client.Enqueue(task)
	return err
}

func (h *Handlers) handleSendSMS(ctx context.Context, t *asynq.Task) error {
	var p SMSPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("sms payload: %w", err)
	}
	if p.Phone == "" || p.Message == "" {
		// Malformed payload — do not retry, it will never succeed.
		slog.Error("worker: sms task has empty phone or message — skipping")
		return nil
	}
	if h.sms == nil {
		// SMS client not wired — treat as disabled, not an error.
		return nil
	}
	return h.sms.Send(ctx, p.Phone, p.Message)
}
