package service

// Unit tests for VelocityService.Check — the AML / velocity-limit gate.
//
// These use a hand-rolled mock of the velocityRepo interface rather than a real
// database: every branch under test is pure decision logic, and a mock lets us
// provoke the failure modes that matter most here (limit row missing, counter
// query erroring) which are awkward to reproduce against a live DB.
//
// The central regression guarded here is that Check FAILS CLOSED. An earlier
// version returned nil — allowing unlimited money movement — when the limit row
// was missing or a counter read errored. A regulatory control that silently
// disables itself is worse than no control, because it reads as enforced.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

// ── Mock repo ─────────────────────────────────────────────────────────────────

type mockVelocityRepo struct {
	limit    *model.VelocityLimit
	limitErr error

	// sumKobo/txnCount answer the 24h window; monthlySum answers the 30d window
	// when non-zero. The service calls SumVelocityWindow twice with different
	// `since` values, so we discriminate on call order.
	sumKobo    int64
	txnCount   int
	sumErr     error
	monthlySum int64
	monthlyErr error
	sumCalls   int

	counterErr error
	flagErr    error
	flagged    []*model.SuspiciousActivity
	counters   []*model.VelocityCounter
}

func (m *mockVelocityRepo) FindVelocityLimit(_ context.Context, _ int, _ string) (*model.VelocityLimit, error) {
	if m.limitErr != nil {
		return nil, m.limitErr
	}
	return m.limit, nil
}

func (m *mockVelocityRepo) ListVelocityLimits(_ context.Context) ([]model.VelocityLimit, error) {
	return nil, nil
}

func (m *mockVelocityRepo) UpsertVelocityLimit(_ context.Context, _ *model.VelocityLimit) error {
	return nil
}

func (m *mockVelocityRepo) UpsertVelocityCounter(_ context.Context, _ *gorm.DB, c *model.VelocityCounter) error {
	if m.counterErr != nil {
		return m.counterErr
	}
	m.counters = append(m.counters, c)
	return nil
}

func (m *mockVelocityRepo) SumVelocityWindow(_ context.Context, _ *gorm.DB, _ uuid.UUID, _ string, _ time.Time) (int64, int, error) {
	m.sumCalls++
	if m.sumCalls == 1 { // 24h window
		return m.sumKobo, m.txnCount, m.sumErr
	}
	// 30d window
	if m.monthlyErr != nil {
		return 0, 0, m.monthlyErr
	}
	if m.monthlySum != 0 {
		return m.monthlySum, m.txnCount, nil
	}
	return m.sumKobo, m.txnCount, nil
}

func (m *mockVelocityRepo) CreateSuspiciousActivity(_ context.Context, _ *gorm.DB, sa *model.SuspiciousActivity) error {
	if m.flagErr != nil {
		return m.flagErr
	}
	m.flagged = append(m.flagged, sa)
	return nil
}

func (m *mockVelocityRepo) ListSuspiciousActivity(_ context.Context, _ *uuid.UUID, _ int) ([]model.SuspiciousActivity, error) {
	return nil, nil
}

// standardLimit mirrors the TierNew 'transfer' row seeded by migration 039:
// ₦50k per txn / ₦100k daily / ₦500k monthly, expressed in kobo.
func standardLimit() *model.VelocityLimit {
	return &model.VelocityLimit{
		ID:          uuid.New(),
		Tier:        int(model.TierNew),
		Operation:   "transfer",
		PerTxnKobo:  5_000_000,
		DailyKobo:   10_000_000,
		MonthlyKobo: 50_000_000,
	}
}

// ── Limit enforcement ─────────────────────────────────────────────────────────

