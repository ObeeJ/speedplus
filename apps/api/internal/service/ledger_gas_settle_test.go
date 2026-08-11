package service

// Gas settlement tests — these call the real LedgerService.Settle against a
// real Postgres transaction (see testDB/withTx in ledger_money_test.go),
// unlike the removed ledger_gas_test.go which reimplemented Settle's formula
// in the test file and asserted it against itself. These would fail if Settle
// were deleted; the old ones would not.
//
// Requires DATABASE_URL — skips locally if unset, same as ledger_money_test.go.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

// seedGasOrder mirrors seedOrder but builds a gas-vertical order with a
// single cylinder item of the given ordered weight, in_transit and with a
// driver assigned — the state Settle expects to be called from.
func seedGasOrder(t *testing.T, tx *gorm.DB, orderedKg float64, subtotalKobo, deliveryKobo, serviceKobo int64, gasMode string) *model.Order {
	t.Helper()
	unique := uuid.NewString()[:8]
	total := subtotalKobo + deliveryKobo + serviceKobo

	customer := &model.User{ID: uuid.New(), Role: model.RoleCustomer, FirstName: "Gas", LastName: "Customer", Phone: "+234810" + unique, PasswordHash: "x"}
	mustCreate(t, tx, customer)

	driverUser := &model.User{ID: uuid.New(), Role: model.RoleDriver, FirstName: "Gas", LastName: "Driver", Phone: "+234811" + unique, PasswordHash: "x"}
	mustCreate(t, tx, driverUser)

	merchantUser := &model.User{ID: uuid.New(), Role: model.RoleMerchant, FirstName: "Gas", LastName: "Merchant", Phone: "+234812" + unique, PasswordHash: "x"}
	mustCreate(t, tx, merchantUser)

	merchant := &model.Merchant{ID: uuid.New(), UserID: merchantUser.ID, BusinessName: "Test Gas Plant " + unique, Vertical: model.VerticalGas}
	mustCreate(t, tx, merchant)

	addr := &model.Address{ID: uuid.New(), UserID: customer.ID, Street: "1 Test St", City: "Lagos", State: "Lagos", Lat: 6.5, Lng: 3.3}
	mustCreate(t, tx, addr)

	quote := &model.PricingQuote{
		ID: uuid.New(), CustomerID: customer.ID, MerchantID: merchant.ID,
		SubtotalKobo: subtotalKobo, DeliveryKobo: deliveryKobo, ServiceKobo: serviceKobo, TotalKobo: total,
		QuoteHash: "test-gas-hash-" + unique, ExpiresAt: time.Now().Add(time.Hour),
	}
	mustCreate(t, tx, quote)

	order := &model.Order{
		ID: uuid.New(), CustomerID: customer.ID, MerchantID: merchant.ID, DriverID: &driverUser.ID,
		QuoteID: quote.ID, Vertical: "gas", Status: model.OrderInTransit, GasMode: &gasMode,
		SubtotalKobo: subtotalKobo, DeliveryKobo: deliveryKobo, ServiceKobo: serviceKobo, TotalKobo: total,
		DeliveryAddressID: addr.ID, IdempotencyKey: "test-gas-order-" + unique,
	}
	mustCreate(t, tx, order)

	product := &model.Product{
		ID: uuid.New(), MerchantID: merchant.ID, Name: "Test Cylinder",
		PriceKobo: subtotalKobo, IsAvailable: true,
	}
	mustCreate(t, tx, product)

	item := &model.OrderItem{
		ID: uuid.New(), OrderID: order.ID, ProductID: product.ID, Name: "Test Cylinder",
		Quantity: 1, UnitPriceKobo: subtotalKobo, TotalKobo: subtotalKobo, WeightKg: orderedKg,
	}
	mustCreate(t, tx, item)
	order.Items = []model.OrderItem{*item}

	return order
}

func seedWeightPhoto(t *testing.T, tx *gorm.DB, order *model.Order, measuredKg float64) {
	t.Helper()
	proof := &model.ProofMedia{
		ID: uuid.New(), OrderID: order.ID, Kind: "weight_photo",
		R2Key: "test/weight/" + uuid.NewString(), SHA256: "deadbeef",
		MeasuredKg: &measuredKg, CapturedAt: time.Now(), CapturedBy: *order.DriverID,
	}
	mustCreate(t, tx, proof)
}

func setupGasLedger(t *testing.T, tx *gorm.DB) (*LedgerService, repo.LedgerRepo) {
	t.Helper()
	ledgerRepo := repo.NewLedgerRepo(tx)
	return NewLedgerService(ledgerRepo, nil), ledgerRepo
}

