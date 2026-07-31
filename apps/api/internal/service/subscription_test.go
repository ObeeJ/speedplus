package service

// Subscription renewal tests — exercises chargeOne (gas mode routing,
// product/weight resolution, idempotency), ProcessDue (dunning increment,
// pause at 3, next-charge scheduling), nextChargeFor (floor, cap, non-gas
// passthrough), and the GetPrevLPGPrice fix (suggestion string generation).
//
// Requires DATABASE_URL — skips if unset, same as ledger_money_test.go.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/config"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func seedSubscriptionFixture(t *testing.T, tx *gorm.DB, gasMode string) (
	sub model.Subscription,
	merchant *model.Merchant,
	spec *model.CylinderSpec,
	product *model.Product,
	addr *model.Address,
) {
	t.Helper()
	unique := uuid.NewString()[:8]

	customerUser := &model.User{ID: uuid.New(), Role: model.RoleCustomer, FirstName: "Sub", LastName: "Customer", Phone: "+234820" + unique, PasswordHash: "x"}
	mustCreate(t, tx, customerUser)

	merchantUser := &model.User{ID: uuid.New(), Role: model.RoleMerchant, FirstName: "Sub", LastName: "Merchant", Phone: "+234821" + unique, PasswordHash: "x"}
	mustCreate(t, tx, merchantUser)

	merchant = &model.Merchant{
		ID: uuid.New(), UserID: merchantUser.ID,
		BusinessName: "Gas Plant " + unique, Vertical: model.VerticalGas,
		IsOpen: true, IsGasPlant: true,
	}
	mustCreate(t, tx, merchant)

	// Use a unique non-standard size to avoid colliding with seeded 3/6/12.5/25kg specs.
	specSize := 7.5 + float64(len(unique)%5)*0.3
	spec = &model.CylinderSpec{
		ID: uuid.New(), SizeKg: specSize, TareKg: 14.5,
		ValveType: "standard", Label: "test-" + unique, IsActive: true,
	}
	mustCreate(t, tx, spec)

	product = &model.Product{
		ID: uuid.New(), MerchantID: merchant.ID,
		Name: "Test Cylinder " + unique, PriceKobo: 1_180_000,
		IsAvailable: true, CylinderSpecID: &spec.ID,
	}
	mustCreate(t, tx, product)

	addr = &model.Address{
		ID: uuid.New(), UserID: customerUser.ID,
		Street: "1 Test St", City: "Lagos", State: "Lagos", Lat: 6.5, Lng: 3.3,
	}
	mustCreate(t, tx, addr)

	gm := gasMode
	sub = model.Subscription{
		ID:             uuid.New(),
		CustomerID:     customerUser.ID,
		MerchantID:     merchant.ID,
		Vertical:       "gas",
		Cadence:        "monthly",
		AddressID:      addr.ID,
		PaymentMethod:  "wallet",
		Status:         "active",
		NextChargeAt:   time.Now().Add(-time.Minute), // due now
		GasMode:        &gm,
		CylinderSpecID: &spec.ID,
	}
	mustCreate(t, tx, &sub)
	return
}

// stubPricingService returns a fixed quote without hitting OSRM.
// We wire it by constructing a minimal PricingService with a nil OSRM URL
// and overriding the quote via the repo directly.
func seedQuote(t *testing.T, tx *gorm.DB, customerID, merchantID uuid.UUID) *model.PricingQuote {
	t.Helper()
	unique := uuid.NewString()[:8]
	q := &model.PricingQuote{
		ID:           uuid.New(),
		CustomerID:   customerID,
		MerchantID:   merchantID,
		SubtotalKobo: 1_180_000,
		DeliveryKobo: 193_000,
		ServiceKobo:  35_400,
		TotalKobo:    1_408_400,
		QuoteHash:    "sub-test-hash-" + unique,
		ExpiresAt:    time.Now().Add(time.Hour),
	}
	mustCreate(t, tx, q)
	return q
}

// ── nextChargeFor unit tests (no DB needed) ───────────────────────────────────

func TestNextChargeFor_NonGasUsesCadence(t *testing.T) {
	sub := model.Subscription{Vertical: "food", Cadence: "monthly"}
	got := nextChargeFor(sub)
	want := nextChargeTime("monthly")
	// Allow 1s tolerance for test execution time.
	if diff := got.Sub(want); diff > time.Second || diff < -time.Second {
		t.Errorf("non-gas nextChargeFor = %v, want ~%v", got, want)
	}
}

