package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

func TestBuildProofKey_ScopesToOrderAndStop(t *testing.T) {
	orderID := uuid.New()
	stopID := uuid.New()

	keyNoStop := buildProofKey(orderID, nil, "pickup_photo")
	if !strings.HasPrefix(keyNoStop, "proof/"+orderID.String()+"/pickup_photo-") {
		t.Errorf("unexpected key shape for no-stop order: %s", keyNoStop)
	}

	keyWithStop := buildProofKey(orderID, &stopID, "dropoff_video")
	wantPrefix := "proof/" + orderID.String() + "/" + stopID.String() + "/dropoff_video-"
	if !strings.HasPrefix(keyWithStop, wantPrefix) {
		t.Errorf("unexpected key shape for stop-scoped media: %s, want prefix %s", keyWithStop, wantPrefix)
	}
}

func TestBuildProofKey_IsUniquePerCall(t *testing.T) {
	orderID := uuid.New()
	k1 := buildProofKey(orderID, nil, "pickup_photo")
	k2 := buildProofKey(orderID, nil, "pickup_photo")
	if k1 == k2 {
		t.Fatal("two captures of the same kind must not collide on the same key")
	}
}

func TestProofMediaService_RejectsInvalidKind(t *testing.T) {
	svc := &ProofMediaService{}
	if _, _, err := svc.PresignUpload(context.Background(), uuid.New(), nil, "not_a_real_kind", uuid.New(), "image/jpeg"); err == nil {
		t.Fatal("expected rejection of an invalid proof kind")
	}
}

func TestProofMediaService_PresignFailsClosedWithoutR2(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		unique := uuid.NewString()[:8]
		driver := &model.User{ID: uuid.New(), Role: model.RoleDriver, FirstName: "R", LastName: "T", Phone: "+234820" + unique, PasswordHash: "x"}
		mustCreate(t, tx, driver)
		customer := &model.User{ID: uuid.New(), Role: model.RoleCustomer, FirstName: "C", LastName: "T", Phone: "+234821" + unique, PasswordHash: "x"}
		mustCreate(t, tx, customer)
		merchantUser := &model.User{ID: uuid.New(), Role: model.RoleMerchant, FirstName: "M", LastName: "T", Phone: "+234822" + unique, PasswordHash: "x"}
		mustCreate(t, tx, merchantUser)
		merchant := &model.Merchant{ID: uuid.New(), UserID: merchantUser.ID, BusinessName: "M " + unique, Vertical: model.VerticalFood}
		mustCreate(t, tx, merchant)
		addr := &model.Address{ID: uuid.New(), UserID: customer.ID, Street: "1 St", City: "Lagos", State: "Lagos", Lat: 6.5, Lng: 3.3}
		mustCreate(t, tx, addr)
		quote := &model.PricingQuote{ID: uuid.New(), CustomerID: customer.ID, MerchantID: merchant.ID, StopCount: 1, TotalKobo: 1000, QuoteHash: "h-" + unique, ExpiresAt: time.Now().Add(time.Hour)}
		mustCreate(t, tx, quote)
		order := &model.Order{
			ID: uuid.New(), CustomerID: customer.ID, MerchantID: merchant.ID, DriverID: &driver.ID,
			QuoteID: quote.ID, Vertical: "package", Status: model.OrderInTransit,
			DeliveryAddressID: addr.ID, IdempotencyKey: "test-" + unique,
		}
		mustCreate(t, tx, order)

		svc := NewProofMediaService(tx, nil) // no R2 client configured
		if _, _, err := svc.PresignUpload(context.Background(), order.ID, nil, "pickup_photo", driver.ID, "image/jpeg"); err == nil {
			t.Fatal("PresignUpload must fail closed when R2 is not configured, not silently skip evidence")
		}
	})
}