// TestSettleGasOrder_BlocksWithoutWeightPhoto is the regression test for
// HIGH-1: every gas order must have a weight_photo row before Settle
// releases escrow, regardless of which of the three confirmation paths
// (QR, code, card) called it. Settle itself must refuse, not just one path.
func TestSettleGasOrder_BlocksWithoutWeightPhoto(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		ledger, ledgerRepo := setupGasLedger(t, tx)

		const subtotal, delivery, serviceFee = int64(1_180_000), int64(193_000), int64(35_400)
		order := seedGasOrder(t, tx, 12.5, subtotal, delivery, serviceFee, "swap")
		total := subtotal + delivery + serviceFee

		if err := ledger.CreditWallet(ctx, tx, order.CustomerID, total, "test_seed", nil); err != nil {
			t.Fatalf("seed customer wallet: %v", err)
		}
		if err := ledger.HoldEscrow(ctx, tx, order.ID, order.CustomerID, total); err != nil {
			t.Fatalf("hold escrow: %v", err)
		}

		// No weight_photo seeded — Settle must refuse.
		err := ledger.Settle(ctx, tx, order, uuid.New())
		if err == nil {
			t.Fatal("expected Settle to reject a gas order with no weight photo, got nil error")
		}

		// Escrow must remain fully intact — nothing should have moved.
		escrowAcct, err := ledger.platformAccount(ctx, tx, model.AccountEscrow)
		if err != nil {
			t.Fatalf("get escrow account: %v", err)
		}
		var escrowBal model.WalletBalance
		if err := tx.Where("account_id = ?", escrowAcct.ID).First(&escrowBal).Error; err != nil {
			t.Fatalf("get escrow balance: %v", err)
		}
		if escrowBal.BalanceKobo != total {
			t.Errorf("escrow balance after blocked settle = %d, want %d (untouched)", escrowBal.BalanceKobo, total)
		}
		_ = ledgerRepo
	})
}

// TestSettleGasOrder_ShortfallRefundsCustomerFromMerchantShare is the
// regression test for the shortfall accounting itself: a measured weight
// beyond tolerance must reduce the merchant's payout and credit the
// customer, with the platform's take unaffected and the journal balanced —
// asserted against real posted ledger entries and balances, not a
// hand-copied formula.
func TestSettleGasOrder_ShortfallRefundsCustomerFromMerchantShare(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		ledger, ledgerRepo := setupGasLedger(t, tx)

		const subtotal, delivery, serviceFee = int64(1_180_000), int64(193_000), int64(35_400)
		const orderedKg, measuredKg = 12.5, 10.5 // 2kg short, well beyond 2% tolerance
		order := seedGasOrder(t, tx, orderedKg, subtotal, delivery, serviceFee, "swap")
		total := subtotal + delivery + serviceFee
		seedWeightPhoto(t, tx, order, measuredKg)

		if err := ledger.CreditWallet(ctx, tx, order.CustomerID, total, "test_seed", nil); err != nil {
			t.Fatalf("seed customer wallet: %v", err)
		}
		if err := ledger.HoldEscrow(ctx, tx, order.ID, order.CustomerID, total); err != nil {
			t.Fatalf("hold escrow: %v", err)
		}
		custBalBeforeSettle, err := ledgerRepo.GetBalance(ctx, order.CustomerID)
		if err != nil {
			t.Fatalf("get customer balance: %v", err)
		}

		if err := ledger.Settle(ctx, tx, order, uuid.New()); err != nil {
			t.Fatalf("settle: %v", err)
		}

		assertAllJournalsBalanced(t, tx)

		wantRefund := int64(2.0 * (float64(subtotal) / orderedKg))
		custBalAfterSettle, err := ledgerRepo.GetBalance(ctx, order.CustomerID)
		if err != nil {
			t.Fatalf("get customer balance: %v", err)
		}
		if custBalAfterSettle != custBalBeforeSettle+wantRefund {
			t.Errorf("customer balance after settle = %d, want %d (before %d + refund %d)",
				custBalAfterSettle, custBalBeforeSettle+wantRefund, custBalBeforeSettle, wantRefund)
		}

		var merchantRow model.Merchant
		if err := tx.Select("user_id").First(&merchantRow, "id = ?", order.MerchantID).Error; err != nil {
			t.Fatalf("resolve merchant user: %v", err)
		}
		merchantBal, err := ledgerRepo.GetBalance(ctx, merchantRow.UserID)
		if err != nil {
			t.Fatalf("get merchant balance: %v", err)
		}
		wantMerchantBal := int64(float64(subtotal)*0.92) - wantRefund
		if merchantBal != wantMerchantBal {
			t.Errorf("merchant balance after shortfall settle = %d, want %d", merchantBal, wantMerchantBal)
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
			t.Errorf("escrow balance after settle = %d, want 0", escrowBal.BalanceKobo)
		}
	})
}

