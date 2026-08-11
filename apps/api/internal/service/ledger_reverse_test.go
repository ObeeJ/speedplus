package service

// Unit tests for LedgerService.Reverse — the only sanctioned ledger correction
// path. Direct UPDATE/DELETE on ledger_entries is blocked by a DB trigger
// (migration 038), so a mistake here cannot be patched by editing rows: the
// only remedy is another journal, which compounds the error.
//
// These use an in-memory mock embedding repo.LedgerRepo, so any method Reverse
// starts calling that we have not stubbed panics loudly rather than silently
// returning a zero value.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

// ── Mock ──────────────────────────────────────────────────────────────────────

type mockReverseRepo struct {
	repo.LedgerRepo // embedded: unstubbed methods panic if called

	entries      []model.LedgerEntry
	entriesErr   error
	reversals    int64
	reversalsErr error

	balances  map[uuid.UUID]int64
	created   [][]model.LedgerEntry
	createErr error
}

func newMockReverseRepo() *mockReverseRepo {
	return &mockReverseRepo{balances: map[uuid.UUID]int64{}}
}

func (m *mockReverseRepo) FindEntriesByJournal(_ context.Context, _ *gorm.DB, _ uuid.UUID) ([]model.LedgerEntry, error) {
	if m.entriesErr != nil {
		return nil, m.entriesErr
	}
	return m.entries, nil
}

func (m *mockReverseRepo) CountReversalsForJournal(_ context.Context, _ *gorm.DB, _ uuid.UUID) (int64, error) {
	if m.reversalsErr != nil {
		return 0, m.reversalsErr
	}
	return m.reversals, nil
}

func (m *mockReverseRepo) CreateEntries(_ context.Context, _ *gorm.DB, entries []model.LedgerEntry) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = append(m.created, entries)
	return nil
}

func (m *mockReverseRepo) LockBalance(_ context.Context, _ *gorm.DB, accountID uuid.UUID) (*model.WalletBalance, error) {
	return &model.WalletBalance{AccountID: accountID, BalanceKobo: m.balances[accountID]}, nil
}

func (m *mockReverseRepo) UpdateBalance(_ context.Context, _ *gorm.DB, accountID uuid.UUID, newBalance int64) error {
	m.balances[accountID] = newBalance
	return nil
}

// balancedJournal builds a valid two-legged journal: debit `from`, credit `to`.
func balancedJournal(journalID, from, to uuid.UUID, amount int64) []model.LedgerEntry {
	return []model.LedgerEntry{
		{ID: uuid.New(), JournalID: journalID, AccountID: from, AmountKobo: -amount, Description: "original", RefType: "order"},
		{ID: uuid.New(), JournalID: journalID, AccountID: to, AmountKobo: amount, Description: "original", RefType: "order"},
	}
}

// ── Happy path ────────────────────────────────────────────────────────────────

func TestReverse_PostsEqualAndOppositeEntries(t *testing.T) {
	journalID, from, to := uuid.New(), uuid.New(), uuid.New()
	m := newMockReverseRepo()
	m.entries = balancedJournal(journalID, from, to, 500_000)
	// Post-original balances: `from` was debited, `to` credited.
	m.balances[from] = 0
	m.balances[to] = 500_000

	svc := NewLedgerService(m, nil)

	if err := svc.Reverse(context.Background(), nil, journalID, "customer refund"); err != nil {
		t.Fatalf("Reverse: %v", err)
	}

	if len(m.created) != 1 {
		t.Fatalf("expected 1 reversal journal, got %d", len(m.created))
	}
	reversal := m.created[0]
	if len(reversal) != 2 {
		t.Fatalf("reversal must mirror every original leg: got %d, want 2", len(reversal))
	}

	// The double-entry invariant must hold for the reversal itself.
	var sum int64
	for _, e := range reversal {
		sum += e.AmountKobo
	}
	if sum != 0 {
		t.Errorf("reversal journal sums to %d, must be 0", sum)
	}

	// Every reversal leg must point back at the original journal and be
	// identifiable as a reversal, or the idempotency guard cannot find it.
	sharedJournal := reversal[0].JournalID
	for _, e := range reversal {
		if e.RefType != "reversal" {
			t.Errorf("RefType = %q, want reversal", e.RefType)
		}
		if e.RefID == nil || *e.RefID != journalID {
			t.Errorf("RefID must link back to the original journal %s", journalID)
		}
		if e.JournalID == journalID {
			t.Error("reversal must use a NEW journal id, not reuse the original")
		}
		if e.JournalID != sharedJournal {
			t.Error("all reversal legs must share one journal id")
		}
	}

	// Balances must return to their pre-original values.
	if m.balances[from] != 500_000 {
		t.Errorf("from balance = %d, want 500000 (debit undone)", m.balances[from])
	}
	if m.balances[to] != 0 {
		t.Errorf("to balance = %d, want 0 (credit undone)", m.balances[to])
	}
}

// ── Idempotency guard ─────────────────────────────────────────────────────────

// The regression this guard exists for. Ledger entries are append-only, so a
// second Reverse() cannot overwrite the first — it posts ANOTHER
// equal-and-opposite journal, moving the balance the wrong way by the full
// original amount.
func TestReverse_RejectsDoubleReversal(t *testing.T) {
	journalID := uuid.New()
	m := newMockReverseRepo()
	m.entries = balancedJournal(journalID, uuid.New(), uuid.New(), 500_000)
	m.reversals = 2 // a reversal journal already exists for this journal

	svc := NewLedgerService(m, nil)

	err := svc.Reverse(context.Background(), nil, journalID, "second attempt")
	if err == nil {
		t.Fatal("DOUBLE-REVERSAL REGRESSION: Reverse allowed a journal to be reversed twice")
	}
	if !errors.Is(err, ErrAlreadyReversed) {
		t.Fatalf("expected ErrAlreadyReversed, got %v", err)
	}
	if len(m.created) != 0 {
		t.Fatalf("no entries may be written when the reversal is rejected, got %d journals", len(m.created))
	}
}

