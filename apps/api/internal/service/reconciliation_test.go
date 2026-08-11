package service

// Unit tests for ReconciliationService — the daily provider-vs-ledger check.
//
// Reconciliation is the control that catches money the application never
// noticed losing: if our ledger and the provider's settlement report disagree,
// something upstream failed silently. These tests use in-memory fakes so they
// run without a database.
//
// Behaviours locked down here:
//   - drift is computed as provider - ledger and recorded on every run
//   - clean and drifted runs are distinguishable by status
//   - a provider that errors still produces an "error" run row (audit trail)
//   - one failing provider does not stop the others from reconciling

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/speedplus/api/internal/model"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeSettlementProvider struct {
	name  string
	total int64
	err   error
	calls int
}

func (f *fakeSettlementProvider) Name() string { return f.name }

func (f *fakeSettlementProvider) FetchSettlements(_ context.Context, _ time.Time) (int64, error) {
	f.calls++
	return f.total, f.err
}

type fakeReconRepo struct {
	ledgerTotals map[string]int64 // provider -> sum
	sumErr       error
	createErr    error
	runs         []*model.ReconciliationRun
}

func (f *fakeReconRepo) SumSuccessfulIntents(_ context.Context, provider string, _ time.Time) (int64, error) {
	if f.sumErr != nil {
		return 0, f.sumErr
	}
	return f.ledgerTotals[provider], nil
}

func (f *fakeReconRepo) CreateReconciliationRun(_ context.Context, run *model.ReconciliationRun) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.runs = append(f.runs, run)
	return nil
}

func testReconDate() time.Time {
	return time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)
}

// ── Drift detection ───────────────────────────────────────────────────────────

func TestReconciliation_DriftDetection(t *testing.T) {
	tests := []struct {
		name          string
		providerTotal int64
		ledgerTotal   int64
		wantDrift     int64
		wantStatus    string
	}{
		{
			name:          "totals match exactly is clean",
			providerTotal: 1_500_000,
			ledgerTotal:   1_500_000,
			wantDrift:     0,
			wantStatus:    "clean",
		},
		{
			name:          "provider settled more than we recorded",
			providerTotal: 2_000_000,
			ledgerTotal:   1_500_000,
			wantDrift:     500_000,
			wantStatus:    "drift_detected",
		},
		{
			name:          "we recorded more than the provider settled",
			providerTotal: 1_000_000,
			ledgerTotal:   1_500_000,
			wantDrift:     -500_000,
			wantStatus:    "drift_detected",
		},
		{
			name:          "both zero is clean, not drift",
			providerTotal: 0,
			ledgerTotal:   0,
			wantDrift:     0,
			wantStatus:    "clean",
		},
		{
			name:          "a single kobo of drift is still drift",
			providerTotal: 1_500_001,
			ledgerTotal:   1_500_000,
			wantDrift:     1,
			wantStatus:    "drift_detected",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			provider := &fakeSettlementProvider{name: "paystack", total: tc.providerTotal}
			repo := &fakeReconRepo{ledgerTotals: map[string]int64{"paystack": tc.ledgerTotal}}
			svc := NewReconciliationService(repo, provider)

			if err := svc.RunForDate(context.Background(), testReconDate()); err != nil {
				t.Fatalf("RunForDate returned error: %v", err)
			}

			if len(repo.runs) != 1 {
				t.Fatalf("expected exactly 1 reconciliation run, got %d", len(repo.runs))
			}
			run := repo.runs[0]
			if run.DriftKobo != tc.wantDrift {
				t.Errorf("DriftKobo = %d, want %d", run.DriftKobo, tc.wantDrift)
			}
			if run.Status != tc.wantStatus {
				t.Errorf("Status = %q, want %q", run.Status, tc.wantStatus)
			}
			if run.ProviderTotal != tc.providerTotal {
				t.Errorf("ProviderTotal = %d, want %d", run.ProviderTotal, tc.providerTotal)
			}
			if run.LedgerTotal != tc.ledgerTotal {
				t.Errorf("LedgerTotal = %d, want %d", run.LedgerTotal, tc.ledgerTotal)
			}
		})
	}
}

// ── Provider failure ──────────────────────────────────────────────────────────

// A provider we cannot reach must still leave an audit trail. Without the
// "error" run row, a day that failed is indistinguishable from a day that was
// never attempted — precisely the gap an auditor asks about.
func TestReconciliation_ProviderErrorRecordsErrorRun(t *testing.T) {
	provider := &fakeSettlementProvider{name: "monnify", err: errors.New("502 bad gateway")}
	repo := &fakeReconRepo{ledgerTotals: map[string]int64{}}
	svc := NewReconciliationService(repo, provider)

	err := svc.RunForDate(context.Background(), testReconDate())
	if err == nil {
		t.Fatal("expected RunForDate to surface the provider error")
	}

	if len(repo.runs) != 1 {
		t.Fatalf("expected an error run row to be recorded, got %d runs", len(repo.runs))
	}
	run := repo.runs[0]
	if run.Status != "error" {
		t.Errorf("Status = %q, want error", run.Status)
	}
	if run.ErrorDetail == nil || *run.ErrorDetail == "" {
		t.Error("ErrorDetail must capture why the provider fetch failed")
	}
	if run.Provider != "monnify" {
		t.Errorf("Provider = %q, want monnify", run.Provider)
	}
}

