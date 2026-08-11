package service

// Money-path tests: double-entry invariant, escrow hold->settle lifecycle,
// webhook dedupe. These exercise the exact code paths flagged as having
// zero coverage - a wrong integer or a missed error return here is a silent
// ledger drift that reconciliation only catches days later.
//
// Requires DATABASE_URL (a real Postgres with migrations applied - see
// cmd/server/migrate_ci). Skips locally if unset; CI sets it and runs the
// migrate_ci step before `go test`, matching .github/workflows/api-ci.yml.
//
// Every test runs inside a single outer transaction that is always rolled
// back (withTx). WalletService.ProcessWebhook opens its own nested
// gorm.DB.Transaction() call - when the *gorm.DB passed to the service is
// itself already an open transaction, GORM uses a SAVEPOINT instead of a
// new top-level transaction, so the outer rollback still undoes it. This
// means no cleanup queries of any kind are needed anywhere in this file.

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/db"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/payment"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set - skipping money-path integration tests")
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
	// users.referral_code is uniqueIndex/not-null and is generated at
	// registration in production. Fixtures build User structs directly, so
	// fill it here — otherwise every test seeding more than one user
	// collides on the empty-string default.
	if u, ok := v.(*model.User); ok && u.ReferralCode == "" {
		u.ReferralCode = "TEST" + strings.ToUpper(uuid.NewString()[:8])
	}
	if err := tx.Create(v).Error; err != nil {
		t.Fatalf("create %T: %v", v, err)
	}
}

// seedWalletOwner creates a real user to own a ledger account —
// ledger_accounts.owner_id carries an FK to users(id).
func seedWalletOwner(t *testing.T, tx *gorm.DB) *model.User {
	t.Helper()
	u := &model.User{
		ID: uuid.New(), Role: model.RoleCustomer,
		FirstName: "Wallet", LastName: "Owner",
		Phone: "+234803" + uuid.NewString()[:8], PasswordHash: "x",
	}
	mustCreate(t, tx, u)
	return u
}

func seedOrder(t *testing.T, tx *gorm.DB, totalKobo, subtotalKobo, deliveryKobo, serviceKobo, tipKobo int64) *model.Order {
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

	quote := &model.PricingQuote{
		ID: uuid.New(), CustomerID: customer.ID, MerchantID: merchant.ID,
		SubtotalKobo: subtotalKobo, DeliveryKobo: deliveryKobo, ServiceKobo: serviceKobo, TotalKobo: totalKobo,
		QuoteHash: "test-hash-" + unique, ExpiresAt: time.Now().Add(time.Hour),
	}
	mustCreate(t, tx, quote)

	order := &model.Order{
		ID: uuid.New(), CustomerID: customer.ID, MerchantID: merchant.ID, DriverID: &driverUser.ID,
		QuoteID: quote.ID, Vertical: "food", Status: model.OrderPending,
		SubtotalKobo: subtotalKobo, DeliveryKobo: deliveryKobo, ServiceKobo: serviceKobo, TipKobo: tipKobo, TotalKobo: totalKobo,
		DeliveryAddressID: addr.ID, IdempotencyKey: "test-order-" + unique,
	}
	mustCreate(t, tx, order)
	return order
}

func assertAllJournalsBalanced(t *testing.T, tx *gorm.DB) {
	t.Helper()
	var rows []struct {
		JournalID uuid.UUID
		Sum       int64
	}
	q := "SELECT journal_id, SUM(amount_kobo) AS sum " + "FROM ledger_entries GROUP BY journal_id"
	if err := tx.Raw(q).Scan(&rows).Error; err != nil {
		t.Fatalf("sum journals: %v", err)
	}
	for _, r := range rows {
		if r.Sum != 0 {
			t.Errorf("journal %s sums to %d, want 0 (double-entry invariant violated)", r.JournalID, r.Sum)
		}
	}
}

func TestJournal_RejectsUnbalancedEntries(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ledgerRepo := repo.NewLedgerRepo(tx)
		ledger := NewLedgerService(ledgerRepo, nil)

		// ledger_accounts.owner_id has an FK to users — seed a real owner.
		acctA, err := ledger.EnsureWallet(context.Background(), tx, seedWalletOwner(t, tx).ID)
		if err != nil {
			t.Fatalf("ensure wallet: %v", err)
		}

		unbalanced := []model.LedgerEntry{
			{ID: uuid.New(), JournalID: uuid.New(), AccountID: acctA.ID, AmountKobo: 1000, Description: "test", RefType: "test"},
		}
		if err := ledger.journal(context.Background(), tx, unbalanced); err == nil {
			t.Fatal("expected journal() to reject an unbalanced entry set, got nil error")
		}
	})
}