// Reversing a reversal would re-apply the original amount.
func TestReverse_RejectsReversingAReversal(t *testing.T) {
	journalID, orig := uuid.New(), uuid.New()
	m := newMockReverseRepo()
	m.entries = []model.LedgerEntry{
		{ID: uuid.New(), JournalID: journalID, AccountID: uuid.New(), AmountKobo: -500_000, RefType: "reversal", RefID: &orig},
		{ID: uuid.New(), JournalID: journalID, AccountID: uuid.New(), AmountKobo: 500_000, RefType: "reversal", RefID: &orig},
	}

	svc := NewLedgerService(m, nil)

	if err := svc.Reverse(context.Background(), nil, journalID, "undo the undo"); err == nil {
		t.Fatal("expected reversing a reversal to be rejected")
	}
	if len(m.created) != 0 {
		t.Fatal("no entries may be written when reversing a reversal is rejected")
	}
}

// ── Failure modes ─────────────────────────────────────────────────────────────

func TestReverse_FailsClosedOnRepoErrors(t *testing.T) {
	boom := errors.New("connection refused")

	tests := []struct {
		name  string
		setup func(*mockReverseRepo, uuid.UUID)
	}{
		{
			name: "journal lookup fails",
			setup: func(m *mockReverseRepo, _ uuid.UUID) {
				m.entriesErr = boom
			},
		},
		{
			name: "journal is empty or unknown",
			setup: func(m *mockReverseRepo, _ uuid.UUID) {
				m.entries = nil
			},
		},
		{
			// Cannot prove the journal is unreversed -> must not write.
			name: "existing-reversal count fails",
			setup: func(m *mockReverseRepo, j uuid.UUID) {
				m.entries = balancedJournal(j, uuid.New(), uuid.New(), 500_000)
				m.reversalsErr = boom
			},
		},
		{
			name: "writing the reversal entries fails",
			setup: func(m *mockReverseRepo, j uuid.UUID) {
				m.entries = balancedJournal(j, uuid.New(), uuid.New(), 500_000)
				m.createErr = boom
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			journalID := uuid.New()
			m := newMockReverseRepo()
			tc.setup(m, journalID)

			svc := NewLedgerService(m, nil)

			if err := svc.Reverse(context.Background(), nil, journalID, "test"); err == nil {
				t.Fatal("expected Reverse to fail closed, got nil error")
			}
		})
	}
}

// A multi-leg journal (e.g. settle: escrow -> merchant + platform fee) must be
// mirrored leg for leg, still summing to zero.
func TestReverse_HandlesMultiLegJournal(t *testing.T) {
	journalID := uuid.New()
	escrow, merchant, platform := uuid.New(), uuid.New(), uuid.New()

	m := newMockReverseRepo()
	m.entries = []model.LedgerEntry{
		{ID: uuid.New(), JournalID: journalID, AccountID: escrow, AmountKobo: -1_000_000, RefType: "order"},
		{ID: uuid.New(), JournalID: journalID, AccountID: merchant, AmountKobo: 900_000, RefType: "order"},
		{ID: uuid.New(), JournalID: journalID, AccountID: platform, AmountKobo: 100_000, RefType: "order"},
	}
	m.balances[escrow] = 0
	m.balances[merchant] = 900_000
	m.balances[platform] = 100_000

	svc := NewLedgerService(m, nil)

	if err := svc.Reverse(context.Background(), nil, journalID, "disputed settlement"); err != nil {
		t.Fatalf("Reverse: %v", err)
	}

	reversal := m.created[0]
	if len(reversal) != 3 {
		t.Fatalf("expected 3 reversal legs, got %d", len(reversal))
	}
	var sum int64
	for _, e := range reversal {
		sum += e.AmountKobo
	}
	if sum != 0 {
		t.Errorf("reversal sums to %d, must be 0", sum)
	}

	if m.balances[escrow] != 1_000_000 {
		t.Errorf("escrow = %d, want 1000000", m.balances[escrow])
	}
	if m.balances[merchant] != 0 {
		t.Errorf("merchant = %d, want 0", m.balances[merchant])
	}
	if m.balances[platform] != 0 {
		t.Errorf("platform = %d, want 0", m.balances[platform])
	}
}

// The caller's reason must reach the ledger — it is the audit trail explaining
// why money moved back.
func TestReverse_RecordsReason(t *testing.T) {
	journalID, from, to := uuid.New(), uuid.New(), uuid.New()
	m := newMockReverseRepo()
	m.entries = balancedJournal(journalID, from, to, 250_000)
	// The reversal debits `to`, so it must hold the credit the original gave it.
	// Without this the reversal is correctly refused for insufficient balance.
	m.balances[to] = 250_000

	svc := NewLedgerService(m, nil)

	if err := svc.Reverse(context.Background(), nil, journalID, "duplicate charge"); err != nil {
		t.Fatalf("Reverse: %v", err)
	}
	for _, e := range m.created[0] {
		if !strings.Contains(e.Description, "duplicate charge") {
			t.Errorf("description %q must include the caller's reason", e.Description)
		}
	}
}