func TestProofMediaService_OnlyAssignedDriverCanCapture(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		unique := uuid.NewString()[:8]
		assignedDriver := &model.User{ID: uuid.New(), Role: model.RoleDriver, FirstName: "R1", LastName: "T", Phone: "+234830" + unique, PasswordHash: "x"}
		mustCreate(t, tx, assignedDriver)
		otherDriver := &model.User{ID: uuid.New(), Role: model.RoleDriver, FirstName: "R2", LastName: "T", Phone: "+234831" + unique, PasswordHash: "x"}
		mustCreate(t, tx, otherDriver)
		customer := &model.User{ID: uuid.New(), Role: model.RoleCustomer, FirstName: "C", LastName: "T", Phone: "+234832" + unique, PasswordHash: "x"}
		mustCreate(t, tx, customer)
		merchantUser := &model.User{ID: uuid.New(), Role: model.RoleMerchant, FirstName: "M", LastName: "T", Phone: "+234833" + unique, PasswordHash: "x"}
		mustCreate(t, tx, merchantUser)
		merchant := &model.Merchant{ID: uuid.New(), UserID: merchantUser.ID, BusinessName: "M " + unique, Vertical: model.VerticalFood}
		mustCreate(t, tx, merchant)
		addr := &model.Address{ID: uuid.New(), UserID: customer.ID, Street: "1 St", City: "Lagos", State: "Lagos", Lat: 6.5, Lng: 3.3}
		mustCreate(t, tx, addr)
		quote := &model.PricingQuote{ID: uuid.New(), CustomerID: customer.ID, MerchantID: merchant.ID, StopCount: 1, TotalKobo: 1000, QuoteHash: "h-" + unique, ExpiresAt: time.Now().Add(time.Hour)}
		mustCreate(t, tx, quote)
		order := &model.Order{
			ID: uuid.New(), CustomerID: customer.ID, MerchantID: merchant.ID, DriverID: &assignedDriver.ID,
			QuoteID: quote.ID, Vertical: "package", Status: model.OrderInTransit,
			DeliveryAddressID: addr.ID, IdempotencyKey: "test-" + unique,
		}
		mustCreate(t, tx, order)

		svc := NewProofMediaService(tx, nil)
		_, err := svc.ConfirmUpload(context.Background(), otherDriver.ID, ConfirmUploadInput{
			OrderID: order.ID, Kind: "pickup_photo", Key: "proof/x", SHA256: "deadbeef",
		})
		if err == nil {
			t.Fatal("a driver not assigned to this order must not be able to record proof media")
		}
	})
}

func TestProofMediaService_GetMediaForOrder_ExposureControl(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		unique := uuid.NewString()[:8]
		driver := &model.User{ID: uuid.New(), Role: model.RoleDriver, FirstName: "R", LastName: "T", Phone: "+234840" + unique, PasswordHash: "x"}
		mustCreate(t, tx, driver)
		customer := &model.User{ID: uuid.New(), Role: model.RoleCustomer, FirstName: "C", LastName: "T", Phone: "+234841" + unique, PasswordHash: "x"}
		mustCreate(t, tx, customer)
		otherCustomer := &model.User{ID: uuid.New(), Role: model.RoleCustomer, FirstName: "C2", LastName: "T", Phone: "+234842" + unique, PasswordHash: "x"}
		mustCreate(t, tx, otherCustomer)
		merchantUser := &model.User{ID: uuid.New(), Role: model.RoleMerchant, FirstName: "M", LastName: "T", Phone: "+234843" + unique, PasswordHash: "x"}
		mustCreate(t, tx, merchantUser)
		merchant := &model.Merchant{ID: uuid.New(), UserID: merchantUser.ID, BusinessName: "M " + unique, Vertical: model.VerticalFood}
		mustCreate(t, tx, merchant)
		addr := &model.Address{ID: uuid.New(), UserID: customer.ID, Street: "1 St", City: "Lagos", State: "Lagos", Lat: 6.5, Lng: 3.3}
		mustCreate(t, tx, addr)
		quote := &model.PricingQuote{ID: uuid.New(), CustomerID: customer.ID, MerchantID: merchant.ID, StopCount: 1, TotalKobo: 1000, QuoteHash: "h-" + unique, ExpiresAt: time.Now().Add(time.Hour)}
		mustCreate(t, tx, quote)
		order := &model.Order{
			ID: uuid.New(), CustomerID: customer.ID, MerchantID: merchant.ID, DriverID: &driver.ID,
			QuoteID: quote.ID, Vertical: "package", Status: model.OrderDelivered,
			DeliveryAddressID: addr.ID, IdempotencyKey: "test-" + unique,
		}
		mustCreate(t, tx, order)
		mustCreate(t, tx, &model.ProofMedia{
			ID: uuid.New(), OrderID: order.ID, Kind: "dropoff_photo", R2Key: "proof/x", SHA256: "deadbeef",
			CapturedAt: time.Now(), CapturedBy: driver.ID,
		})

		svc := NewProofMediaService(tx, nil)

		if _, err := svc.GetMediaForOrder(context.Background(), order.ID, otherCustomer.ID, "customer"); err == nil {
			t.Fatal("a different customer must not be able to view this order's proof media")
		}

		media, err := svc.GetMediaForOrder(context.Background(), order.ID, customer.ID, "customer")
		if err != nil {
			t.Fatalf("order owner must be able to view proof media: %v", err)
		}
		if len(media) != 1 {
			t.Fatalf("expected 1 media row, got %d", len(media))
		}

		if _, err := svc.GetMediaForOrder(context.Background(), order.ID, uuid.New(), "admin"); err != nil {
			t.Fatalf("admin must be able to view proof media: %v", err)
		}
	})
}
