package service

// VelocityService enforces per-tier, per-operation transaction velocity limits
// and detects structuring patterns (AML).
//
// Call Check inside the same DB transaction as the money movement — if Check
// returns an error the caller's transaction rolls back and no money moves.
//
// Structuring detection: if the rolling 24h sum of transactions that are each
// individually below the per-txn limit exceeds structuringThresholdKobo, a
// suspicious_activity row is written. This never blocks the transaction.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

// ErrVelocityExceeded is returned when a transaction would breach a limit.
var ErrVelocityExceeded = errors.New("transaction limit exceeded")

// Structuring detection tuning.
//
// These replace a previous fixed constant of 450_000_000 kobo, which was
// unreachable: the daily-cap check above guarantees the rolling total never
// exceeds limit.DailyKobo, whose largest seeded value is 100_000_000 kobo. The
// branch could therefore never be entered for any tier or operation, so the AML
// control silently detected nothing. Thresholds must be relative to the
// configured limits, never absolute.
const (
	// structuringVolumeRatio: percent of the tier's daily cap that the rolling
	// 24h total must reach before the volume signal counts. 80 = 80%.
	structuringVolumeRatio int64 = 80
	// structuringMinTxns: minimum transactions in the window before the
	// fragmentation signal counts. One large transfer is not structuring.
	structuringMinTxns int = 3
)

type velocityRepo interface {
	FindVelocityLimit(ctx context.Context, tier int, operation string) (*model.VelocityLimit, error)
	ListVelocityLimits(ctx context.Context) ([]model.VelocityLimit, error)
	UpsertVelocityLimit(ctx context.Context, limit *model.VelocityLimit) error
	UpsertVelocityCounter(ctx context.Context, tx *gorm.DB, counter *model.VelocityCounter) error
	SumVelocityWindow(ctx context.Context, tx *gorm.DB, userID uuid.UUID, operation string, since time.Time) (sumKobo int64, txnCount int, err error)
	CreateSuspiciousActivity(ctx context.Context, tx *gorm.DB, sa *model.SuspiciousActivity) error
	ListSuspiciousActivity(ctx context.Context, cursor *uuid.UUID, limit int) ([]model.SuspiciousActivity, error)
}

type VelocityService struct {
	repo velocityRepo
}

func NewVelocityService(r velocityRepo) *VelocityService {
	return &VelocityService{repo: r}
}

