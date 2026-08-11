package service

// ReconciliationService runs daily provider reconciliation.
//
// Flow:
//   1. For each configured provider, fetch the settlement report for yesterday.
//   2. Sum the settled amounts from the provider report.
//   3. Sum the corresponding payment_intents in our ledger for the same date.
//   4. Write a reconciliation_runs row with the diff.
//   5. If drift != 0, log at Error level (alerting hooks on structured logs).
//
// This is called by the asynq scheduler (TaskProviderReconcile) at 02:00 WAT daily.

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

// SettlementProvider is the interface each payment provider must satisfy to
// participate in reconciliation. Implement FetchSettlements on each concrete
// provider when the provider's settlement API is available.
type SettlementProvider interface {
	Name() string
	// FetchSettlements returns the total settled amount (in kobo) for the given date.
	// Returns 0, nil when the provider has no settlements for that date.
	FetchSettlements(ctx context.Context, date time.Time) (totalKobo int64, err error)
}

type reconciliationRepo interface {
	// SumSuccessfulIntents returns the sum of payment_intents.amount_kobo
	// for a given provider and calendar date (UTC).
	SumSuccessfulIntents(ctx context.Context, provider string, date time.Time) (int64, error)
	CreateReconciliationRun(ctx context.Context, run *model.ReconciliationRun) error
}

type ReconciliationService struct {
	repo      reconciliationRepo
	providers []SettlementProvider
}

func NewReconciliationService(r reconciliationRepo, providers ...SettlementProvider) *ReconciliationService {
	return &ReconciliationService{repo: r, providers: providers}
}

// RunForDate reconciles all providers for the given date.
// Intended to be called with yesterday's date from the asynq scheduler.
func (s *ReconciliationService) RunForDate(ctx context.Context, date time.Time) error {
	date = date.UTC().Truncate(24 * time.Hour) // normalise to midnight UTC

	var firstErr error
	for _, p := range s.providers {
		if err := s.reconcileProvider(ctx, p, date); err != nil {
			slog.ErrorContext(ctx, "reconciliation: provider failed",
				"provider", p.Name(), "date", date.Format("2006-01-02"), "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (s *ReconciliationService) reconcileProvider(ctx context.Context, p SettlementProvider, date time.Time) error {
	providerTotal, err := p.FetchSettlements(ctx, date)
	if err != nil {
		detail := err.Error()
		run := &model.ReconciliationRun{
			ID:          uuid.New(),
			Provider:    p.Name(),
			RunDate:     date,
			Status:      "error",
			ErrorDetail: &detail,
			CreatedAt:   time.Now(),
		}
		// Do not discard this write. The error run IS the audit trail proving we
		// attempted reconciliation and could not reach the provider; losing it
		// silently makes a failed day indistinguishable from a day that never
		// ran, which is exactly the gap an auditor asks about.
		if saveErr := s.repo.CreateReconciliationRun(ctx, run); saveErr != nil {
			slog.ErrorContext(ctx, "reconciliation: failed to record error run",
				"provider", p.Name(), "date", date.Format("2006-01-02"),
				"fetch_error", err, "save_error", saveErr)
		}
		return fmt.Errorf("fetch settlements from %s: %w", p.Name(), err)
	}

	ledgerTotal, err := s.repo.SumSuccessfulIntents(ctx, p.Name(), date)
	if err != nil {
		// Record an error run here too, mirroring the provider-fetch path above.
		// Without it, a reconciliation that failed because of OUR database is
		// indistinguishable from a day that was never attempted — the provider
		// leg succeeded, so nothing else records that we tried at all.
		detail := err.Error()
		run := &model.ReconciliationRun{
			ID:            uuid.New(),
			Provider:      p.Name(),
			RunDate:       date,
			ProviderTotal: providerTotal,
			Status:        "error",
			ErrorDetail:   &detail,
			CreatedAt:     time.Now(),
		}
		if saveErr := s.repo.CreateReconciliationRun(ctx, run); saveErr != nil {
			slog.ErrorContext(ctx, "reconciliation: failed to record ledger-sum error run",
				"provider", p.Name(), "date", date.Format("2006-01-02"),
				"sum_error", err, "save_error", saveErr)
		}
		return fmt.Errorf("sum ledger intents for %s: %w", p.Name(), err)
	}

	drift := providerTotal - ledgerTotal
	status := "clean"
	if drift != 0 {
		status = "drift_detected"
	}

	run := &model.ReconciliationRun{
		ID:            uuid.New(),
		Provider:      p.Name(),
		RunDate:       date,
		ProviderTotal: providerTotal,
		LedgerTotal:   ledgerTotal,
		DriftKobo:     drift,
		Status:        status,
		CreatedAt:     time.Now(),
	}
	if err := s.repo.CreateReconciliationRun(ctx, run); err != nil {
		return fmt.Errorf("save reconciliation run: %w", err)
	}

	if drift != 0 {
		// Structured log at Error so any log-based alerting (Sentry, Datadog, etc.)
		// fires immediately. The run row is the audit trail.
		slog.ErrorContext(ctx, "reconciliation: drift detected",
			"provider", p.Name(),
			"date", date.Format("2006-01-02"),
			"provider_total_kobo", providerTotal,
			"ledger_total_kobo", ledgerTotal,
			"drift_kobo", drift,
		)
	} else {
		slog.InfoContext(ctx, "reconciliation: clean",
			"provider", p.Name(),
			"date", date.Format("2006-01-02"),
			"total_kobo", ledgerTotal,
		)
	}
	return nil
}

// ── Repo implementation (wired in main.go via LedgerRepo) ────────────────────

// reconciliationLedgerRepo adapts the existing gorm.DB to the reconciliationRepo interface.
// Kept here to avoid polluting repo/ledger.go with a concern that belongs to this service.
type reconciliationLedgerRepo struct{ db *gorm.DB }

func NewReconciliationRepo(db *gorm.DB) reconciliationRepo {
	return &reconciliationLedgerRepo{db: db}
}

func (r *reconciliationLedgerRepo) SumSuccessfulIntents(ctx context.Context, provider string, date time.Time) (int64, error) {
	// Half-open UTC range [date, date+24h) rather than DATE(created_at) = ?.
	//
	// Correctness: DATE() on a TIMESTAMPTZ casts using the session TimeZone, so
	// on any connection not set to UTC it buckets rows into a different calendar
	// day than the UTC date the caller normalised — reconciliation would compare
	// the provider's day against a shifted slice of ours and report phantom
	// drift. Performance: wrapping the column in DATE() makes the predicate
	// non-sargable, forcing a full scan; a plain range can use an index.
	start := date.UTC().Truncate(24 * time.Hour)
	end := start.Add(24 * time.Hour)

	var sum int64
	err := r.db.WithContext(ctx).
		Table("payment_intents").
		Where("provider = ? AND status = 'success' AND created_at >= ? AND created_at < ?", provider, start, end).
		Select("COALESCE(SUM(amount_kobo), 0)").
		Scan(&sum).Error
	return sum, err
}

func (r *reconciliationLedgerRepo) CreateReconciliationRun(ctx context.Context, run *model.ReconciliationRun) error {
	return r.db.WithContext(ctx).Create(run).Error
}
