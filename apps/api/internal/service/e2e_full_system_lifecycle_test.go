package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/config"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestFullSystemLifecycle_EndToEnd(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()

		// Repos
		userRepo := repo.NewUserRepo(tx)
		ledgerRepo := repo.NewLedgerRepo(tx)
		orderRepo := repo.NewOrderRepo(tx)
		dispatchRepo := repo.NewDispatchRepo(tx)
		walletRepo := repo.NewWalletRepo(tx)

		// Services
		cfg := &config.Config{
			JWTSecret:         "test-secret-at-least-32-bytes-long!!",
			JWTAccessTTLMin:   15,
			JWTRefreshTTLDays: 30,
			QuoteSecret:       "devlocalquotesecretkeymustbe32char",
		}
		authSvc := NewAuthService(userRepo, cfg, nil, nil)
		ledgerSvc := NewLedgerService(ledgerRepo, nil)
		orderSvc := NewOrderService(orderRepo, nil, ledgerSvc, nil)
		walletSvc := NewWalletService(walletRepo, ledgerSvc, nil, nil, nil, userRepo)
		_ = orderSvc

		// ── STEP 1: CUSTOMER ONBOARDING ──────────────────────────────────────
		customerPhone := "+234801" + uuid.NewString()[:8]
		custInput := RegisterInput{
			FirstName:    "E2E",
			LastName:     "Customer",
			Phone:        customerPhone,
			Password:     "password123",
			Role:         model.RoleCustomer,
			ReferralCode: "",
		}
		customer, _, _, err := authSvc.Register(ctx, custInput)
		if err != nil {
			t.Fatalf("Step 1 - Register customer failed: %v", err)
		}

		// Verify Customer OTP
		otpCode := "123456"
		hash, _ := bcrypt.GenerateFromPassword([]byte(otpCode), 12)
		mustCreate(t, tx, &model.OTPCode{
			ID:        uuid.New(),
			Phone:     customerPhone,
			CodeHash:  string(hash),
			Purpose:   "phone_verification",
			ExpiresAt: time.Now().Add(5 * time.Minute),
		})
		verifiedCust, accessTok, _, err := authSvc.VerifyOTP(ctx, customerPhone, otpCode, "phone_verification")
		if err != nil {
			t.Fatalf("Step 1 - VerifyOTP failed: %v", err)
		}
		if !verifiedCust.IsVerified || accessTok == "" {
			t.Fatalf("Step 1 - Customer verified claim invalid")
		}

		// ── STEP 2: DRIVER ONBOARDING & KYC APPROVAL ─────────────────────────
		driverPhone := "+234802" + uuid.NewString()[:8]
		driverInput := RegisterInput{
			FirstName:    "E2E",
			LastName:     "Driver",
			Phone:        driverPhone,
			Password:     "password123",
			Role:         model.RoleDriver,
			VehicleType:  "motorcycle",
			VehiclePlate: "LAG-992-E2E",
		}
		driver, _, _, err := authSvc.Register(ctx, driverInput)
		if err != nil {
			t.Fatalf("Step 2 - Register driver failed: %v", err)
		}

		// Approve Driver Profile
		if err := tx.Model(&model.DriverProfile{}).
			Where("user_id = ?", driver.ID).
			Update("status", model.DriverApproved).Error; err != nil {
			t.Fatalf("Step 2 - Approve driver failed: %v", err)
		}

		// Driver goes online
		if err := dispatchRepo.SetDriverOnline(ctx, driver.ID, true); err != nil {
			t.Fatalf("Step 2 - Set driver online failed: %v", err)
		}

		// ── STEP 3: MERCHANT ONBOARDING ──────────────────────────────────────
		merchantPhone := "+234803" + uuid.NewString()[:8]
		merchInput := RegisterInput{
			FirstName: "E2E",
			LastName:  "Merchant",
			Phone:     merchantPhone,
			Password:  "password123",
			Role:      model.RoleMerchant,
		}
		merchantUser, _, _, err := authSvc.Register(ctx, merchInput)
		if err != nil {
			t.Fatalf("Step 3 - Register merchant user failed: %v", err)
		}

		merchant := &model.Merchant{
			ID:           uuid.New(),
			UserID:       merchantUser.ID,
			BusinessName: "E2E Delicacies Restaurant",
			Vertical:     model.VerticalFood,
			Status:       model.MerchantActive,
			IsOpen:       true,
			Lat:          6.5244,
			Lng:          3.3792,
		}
		mustCreate(t, tx, merchant)

		const subtotal int64 = 8_000_000 // 80,000 NGN
		const delivery int64 = 1_500_000 // 15,000 NGN
		const serviceFee int64 = 500_000 // 5,000 NGN
		const total int64 = 10_000_000   // 100,000 NGN

		product := &model.Product{
			ID:          uuid.New(),
			MerchantID:  merchant.ID,
			Name:        "Special Jollof Combo",
			PriceKobo:   subtotal,
			Category:    "Main Dishes",
			IsAvailable: true,
		}
		mustCreate(t, tx, product)

		// ── STEP 4: WALLET INBOUND FUNDING (SIMULATING MONNIFY WEBHOOK) ──────
		const fundAmountKobo int64 = 20_000_000 // 200,000 NGN
		if err := ledgerSvc.CreditWallet(ctx, tx, customer.ID, fundAmountKobo, "monnify_dva_topup", nil); err != nil {
			t.Fatalf("Step 4 - Fund customer wallet failed: %v", err)
		}

		custWallet, err := ledgerSvc.EnsureWallet(ctx, tx, customer.ID)
		if err != nil {
			t.Fatalf("Step 4 - Ensure customer wallet failed: %v", err)
		}
		custBal, err := ledgerRepo.LockBalance(ctx, tx, custWallet.ID)
		if err != nil {
			t.Fatalf("Step 4 - Lock customer balance failed: %v", err)
		}
		if custBal.BalanceKobo != fundAmountKobo {
			t.Fatalf("Step 4 - Customer balance = %d, want %d", custBal.BalanceKobo, fundAmountKobo)
		}

		// ── STEP 5: ORDER PLACEMENT & ESCROW HOLD ────────────────────────────
		address := &model.Address{
			ID:      uuid.New(),
			UserID:  customer.ID,
			Street:  "15 Marina Street",
			City:    "Lagos",
			State:   "Lagos",
			Country: "Nigeria",
			Lat:     6.5244,
			Lng:     3.3792,
		}
		mustCreate(t, tx, address)

		quote := &model.PricingQuote{
			ID:           uuid.New(),
			CustomerID:   customer.ID,
			MerchantID:   merchant.ID,
			SubtotalKobo: subtotal,
			DeliveryKobo: delivery,
			ServiceKobo:  serviceFee,
			TotalKobo:    total,
			ExpiresAt:    time.Now().Add(30 * time.Minute),
			QuoteHash:    "valid-hash",
		}
		mustCreate(t, tx, quote)

		order := &model.Order{
			ID:                uuid.New(),
			CustomerID:        customer.ID,
			MerchantID:        merchant.ID,
			QuoteID:           quote.ID,
			Vertical:          "food",
			Status:            model.OrderPending,
			SubtotalKobo:      subtotal,
			DeliveryKobo:      delivery,
			ServiceKobo:       serviceFee,
			TotalKobo:         total,
			DeliveryAddressID: address.ID,
			PaymentMethod:     "wallet",
			IdempotencyKey:    "e2e-idem-" + uuid.NewString()[:8],
		}
		mustCreate(t, tx, order)

		// Add item to order
		item := &model.OrderItem{
			ID:            uuid.New(),
			OrderID:       order.ID,
			ProductID:     product.ID,
			Name:          "Special Jollof Combo",
			Quantity:      1,
			UnitPriceKobo: subtotal,
			TotalKobo:     subtotal,
		}
		mustCreate(t, tx, item)

		// Add proof media so Settle non-gas requirement is satisfied
		proof := &model.ProofMedia{
			ID:         uuid.New(),
			OrderID:    order.ID,
			Kind:       "dropoff_photo",
			R2Key:      "test-proof-key",
			SHA256:     "d3b07384d113edec49eaa6238ad5ff00",
			CapturedBy: driver.ID,
			CapturedAt: time.Now(),
		}
		mustCreate(t, tx, proof)

		// Hold Escrow
		if err := ledgerSvc.HoldEscrow(ctx, tx, order.ID, customer.ID, total); err != nil {
			t.Fatalf("Step 5 - Hold escrow failed: %v", err)
		}

		// ── STEP 6: MERCHANT ACCEPTANCE & PREPARATION ────────────────────────
		if err := tx.Model(&model.Order{}).Where("id = ?", order.ID).
			Update("status", model.OrderPreparing).Error; err != nil {
			t.Fatalf("Step 6 - Merchant prepare order failed: %v", err)
		}

		// ── STEP 7: RIDER DISPATCH & ORDER PICKUP ────────────────────────────
		offer := &model.DeliveryOffer{
			ID:        uuid.New(),
			OrderID:   order.ID,
			DriverID:  &driver.ID,
			Status:    "accepted",
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}
		mustCreate(t, tx, offer)

		if err := dispatchRepo.AssignDriverToOrder(ctx, tx, order.ID, driver.ID); err != nil {
			t.Fatalf("Step 7 - Assign driver to order failed: %v", err)
		}

		if err := tx.Model(&model.Order{}).Where("id = ?", order.ID).
			Update("status", model.OrderInTransit).Error; err != nil {
			t.Fatalf("Step 7 - Transition to in_transit failed: %v", err)
		}

		// ── STEP 8: DELIVERY CONFIRMATION & ESCROW SETTLEMENT ────────────────
		if err := tx.Model(&model.Order{}).Where("id = ?", order.ID).
			Update("status", model.OrderDelivered).Error; err != nil {
			t.Fatalf("Step 8 - Mark delivered failed: %v", err)
		}

		// Update in-memory order struct for Settle helper
		order.DriverID = &driver.ID
		order.Status = model.OrderDelivered

		// Settle Escrow to Merchant & Driver via Settle
		paycodeID := uuid.New()
		if err := ledgerSvc.Settle(ctx, tx, order, paycodeID); err != nil {
			t.Fatalf("Step 8 - Settle escrow failed: %v", err)
		}

		// ── STEP 9: VERIFY FINAL BALANCES & DRIVER CASHOUT ───────────────────
		merchantWallet, err := ledgerSvc.EnsureWallet(ctx, tx, merchant.UserID)
		if err != nil {
			t.Fatalf("Step 9 - Ensure merchant wallet failed: %v", err)
		}
		merchBal, err := ledgerRepo.LockBalance(ctx, tx, merchantWallet.ID)
		if err != nil {
			t.Fatalf("Step 9 - Lock merchant balance failed: %v", err)
		}
		if merchBal.BalanceKobo <= 0 {
			t.Errorf("Step 9 - Merchant balance = %d, want > 0", merchBal.BalanceKobo)
		}

		driverWallet, err := ledgerSvc.EnsureWallet(ctx, tx, driver.ID)
		if err != nil {
			t.Fatalf("Step 9 - Ensure driver wallet failed: %v", err)
		}
		drvBal, err := ledgerRepo.LockBalance(ctx, tx, driverWallet.ID)
		if err != nil {
			t.Fatalf("Step 9 - Lock driver balance failed: %v", err)
		}
		if drvBal.BalanceKobo <= 0 {
			t.Errorf("Step 9 - Driver balance = %d, want > 0", drvBal.BalanceKobo)
		}

		// Seed unpaid DriverEarning record for EWA cashout validator
		earning := &model.DriverEarning{
			ID:         uuid.New(),
			DriverID:   driver.ID,
			OrderID:    order.ID,
			AmountKobo: drvBal.BalanceKobo,
			PaidOutAt:  nil,
		}
		mustCreate(t, tx, earning)

		// Seed Driver Bank Account for payout
		bankAcct := &model.DriverBankAccount{
			ID:            uuid.New(),
			DriverID:      driver.ID,
			BankCode:      "058",
			BankName:      "GTBank",
			AccountNumber: "0123456789",
			AccountName:   "E2E DRIVER",
			Provider:      "paystack",
			IsVerified:    true,
		}
		mustCreate(t, tx, bankAcct)

		// Driver EWA Cashout Test (cashout amount + fee <= earned balance)
		cashoutAmount := drvBal.BalanceKobo - 10000
		if err := walletSvc.EWACashout(ctx, driver.ID, cashoutAmount, "PAYOUT-E2E-001"); err != nil {
			t.Fatalf("Step 9 - Driver EWA cashout failed: %v", err)
		}
	})
}
