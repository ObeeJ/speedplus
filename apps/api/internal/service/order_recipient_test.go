package service

// Recipient PII exposure-control tests: recipient name/phone for package
// stops must be encrypted at rest and only decrypted for an authorized
// reader (order owner, admin, or the assigned driver on the single active
// stop while in transit). A driver must never see other stops' recipients,
// nor any recipient after the order is delivered.
//
// Requires DATABASE_URL (see ledger_money_test.go for the shared fixtures
// testDB/withTx/mustCreate). Skips locally if unset.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	appcrypto "github.com/speedplus/api/internal/crypto"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

func testRecipientCipher(t *testing.T) *appcrypto.Cipher {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	c, err := appcrypto.NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	return c
}

func TestGetStops_RecipientExposureControl(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		unique := uuid.NewString()[:8]
		cipher := testRecipientCipher(t)

		customer := &model.User{ID: uuid.New(), Role: model.RoleCustomer, FirstName: "Sender", LastName: "Test", Phone: "+234810" + unique, PasswordHash: "x"}
		mustCreate(t, tx, customer)
		driverUser := &model.User{ID: uuid.New(), Role: model.RoleDriver, FirstName: "Rider", LastName: "Test", Phone: "+234811" + unique, PasswordHash: "x"}
		mustCreate(t, tx, driverUser)
		otherDriver := &model.User{ID: uuid.New(), Role: model.RoleDriver, FirstName: "Other", LastName: "Rider", Phone: "+234812" + unique, PasswordHash: "x"}
		mustCreate(t, tx, otherDriver)
		merchantUser := &model.User{ID: uuid.New(), Role: model.RoleMerchant, FirstName: "Logistics", LastName: "Test", Phone: "+234813" + unique, PasswordHash: "x"}
		mustCreate(t, tx, merchantUser)
		merchant := &model.Merchant{ID: uuid.New(), UserID: merchantUser.ID, BusinessName: "Test Logistics " + unique, Vertical: model.VerticalFood}
		mustCreate(t, tx, merchant)
		addr := &model.Address{ID: uuid.New(), UserID: customer.ID, Street: "1 Test St", City: "Lagos", State: "Lagos", Lat: 6.5, Lng: 3.3}
		mustCreate(t, tx, addr)
		quote := &model.PricingQuote{ID: uuid.New(), CustomerID: customer.ID, MerchantID: merchant.ID, SubtotalKobo: 1000, DeliveryKobo: 500, ServiceKobo: 50, TotalKobo: 1550, QuoteHash: "h-" + unique, ExpiresAt: time.Now().Add(time.Hour)}
		mustCreate(t, tx, quote)

		order := &model.Order{
			ID: uuid.New(), CustomerID: customer.ID, MerchantID: merchant.ID, DriverID: &driverUser.ID,
			QuoteID: quote.ID, Vertical: "package", Status: model.OrderInTransit,
			SubtotalKobo: 1000, DeliveryKobo: 500, ServiceKobo: 50, TotalKobo: 1550,
			DeliveryAddressID: addr.ID, IdempotencyKey: "test-order-" + unique,
		}
		mustCreate(t, tx, order)

		nameEnc1, _ := cipher.Encrypt("Ada Okafor")
		phoneEnc1, _ := cipher.Encrypt("+2348011111111")
		nameEnc2, _ := cipher.Encrypt("Emeka Nwosu")
		phoneEnc2, _ := cipher.Encrypt("+2348022222222")

		stopConfirmed := &model.OrderStop{
			ID: uuid.New(), OrderID: order.ID, Sequence: 1, AddressID: addr.ID,
			RecipientNameEnc: &nameEnc1, RecipientPhoneEnc: &phoneEnc1,
			QRCode: "qr1", Status: "confirmed",
		}
		mustCreate(t, tx, stopConfirmed)

		stopActive := &model.OrderStop{
			ID: uuid.New(), OrderID: order.ID, Sequence: 2, AddressID: addr.ID,
			RecipientNameEnc: &nameEnc2, RecipientPhoneEnc: &phoneEnc2,
			QRCode: "qr2", Status: "pending",
		}
		mustCreate(t, tx, stopActive)

		svc := &OrderService{orders: repo.NewOrderRepo(tx), recipients: cipher}

		t.Run("customer sees all recipients", func(t *testing.T) {
			stops, err := svc.GetStops(context.Background(), order.ID, customer.ID, "customer")
			if err != nil {
				t.Fatalf("GetStops: %v", err)
			}
			for _, s := range stops {
				if s.RecipientName == nil || s.RecipientPhone == nil {
					t.Errorf("stop %d: customer (order owner) must see recipient, got nil", s.Sequence)
				}
			}
		})

		t.Run("admin sees all recipients", func(t *testing.T) {
			stops, err := svc.GetStops(context.Background(), order.ID, uuid.New(), "admin")
			if err != nil {
				t.Fatalf("GetStops: %v", err)
			}
			for _, s := range stops {
				if s.RecipientName == nil {
					t.Errorf("stop %d: admin must see recipient", s.Sequence)
				}
			}
		})

		t.Run("assigned driver sees only the active stop's recipient", func(t *testing.T) {
			stops, err := svc.GetStops(context.Background(), order.ID, driverUser.ID, "driver")
			if err != nil {
				t.Fatalf("GetStops: %v", err)
			}
			for _, s := range stops {
				if s.Sequence == 1 && s.RecipientName != nil {
					t.Error("driver must NOT see the already-confirmed stop's recipient")
				}
				if s.Sequence == 2 {
					if s.RecipientName == nil || *s.RecipientName != "Emeka Nwosu" {
						t.Errorf("driver must see the active stop's recipient, got %v", s.RecipientName)
					}
				}
			}
		})

		t.Run("a driver not assigned to this order sees nothing", func(t *testing.T) {
			stops, err := svc.GetStops(context.Background(), order.ID, otherDriver.ID, "driver")
			if err != nil {
				t.Fatalf("GetStops: %v", err)
			}
			for _, s := range stops {
				if s.RecipientName != nil || s.RecipientPhone != nil {
					t.Errorf("stop %d: unassigned driver must never see recipient PII", s.Sequence)
				}
			}
		})

		t.Run("no recipient is ever exposed once the order is delivered", func(t *testing.T) {
			if err := tx.Model(&model.Order{}).Where("id = ?", order.ID).Update("status", model.OrderDelivered).Error; err != nil {
				t.Fatalf("update order status: %v", err)
			}
			stops, err := svc.GetStops(context.Background(), order.ID, driverUser.ID, "driver")
			if err != nil {
				t.Fatalf("GetStops: %v", err)
			}
			for _, s := range stops {
				if s.RecipientName != nil || s.RecipientPhone != nil {
					t.Errorf("stop %d: driver must not see recipient PII after delivery", s.Sequence)
				}
			}
		})
	})
}

func TestOrder_EncryptRecipient_FailsClosedWithoutCipher(t *testing.T) {
	svc := &OrderService{} // no recipients cipher injected
	name := "Should Not Store Plaintext"
	if _, err := svc.encryptRecipient(&name); err == nil {
		t.Fatal("encryptRecipient must fail rather than silently store plaintext when no cipher is configured")
	}
	if enc, err := svc.encryptRecipient(nil); err != nil || enc != nil {
		t.Fatalf("encryptRecipient(nil) should return nil, nil even without a cipher; got %v, %v", enc, err)
	}
}