// One bad provider must not abort the others: a Paystack outage should not
// leave Flutterwave and Monnify unreconciled for the day.
func TestReconciliation_OneProviderFailureDoesNotStopOthers(t *testing.T) {
	failing := &fakeSettlementProvider{name: "paystack", err: errors.New("timeout")}
	healthy1 := &fakeSettlementProvider{name: "flutterwave", total: 800_000}
	healthy2 := &fakeSettlementProvider{name: "monnify", total: 300_000}

	repo := &fakeReconRepo{ledgerTotals: map[string]int64{
		"flutterwave": 800_000,
		"monnify":     300_000,
	}}
	svc := NewReconciliationService(repo, failing, healthy1, healthy2)

	err := svc.RunForDate(context.Background(), testReconDate())
	if err == nil {
		t.Fatal("expected the failing provider's error to be surfaced")
	}

	if healthy1.calls != 1 || healthy2.calls != 1 {
		t.Fatalf("healthy providers must still be reconciled: flutterwave=%d monnify=%d",
			healthy1.calls, healthy2.calls)
	}
	// 1 error run + 2 clean runs
	if len(repo.runs) != 3 {
		t.Fatalf("expected 3 run rows (1 error + 2 clean), got %d", len(repo.runs))
	}
	for _, r := range repo.runs {
		if r.Provider != "paystack" && r.Status != "clean" {
			t.Errorf("provider %s: status = %q, want clean", r.Provider, r.Status)
		}
	}
}

// A ledger-sum failure must surface, not be silently treated as zero — zero
// would manufacture drift equal to the provider's entire day.
func TestReconciliation_LedgerSumErrorSurfaces(t *testing.T) {
	provider := &fakeSettlementProvider{name: "paystack", total: 1_000_000}
	repo := &fakeReconRepo{sumErr: errors.New("connection refused")}
	svc := NewReconciliationService(repo, provider)

	err := svc.RunForDate(context.Background(), testReconDate())
	if err == nil {
		t.Fatal("expected the ledger sum error to be surfaced")
	}

	// It must also leave an audit trail. The provider leg succeeded, so without
	// an "error" run row nothing records that we attempted this day at all, and
	// a reconciliation that failed on our own database would look identical to
	// one that was never scheduled.
	if len(repo.runs) != 1 {
		t.Fatalf("expected an error run row for the audit trail, got %d runs", len(repo.runs))
	}
	run := repo.runs[0]
	if run.Status != "error" {
		t.Errorf("Status = %q, want error", run.Status)
	}
	if run.ErrorDetail == nil || *run.ErrorDetail == "" {
		t.Error("ErrorDetail must capture why the ledger sum failed")
	}
	// Keep the leg that succeeded; the ledger side is unknown, so drift must
	// NOT be fabricated by treating it as zero.
	if run.ProviderTotal != 1_000_000 {
		t.Errorf("ProviderTotal = %d, want 1000000 (the leg that succeeded)", run.ProviderTotal)
	}
	if run.DriftKobo != 0 {
		t.Errorf("DriftKobo = %d; drift must not be computed from an unknown ledger total", run.DriftKobo)
	}
}

// The run date must be normalised to midnight UTC so run_date (a DATE column
// with UNIQUE(provider, run_date)) is stable regardless of when the job fires.
func TestReconciliation_NormalisesDateToMidnightUTC(t *testing.T) {
	provider := &fakeSettlementProvider{name: "paystack", total: 0}
	repo := &fakeReconRepo{ledgerTotals: map[string]int64{"paystack": 0}}
	svc := NewReconciliationService(repo, provider)

	// Fire with a mid-afternoon timestamp.
	messy := time.Date(2026, 8, 6, 14, 37, 22, 500, time.UTC)
	if err := svc.RunForDate(context.Background(), messy); err != nil {
		t.Fatalf("RunForDate: %v", err)
	}

	got := repo.runs[0].RunDate
	want := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("RunDate = %v, want %v (must be midnight UTC)", got, want)
	}
}

// No providers configured is a no-op, not a crash.
func TestReconciliation_NoProvidersIsNoOp(t *testing.T) {
	repo := &fakeReconRepo{ledgerTotals: map[string]int64{}}
	svc := NewReconciliationService(repo)

	if err := svc.RunForDate(context.Background(), testReconDate()); err != nil {
		t.Fatalf("expected no error with zero providers, got %v", err)
	}
	if len(repo.runs) != 0 {
		t.Fatalf("expected no runs, got %d", len(repo.runs))
	}
}