// TestSettleGasOrder_NoRefundWithinTolerance confirms a measured weight
// inside the 2% band settles at full merchant payout — no shortfall entries,
// no customer credit.
func TestSettleGasOrder_NoRefundWithinTolerance(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		ledger, ledgerRepo := setupGasLedger(t, tx)

		const subtotal, delivery, serviceFee = int64(1_180_000), int64(193_000), int64(35_400)
		const orderedKg, measuredKg = 12.5, 12.3 // 0.2kg short — within 2% tolerance
		order := seedGasOrder(t, tx, orderedKg, subtotal, delivery, serviceFee, "swap")
		total := subtotal + delivery + serviceFee
		seedWeightPhoto(t, tx, order, measuredKg)

		if err := ledger.CreditWallet(ctx, tx, order.CustomerID, total, "test_seed", nil); err != nil {
			t.Fatalf("seed customer wallet: %v", err)
		}
		if err := ledger.HoldEscrow(ctx, tx, order.ID, order.CustomerID, total); err != nil {
			t.Fatalf("hold escrow: %v", err)
		}

		if err := ledger.Settle(ctx, tx, order, uuid.New()); err != nil {
			t.Fatalf("settle: %v", err)
		}
		assertAllJournalsBalanced(t, tx)

		var merchantRow model.Merchant
		if err := tx.Select("user_id").First(&merchantRow, "id = ?", order.MerchantID).Error; err != nil {
			t.Fatalf("resolve merchant user: %v", err)
		}
		merchantBal, err := ledgerRepo.GetBalance(ctx, merchantRow.UserID)
		if err != nil {
			t.Fatalf("get merchant balance: %v", err)
		}
		wantMerchantBal := int64(float64(subtotal) * 0.92)
		if merchantBal != wantMerchantBal {
			t.Errorf("merchant balance within tolerance = %d, want %d (full payout, no shortfall)", merchantBal, wantMerchantBal)
		}
	})
}

// TestSettleGasOrder_RejectsImplausibleMeasuredWeight is the regression test
// for MED-2: a measured_kg outside the 50%–150% plausibility band must cause
// Settle to return an error before any money moves, so a driver typo or a
// tampered weight photo cannot zero a merchant's entire payout.
func TestSettleGasOrder_RejectsImplausibleMeasuredWeight(t *testing.T) {
	cases := []struct {
		name       string
		orderedKg  float64
		measuredKg float64
	}{
		{"too low — 10% of ordered", 12.5, 1.25},
		{"too high — 200% of ordered", 12.5, 25.0},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gdb := testDB(t)
			withTx(t, gdb, func(tx *gorm.DB) {
				ctx := context.Background()
				ledger, _ := setupGasLedger(t, tx)

				const subtotal, delivery, serviceFee = int64(1_180_000), int64(193_000), int64(35_400)
				order := seedGasOrder(t, tx, tc.orderedKg, subtotal, delivery, serviceFee, "swap")
				total := subtotal + delivery + serviceFee
				seedWeightPhoto(t, tx, order, tc.measuredKg)

				if err := ledger.CreditWallet(ctx, tx, order.CustomerID, total, "test_seed", nil); err != nil {
					t.Fatalf("seed customer wallet: %v", err)
				}
				if err := ledger.HoldEscrow(ctx, tx, order.ID, order.CustomerID, total); err != nil {
					t.Fatalf("hold escrow: %v", err)
				}

				err := ledger.Settle(ctx, tx, order, uuid.New())
				if err == nil {
					t.Fatalf("expected Settle to reject implausible measured weight %.2f kg for ordered %.2f kg, got nil",
						tc.measuredKg, tc.orderedKg)
				}

				// Escrow must be untouched.
				escrowAcct, _ := ledger.platformAccount(ctx, tx, model.AccountEscrow)
				var escrowBal model.WalletBalance
				if err := tx.Where("account_id = ?", escrowAcct.ID).First(&escrowBal).Error; err != nil {
					t.Fatalf("get escrow balance: %v", err)
				}
				if escrowBal.BalanceKobo != total {
					t.Errorf("escrow balance after rejected settle = %d, want %d (untouched)", escrowBal.BalanceKobo, total)
				}
			})
		})
	}
}
