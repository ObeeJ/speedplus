package service

// NDPR purge tests: recipient PII must be nulled after the retention window,
// but never while a dispute (frozen escrow) is open on that order.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

func seedDeliveredPackageOrder(t *testing.T, tx *gorm.DB, deliveredAt time.Time, cipher interface {
	Encrypt(string) (string, error)
}) *model.Order {
	t.Helper()
	unique := uuid.NewString()[:8]

	customer := &model.User{ID: uuid.New(), Role: model.RoleCustomer, FirstName: "C", LastName: "T", Phone: "+234850" + unique, PasswordHash: "x"}
	mustCreate(t, tx, customer)
	merchantUser := &model.User{ID: uuid.New(), Role: model.RoleMerchant, FirstName: "M", LastName: "T", Phone: "+234851" + unique, PasswordHash: "x"}
	mustCreate(t, tx, merchantUser)
	merchant := &model.Merchant{ID: uuid.New(), UserID: merchantUser.ID, BusinessName: "M " + unique, Vertical: model.VerticalFood}
	mustCreate(t, tx, merchant)
	addr := &model.Address{ID: uuid.New(), UserID: customer.ID, Street: "1 St", City: "Lagos", State: "Lagos", Lat: 6.5, Lng: 3.3}
	mustCreate(t, tx, addr)
	quote := &model.PricingQuote{ID: uuid.New(), CustomerID: customer.ID, MerchantID: merchant.ID, StopCount: 1, TotalKobo: 1000, QuoteHash: "h-" + unique, ExpiresAt: time.Now().Add(time.Hour)}
	mustCreate(t, tx, quote)

	nameEnc, _ := cipher.Encrypt("Recipient Name")
	phoneEnc, _ := cipher.Encrypt("+2348099999999")

	order := &model.Order{
		ID: uuid.New(), CustomerID: customer.ID, MerchantID: merchant.ID,
		QuoteID: quote.ID, Vertical: "package", Status: model.OrderDelivered,
		DeliveryAddressID: addr.ID, IdempotencyKey: "test-" + unique,
		RecipientNameEnc: &nameEnc, RecipientPhoneEnc: &phoneEnc,
		DeliveredAt: &deliveredAt,
	}
	mustCreate(t, tx, order)
	return order
}

func TestPurgeStaleRecipientPII_PurgesOldDeliveredOrders(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		cipher := testRecipientCipher(t)
		old := seedDeliveredPackageOrder(t, tx, time.Now().AddDate(0, 0, -recipientPIIRetentionDays-1), cipher)

		svc := &OrderService{orders: repo.NewOrderRepo(tx)}
		n, err := svc.PurgeStaleRecipientPII(context.Background())
		if err != nil {
			t.Fatalf("PurgeStaleRecipientPII: %v", err)
		}
		if n != 1 {
			t.Fatalf("expected 1 order purged, got %d", n)
		}

		var reloaded model.Order
		if err := tx.First(&reloaded, old.ID).Error; err != nil {
			t.Fatalf("reload order: %v", err)
		}
		if reloaded.RecipientNameEnc != nil || reloaded.RecipientPhoneEnc != nil {
			t.Fatal("recipient PII must be nulled after the retention window")
		}
	})
}

func TestPurgeStaleRecipientPII_SkipsRecentOrders(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		cipher := testRecipientCipher(t)
		recent := seedDeliveredPackageOrder(t, tx, time.Now().AddDate(0, 0, -1), cipher)

		svc := &OrderService{orders: repo.NewOrderRepo(tx)}
		if _, err := svc.PurgeStaleRecipientPII(context.Background()); err != nil {
			t.Fatalf("PurgeStaleRecipientPII: %v", err)
		}

		var reloaded model.Order
		if err := tx.First(&reloaded, recent.ID).Error; err != nil {
			t.Fatalf("reload order: %v", err)
		}
		if reloaded.RecipientNameEnc == nil {
			t.Fatal("recipient PII must survive within the retention window")
		}
	})
}

func TestPurgeStaleRecipientPII_SkipsOpenDisputes(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		cipher := testRecipientCipher(t)
		disputed := seedDeliveredPackageOrder(t, tx, time.Now().AddDate(0, 0, -recipientPIIRetentionDays-1), cipher)

		// escrow_holds.account_id has an FK to ledger_accounts — create a real
		// account rather than a free-floating UUID.
		ledger := NewLedgerService(repo.NewLedgerRepo(tx), nil)
		holdAcct, err := ledger.EnsureWallet(context.Background(), tx, seedWalletOwner(t, tx).ID)
		if err != nil {
			t.Fatalf("ensure hold account: %v", err)
		}
		hold := &model.EscrowHold{
			ID: uuid.New(), OrderID: disputed.ID, AccountID: holdAcct.ID,
			AmountKobo: 1000, Status: model.EscrowFrozen,
		}
		mustCreate(t, tx, hold)

		svc := &OrderService{orders: repo.NewOrderRepo(tx)}
		n, err := svc.PurgeStaleRecipientPII(context.Background())
		if err != nil {
			t.Fatalf("PurgeStaleRecipientPII: %v", err)
		}
		if n != 0 {
			t.Fatalf("expected 0 orders purged (open dispute), got %d", n)
		}

		var reloaded model.Order
		if err := tx.First(&reloaded, disputed.ID).Error; err != nil {
			t.Fatalf("reload order: %v", err)
		}
		if reloaded.RecipientNameEnc == nil {
			t.Fatal("recipient PII must survive while a dispute (frozen escrow) is open")
		}
	})
}
