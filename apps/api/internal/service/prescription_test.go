package service

// Prescription (Rx) tests — the pharmacy vertical previously shipped with
// zero coverage of any kind, which is how the merchantId/review/single-use
// defects (see docs audit) went unnoticed. These call the real CatalogService
// and the real repo atomic-update methods against a Postgres transaction,
// following the withTx/mustCreate pattern in ledger_money_test.go — not
// mocks, so a regression in the actual SQL WHERE clauses fails these tests.
//
// Requires DATABASE_URL; skips locally if unset, same as ledger_money_test.go.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

// seedPharmacyMerchant creates a pharmacy-vertical merchant + owning user.
func seedPharmacyMerchant(t *testing.T, tx *gorm.DB) *model.Merchant {
	t.Helper()
	unique := uuid.NewString()[:8]
	u := &model.User{ID: uuid.New(), Role: model.RoleMerchant, FirstName: "Rx", LastName: "Pharmacy", Phone: "+234813" + unique, PasswordHash: "x"}
	mustCreate(t, tx, u)
	m := &model.Merchant{ID: uuid.New(), UserID: u.ID, BusinessName: "Test Pharmacy " + unique, Vertical: model.VerticalPharmacy}
	mustCreate(t, tx, m)
	return m
}

// seedFoodMerchant creates a non-pharmacy merchant, used to prove
// CreatePrescription rejects the wrong vertical.
func seedFoodMerchant(t *testing.T, tx *gorm.DB) *model.Merchant {
	t.Helper()
	unique := uuid.NewString()[:8]
	u := &model.User{ID: uuid.New(), Role: model.RoleMerchant, FirstName: "Rx", LastName: "Food", Phone: "+234814" + unique, PasswordHash: "x"}
	mustCreate(t, tx, u)
	m := &model.Merchant{ID: uuid.New(), UserID: u.ID, BusinessName: "Test Restaurant " + unique, Vertical: model.VerticalFood}
	mustCreate(t, tx, m)
	return m
}

// seedMinimalOrder creates just enough of an order row to satisfy
// prescriptions.consumed_order_id's FK — ConsumePrescriptionTx runs inside
// OrderService.Create's transaction *after* the order row is inserted
// (order.go), so in production the FK is always satisfied by the time this
// UPDATE runs; here we replicate that ordering explicitly.
func seedMinimalOrder(t *testing.T, tx *gorm.DB, customerID, merchantID uuid.UUID) *model.Order {
	t.Helper()
	unique := uuid.NewString()[:8]
	addr := &model.Address{ID: uuid.New(), UserID: customerID, Street: "1 Test St", City: "Lagos", State: "Lagos", Lat: 6.5, Lng: 3.3}
	mustCreate(t, tx, addr)
	quote := &model.PricingQuote{
		ID: uuid.New(), CustomerID: customerID, MerchantID: merchantID,
		SubtotalKobo: 100000, DeliveryKobo: 50000, ServiceKobo: 5000, TotalKobo: 155000,
		QuoteHash: "test-rx-hash-" + unique, ExpiresAt: time.Now().Add(time.Hour),
	}
	mustCreate(t, tx, quote)
	order := &model.Order{
		ID: uuid.New(), CustomerID: customerID, MerchantID: merchantID, QuoteID: quote.ID,
		Vertical: "pharmacy", Status: model.OrderPending,
		SubtotalKobo: 100000, DeliveryKobo: 50000, ServiceKobo: 5000, TotalKobo: 155000,
		DeliveryAddressID: addr.ID, IdempotencyKey: "test-rx-order-" + unique,
	}
	mustCreate(t, tx, order)
	return order
}

func seedRxCustomer(t *testing.T, tx *gorm.DB) *model.User {
	t.Helper()
	unique := uuid.NewString()[:8]
	c := &model.User{ID: uuid.New(), Role: model.RoleCustomer, FirstName: "Rx", LastName: "Customer", Phone: "+234815" + unique, PasswordHash: "x"}
	mustCreate(t, tx, c)
	return c
}

func TestCreatePrescription_RejectsNonPharmacyMerchant(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		svc := NewCatalogService(repo.NewCatalogRepo(tx), nil)
		merchant := seedFoodMerchant(t, tx)
		customer := seedRxCustomer(t, tx)

		_, err := svc.CreatePrescription(t.Context(), customer.ID, "prescriptions/x/key", merchant.ID)
		if err == nil {
			t.Fatal("expected error creating a prescription against a non-pharmacy merchant, got nil")
		}
	})
}

func TestCreatePrescription_RejectsUnknownMerchant(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		svc := NewCatalogService(repo.NewCatalogRepo(tx), nil)
		customer := seedRxCustomer(t, tx)

		_, err := svc.CreatePrescription(t.Context(), customer.ID, "prescriptions/x/key", uuid.New())
		if err == nil {
			t.Fatal("expected error creating a prescription against a nonexistent merchant, got nil")
		}
	})
}