func TestVelocityCheck_LimitEnforcement(t *testing.T) {
	tests := []struct {
		name       string
		amount     int64
		dailySum   int64
		monthlySum int64
		wantErr    bool
		wantReason string
	}{
		{
			name:   "within all limits passes",
			amount: 1_000_000, // ₦10k
		},
		{
			name:   "amount exactly at per-txn cap passes",
			amount: 5_000_000,
		},
		{
			name:       "amount one kobo over per-txn cap is blocked",
			amount:     5_000_001,
			wantErr:    true,
			wantReason: "per-txn",
		},
		{
			name:     "daily cap exactly reached passes",
			amount:   1_000_000,
			dailySum: 9_000_000, // 9M + 1M == 10M cap
		},
		{
			name:       "daily cap exceeded by one kobo is blocked",
			amount:     1_000_001,
			dailySum:   9_000_000,
			wantErr:    true,
			wantReason: "daily",
		},
		{
			name:       "monthly cap exceeded is blocked even when daily is fine",
			amount:     1_000_000,
			monthlySum: 49_500_000, // 49.5M + 1M > 50M cap
			wantErr:    true,
			wantReason: "monthly",
		},
		{
			name:       "zero amount is rejected",
			amount:     0,
			wantErr:    true,
			wantReason: "positive",
		},
		{
			name:       "negative amount is rejected",
			amount:     -5_000_000,
			wantErr:    true,
			wantReason: "positive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockVelocityRepo{
				limit:      standardLimit(),
				sumKobo:    tc.dailySum,
				monthlySum: tc.monthlySum,
			}
			svc := NewVelocityService(repo)

			err := svc.Check(context.Background(), nil, uuid.New(), tc.amount, model.TierNew, "transfer")

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error (%s) but Check allowed the transaction", tc.wantReason)
				}
				if !errors.Is(err, ErrVelocityExceeded) {
					t.Fatalf("expected ErrVelocityExceeded, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected transaction to pass, got %v", err)
			}
		})
	}
}

// ── Fail-closed behaviour ─────────────────────────────────────────────────────

// A missing limit row or an unreadable counter must BLOCK. Each of these
// previously returned nil, letting unlimited amounts past an AML control.
func TestVelocityCheck_FailsClosed(t *testing.T) {
	dbDown := errors.New("connection refused")

	tests := []struct {
		name string
		repo *mockVelocityRepo
	}{
		{
			name: "no limit configured for tier/operation",
			repo: &mockVelocityRepo{limitErr: gorm.ErrRecordNotFound},
		},
		{
			name: "limit lookup hits a database error",
			repo: &mockVelocityRepo{limitErr: dbDown},
		},
		{
			name: "daily counter read fails",
			repo: &mockVelocityRepo{limit: standardLimit(), sumErr: dbDown},
		},
		{
			name: "monthly counter read fails",
			repo: &mockVelocityRepo{limit: standardLimit(), monthlyErr: dbDown},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewVelocityService(tc.repo)

			err := svc.Check(context.Background(), nil, uuid.New(), 1_000_000, model.TierNew, "transfer")

			if err == nil {
				t.Fatal("FAIL-OPEN REGRESSION: Check allowed a transaction it could not validate")
			}
			if !errors.Is(err, ErrVelocityExceeded) {
				t.Fatalf("expected ErrVelocityExceeded, got %v", err)
			}
		})
	}
}

// ── Structuring detection ─────────────────────────────────────────────────────

// Regression test for a threshold that could never be reached.
//
// The original code compared the rolling window against a fixed 450_000_000
// kobo, but the daily-cap check above already bounds that window by
// limit.DailyKobo, whose largest seeded value is 100_000_000. The branch was
// unreachable for every tier and operation, so no suspicious_activity row could
// ever be written. Thresholds must be relative to the configured limit.
func TestVelocityCheck_StructuringDetection(t *testing.T) {
	limit := standardLimit() // DailyKobo = 10_000_000; 80% => 8_000_000

	tests := []struct {
		name        string
		amount      int64
		dailySum    int64
		txnCount    int
		wantFlagged bool
	}{
		{
			name:        "high volume via many small txns is flagged",
			amount:      1_000_000,
			dailySum:    8_000_000, // total 9M >= 8M threshold
			txnCount:    5,         // 6 total >= 3 minimum
			wantFlagged: true,
		},
		{
			name:        "high volume via a single large txn is NOT structuring",
			amount:      5_000_000,
			dailySum:    4_000_000, // total 9M >= threshold
			txnCount:    1,         // 2 total < 3 minimum
			wantFlagged: false,
		},
		{
			name:        "many txns but low volume is not flagged",
			amount:      100_000,
			dailySum:    500_000, // total 600k < 8M threshold
			txnCount:    10,
			wantFlagged: false,
		},
		{
			name:        "exactly at both thresholds is flagged",
			amount:      1_000_000,
			dailySum:    7_000_000, // total exactly 8M
			txnCount:    2,         // 3 total, exactly the minimum
			wantFlagged: true,
		},
		{
			name:        "quiet account is not flagged",
			amount:      50_000,
			dailySum:    0,
			txnCount:    0,
			wantFlagged: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockVelocityRepo{
				limit:    limit,
				sumKobo:  tc.dailySum,
				txnCount: tc.txnCount,
			}
			svc := NewVelocityService(repo)

			err := svc.Check(context.Background(), nil, uuid.New(), tc.amount, model.TierNew, "transfer")
			if err != nil {
				t.Fatalf("structuring detection must not block a within-limits transaction, got %v", err)
			}

			gotFlagged := len(repo.flagged) > 0
			if gotFlagged != tc.wantFlagged {
				t.Fatalf("flagged = %v, want %v (window total %d, txns %d)",
					gotFlagged, tc.wantFlagged, tc.dailySum+tc.amount, tc.txnCount+1)
			}
			if tc.wantFlagged {
				sa := repo.flagged[0]
				if sa.Reason != "structuring_detected" {
					t.Errorf("reason = %q, want structuring_detected", sa.Reason)
				}
				if sa.WindowKobo != tc.dailySum+tc.amount {
					t.Errorf("WindowKobo = %d, want %d", sa.WindowKobo, tc.dailySum+tc.amount)
				}
			}
		})
	}
}