func TestJournal_AcceptsBalancedEntries(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ledgerRepo := repo.NewLedgerRepo(tx)
		ledger := NewLedgerService(ledgerRepo, nil)

		acctA, err := ledger.EnsureWallet(context.Background(), tx, seedWalletOwner(t, tx).ID)
		if err != nil {
			t.Fatalf("ensure wallet: %v", err)
		}
		acctB, err := ledger.EnsureWallet(context.Background(), tx, seedWalletOwner(t, tx).ID)
		if err != nil {
			t.Fatalf("ensure wallet: %v", err)
		}

		journalID := uuid.New()
		balanced := []model.LedgerEntry{
			{ID: uuid.New(), JournalID: journalID, AccountID: acctA.ID, AmountKobo: -500, Description: "test debit", RefType: "test"},
			{ID: uuid.New(), JournalID: journalID, AccountID: acctB.ID, AmountKobo: 500, Description: "test credit", RefType: "test"},
		}
		if err := ledger.journal(context.Background(), tx, balanced); err != nil {
			t.Fatalf("expected balanced journal to succeed, got: %v", err)
		}
		assertAllJournalsBalanced(t, tx)
	})
}

func TestEscrowHoldAndSettle_FullLifecycle(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		ledgerRepo := repo.NewLedgerRepo(tx)
		ledger := NewLedgerService(ledgerRepo, nil)

		const subtotal, delivery, service, tip = int64(500000), int64(80000), int64(20000), int64(10000)
		total := subtotal + delivery + service + tip
		order := seedOrder(t, tx, total, subtotal, delivery, service, tip)

		if err := ledger.CreditWallet(ctx, tx, order.CustomerID, total, "test_seed", nil); err != nil {
			t.Fatalf("seed customer wallet: %v", err)
		}

		if err := ledger.HoldEscrow(ctx, tx, order.ID, order.CustomerID, total); err != nil {
			t.Fatalf("hold escrow: %v", err)
		}

		custBalAfterHold, err := ledgerRepo.GetBalance(ctx, order.CustomerID)
		if err != nil {
			t.Fatalf("get customer balance: %v", err)
		}
		if custBalAfterHold != 0 {
			t.Errorf("customer balance after full-total hold = %d, want 0", custBalAfterHold)
		}

		// Settlement requires proof of delivery for non-gas orders (tier-0
		// prevention added with the dispute work): at least one dropoff_photo
		// must exist before escrow is released. This fixture predates that gate,
		// so seed the evidence rather than weaken the invariant.
		mustCreate(t, tx, &model.ProofMedia{
			ID: uuid.New(), OrderID: order.ID, Kind: "dropoff_photo",
			R2Key: "test/dropoff/" + uuid.NewString(), SHA256: "deadbeef",
			CapturedAt: time.Now(), CapturedBy: *order.DriverID,
		})

		paycodeEventID := uuid.New()
		if err := ledger.Settle(ctx, tx, order, paycodeEventID); err != nil {
			t.Fatalf("settle: %v", err)
		}

		escrowAcct, err := ledger.platformAccount(ctx, tx, model.AccountEscrow)
		if err != nil {
			t.Fatalf("get escrow account: %v", err)
		}
		var escrowBal model.WalletBalance
		if err := tx.Where("account_id = ?", escrowAcct.ID).First(&escrowBal).Error; err != nil {
			t.Fatalf("get escrow balance: %v", err)
		}
		if escrowBal.BalanceKobo != 0 {
			t.Errorf("escrow balance after settle = %d, want 0 (money left stranded in escrow)", escrowBal.BalanceKobo)
		}

		// Ledger accounts are keyed by User.ID (ledger_accounts.owner_id has an
		// FK to users), so read the merchant's balance through its owning user
		// — the same resolution LedgerService.EnsureMerchantWallet performs.
		var merchantRow model.Merchant
		if err := tx.Select("user_id").First(&merchantRow, "id = ?", order.MerchantID).Error; err != nil {
			t.Fatalf("resolve merchant user: %v", err)
		}
		merchantBal, err := ledgerRepo.GetBalance(ctx, merchantRow.UserID)
		if err != nil {
			t.Fatalf("get merchant balance: %v", err)
		}
		if merchantBal <= 0 {
			t.Errorf("merchant balance after settle = %d, want > 0", merchantBal)
		}
		driverBal, err := ledgerRepo.GetBalance(ctx, *order.DriverID)
		if err != nil {
			t.Fatalf("get driver balance: %v", err)
		}
		if driverBal <= tip {
			t.Errorf("driver balance after settle = %d, want > tip (%d)", driverBal, tip)
		}

		var hold model.EscrowHold
		if err := tx.Where("order_id = ?", order.ID).First(&hold).Error; err != nil {
			t.Fatalf("get escrow hold: %v", err)
		}
		if hold.Status != model.EscrowReleased {
			t.Errorf("escrow hold status = %s, want %s", hold.Status, model.EscrowReleased)
		}
		if hold.ReleasedAt == nil {
			t.Error("escrow hold ReleasedAt is nil after settle")
		}

		assertAllJournalsBalanced(t, tx)
	})
}