func TestCreatePrescription_SucceedsAgainstPharmacy(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		svc := NewCatalogService(repo.NewCatalogRepo(tx), nil)
		merchant := seedPharmacyMerchant(t, tx)
		customer := seedRxCustomer(t, tx)

		p, err := svc.CreatePrescription(t.Context(), customer.ID, "prescriptions/x/key", merchant.ID)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if p.MerchantID == nil || *p.MerchantID != merchant.ID {
			t.Fatalf("expected MerchantID %s, got %v", merchant.ID, p.MerchantID)
		}
		if p.Status != "pending" {
			t.Fatalf("expected status pending, got %s", p.Status)
		}
	})
}

func TestReviewPrescription_ApproveSetsExpiryAndStatus(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		catalogSvc := NewCatalogService(repo.NewCatalogRepo(tx), nil)
		merchant := seedPharmacyMerchant(t, tx)
		customer := seedRxCustomer(t, tx)
		reviewer := uuid.New()
		mustCreate(t, tx, &model.User{ID: reviewer, Role: model.RoleMerchant, FirstName: "Pharm", LastName: "Acist", Phone: "+234816" + uuid.NewString()[:8], PasswordHash: "x"})

		p, err := catalogSvc.CreatePrescription(t.Context(), customer.ID, "prescriptions/x/key", merchant.ID)
		if err != nil {
			t.Fatalf("create prescription: %v", err)
		}

		reviewed, err := catalogSvc.ReviewPrescription(t.Context(), reviewer, merchant.ID, p.ID, true, nil)
		if err != nil {
			t.Fatalf("review: %v", err)
		}
		if reviewed.Status != "approved" {
			t.Fatalf("expected approved, got %s", reviewed.Status)
		}
		if reviewed.ExpiresAt == nil || !reviewed.ExpiresAt.After(time.Now()) {
			t.Fatalf("expected a future ExpiresAt on approval, got %v", reviewed.ExpiresAt)
		}
	})
}

func TestReviewPrescription_SecondReviewConflicts(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		catalogSvc := NewCatalogService(repo.NewCatalogRepo(tx), nil)
		merchant := seedPharmacyMerchant(t, tx)
		customer := seedRxCustomer(t, tx)
		reviewer := uuid.New()
		mustCreate(t, tx, &model.User{ID: reviewer, Role: model.RoleMerchant, FirstName: "Pharm", LastName: "Acist", Phone: "+234817" + uuid.NewString()[:8], PasswordHash: "x"})

		p, err := catalogSvc.CreatePrescription(t.Context(), customer.ID, "prescriptions/x/key", merchant.ID)
		if err != nil {
			t.Fatalf("create prescription: %v", err)
		}
		if _, err := catalogSvc.ReviewPrescription(t.Context(), reviewer, merchant.ID, p.ID, true, nil); err != nil {
			t.Fatalf("first review: %v", err)
		}

		// This is the same conditional-UPDATE codepath a genuinely concurrent
		// second reviewer would hit — the WHERE status='pending' clause is
		// what makes exactly one review win, proven here by the second call
		// (against already-'approved') getting zero rows affected and
		// ErrPrescriptionUsed, not by re-reading and re-checking in Go.
		if _, err := catalogSvc.ReviewPrescription(t.Context(), reviewer, merchant.ID, p.ID, false, nil); err != ErrPrescriptionUsed {
			t.Fatalf("expected ErrPrescriptionUsed on second review, got %v", err)
		}
	})
}

func TestReviewPrescription_WrongMerchantForbidden(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		catalogSvc := NewCatalogService(repo.NewCatalogRepo(tx), nil)
		merchant := seedPharmacyMerchant(t, tx)
		otherMerchant := seedPharmacyMerchant(t, tx)
		customer := seedRxCustomer(t, tx)
		reviewer := uuid.New()
		mustCreate(t, tx, &model.User{ID: reviewer, Role: model.RoleMerchant, FirstName: "Pharm", LastName: "Acist", Phone: "+234818" + uuid.NewString()[:8], PasswordHash: "x"})

		p, err := catalogSvc.CreatePrescription(t.Context(), customer.ID, "prescriptions/x/key", merchant.ID)
		if err != nil {
			t.Fatalf("create prescription: %v", err)
		}

		if _, err := catalogSvc.ReviewPrescription(t.Context(), reviewer, otherMerchant.ID, p.ID, true, nil); err != ErrForbidden {
			t.Fatalf("expected ErrForbidden when a different pharmacy reviews, got %v", err)
		}
	})
}