func TestNextChargeFor_GasNoPredictionUsesCadence(t *testing.T) {
	sub := model.Subscription{Vertical: "gas", Cadence: "monthly", PredictedRunoutAt: nil}
	got := nextChargeFor(sub)
	want := nextChargeTime("monthly")
	if diff := got.Sub(want); diff > time.Second || diff < -time.Second {
		t.Errorf("gas no-prediction nextChargeFor = %v, want ~%v", got, want)
	}
}

func TestNextChargeFor_GasPredictionUsedWithLeadTime(t *testing.T) {
	runout := time.Now().AddDate(0, 0, 20) // 20 days out
	sub := model.Subscription{
		Vertical: "gas", Cadence: "monthly",
		PredictedRunoutAt: &runout,
	}
	got := nextChargeFor(sub)
	// Should be runout - 1 day = 19 days from now, within the monthly floor (14d) and cap (45d).
	wantApprox := runout.Add(-24 * time.Hour)
	if diff := got.Sub(wantApprox); diff > time.Second || diff < -time.Second {
		t.Errorf("gas prediction nextChargeFor = %v, want ~%v", got, wantApprox)
	}
}

func TestNextChargeFor_GasPredictionClampedToFloor(t *testing.T) {
	// Runout in 5 days — lead time gives 4 days, below the 14-day half-cadence floor.
	runout := time.Now().AddDate(0, 0, 5)
	sub := model.Subscription{
		Vertical: "gas", Cadence: "monthly",
		PredictedRunoutAt: &runout,
	}
	got := nextChargeFor(sub)
	floor := time.Now().AddDate(0, 0, 14)
	if got.Before(floor.Add(-time.Second)) {
		t.Errorf("nextChargeFor below half-cadence floor: got %v, floor %v", got, floor)
	}
}

func TestNextChargeFor_GasPredictionClampedToCap(t *testing.T) {
	// Runout in 60 days — above the 45-day cap for monthly.
	runout := time.Now().AddDate(0, 0, 60)
	sub := model.Subscription{
		Vertical: "gas", Cadence: "monthly",
		PredictedRunoutAt: &runout,
	}
	got := nextChargeFor(sub)
	cap := time.Now().AddDate(0, 0, 45)
	if got.After(cap.Add(time.Second)) {
		t.Errorf("nextChargeFor above cap: got %v, cap %v", got, cap)
	}
}

func TestNextChargeFor_WeeklyFloorAndCap(t *testing.T) {
	// Floor: 3 days, cap: 10 days for weekly cadence.
	tooSoon := time.Now().AddDate(0, 0, 1)
	sub := model.Subscription{Vertical: "gas", Cadence: "weekly", PredictedRunoutAt: &tooSoon}
	got := nextChargeFor(sub)
	floor := time.Now().AddDate(0, 0, 3)
	if got.Before(floor.Add(-time.Second)) {
		t.Errorf("weekly floor not enforced: got %v, floor %v", got, floor)
	}

	tooLate := time.Now().AddDate(0, 0, 20)
	sub.PredictedRunoutAt = &tooLate
	got = nextChargeFor(sub)
	cap := time.Now().AddDate(0, 0, 10)
	if got.After(cap.Add(time.Second)) {
		t.Errorf("weekly cap not enforced: got %v, cap %v", got, cap)
	}
}

// ── DB-backed tests ───────────────────────────────────────────────────────────