type stubProvider struct{ verifyCalls int }

func (s *stubProvider) Name() string { return "stub" }
func (s *stubProvider) InitiateCharge(ctx context.Context, req payment.ChargeRequest) (*payment.ChargeResponse, error) {
	return &payment.ChargeResponse{Reference: req.Reference}, nil
}
func (s *stubProvider) VerifyTransaction(ctx context.Context, ref string) (*payment.VerifyResponse, error) {
	s.verifyCalls++
	return &payment.VerifyResponse{Reference: ref, AmountKobo: 100000, Status: "success"}, nil
}
func (s *stubProvider) InitiateTransfer(ctx context.Context, req payment.TransferRequest) (*payment.TransferResponse, error) {
	return &payment.TransferResponse{}, nil
}
func (s *stubProvider) VerifyWebhookSignature(payload []byte, signature string) bool { return true }

type noopPINVerifier struct{}

func (noopPINVerifier) VerifyPIN(ctx context.Context, userID uuid.UUID, pin string) error { return nil }

type noopWalletEmail struct{}

func (noopWalletEmail) SendWalletFunded(ctx context.Context, toEmail, firstName string, amountKobo, newBalanceKobo int64) {
}
func (noopWalletEmail) SendTransferReceived(ctx context.Context, toEmail, firstName string, amountKobo int64, senderName string) {
}

func TestProcessWebhook_DuplicateEventCreditedOnce(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		unique := uuid.NewString()[:8]

		user := &model.User{ID: uuid.New(), Role: model.RoleCustomer, FirstName: "Webhook", LastName: "Test", Phone: "+234803" + unique, PasswordHash: "x"}
		mustCreate(t, tx, user)

		ref := "test-ref-" + unique
		intent := &model.PaymentIntent{
			ID: uuid.New(), UserID: user.ID, AmountKobo: 100000, Provider: "stub",
			Status: "pending", IdempotencyKey: "test-idem-" + unique, ProviderRef: &ref,
		}
		mustCreate(t, tx, intent)

		// Pass `tx` (already an open transaction) as the service's db, not
		// `gdb` — WalletService.ProcessWebhook calls db.Transaction(...)
		// internally, and GORM treats that as a SAVEPOINT when the db handle
		// is already mid-transaction, keeping everything inside the outer
		// rollback boundary.
		ledgerRepo := repo.NewLedgerRepo(tx)
		ledger := NewLedgerService(ledgerRepo, nil)
		provider := &stubProvider{}
		userRepo := repo.NewUserRepo(tx)
		wallet := NewWalletService(repo.NewWalletRepo(tx), ledger, noopPINVerifier{}, provider, noopWalletEmail{}, userRepo)

		payload := WebhookPayload{
			Provider: "stub", EventID: "evt-" + unique, EventType: "charge.success",
			Reference: ref, RawBody: []byte("{}"),
		}

		if err := wallet.ProcessWebhook(ctx, payload); err != nil {
			t.Fatalf("first ProcessWebhook call: %v", err)
		}
		if err := wallet.ProcessWebhook(ctx, payload); err != nil {
			t.Fatalf("second (duplicate) ProcessWebhook call returned error instead of no-op: %v", err)
		}

		if provider.verifyCalls != 1 {
			t.Errorf("provider.VerifyTransaction called %d times, want 1 - dedupe by (provider, event_id) should prevent the second call from re-verifying", provider.verifyCalls)
		}

		bal, err := ledger.GetBalance(ctx, user.ID)
		if err != nil {
			t.Fatalf("get balance: %v", err)
		}
		if bal != 100000 {
			t.Errorf("balance after duplicate webhook delivery = %d, want 100000 (credited exactly once)", bal)
		}
	})
}
