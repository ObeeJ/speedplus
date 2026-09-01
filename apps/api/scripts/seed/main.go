// scripts/seed/main.go — E2E fixture seeder.
// Drops all public tables, re-runs migrations, then inserts the minimum
// set of rows required for the customer→order→driver-accept E2E flow.
//
// Usage:
//
//	DATABASE_URL=postgres://... go run ./scripts/seed/main.go
package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/speedplus/api/internal/migrations"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Fixture credentials — read from env so CI can override; safe defaults for local dev only.
func seedPassword() string  { return getEnvOr("E2E_SEED_PASSWORD", "Test1234!") }
func customerPhone() string { return getEnvOr("E2E_CUSTOMER_PHONE", "+2349000000001") }
func driverPhone() string   { return getEnvOr("E2E_DRIVER_PHONE", "+2349000000002") }
func merchantPhone() string { return getEnvOr("E2E_MERCHANT_PHONE", "+2349000000003") }

const walletFundKobo = int64(5_000_000) // ₦50,000 in kobo

func getEnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// Safety guard — must be explicitly set to prevent accidental wipe of non-test DBs.
	if os.Getenv("ALLOW_DESTRUCTIVE_SEED") != "true" {
		slog.Error("refusing to run: set ALLOW_DESTRUCTIVE_SEED=true to confirm this is a test database")
		os.Exit(1)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://fourdat:fourdat@localhost:5433/fourdat?sslmode=disable"
	}

	ctx := context.Background()

	if err := resetAndMigrate(dsn); err != nil {
		slog.Error("reset/migrate failed", "err", err)
		os.Exit(1)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		slog.Error("gorm connect failed", "err", err)
		os.Exit(1)
	}

	if err := seed(ctx, db); err != nil {
		slog.Error("seed failed", "err", err)
		os.Exit(1)
	}

	slog.Info("seed complete")
}

