package service

// DB-backed tests for AdminService dispute operations (FreezeEscrow / ReleaseEscrow).
//
// These are the highest-blast-radius admin actions: they move money out of
// escrow. Requires DATABASE_URL; skips when unset, same as ledger_money_test.go.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

func seedAdmin(t *testing.T, tx *gorm.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	u := &model.User{
		ID: id, Role: "admin", FirstName: "Admin", LastName: "User",
		Phone: "+234700" + id.String()[:7], PasswordHash: "x",
		ReferralCode: "ADM" + id.String()[:8],
	}
	mustCreate(t, tx, u)
	return id
}

func setupDispute(t *testing.T, tx *gorm.DB) (*AdminService, *LedgerService, *model.Order) {
	t.Helper()
	ledgerRepo := repo.NewLedgerRepo(tx)
	ledger := NewLedgerService(ledgerRepo, nil)
	adminSvc := NewAdminService(repo.NewAdminRepo(tx), ledger)

	const total int64 = 10_000_000 // above disputeSingleAdminThresholdKobo (5_000_000) to exercise dual-admin path
	order := seedOrder(t, tx, total, 8_000_000, 1_000_000, 800_000, 200_000)

	// Fund customer and hold escrow so there's something to freeze.
	ctx := context.Background()
	if err := ledger.CreditWallet(ctx, tx, order.CustomerID, total, "test_seed", nil); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}
	if err := ledger.HoldEscrow(ctx, tx, order.ID, order.CustomerID, total); err != nil {
		t.Fatalf("hold escrow: %v", err)
	}
	return adminSvc, ledger, order
}

func TestAdminDispute_FreezeEscrow(t *testing.T) {
	gdb := testDB(t)

	t.Run("freezes a held escrow", func(t *testing.T) {
		withTx(t, gdb, func(tx *gorm.DB) {
			svc, _, order := setupDispute(t, tx)
			adminID := seedAdmin(t, tx)
			if err := svc.FreezeEscrow(context.Background(), order.ID, adminID, "suspected fraud"); err != nil {
				t.Fatalf("FreezeEscrow: %v", err)
			}
		})
	})

	t.Run("errors when no held escrow exists", func(t *testing.T) {
		withTx(t, gdb, func(tx *gorm.DB) {
			ledger := NewLedgerService(repo.NewLedgerRepo(tx), nil)
			svc := NewAdminService(repo.NewAdminRepo(tx), ledger)
			if err := svc.FreezeEscrow(context.Background(), uuid.New(), uuid.New(), "test"); err == nil {
				t.Fatal("expected error for non-existent escrow")
			}
		})
	})

	t.Run("cannot freeze an already-frozen escrow", func(t *testing.T) {
		withTx(t, gdb, func(tx *gorm.DB) {
			svc, _, order := setupDispute(t, tx)
			adminID := seedAdmin(t, tx)
			if err := svc.FreezeEscrow(context.Background(), order.ID, adminID, "first freeze"); err != nil {
				t.Fatalf("first FreezeEscrow: %v", err)
			}
			// second freeze must fail — escrow is now frozen, not held
			if err := svc.FreezeEscrow(context.Background(), order.ID, adminID, "second freeze"); err == nil {
				t.Fatal("expected error freezing an already-frozen escrow")
			}
		})
	})
}

func TestAdminDispute_ReleaseEscrow(t *testing.T) {
	gdb := testDB(t)

	t.Run("release to customer refunds the buyer", func(t *testing.T) {
		withTx(t, gdb, func(tx *gorm.DB) {
			svc, ledger, order := setupDispute(t, tx)
			adminID := seedAdmin(t, tx)
			if err := svc.FreezeEscrow(context.Background(), order.ID, adminID, "dispute"); err != nil {
				t.Fatalf("FreezeEscrow: %v", err)
			}
			if _, err := svc.ReleaseEscrow(context.Background(), order.ID, adminID, "customer", "refund"); err != nil {
				t.Fatalf("ReleaseEscrow (first approval): %v", err)
			}
			admin2ID := seedAdmin(t, tx)
			if _, err := svc.ReleaseEscrow(context.Background(), order.ID, admin2ID, "customer", "refund confirmed"); err != nil {
				t.Fatalf("ReleaseEscrow (second approval): %v", err)
			}
			bal, err := ledger.GetBalance(context.Background(), order.CustomerID)
			if err != nil {
				t.Fatalf("GetBalance: %v", err)
			}
			if bal != 10_000_000 {
				t.Errorf("customer balance = %d, want 10000000 after refund", bal)
			}
		})
	})

	t.Run("invalid recipient is rejected", func(t *testing.T) {
		withTx(t, gdb, func(tx *gorm.DB) {
			svc, _, order := setupDispute(t, tx)
			admin1 := seedAdmin(t, tx)
			admin2 := seedAdmin(t, tx)
			if err := svc.FreezeEscrow(context.Background(), order.ID, admin1, "dispute"); err != nil {
				t.Fatalf("FreezeEscrow: %v", err)
			}
			// first approval
			if _, err := svc.ReleaseEscrow(context.Background(), order.ID, admin1, "invalid", "test"); err != nil {
				t.Fatalf("first approval: %v", err)
			}
			// second approval with invalid recipient — should error
			if _, err := svc.ReleaseEscrow(context.Background(), order.ID, admin2, "invalid", "test"); err == nil {
				t.Fatal("expected error for invalid recipient")
			}
		})
	})

	t.Run("same admin cannot approve twice", func(t *testing.T) {
		withTx(t, gdb, func(tx *gorm.DB) {
			svc, _, order := setupDispute(t, tx)
			adminID := seedAdmin(t, tx)
			if err := svc.FreezeEscrow(context.Background(), order.ID, adminID, "dispute"); err != nil {
				t.Fatalf("FreezeEscrow: %v", err)
			}
			if _, err := svc.ReleaseEscrow(context.Background(), order.ID, adminID, "customer", "first"); err != nil {
				t.Fatalf("first approval: %v", err)
			}
			// same admin tries to be the second approver
			if _, err := svc.ReleaseEscrow(context.Background(), order.ID, adminID, "customer", "second"); err == nil {
				t.Fatal("same admin must not be able to approve twice")
			}
		})
	})
}
