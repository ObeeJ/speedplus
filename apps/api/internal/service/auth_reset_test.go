package service

// Tests for AuthService.ResetPassword.
// Requires DATABASE_URL — skips locally if unset, same as auth_verify_test.go.

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

func TestResetPassword_ValidOTP(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		userRepo := repo.NewUserRepo(tx)
		cfg := &config.Config{JWTSecret: "test-secret-at-least-32-bytes-long!!", JWTAccessTTLMin: 15, JWTRefreshTTLDays: 30}
		auth := NewAuthService(userRepo, cfg, nil, nil)

		phone := "+234801" + uuid.NewString()[:8]
		oldHash, _ := bcrypt.GenerateFromPassword([]byte("oldpassword"), 12)
		user := &model.User{
			ID: uuid.New(), Role: model.RoleCustomer,
			FirstName: "Reset", LastName: "Test", Phone: phone,
			PasswordHash: string(oldHash), ReferralCode: "RST" + uuid.NewString()[:8],
		}
		mustCreate(t, tx, user)

		code := "654321"
		hash, _ := bcrypt.GenerateFromPassword([]byte(code), 12)
		otp := &model.OTPCode{
			ID: uuid.New(), Phone: phone, CodeHash: string(hash),
			Purpose: "password_reset", ExpiresAt: time.Now().Add(5 * time.Minute),
		}
		mustCreate(t, tx, otp)

		if err := auth.ResetPassword(ctx, phone, code, "newpassword123"); err != nil {
			t.Fatalf("ResetPassword: %v", err)
		}

		var reloaded model.User
		if err := tx.First(&reloaded, "id = ?", user.ID).Error; err != nil {
			t.Fatalf("reload user: %v", err)
		}
		if reloaded.PasswordHash == string(oldHash) {
			t.Error("password hash unchanged after reset")
		}
		if !verifyPassword("newpassword123", reloaded.PasswordHash) {
			t.Error("new password does not verify against stored hash")
		}

		// OTP must be consumed — a second call with the same code must fail.
		if err := auth.ResetPassword(ctx, phone, code, "anotherpassword"); err == nil {
			t.Error("expected error on reused OTP, got nil")
		}
	})
}

func TestResetPassword_WrongOTPRejected(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		userRepo := repo.NewUserRepo(tx)
		cfg := &config.Config{JWTSecret: "test-secret-at-least-32-bytes-long!!", JWTAccessTTLMin: 15, JWTRefreshTTLDays: 30}
		auth := NewAuthService(userRepo, cfg, nil, nil)

		phone := "+234802" + uuid.NewString()[:8]
		user := &model.User{
			ID: uuid.New(), Role: model.RoleCustomer,
			FirstName: "Wrong", LastName: "OTP", Phone: phone,
			PasswordHash: "x", ReferralCode: "WRG" + uuid.NewString()[:8],
		}
		mustCreate(t, tx, user)

		hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), 12)
		otp := &model.OTPCode{
			ID: uuid.New(), Phone: phone, CodeHash: string(hash),
			Purpose: "password_reset", ExpiresAt: time.Now().Add(5 * time.Minute),
		}
		mustCreate(t, tx, otp)

		if err := auth.ResetPassword(ctx, phone, "000000", "newpassword123"); err == nil {
			t.Fatal("expected error for wrong OTP, got nil")
		}
	})
}

func TestResetPassword_ExpiredOTPRejected(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		ctx := context.Background()
		userRepo := repo.NewUserRepo(tx)
		cfg := &config.Config{JWTSecret: "test-secret-at-least-32-bytes-long!!", JWTAccessTTLMin: 15, JWTRefreshTTLDays: 30}
		auth := NewAuthService(userRepo, cfg, nil, nil)

		phone := "+234805" + uuid.NewString()[:8]
		user := &model.User{
			ID: uuid.New(), Role: model.RoleCustomer,
			FirstName: "Expired", LastName: "OTP", Phone: phone,
			PasswordHash: "x", ReferralCode: "EXP" + uuid.NewString()[:8],
		}
		mustCreate(t, tx, user)

		code := "111111"
		hash, _ := bcrypt.GenerateFromPassword([]byte(code), 12)
		otp := &model.OTPCode{
			ID: uuid.New(), Phone: phone, CodeHash: string(hash),
			Purpose: "password_reset", ExpiresAt: time.Now().Add(-1 * time.Minute), // already expired
		}
		mustCreate(t, tx, otp)

		if err := auth.ResetPassword(ctx, phone, code, "newpassword123"); err == nil {
			t.Fatal("expected error for expired OTP, got nil")
		}
	})
}