// resetAndMigrate drops all tables in the public schema then re-runs migrations.
func resetAndMigrate(dsn string) error {
	// Use a dedicated connection for the destructive drop so it is fully
	// committed before migrate opens its own connection pool.
	dropDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open for drop: %w", err)
	}
	// Ping forces the connection to actually establish before we run DDL.
	if err := dropDB.Ping(); err != nil {
		dropDB.Close()
		return fmt.Errorf("ping for drop: %w", err)
	}
	for _, stmt := range []string{
		`DROP SCHEMA public CASCADE`,
		`CREATE SCHEMA public`,
		`GRANT ALL ON SCHEMA public TO PUBLIC`,
	} {
		if _, err := dropDB.Exec(stmt); err != nil {
			dropDB.Close()
			return fmt.Errorf("drop schema (%s): %w", stmt, err)
		}
	}
	// Close and wait for the connection to fully release before migrate connects.
	if err := dropDB.Close(); err != nil {
		return fmt.Errorf("close drop conn: %w", err)
	}
	slog.Info("schema dropped")

	// Fresh connection for migrate after the drop is committed.
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open for migrate: %w", err)
	}
	defer sqlDB.Close()

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}
	driver, err := migratepg.WithInstance(sqlDB, &migratepg.Config{})
	if err != nil {
		return fmt.Errorf("migration driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate instance: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}

	slog.Info("schema reset and migrations applied")
	return nil
}

func seed(ctx context.Context, db *gorm.DB) error {
	pwd := seedPassword()
	hash, err := hashPassword(pwd)
	if err != nil {
		return err
	}

	// ── Users ─────────────────────────────────────────────────────────────────
	customerID := uuid.New()
	driverID := uuid.New()
	merchantUserID := uuid.New()
	adminUserID := uuid.New() // needed as updated_by FK in fee_configs

	// Insert users one at a time — GORM batch insert with map[string]any
	// can scramble column ordering on some driver versions.
	for _, u := range []map[string]any{
		{
			"id": customerID, "role": "customer",
			"first_name": "Test", "last_name": "Customer",
			"phone": customerPhone(), "password_hash": hash,
			"referral_code": "TESTCUST", "is_verified": true, "is_active": true,
			"created_at": time.Now(), "updated_at": time.Now(),
		},
		{
			"id": driverID, "role": "driver",
			"first_name": "Test", "last_name": "Driver",
			"phone": driverPhone(), "password_hash": hash,
			"referral_code": "TESTDRVR", "is_verified": true, "is_active": true,
			"created_at": time.Now(), "updated_at": time.Now(),
		},
		{
			"id": merchantUserID, "role": "merchant",
			"first_name": "Test", "last_name": "Merchant",
			"phone": merchantPhone(), "password_hash": hash,
			"referral_code": "TESTMRCH", "is_verified": true, "is_active": true,
			"created_at": time.Now(), "updated_at": time.Now(),
		},
		{
			"id": adminUserID, "role": "admin",
			"first_name": "Test", "last_name": "Admin",
			"phone": "+2349000000004", "password_hash": hash,
			"referral_code": "TESTADMN", "is_verified": true, "is_active": true,
			"created_at": time.Now(), "updated_at": time.Now(),
		},
	} {
		if err := db.WithContext(ctx).Table("users").Create(u).Error; err != nil {
			return fmt.Errorf("user %v: %w", u["phone"], err)
		}
	}

	// ── Driver profile (approved + online) ────────────────────────────────────
	driverProfileID := uuid.New()
	if err := db.WithContext(ctx).Table("driver_profiles").Create(map[string]any{
		"id": driverProfileID, "user_id": driverID,
		"status": "approved", "vehicle_type": "motorcycle",
		"vehicle_plate": "LAG-E2E-01", "rating": 5.0,
		"total_deliveries": 0, "is_online": true,
		"created_at": time.Now(), "updated_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("driver_profile: %w", err)
	}

	// ── Driver location (Lagos Island — inside the seed zone) ─────────────────
	if err := db.WithContext(ctx).Exec(`
		INSERT INTO driver_locations (id, driver_id, location, heading, updated_at)
		VALUES (?, ?, ST_SetSRID(ST_MakePoint(3.3958, 6.4531), 4326), 0, NOW())
	`, uuid.New(), driverID).Error; err != nil {
		return fmt.Errorf("driver_location: %w", err)
	}

	// ── Service zone (Lagos Island bounding box) ───────────────────────────────
	zoneID := uuid.New()
	if err := db.WithContext(ctx).Exec(`
		INSERT INTO service_zones (id, name, boundary, active_days, window_start, window_end, launch_status, is_active, created_at, updated_at)
		VALUES (
			?, 'Lagos Island E2E Zone',
			ST_SetSRID(ST_GeomFromText('POLYGON((3.35 6.42, 3.45 6.42, 3.45 6.48, 3.35 6.48, 3.35 6.42))'), 4326),
			127, 0, 1440, 'live', true, NOW(), NOW()
		)
	`, zoneID).Error; err != nil {
		return fmt.Errorf("service_zone: %w", err)
	}

	// ── Merchant profile ───────────────────────────────────────────────────────
	merchantID := uuid.New()
	if err := db.WithContext(ctx).Table("merchants").Create(map[string]any{
		"id": merchantID, "user_id": merchantUserID,
		"business_name": "E2E Food Shop", "vertical": "food",
		"status": "active", "rating": 5.0, "is_open": true,
		"lat": 6.4531, "lng": 3.3958,
		"fill_status": "good",
		"created_at": time.Now(), "updated_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("merchant: %w", err)
	}

	// ── Product ────────────────────────────────────────────────────────────────
	productID := uuid.New()
	if err := db.WithContext(ctx).Table("products").Create(map[string]any{
		"id": productID, "merchant_id": merchantID,
		"name": "E2E Jollof Rice", "price_kobo": int64(2500_00),
		"category": "food", "is_available": true,
		"created_at": time.Now(), "updated_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("product: %w", err)
	}

	// ── Pharmacy merchant + OTC product ────────────────────────────────────────
	// Gas (migration 022) and package (migration 015) verticals already have a
	// platform merchant seeded via migration. Pharmacy has none anywhere in the
	// migration history — apps/customer/app/pharmacy/page.tsx calls
	// catalogApi.listMerchants('pharmacy') and renders an empty state when no
	// pharmacy merchant exists, so the pharmacy customer flow was previously
	// impossible to complete end-to-end. Seed one here, mirroring the food
	// merchant/product pattern above.
	pharmacyMerchantUserID := uuid.New()
	if err := db.WithContext(ctx).Table("users").Create(map[string]any{
		"id": pharmacyMerchantUserID, "role": "merchant",
		"first_name": "E2E", "last_name": "Pharmacy",
		"phone": "+2349000000005", "password_hash": hash,
		"referral_code": "TESTPHRM", "is_verified": true, "is_active": true,
		"created_at": time.Now(), "updated_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("pharmacy merchant user: %w", err)
	}

	pharmacyMerchantID := uuid.New()
	if err := db.WithContext(ctx).Table("merchants").Create(map[string]any{
		"id": pharmacyMerchantID, "user_id": pharmacyMerchantUserID,
		"business_name": "E2E Pharmacy", "vertical": "pharmacy",
		"status": "active", "rating": 5.0, "is_open": true,
		"lat": 6.4531, "lng": 3.3958,
		"created_at": time.Now(), "updated_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("pharmacy merchant: %w", err)
	}

	pharmacyProductID := uuid.New()
	if err := db.WithContext(ctx).Table("products").Create(map[string]any{
		"id": pharmacyProductID, "merchant_id": pharmacyMerchantID,
		"name": "E2E Paracetamol 500mg", "description": "Pack of 20 tablets",
		"price_kobo": int64(1500_00), "category": "otc", "is_available": true,
		"created_at": time.Now(), "updated_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("pharmacy product: %w", err)
	}

	// ── Customer address (inside the zone) ────────────────────────────────────
	addressID := uuid.New()
	if err := db.WithContext(ctx).Table("addresses").Create(map[string]any{
		"id": addressID, "user_id": customerID,
		"label": "E2E Home", "street": "1 Test Street",
		"city": "Lagos Island", "state": "Lagos", "country": "Nigeria",
		"lat": 6.4550, "lng": 3.3980,
		"is_default": true,
		"created_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("address: %w", err)
	}

	// ── Fee config (food vertical) ─────────────────────────────────────────────
	if err := db.WithContext(ctx).Table("fee_configs").Create(map[string]any{
		"id": uuid.New(), "vertical": "food",
		"base_fee_kobo": int64(5000), "per_km_kobo": int64(3000),
		"per_kg_kobo": int64(0), "per_stop_kobo": int64(0),
		"service_pct": 0.05,
		// driver_take_rate + platform_take_rate must equal 1.0 (DB constraint)
		"merchant_take_rate": 0.80,
		"driver_take_rate": 0.85, "platform_take_rate": 0.15,
		"fuel_price_ref_kobo": int64(0),
		"effective_at": time.Now().Add(-1 * time.Hour),
		"updated_by": adminUserID, "reason": "e2e seed",
		"created_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("fee_config: %w", err)
	}

	// ── Ledger accounts + wallets ──────────────────────────────────────────────
	// Customer wallet
	customerWalletAccID := uuid.New()
	// Driver earnings account
	driverEarningsAccID := uuid.New()
	// Platform revenue account
	platformRevenueAccID := uuid.New()
	// Platform escrow account
	platformEscrowAccID := uuid.New()

	// Use typed nil so GORM emits NULL rather than omitting the column.
	var noOwner *uuid.UUID
	ledgerAccounts := []map[string]any{
		{"id": customerWalletAccID, "owner_id": customerID, "type": "wallet", "currency": "NGN", "created_at": time.Now()},
		{"id": driverEarningsAccID, "owner_id": driverID, "type": "earnings", "currency": "NGN", "created_at": time.Now()},
		{"id": platformRevenueAccID, "owner_id": noOwner, "type": "revenue", "currency": "NGN", "created_at": time.Now()},
		{"id": platformEscrowAccID, "owner_id": noOwner, "type": "escrow", "currency": "NGN", "created_at": time.Now()},
	}
	if err := db.WithContext(ctx).Table("ledger_accounts").Create(&ledgerAccounts).Error; err != nil {
		return fmt.Errorf("ledger_accounts: %w", err)
	}

	// Fund customer wallet so they can place an order
	walletBalances := []map[string]any{
		{"account_id": customerWalletAccID, "balance_kobo": walletFundKobo, "updated_at": time.Now()},
		{"account_id": driverEarningsAccID, "balance_kobo": int64(0), "updated_at": time.Now()},
	}
	if err := db.WithContext(ctx).Table("wallet_balances").Create(&walletBalances).Error; err != nil {
		return fmt.Errorf("wallet_balances: %w", err)
	}

	// ── Wallet PINs (customer + driver) ────────────────────────────────────────
	// WalletService.Transfer requires PINVerifier.VerifyPIN, which does
	// repo.FindPIN(userID) and fails with "pin not set" if no `pins` row
	// exists (see internal/service/auth.go VerifyPIN). No prior seed data set
	// one, so wallet-to-wallet transfer (walletApi.transfer, used by
	// /wallet/transfer) could never succeed against this fixture set. Set the
	// same PIN "1234" for both the customer and driver so wallet-flow.spec.ts
	// can transfer between real seeded accounts.
	pinHash, err := bcrypt.GenerateFromPassword([]byte("1234"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash wallet pin: %w", err)
	}
	walletPins := []map[string]any{
		{"id": uuid.New(), "user_id": customerID, "pin_hash": string(pinHash), "created_at": time.Now(), "updated_at": time.Now()},
		{"id": uuid.New(), "user_id": driverID, "pin_hash": string(pinHash), "created_at": time.Now(), "updated_at": time.Now()},
	}
	if err := db.WithContext(ctx).Table("pins").Create(&walletPins).Error; err != nil {
		return fmt.Errorf("wallet pins: %w", err)
	}

	// ── User trust tiers ───────────────────────────────────────────────────────
	trustTiers := []map[string]any{
		{"user_id": customerID, "tier": 1, "completed_orders": 3, "fraud_flags": 0, "frozen": false, "updated_at": time.Now()},
		{"user_id": driverID, "tier": 1, "completed_orders": 0, "fraud_flags": 0, "frozen": false, "updated_at": time.Now()},
	}
	if err := db.WithContext(ctx).Table("user_trust_tiers").Create(&trustTiers).Error; err != nil {
		return fmt.Errorf("trust_tiers: %w", err)
	}

	// ── Package vertical fee config ─────────────────────────────────────────────
	// Migration 015 already seeds the platform "Fourdat Logistics" merchant at
	// the deterministic ID 00000000-0000-0000-0000-000000000002 — which is what
	// NEXT_PUBLIC_PACKAGE_MERCHANT_ID (apps/customer/.env.local) points at — so
	// this file must NOT re-insert that merchant/user (would violate the PK and
	// collide with any other seed rows on the same phone number). What's missing
	// is a fee_config row for vertical='package': without one, /package/price's
	// quote request has no pricing rule to apply and the package E2E flow can
	// never reach a price. Same gap exists for the gas vertical's hardcoded
	// merchant id (...004 in apps/customer/app/gas/price/page.tsx) but that one
	// already has a fee_config from migration 022, so only 'package' needs this.
	if err := db.WithContext(ctx).Table("fee_configs").Create(map[string]any{
		"id": uuid.New(), "vertical": "package",
		"base_fee_kobo": int64(5000), "per_km_kobo": int64(3000),
		"per_kg_kobo": int64(500), "per_stop_kobo": int64(2000),
		"service_pct": 0.05,
		"merchant_take_rate": 0.80,
		"driver_take_rate": 0.85, "platform_take_rate": 0.15,
		"fuel_price_ref_kobo": int64(0),
		"effective_at": time.Now().Add(-1 * time.Hour),
		"updated_by": adminUserID, "reason": "e2e seed",
		"created_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("package fee_config: %w", err)
	}

	// ── Pending merchant + pending driver (admin approval flow fixtures) ───────
	pendingMerchantUserID := uuid.New()
	if err := db.WithContext(ctx).Table("users").Create(map[string]any{
		"id": pendingMerchantUserID, "role": "merchant",
		"first_name": "Pending", "last_name": "Merchant",
		"phone": "+2349000000008", "password_hash": hash,
		"referral_code": "TESTPEND", "is_verified": true, "is_active": true,
		"created_at": time.Now(), "updated_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("pending merchant user: %w", err)
	}
	if err := db.WithContext(ctx).Table("merchants").Create(map[string]any{
		"id": uuid.New(), "user_id": pendingMerchantUserID,
		"business_name": "E2E Pending Merchant", "vertical": "food",
		"status": "pending", "rating": 0.0, "is_open": false,
		"lat": 6.4531, "lng": 3.3958,
		"fill_status": "good",
		"created_at": time.Now(), "updated_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("pending merchant: %w", err)
	}

	pendingDriverUserID := uuid.New()
	if err := db.WithContext(ctx).Table("users").Create(map[string]any{
		"id": pendingDriverUserID, "role": "driver",
		"first_name": "Pending", "last_name": "Driver",
		"phone": "+2349000000009", "password_hash": hash,
		"referral_code": "TESTPDRV", "is_verified": true, "is_active": true,
		"created_at": time.Now(), "updated_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("pending driver user: %w", err)
	}
	if err := db.WithContext(ctx).Table("driver_profiles").Create(map[string]any{
		"id": uuid.New(), "user_id": pendingDriverUserID,
		"status": "pending", "vehicle_type": "motorcycle",
		"vehicle_plate": "LAG-E2E-99", "rating": 0.0,
		"total_deliveries": 0, "is_online": false,
		"created_at": time.Now(), "updated_at": time.Now(),
	}).Error; err != nil {
		return fmt.Errorf("pending driver_profile: %w", err)
	}

	// ── Write fixture IDs to a temp file so tests can read them ─────────────
	fixtureContent := fmt.Sprintf(`CUSTOMER_PHONE=%s
DRIVER_PHONE=%s
SEED_PASSWORD=%s
MERCHANT_ID=%s
PRODUCT_ID=%s
ADDRESS_ID=%s
CUSTOMER_ID=%s
DRIVER_ID=%s
PHARMACY_MERCHANT_ID=%s
PHARMACY_PRODUCT_ID=%s
`, customerPhone(), driverPhone(), pwd, merchantID, productID, addressID, customerID, driverID, pharmacyMerchantID, pharmacyProductID)

	fixturePath := filepath.Join(os.TempDir(), "fourdat-e2e-fixtures.env")
	if err := os.WriteFile(fixturePath, []byte(fixtureContent), 0600); err != nil {
		slog.Warn("could not write fixtures file", "path", fixturePath, "err", err)
	} else {
		slog.Info("fixtures written", "path", fixturePath)
	}

	slog.Info("seeded",
		"customer_id", customerID,
		"driver_id", driverID,
		"merchant_id", merchantID,
		"product_id", productID,
		"address_id", addressID,
	)
	return nil
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=1,p=4$%s$%s",
		hex.EncodeToString(salt),
		hex.EncodeToString(hash),
	), nil
}