// Flagging is advisory. If the suspicious_activity insert fails, a legitimate
// within-limits transaction must still succeed rather than be rolled back.
func TestVelocityCheck_StructuringFlagFailureDoesNotBlock(t *testing.T) {
	repo := &mockVelocityRepo{
		limit:    standardLimit(),
		sumKobo:  8_000_000,
		txnCount: 5,
		flagErr:  errors.New("insert failed"),
	}
	svc := NewVelocityService(repo)

	if err := svc.Check(context.Background(), nil, uuid.New(), 1_000_000, model.TierNew, "transfer"); err != nil {
		t.Fatalf("advisory flag failure must not block the transaction, got %v", err)
	}
}

// A counter write failure is likewise non-fatal.
func TestVelocityCheck_CounterWriteFailureDoesNotBlock(t *testing.T) {
	repo := &mockVelocityRepo{
		limit:      standardLimit(),
		counterErr: errors.New("upsert failed"),
	}
	svc := NewVelocityService(repo)

	if err := svc.Check(context.Background(), nil, uuid.New(), 1_000_000, model.TierNew, "transfer"); err != nil {
		t.Fatalf("counter write failure must not block the transaction, got %v", err)
	}
}

// The transaction must be recorded so subsequent windows include it.
func TestVelocityCheck_RecordsCounter(t *testing.T) {
	repo := &mockVelocityRepo{limit: standardLimit()}
	svc := NewVelocityService(repo)
	userID := uuid.New()

	if err := svc.Check(context.Background(), nil, userID, 2_500_000, model.TierNew, "transfer"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.counters) != 1 {
		t.Fatalf("expected exactly 1 counter write, got %d", len(repo.counters))
	}
	c := repo.counters[0]
	if c.UserID != userID {
		t.Errorf("counter UserID = %v, want %v", c.UserID, userID)
	}
	if c.AmountKobo != 2_500_000 {
		t.Errorf("counter AmountKobo = %d, want 2500000", c.AmountKobo)
	}
	if c.Operation != "transfer" {
		t.Errorf("counter Operation = %q, want transfer", c.Operation)
	}
	// Buckets must be hour-truncated so the ON CONFLICT key groups correctly.
	if !c.WindowStart.Equal(c.WindowStart.Truncate(time.Hour)) {
		t.Errorf("WindowStart %v is not truncated to the hour", c.WindowStart)
	}
}

// A blocked transaction must not be counted — otherwise a rejected attempt
// would consume the user's remaining daily allowance.
func TestVelocityCheck_BlockedTransactionIsNotCounted(t *testing.T) {
	repo := &mockVelocityRepo{limit: standardLimit()}
	svc := NewVelocityService(repo)

	err := svc.Check(context.Background(), nil, uuid.New(), 6_000_000, model.TierNew, "transfer") // over per-txn cap
	if err == nil {
		t.Fatal("expected the over-limit transaction to be blocked")
	}
	if len(repo.counters) != 0 {
		t.Fatalf("a blocked transaction must not be recorded, got %d counter writes", len(repo.counters))
	}
}

// Tiers are independent: the service must apply the limit for the tier passed.
func TestVelocityCheck_UsesTierSpecificLimit(t *testing.T) {
	// TierRegular 'transfer' from migration 039: ₦200k per txn.
	regular := &model.VelocityLimit{
		Tier:        int(model.TierRegular),
		Operation:   "transfer",
		PerTxnKobo:  20_000_000,
		DailyKobo:   50_000_000,
		MonthlyKobo: 200_000_000,
	}
	repo := &mockVelocityRepo{limit: regular}
	svc := NewVelocityService(repo)

	// ₦100k would breach TierNew's ₦50k per-txn cap but is fine for TierRegular.
	if err := svc.Check(context.Background(), nil, uuid.New(), 10_000_000, model.TierRegular, "transfer"); err != nil {
		t.Fatalf("TierRegular should permit this amount, got %v", err)
	}
}