func TestConsumePrescriptionTx_SingleUse(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		catalogSvc := NewCatalogService(repo.NewCatalogRepo(tx), nil)
		orderRepo := repo.NewOrderRepo(tx)
		merchant := seedPharmacyMerchant(t, tx)
		customer := seedRxCustomer(t, tx)
		reviewer := uuid.New()
		mustCreate(t, tx, &model.User{ID: reviewer, Role: model.RoleMerchant, FirstName: "Pharm", LastName: "Acist", Phone: "+234819" + uuid.NewString()[:8], PasswordHash: "x"})

		p, err := catalogSvc.CreatePrescription(t.Context(), customer.ID, "prescriptions/x/key", merchant.ID)
		if err != nil {
			t.Fatalf("create prescription: %v", err)
		}
		if _, err := catalogSvc.ReviewPrescription(t.Context(), reviewer, merchant.ID, p.ID, true, nil); err != nil {
			t.Fatalf("review: %v", err)
		}

		order1 := seedMinimalOrder(t, tx, customer.ID, merchant.ID)
		rows, err := orderRepo.ConsumePrescriptionTx(t.Context(), tx, p.ID, customer.ID, merchant.ID, order1.ID)
		if err != nil {
			t.Fatalf("first consume: %v", err)
		}
		if rows != 1 {
			t.Fatalf("expected 1 row consumed on first use, got %d", rows)
		}

		// Second order attempting to reuse the same (now-consumed) Rx must
		// affect zero rows — this is the actual single-use guarantee (P4).
		order2 := seedMinimalOrder(t, tx, customer.ID, merchant.ID)
		rows, err = orderRepo.ConsumePrescriptionTx(t.Context(), tx, p.ID, customer.ID, merchant.ID, order2.ID)
		if err != nil {
			t.Fatalf("second consume: %v", err)
		}
		if rows != 0 {
			t.Fatalf("expected 0 rows on reuse of a consumed prescription, got %d", rows)
		}
	})
}

func TestConsumePrescriptionTx_WrongMerchantRejected(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		catalogSvc := NewCatalogService(repo.NewCatalogRepo(tx), nil)
		orderRepo := repo.NewOrderRepo(tx)
		merchantA := seedPharmacyMerchant(t, tx)
		merchantB := seedPharmacyMerchant(t, tx)
		customer := seedRxCustomer(t, tx)
		reviewer := uuid.New()
		mustCreate(t, tx, &model.User{ID: reviewer, Role: model.RoleMerchant, FirstName: "Pharm", LastName: "Acist", Phone: "+234820" + uuid.NewString()[:8], PasswordHash: "x"})

		// Approved by pharmacy A.
		p, err := catalogSvc.CreatePrescription(t.Context(), customer.ID, "prescriptions/x/key", merchantA.ID)
		if err != nil {
			t.Fatalf("create prescription: %v", err)
		}
		if _, err := catalogSvc.ReviewPrescription(t.Context(), reviewer, merchantA.ID, p.ID, true, nil); err != nil {
			t.Fatalf("review: %v", err)
		}

		// An order against pharmacy B must not be able to consume it (P4's
		// merchant-binding half — reuse "at any pharmacy").
		rows, err := orderRepo.ConsumePrescriptionTx(t.Context(), tx, p.ID, customer.ID, merchantB.ID, uuid.New())
		if err != nil {
			t.Fatalf("consume: %v", err)
		}
		if rows != 0 {
			t.Fatalf("expected 0 rows consuming an Rx approved by a different pharmacy, got %d", rows)
		}
	})
}

func TestConsumePrescriptionTx_ExpiredRejected(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		merchant := seedPharmacyMerchant(t, tx)
		customer := seedRxCustomer(t, tx)
		orderRepo := repo.NewOrderRepo(tx)

		past := time.Now().Add(-time.Hour)
		p := &model.Prescription{
			ID: uuid.New(), CustomerID: customer.ID, MerchantID: &merchant.ID,
			R2Key: "prescriptions/x/key", Status: "approved", ExpiresAt: &past,
		}
		mustCreate(t, tx, p)

		rows, err := orderRepo.ConsumePrescriptionTx(t.Context(), tx, p.ID, customer.ID, merchant.ID, uuid.New())
		if err != nil {
			t.Fatalf("consume: %v", err)
		}
		if rows != 0 {
			t.Fatalf("expected 0 rows consuming an expired prescription, got %d", rows)
		}
	})
}

func TestConsumePrescriptionTx_PendingNotConsumable(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		catalogSvc := NewCatalogService(repo.NewCatalogRepo(tx), nil)
		orderRepo := repo.NewOrderRepo(tx)
		merchant := seedPharmacyMerchant(t, tx)
		customer := seedRxCustomer(t, tx)

		p, err := catalogSvc.CreatePrescription(t.Context(), customer.ID, "prescriptions/x/key", merchant.ID)
		if err != nil {
			t.Fatalf("create prescription: %v", err)
		}

		// Never reviewed — still 'pending'.
		rows, err := orderRepo.ConsumePrescriptionTx(t.Context(), tx, p.ID, customer.ID, merchant.ID, uuid.New())
		if err != nil {
			t.Fatalf("consume: %v", err)
		}
		if rows != 0 {
			t.Fatalf("expected 0 rows consuming a pending (unapproved) prescription, got %d", rows)
		}
	})
}
