package repo

// DB-backed tests for the escrow-hold half of ledgerRepo — CreateEscrowHold,
// LockEscrowHold (the SELECT ... FOR UPDATE the freeze/release/settle paths
// depend on to avoid a double-freeze race), and FindEscrowHoldByOrder (added
// in b43056f as the read path behind GetEscrowStatus). Before this file,
// internal/repo had zero test files despite being the layer every money-path
// service call ultimately hits.
//
// Requires DATABASE_URL (a real Postgres with migrations applied); skips
// locally if unset, matching the convention in internal/service.
//
// Every test runs inside a single outer transaction that is always rolled
// back, so no cleanup queries are needed.

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/db"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set - skipping repo integration tests")
	}
	gdb, err := db.Connect(dsn, false)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	return gdb
}

func withTx(t *testing.T, gdb *gorm.DB, fn func(tx *gorm.DB)) {
	t.Helper()
	tx := gdb.Begin()
	defer tx.Rollback()
	fn(tx)
}

func mustCreate(t *testing.T, tx *gorm.DB, v interface{}) {
	t.Helper()
	if u, ok := v.(*model.User); ok && u.ReferralCode == "" {
		u.ReferralCode = "TEST" + uuid.NewString()[:8]
	}
	if err := tx.Create(v).Error; err != nil {
		t.Fatalf("create %T: %v", v, err)
	}
}

// seedOrderWithAccount creates the minimum fixture graph escrow_holds' FKs
// require: a customer, driver, merchant (+ merchant user), a pricing quote,
// an order, and a ledger account to hold the escrow.
func seedOrderWithAccount(t *testing.T, tx *gorm.DB) (*model.Order, *model.LedgerAccount) {
	t.Helper()
	unique := uuid.NewString()[:8]

	customer := &model.User{ID: uuid.New(), Role: model.RoleCustomer, FirstName: "Test", LastName: "Customer", Phone: "+234800" + unique, PasswordHash: "x"}
	mustCreate(t, tx, customer)

	driverUser := &model.User{ID: uuid.New(), Role: model.RoleDriver, FirstName: "Test", LastName: "Driver", Phone: "+234801" + unique, PasswordHash: "x"}
	mustCreate(t, tx, driverUser)

	merchantUser := &model.User{ID: uuid.New(), Role: model.RoleMerchant, FirstName: "Test", LastName: "Merchant", Phone: "+234802" + unique, PasswordHash: "x"}
	mustCreate(t, tx, merchantUser)

	merchant := &model.Merchant{ID: uuid.New(), UserID: merchantUser.ID, BusinessName: "Test Merchant " + unique, Vertical: model.VerticalFood}
	mustCreate(t, tx, merchant)

	addr := &model.Address{ID: uuid.New(), UserID: customer.ID, Street: "1 Test St", City: "Lagos", State: "Lagos", Lat: 6.5, Lng: 3.3}
	mustCreate(t, tx, addr)

	const total int64 = 10_000_000
	quote := &model.PricingQuote{
		ID: uuid.New(), CustomerID: customer.ID, MerchantID: merchant.ID,
		SubtotalKobo: 8_000_000, DeliveryKobo: 1_000_000, ServiceKobo: 1_000_000, TotalKobo: total,
		QuoteHash: "test-hash-" + unique, ExpiresAt: time.Now().Add(time.Hour),
	}
	mustCreate(t, tx, quote)

	order := &model.Order{
		ID: uuid.New(), CustomerID: customer.ID, MerchantID: merchant.ID, DriverID: &driverUser.ID,
		QuoteID: quote.ID, Vertical: "food", Status: model.OrderPending,
		SubtotalKobo: 8_000_000, DeliveryKobo: 1_000_000, ServiceKobo: 1_000_000, TotalKobo: total,
		DeliveryAddressID: addr.ID, IdempotencyKey: "test-order-" + unique,
	}
	mustCreate(t, tx, order)

	acct := &model.LedgerAccount{ID: uuid.New(), OwnerID: &customer.ID, Type: model.AccountWallet}
	mustCreate(t, tx, acct)

	return order, acct
}