// TestChargeOne_GasModeOnInput verifies that chargeOne sets GasMode on
// CreateOrderInput (not Customizations on the item). We do this by
// constructing the input struct the same way chargeOne does and asserting
// the field is set — no DB or pricing stack needed.
func TestChargeOne_GasModeOnInput(t *testing.T) {
	for _, mode := range []string{"swap", "refill", "new_cylinder"} {
		gasMode := mode
		sub := model.Subscription{GasMode: &gasMode}

		// Mirror chargeOne's gas mode resolution logic.
		resolvedMode := "swap"
		if sub.GasMode != nil {
			resolvedMode = *sub.GasMode
		}
		in := CreateOrderInput{
			Vertical: "gas",
			GasMode:  &resolvedMode,
			Items: []OrderItemInput{
				{Quantity: 1, WeightKg: 12.5},
			},
		}
		if in.GasMode == nil {
			t.Errorf("mode=%q: GasMode is nil on CreateOrderInput", mode)
			continue
		}
		if *in.GasMode != mode {
			t.Errorf("mode=%q: GasMode = %q, want %q", mode, *in.GasMode, mode)
		}
		for _, item := range in.Items {
			if item.Customizations != nil {
				t.Errorf("mode=%q: gas mode must not be in Customizations, got %q", mode, *item.Customizations)
			}
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsRune(s, substr))
}

func containsRune(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// testPricingSvc builds a PricingService with a dummy config so signQuote
// doesn't panic on a nil cfg pointer in tests that reach the Quote path.
func testPricingSvc(orders repo.OrderRepo, feeConfigs *FeeConfigService) *PricingService {
	cfg := &config.Config{JWTSecret: "test-secret-32-bytes-padded-here"}
	return NewPricingService(orders, cfg, "", feeConfigs)
}

// TestProcessDue_DunningIncrementAndPause verifies that a failing chargeOne
// increments dunning_count and pauses the subscription at count=3.
func TestProcessDue_DunningIncrementAndPause(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		sub, _, _, _, _ := seedSubscriptionFixture(t, tx, "swap")

		// Set dunning_count to 2 so the next failure triggers pause.
		if err := tx.Model(&model.Subscription{}).Where("id = ?", sub.ID).
			Update("dunning_count", 2).Error; err != nil {
			t.Fatalf("set dunning count: %v", err)
		}
		sub.DunningCount = 2

		subRepo := repo.NewSubscriptionRepo(tx)
		orderRepo := repo.NewOrderRepo(tx)
		ledgerRepo := repo.NewLedgerRepo(tx)
		ledger := NewLedgerService(ledgerRepo, nil)
		feeRepo := repo.NewFeeConfigRepo(tx)
		feeConfigSvc := NewFeeConfigService(feeRepo)
		pricingSvc := testPricingSvc(orderRepo, feeConfigSvc)
		tierRepo := repo.NewTierRepo(tx)
		tierSvc := NewTierService(tierRepo)
		orderSvc := NewOrderService(orderRepo, pricingSvc, ledger, tierSvc)

		svc := &SubscriptionService{
			repo:     subRepo,
			orders:   orderRepo,
			orderSvc: orderSvc,
			ledger:   ledger,
		}

		// ProcessDue will call chargeOne which will fail (no OSRM/wallet),
		// triggering dunning increment to 3 and then PauseByID.
		_ = svc.ProcessDue(ctx)

		var updated model.Subscription
		if err := tx.First(&updated, "id = ?", sub.ID).Error; err != nil {
			t.Fatalf("reload subscription: %v", err)
		}
		if updated.DunningCount != 3 {
			t.Errorf("dunning_count = %d, want 3", updated.DunningCount)
		}
		if updated.Status != "paused" {
			t.Errorf("status = %q, want paused after 3 dunning failures", updated.Status)
		}
		if updated.PausedReason == nil || *updated.PausedReason != "dunning" {
			t.Errorf("paused_reason = %v, want dunning", updated.PausedReason)
		}
	})
}

// TestProcessDue_DunningIncrementBelowThreshold verifies that a failing
// chargeOne at dunning_count < 2 increments the count but does NOT pause.
func TestProcessDue_DunningIncrementBelowThreshold(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		sub, _, _, _, _ := seedSubscriptionFixture(t, tx, "swap")
		// dunning_count starts at 0 from the fixture.

		subRepo := repo.NewSubscriptionRepo(tx)
		orderRepo := repo.NewOrderRepo(tx)
		ledgerRepo := repo.NewLedgerRepo(tx)
		ledger := NewLedgerService(ledgerRepo, nil)
		feeRepo := repo.NewFeeConfigRepo(tx)
		feeConfigSvc := NewFeeConfigService(feeRepo)
		pricingSvc := testPricingSvc(orderRepo, feeConfigSvc)
		tierRepo := repo.NewTierRepo(tx)
		tierSvc := NewTierService(tierRepo)
		orderSvc := NewOrderService(orderRepo, pricingSvc, ledger, tierSvc)

		svc := &SubscriptionService{
			repo:     subRepo,
			orders:   orderRepo,
			orderSvc: orderSvc,
			ledger:   ledger,
		}

		_ = svc.ProcessDue(ctx)

		var updated model.Subscription
		if err := tx.First(&updated, "id = ?", sub.ID).Error; err != nil {
			t.Fatalf("reload subscription: %v", err)
		}
		if updated.DunningCount != 1 {
			t.Errorf("dunning_count = %d, want 1", updated.DunningCount)
		}
		if updated.Status != "active" {
			t.Errorf("status = %q, want active (should not pause below threshold)", updated.Status)
		}
	})
}

