package service

// DB-backed tests for OrderService.RaiseDispute / GetDisputeStatus — the
// customer-facing entry point into the dispute/freeze flow tested from the
// admin side in admin_dispute_test.go. These guard the ownership check and
// the "delivered only" precondition: get either wrong and a customer could
// freeze escrow on someone else's order, or freeze an order that hasn't
// even been delivered yet. Requires DATABASE_URL; skips when unset, same as
// ledger_money_test.go.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

func TestRaiseDispute_RejectsNonDeliveredOrder(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		ledger := NewLedgerService(repo.NewLedgerRepo(tx), nil)
		orderRepo := repo.NewOrderRepo(tx)
		orderSvc := NewOrderService(orderRepo, nil, ledger, nil)

		order := seedOrder(t, tx, 10_000_000, 8_000_000, 1_000_000, 1_000_000, 0)
		// seedOrder leaves Status = OrderPending, not Delivered.

		if err := orderSvc.RaiseDispute(ctx, order.ID, order.CustomerID, "never arrived"); err == nil {
			t.Fatal("expected RaiseDispute to reject a non-delivered order, got nil error")
		}
	})
}

func TestRaiseDispute_RejectsNonOwner(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		ledger := NewLedgerService(repo.NewLedgerRepo(tx), nil)
		orderRepo := repo.NewOrderRepo(tx)
		orderSvc := NewOrderService(orderRepo, nil, ledger, nil)

		order := seedOrder(t, tx, 10_000_000, 8_000_000, 1_000_000, 1_000_000, 0)
		if err := tx.Model(&model.Order{}).Where("id = ?", order.ID).
			Update("status", model.OrderDelivered).Error; err != nil {
			t.Fatalf("mark delivered: %v", err)
		}

		imposter := uuid.New()
		u := &model.User{ID: imposter, Role: model.RoleCustomer, FirstName: "Not", LastName: "Owner", Phone: "+234809" + uuid.NewString()[:8], PasswordHash: "x"}
		mustCreate(t, tx, u)

		if err := orderSvc.RaiseDispute(ctx, order.ID, imposter, "trying to dispute someone else's order"); err != ErrOrderForbidden {
			t.Fatalf("expected ErrOrderForbidden for non-owner, got: %v", err)
		}
	})
}

func TestRaiseDispute_FreezesEscrowOnDeliveredOwnedOrder(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		ledger := NewLedgerService(repo.NewLedgerRepo(tx), nil)
		orderRepo := repo.NewOrderRepo(tx)
		orderSvc := NewOrderService(orderRepo, nil, ledger, nil)

		const total int64 = 10_000_000
		order := seedOrder(t, tx, total, 8_000_000, 1_000_000, 1_000_000, 0)
		if err := tx.Model(&model.Order{}).Where("id = ?", order.ID).
			Update("status", model.OrderDelivered).Error; err != nil {
			t.Fatalf("mark delivered: %v", err)
		}
		if err := ledger.CreditWallet(ctx, tx, order.CustomerID, total, "test_seed", nil); err != nil {
			t.Fatalf("seed wallet: %v", err)
		}
		if err := ledger.HoldEscrow(ctx, tx, order.ID, order.CustomerID, total); err != nil {
			t.Fatalf("hold escrow: %v", err)
		}

		if err := orderSvc.RaiseDispute(ctx, order.ID, order.CustomerID, "item damaged"); err != nil {
			t.Fatalf("RaiseDispute: %v", err)
		}

		status, err := orderSvc.GetDisputeStatus(ctx, order.ID, order.CustomerID)
		if err != nil {
			t.Fatalf("GetDisputeStatus: %v", err)
		}
		if status != string(model.EscrowFrozen) {
			t.Errorf("dispute status = %q, want %q", status, model.EscrowFrozen)
		}

		// Raising a second dispute on an already-frozen escrow must fail —
		// otherwise a customer could re-freeze indefinitely or double-count
		// the freeze journal entries.
		if err := orderSvc.RaiseDispute(ctx, order.ID, order.CustomerID, "again"); err == nil {
			t.Fatal("expected second RaiseDispute on an already-frozen escrow to fail")
		}
	})
}

func TestGetDisputeStatus_RejectsNonOwner(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		ledger := NewLedgerService(repo.NewLedgerRepo(tx), nil)
		orderRepo := repo.NewOrderRepo(tx)
		orderSvc := NewOrderService(orderRepo, nil, ledger, nil)

		order := seedOrder(t, tx, 10_000_000, 8_000_000, 1_000_000, 1_000_000, 0)

		imposter := uuid.New()
		u := &model.User{ID: imposter, Role: model.RoleCustomer, FirstName: "Not", LastName: "Owner", Phone: "+234810" + uuid.NewString()[:8], PasswordHash: "x"}
		mustCreate(t, tx, u)

		if _, err := orderSvc.GetDisputeStatus(ctx, order.ID, imposter); err != ErrOrderForbidden {
			t.Fatalf("expected ErrOrderForbidden for non-owner, got: %v", err)
		}
	})
}
