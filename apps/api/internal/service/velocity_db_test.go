package service

// Database-backed tests for the velocity counter.
//
// velocity_test.go mocks the repo, which is right for exercising decision
// logic but structurally blind to one whole class of defect: a Go type that
// cannot be written to its Postgres column. VelocityCounter.WindowSize is a
// time.Duration (int64 nanoseconds to GORM) against a column declared
// INTERVAL NOT NULL, and the counter-upsert error is deliberately swallowed as
// non-fatal inside Check. If that write fails, no counter row is ever created,
// SumVelocityWindow always returns 0, and the daily/monthly limits silently
// never bind while the service logs one line per transaction and looks healthy.
//
// Only a real insert can settle it. Requires DATABASE_URL with migrations
// applied (see cmd/server/migrate_ci); skips when unset, matching
// ledger_money_test.go.

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/db"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

func velocityTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping velocity DB tests")
	}
	gdb, err := db.Connect(dsn, false)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	return gdb
}

// seedVelocityUser inserts a minimal user so the counter's FK is satisfiable.
func seedVelocityUser(t *testing.T, tx *gorm.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	u := &model.User{
		ID:           id,
		Phone:        "+234900" + id.String()[:7],
		FirstName:    "Velocity",
		LastName:     "Test",
		Role:         "customer",
		ReferralCode: "VEL" + id.String()[:8],
		IsActive:     true,
	}
	if err := tx.Create(u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return id
}

// The decisive test: can a VelocityCounter actually be written?
func TestVelocityCounter_WritesToPostgres(t *testing.T) {
	gdb := velocityTestDB(t)
	tx := gdb.Begin()
	defer tx.Rollback()

	userID := seedVelocityUser(t, tx)
	r := repo.NewVelocityRepo(gdb)
	ctx := context.Background()
	windowStart := time.Now().UTC().Truncate(time.Hour)

	counter := &model.VelocityCounter{
		ID:          uuid.New(),
		UserID:      userID,
		Operation:   "transfer",
		WindowStart: windowStart,
		WindowSize:  24 * time.Hour,
		AmountKobo:  250_000,
		TxnCount:    1,
		UpdatedAt:   time.Now().UTC(),
	}

	if err := r.UpsertVelocityCounter(ctx, tx, counter); err != nil {
		t.Fatalf("VELOCITY COUNTERS ARE NOT PERSISTABLE: %v\n"+
			"WindowSize is time.Duration against an INTERVAL column. Because the "+
			"counter-upsert error is swallowed as non-fatal in Check, this failure "+
			"is invisible at runtime and leaves daily/monthly limits permanently "+
			"unbound (SumVelocityWindow always sees 0).", err)
	}

	// It must be readable back through the same aggregate Check relies on.
	sum, count, err := r.SumVelocityWindow(ctx, tx, userID, "transfer", windowStart.Add(-time.Hour))
	if err != nil {
		t.Fatalf("SumVelocityWindow: %v", err)
	}
	if sum != 250_000 {
		t.Errorf("sum = %d, want 250000 — the counter did not round-trip", sum)
	}
	if count != 1 {
		t.Errorf("txnCount = %d, want 1", count)
	}
}

// Repeated transactions in the same hour must accumulate onto one row via
// ON CONFLICT, not create duplicates — the daily cap depends on it.
func TestVelocityCounter_AccumulatesWithinWindow(t *testing.T) {
	gdb := velocityTestDB(t)
	tx := gdb.Begin()
	defer tx.Rollback()

	userID := seedVelocityUser(t, tx)
	r := repo.NewVelocityRepo(gdb)
	ctx := context.Background()
	windowStart := time.Now().UTC().Truncate(time.Hour)

	for i := 0; i < 3; i++ {
		c := &model.VelocityCounter{
			ID:          uuid.New(),
			UserID:      userID,
			Operation:   "transfer",
			WindowStart: windowStart,
			WindowSize:  24 * time.Hour,
			AmountKobo:  100_000,
			TxnCount:    1,
			UpdatedAt:   time.Now().UTC(),
		}
		if err := r.UpsertVelocityCounter(ctx, tx, c); err != nil {
			t.Fatalf("upsert %d: %v", i, err)
		}
	}

	sum, count, err := r.SumVelocityWindow(ctx, tx, userID, "transfer", windowStart.Add(-time.Hour))
	if err != nil {
		t.Fatalf("SumVelocityWindow: %v", err)
	}
	if sum != 300_000 {
		t.Errorf("sum = %d, want 300000 (3 x 100000 accumulated)", sum)
	}
	if count != 3 {
		t.Errorf("txnCount = %d, want 3", count)
	}
}

// Counters must be scoped per operation: a cashout must not consume the
// transfer allowance.
func TestVelocityCounter_IsolatesOperations(t *testing.T) {
	gdb := velocityTestDB(t)
	tx := gdb.Begin()
	defer tx.Rollback()

	userID := seedVelocityUser(t, tx)
	r := repo.NewVelocityRepo(gdb)
	ctx := context.Background()
	windowStart := time.Now().UTC().Truncate(time.Hour)

	for _, op := range []string{"transfer", "cashout"} {
		c := &model.VelocityCounter{
			ID: uuid.New(), UserID: userID, Operation: op,
			WindowStart: windowStart, WindowSize: 24 * time.Hour,
			AmountKobo: 100_000, TxnCount: 1, UpdatedAt: time.Now().UTC(),
		}
		if err := r.UpsertVelocityCounter(ctx, tx, c); err != nil {
			t.Fatalf("upsert %s: %v", op, err)
		}
	}

	sum, count, err := r.SumVelocityWindow(ctx, tx, userID, "transfer", windowStart.Add(-time.Hour))
	if err != nil {
		t.Fatalf("SumVelocityWindow: %v", err)
	}
	if sum != 100_000 || count != 1 {
		t.Errorf("transfer window = (%d, %d), want (100000, 1) — operations are leaking into each other", sum, count)
	}
}

// The seeded limits from migration 039 must actually be readable, or Check
// fails closed and blocks every money movement for that tier/operation.
func TestVelocityLimits_SeededByMigration(t *testing.T) {
	gdb := velocityTestDB(t)
	r := repo.NewVelocityRepo(gdb)
	ctx := context.Background()

	for _, op := range []string{"transfer", "cashout", "withdraw", "fund", "gift_card"} {
		for _, tier := range []int{0, 1} {
			limit, err := r.FindVelocityLimit(ctx, tier, op)
			if err != nil {
				t.Errorf("no seeded limit for tier %d operation %q: %v — Check fails closed and blocks this operation entirely", tier, op, err)
				continue
			}
			if limit.PerTxnKobo > limit.DailyKobo {
				t.Errorf("tier %d %q: per-txn %d > daily %d — the per-txn cap can never bind",
					tier, op, limit.PerTxnKobo, limit.DailyKobo)
			}
			if limit.DailyKobo > limit.MonthlyKobo {
				t.Errorf("tier %d %q: daily %d > monthly %d — the daily cap can never bind",
					tier, op, limit.DailyKobo, limit.MonthlyKobo)
			}
		}
	}
}