func TestLedgerRepo_CreateAndFindEscrowHoldByOrder(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		r := NewLedgerRepo(tx)
		order, acct := seedOrderWithAccount(t, tx)

		hold := &model.EscrowHold{ID: uuid.New(), OrderID: order.ID, AccountID: acct.ID, AmountKobo: 10_000_000, Status: model.EscrowHeld}
		if err := r.CreateEscrowHold(context.Background(), tx, hold); err != nil {
			t.Fatalf("CreateEscrowHold: %v", err)
		}

		found, err := r.FindEscrowHoldByOrder(context.Background(), order.ID)
		if err != nil {
			t.Fatalf("FindEscrowHoldByOrder: %v", err)
		}
		if found.ID != hold.ID {
			t.Errorf("found hold ID = %s, want %s", found.ID, hold.ID)
		}
		if found.Status != model.EscrowHeld {
			t.Errorf("found hold status = %q, want %q", found.Status, model.EscrowHeld)
		}
		if found.AmountKobo != 10_000_000 {
			t.Errorf("found hold amount = %d, want 10000000", found.AmountKobo)
		}
	})
}

func TestLedgerRepo_FindEscrowHoldByOrder_NotFound(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		r := NewLedgerRepo(tx)
		if _, err := r.FindEscrowHoldByOrder(context.Background(), uuid.New()); err == nil {
			t.Fatal("expected error for non-existent escrow hold, got nil")
		}
	})
}

func TestLedgerRepo_LockEscrowHold_OnlyMatchesRequestedStatus(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		r := NewLedgerRepo(tx)
		order, acct := seedOrderWithAccount(t, tx)

		hold := &model.EscrowHold{ID: uuid.New(), OrderID: order.ID, AccountID: acct.ID, AmountKobo: 10_000_000, Status: model.EscrowHeld}
		if err := r.CreateEscrowHold(context.Background(), tx, hold); err != nil {
			t.Fatalf("CreateEscrowHold: %v", err)
		}

		// Locking for the wrong status must fail — this is exactly the guard
		// FreezeEscrowForCustomer relies on to reject a double-freeze.
		if _, err := r.LockEscrowHold(context.Background(), tx, order.ID, model.EscrowFrozen); err == nil {
			t.Fatal("expected LockEscrowHold to fail when no hold has the requested status")
		}

		locked, err := r.LockEscrowHold(context.Background(), tx, order.ID, model.EscrowHeld)
		if err != nil {
			t.Fatalf("LockEscrowHold(held): %v", err)
		}
		if locked.ID != hold.ID {
			t.Errorf("locked hold ID = %s, want %s", locked.ID, hold.ID)
		}
	})
}

func TestLedgerRepo_SaveEscrowHold_UpdatesStatus(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		r := NewLedgerRepo(tx)
		order, acct := seedOrderWithAccount(t, tx)

		hold := &model.EscrowHold{ID: uuid.New(), OrderID: order.ID, AccountID: acct.ID, AmountKobo: 10_000_000, Status: model.EscrowHeld}
		if err := r.CreateEscrowHold(context.Background(), tx, hold); err != nil {
			t.Fatalf("CreateEscrowHold: %v", err)
		}

		hold.Status = model.EscrowFrozen
		if err := r.SaveEscrowHold(context.Background(), tx, hold); err != nil {
			t.Fatalf("SaveEscrowHold: %v", err)
		}

		found, err := r.FindEscrowHoldByOrder(context.Background(), order.ID)
		if err != nil {
			t.Fatalf("FindEscrowHoldByOrder: %v", err)
		}
		if found.Status != model.EscrowFrozen {
			t.Errorf("status after save = %q, want %q", found.Status, model.EscrowFrozen)
		}
	})
}