// Check enforces velocity limits for a single transaction.
// Must be called inside the caller's DB transaction (tx must not be nil).
// tier is the user's current TrustTier (model.TierNew or model.TierRegular).
func (s *VelocityService) Check(ctx context.Context, tx *gorm.DB, userID uuid.UUID, amountKobo int64, tier model.TrustTier, operation string) error {
	if amountKobo <= 0 {
		return fmt.Errorf("%w: amount must be positive, got %d", ErrVelocityExceeded, amountKobo)
	}

	limit, err := s.repo.FindVelocityLimit(ctx, int(tier), operation)
	if err != nil {
		// FAIL CLOSED. A missing limit row means this tier/operation pair was
		// never configured (typo, new operation, unseeded environment) — or the
		// database is unreachable. Both are indistinguishable here, and both
		// previously allowed unlimited money movement past an AML control.
		// A regulatory limit that silently disables itself is worse than no
		// limit at all, because it reads as enforced.
		slog.ErrorContext(ctx, "velocity: limit lookup failed — blocking transaction",
			"error", err, "tier", int(tier), "operation", operation)
		return fmt.Errorf("%w: no velocity limit configured for tier %d operation %q", ErrVelocityExceeded, int(tier), operation)
	}

	// 1. Per-transaction cap.
	if amountKobo > limit.PerTxnKobo {
		return fmt.Errorf("%w: single transaction of %d kobo exceeds per-txn limit of %d kobo for operation %s",
			ErrVelocityExceeded, amountKobo, limit.PerTxnKobo, operation)
	}

	now := time.Now().UTC()

	// 2. Rolling 24h cap.
	// Buckets are hour-truncated, so the effective window is [24h, 25h) —
	// deliberately conservative: it can only over-count, never under-count.
	dailySum, dailyCount, err := s.repo.SumVelocityWindow(ctx, tx, userID, operation, now.Add(-24*time.Hour))
	if err != nil {
		// FAIL CLOSED: an unreadable counter means we cannot prove the
		// transaction is within limits. Allowing it would let anyone who can
		// induce a query error bypass every velocity cap.
		slog.ErrorContext(ctx, "velocity: daily sum failed — blocking transaction", "error", err, "user_id", userID)
		return fmt.Errorf("%w: unable to verify daily limit", ErrVelocityExceeded)
	}
	if dailySum+amountKobo > limit.DailyKobo {
		return fmt.Errorf("%w: daily limit of %d kobo would be exceeded (current: %d, adding: %d)",
			ErrVelocityExceeded, limit.DailyKobo, dailySum, amountKobo)
	}

	// 3. Rolling 30d cap.
	monthlySum, _, err := s.repo.SumVelocityWindow(ctx, tx, userID, operation, now.Add(-30*24*time.Hour))
	if err != nil {
		slog.ErrorContext(ctx, "velocity: monthly sum failed — blocking transaction", "error", err, "user_id", userID)
		return fmt.Errorf("%w: unable to verify monthly limit", ErrVelocityExceeded)
	}
	if monthlySum+amountKobo > limit.MonthlyKobo {
		return fmt.Errorf("%w: monthly limit of %d kobo would be exceeded (current: %d, adding: %d)",
			ErrVelocityExceeded, limit.MonthlyKobo, monthlySum, amountKobo)
	}

	// 4. Structuring detection (non-blocking, advisory).
	//
	// Structuring is splitting one large transfer into many smaller ones to stay
	// under a reporting threshold. Two signals must BOTH hold:
	//   (a) volume — the rolling 24h total has reached most of the daily cap;
	//   (b) fragmentation — it got there via many transactions, not one or two.
	//
	// The threshold is derived from THIS tier's configured daily limit rather
	// than a global constant. A fixed constant cannot work here: limits are
	// admin-configurable per tier, so any hardcoded value either sits above
	// every daily cap (making detection unreachable) or below normal activity
	// (making it fire constantly).
	windowTotal := dailySum + amountKobo
	windowCount := dailyCount + 1
	if windowTotal >= structuringVolumeRatio*limit.DailyKobo/100 && windowCount >= structuringMinTxns {
		sa := &model.SuspiciousActivity{
			ID:         uuid.New(),
			UserID:     userID,
			Operation:  operation,
			Reason:     "structuring_detected",
			AmountKobo: amountKobo,
			WindowKobo: windowTotal,
			CreatedAt:  now,
		}
		if flagErr := s.repo.CreateSuspiciousActivity(ctx, tx, sa); flagErr != nil {
			// Advisory only: a failed flag must not roll back a legitimate,
			// within-limits transaction. Logged loudly so it is not lost.
			slog.ErrorContext(ctx, "velocity: failed to write suspicious_activity",
				"error", flagErr, "user_id", userID, "operation", operation)
		}
	}

	// 5. Record this transaction in the counter.
	// Window is truncated to the current hour so rows are naturally bucketed.
	windowStart := now.Truncate(time.Hour)
	counter := &model.VelocityCounter{
		ID:          uuid.New(),
		UserID:      userID,
		Operation:   operation,
		WindowStart: windowStart,
		WindowSize:  24 * time.Hour,
		AmountKobo:  amountKobo,
		TxnCount:    1,
		UpdatedAt:   now,
	}
	if err := s.repo.UpsertVelocityCounter(ctx, tx, counter); err != nil {
		// Non-fatal: counter write failure should not block the transaction.
		slog.ErrorContext(ctx, "velocity: counter upsert failed", "error", err, "user_id", userID)
	}

	return nil
}

// ── Admin surface ─────────────────────────────────────────────────────────────

func (s *VelocityService) ListLimits(ctx context.Context) ([]model.VelocityLimit, error) {
	return s.repo.ListVelocityLimits(ctx)
}

func (s *VelocityService) UpsertLimit(ctx context.Context, tier int, operation string, perTxn, daily, monthly int64) (*model.VelocityLimit, error) {
	// Reject nonsensical configurations. A per-txn cap above the daily cap can
	// never bind, and a daily cap above the monthly cap can never bind either —
	// in both cases an admin believes they tightened a control that has been
	// made unreachable. Same class of defect as the hardcoded structuring
	// threshold that sat above every possible daily total.
	if perTxn <= 0 || daily <= 0 || monthly <= 0 {
		return nil, fmt.Errorf("velocity limits must be positive (per-txn %d, daily %d, monthly %d)", perTxn, daily, monthly)
	}
	if perTxn > daily {
		return nil, fmt.Errorf("per-txn limit %d exceeds daily limit %d — the per-txn cap could never bind", perTxn, daily)
	}
	if daily > monthly {
		return nil, fmt.Errorf("daily limit %d exceeds monthly limit %d — the daily cap could never bind", daily, monthly)
	}

	limit := &model.VelocityLimit{
		ID:          uuid.New(),
		Tier:        tier,
		Operation:   operation,
		PerTxnKobo:  perTxn,
		DailyKobo:   daily,
		MonthlyKobo: monthly,
	}
	if err := s.repo.UpsertVelocityLimit(ctx, limit); err != nil {
		return nil, err
	}

	// Read back rather than returning the struct we just built. The upsert
	// conflicts on (tier, operation) and updates only the three amount columns,
	// so on an UPDATE the row keeps its original id — the uuid.New() above was
	// never persisted. Returning it hands the admin API an identifier that does
	// not exist in the database.
	stored, err := s.repo.FindVelocityLimit(ctx, tier, operation)
	if err != nil {
		return nil, fmt.Errorf("read back velocity limit after upsert: %w", err)
	}
	return stored, nil
}

func (s *VelocityService) ListSuspiciousActivity(ctx context.Context, cursor *uuid.UUID, limit int) ([]model.SuspiciousActivity, error) {
	return s.repo.ListSuspiciousActivity(ctx, cursor, limit)
}