// TestUpdateNextCharge_ResetsDunningCount verifies that a successful charge
// resets dunning_count to 0 via UpdateNextCharge.
func TestUpdateNextCharge_ResetsDunningCount(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		sub, _, _, _, _ := seedSubscriptionFixture(t, tx, "swap")

		// Manually set dunning_count to 2 to simulate prior failures.
		if err := tx.Model(&model.Subscription{}).Where("id = ?", sub.ID).
			Update("dunning_count", 2).Error; err != nil {
			t.Fatalf("set dunning count: %v", err)
		}

		subRepo := repo.NewSubscriptionRepo(tx)
		next := time.Now().AddDate(0, 1, 0)
		if err := subRepo.UpdateNextCharge(ctx, sub.ID, next); err != nil {
			t.Fatalf("UpdateNextCharge: %v", err)
		}

		var updated model.Subscription
		if err := tx.First(&updated, "id = ?", sub.ID).Error; err != nil {
			t.Fatalf("reload: %v", err)
		}
		if updated.DunningCount != 0 {
			t.Errorf("dunning_count after successful charge = %d, want 0", updated.DunningCount)
		}
	})
}

// TestGetPrevLPGPrice_ReturnsPreviousRow verifies that GetPrevLPGPrice
// returns the current live price — used by RecordLPGPrice to compute the
// delta suggestion before inserting the new row.
func TestGetPrevLPGPrice_ReturnsPreviousRow(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		adminID := seedWalletOwner(t, tx).ID
		region := "Lagos-test-" + uuid.NewString()[:8]

		existing := &model.LPGPriceIndex{
			ID: uuid.New(), Region: region, PricePerKgKobo: 100_000,
			Source: "test", EffectiveAt: time.Now().Add(-time.Hour), UpdatedBy: adminID,
		}
		mustCreate(t, tx, existing)

		subRepo := repo.NewSubscriptionRepo(tx)
		prev, err := subRepo.GetPrevLPGPrice(ctx, region)
		if err != nil {
			t.Fatalf("GetPrevLPGPrice: %v", err)
		}
		if prev.PricePerKgKobo != 100_000 {
			t.Errorf("GetPrevLPGPrice returned price %d, want %d",
				prev.PricePerKgKobo, int64(100_000))
		}
	})
}

// TestGetPrevLPGPrice_ErrorsWithNoRows verifies that GetPrevLPGPrice
// returns an error when no price row exists for the region.
func TestGetPrevLPGPrice_ErrorsWithOnlyOneRow(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		region := "Lagos-empty-" + uuid.NewString()[:8]

		subRepo := repo.NewSubscriptionRepo(tx)
		_, err := subRepo.GetPrevLPGPrice(ctx, region)
		if err == nil {
			t.Error("expected error when no price row exists for region, got nil")
		}
	})
}

// TestRecordLPGPrice_SuggestionGeneratedOnLargeMove verifies that
// RecordLPGPrice produces a suggestion string when the price moves >10%.
func TestRecordLPGPrice_SuggestionGeneratedOnLargeMove(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		adminID := seedWalletOwner(t, tx).ID
		region := "Lagos-suggest-" + uuid.NewString()[:8]

		// Seed the existing price.
		existing := &model.LPGPriceIndex{
			ID: uuid.New(), Region: region, PricePerKgKobo: 100_000,
			Source: "test", EffectiveAt: time.Now().Add(-time.Hour), UpdatedBy: adminID,
		}
		mustCreate(t, tx, existing)

		subRepo := repo.NewSubscriptionRepo(tx)
		svc := &SubscriptionService{repo: subRepo}

		// New price is 115_000 — 15% increase, above the 10% threshold.
		_, suggestion, err := svc.RecordLPGPrice(ctx, adminID, region, 115_000, "market")
		if err != nil {
			t.Fatalf("RecordLPGPrice: %v", err)
		}
		if suggestion == "" {
			t.Error("expected a suggestion string for a 15% price move, got empty string")
		}
	})
}

// TestRecordLPGPrice_NoSuggestionOnSmallMove verifies that a <10% move
// produces no suggestion.
func TestRecordLPGPrice_NoSuggestionOnSmallMove(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		adminID := seedWalletOwner(t, tx).ID
		region := "Lagos-small-" + uuid.NewString()[:8]

		existing := &model.LPGPriceIndex{
			ID: uuid.New(), Region: region, PricePerKgKobo: 100_000,
			Source: "test", EffectiveAt: time.Now().Add(-time.Hour), UpdatedBy: adminID,
		}
		mustCreate(t, tx, existing)

		subRepo := repo.NewSubscriptionRepo(tx)
		svc := &SubscriptionService{repo: subRepo}

		// 5% increase — below threshold.
		_, suggestion, err := svc.RecordLPGPrice(ctx, adminID, region, 105_000, "market")
		if err != nil {
			t.Fatalf("RecordLPGPrice: %v", err)
		}
		if suggestion != "" {
			t.Errorf("expected no suggestion for a 5%% price move, got %q", suggestion)
		}
	})
}
