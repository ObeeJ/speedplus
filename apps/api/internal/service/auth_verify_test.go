package service

// Regression test for the production blocker found in the prod-readiness
// audit: VerifyOTP validated the code but never set users.is_verified, and
// RequireVerified (middleware/auth.go) gates order creation and wallet
// funding on that flag — so no user could ever place an order. This proves
// the fix end to end against a real Postgres: register → request OTP →
// verify → IsVerified is true and fresh tokens are returned.
//
// Requires DATABASE_URL — skips locally if unset, same as ledger_money_test.go.

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

func TestVerifyOTP_MarksUserVerifiedAndIssuesFreshTokens(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		userRepo := repo.NewUserRepo(tx)
		cfg := &config.Config{JWTSecret: "test-secret-at-least-32-bytes-long!!", JWTAccessTTLMin: 15, JWTRefreshTTLDays: 30}
		auth := NewAuthService(userRepo, cfg, nil, nil)

		phone := "+234803" + uuid.NewString()[:8]
		user := &model.User{
			ID: uuid.New(), Role: model.RoleCustomer,
			FirstName: "Verify", LastName: "Test", Phone: phone,
			PasswordHash: "x", ReferralCode: "TEST" + uuid.NewString()[:8],
			IsVerified: false,
		}
		mustCreate(t, tx, user)

		code := "123456"
		hash, err := bcrypt.GenerateFromPassword([]byte(code), 12)
		if err != nil {
			t.Fatalf("hash otp: %v", err)
		}
		otp := &model.OTPCode{
			ID: uuid.New(), Phone: phone, CodeHash: string(hash),
			Purpose: "phone_verification", ExpiresAt: time.Now().Add(5 * time.Minute),
		}
		mustCreate(t, tx, otp)

		_, access, refresh, err := auth.VerifyOTP(ctx, phone, code, "phone_verification")
		if err != nil {
			t.Fatalf("VerifyOTP: %v", err)
		}
		if access == "" || refresh == "" {
			t.Errorf("VerifyOTP returned empty tokens: access=%q refresh=%q", access, refresh)
		}

		var reloaded model.User
		if err := tx.First(&reloaded, "id = ?", user.ID).Error; err != nil {
			t.Fatalf("reload user: %v", err)
		}
		if !reloaded.IsVerified {
			t.Error("user.IsVerified = false after VerifyOTP, want true")
		}
	})
}

func TestVerifyOTP_WrongCodeRejected(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		userRepo := repo.NewUserRepo(tx)
		cfg := &config.Config{JWTSecret: "test-secret-at-least-32-bytes-long!!", JWTAccessTTLMin: 15, JWTRefreshTTLDays: 30}
		auth := NewAuthService(userRepo, cfg, nil, nil)

		phone := "+234804" + uuid.NewString()[:8]
		user := &model.User{
			ID: uuid.New(), Role: model.RoleCustomer,
			FirstName: "Wrong", LastName: "Code", Phone: phone,
			PasswordHash: "x", ReferralCode: "TEST" + uuid.NewString()[:8],
		}
		mustCreate(t, tx, user)

		hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), 12)
		otp := &model.OTPCode{
			ID: uuid.New(), Phone: phone, CodeHash: string(hash),
			Purpose: "phone_verification", ExpiresAt: time.Now().Add(5 * time.Minute),
		}
		mustCreate(t, tx, otp)

		_, _, _, err := auth.VerifyOTP(ctx, phone, "000000", "phone_verification")
		if err == nil {
			t.Fatal("expected error for wrong OTP code, got nil")
		}

		var reloaded model.User
		tx.First(&reloaded, "id = ?", user.ID)
		if reloaded.IsVerified {
			t.Error("user.IsVerified = true after a WRONG code, want false")
		}
	})
}
