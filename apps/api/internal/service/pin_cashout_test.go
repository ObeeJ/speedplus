package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ── PIN lockout ───────────────────────────────────────────────────────────────

// pinOnlyRepo satisfies UserRepo with only the PIN methods implemented.
// All other methods panic so any unexpected call is immediately visible.
type pinOnlyRepo struct {
	pin             *model.PIN
	incrementCalled int
	resetCalled     int
	lastLockUntil   *time.Time
}

func (r *pinOnlyRepo) FindPIN(_ context.Context, _ uuid.UUID) (*model.PIN, error) {
	if r.pin == nil {
		return nil, errors.New("not found")
	}
	return r.pin, nil
}
func (r *pinOnlyRepo) IncrementPINFailure(_ context.Context, _ uuid.UUID, lockUntil *time.Time) error {
	r.incrementCalled++
	r.lastLockUntil = lockUntil
	r.pin.FailedAttempts++
	r.pin.LockedUntil = lockUntil
	return nil
}
func (r *pinOnlyRepo) ResetPINFailures(_ context.Context, _ uuid.UUID) error {
	r.resetCalled++
	r.pin.FailedAttempts = 0
	r.pin.LockedUntil = nil
	return nil
}
func (r *pinOnlyRepo) UpsertPIN(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (r *pinOnlyRepo) PhoneByID(_ context.Context, _ uuid.UUID) (string, error) { return "", nil }

// Satisfy the rest of UserRepo with panics.
func (r *pinOnlyRepo) Create(_ context.Context, _ *model.User) error                { panic("unexpected") }
func (r *pinOnlyRepo) FindByPhone(_ context.Context, _ string) (*model.User, error) { panic("unexpected") }
func (r *pinOnlyRepo) FindByID(_ context.Context, _ uuid.UUID) (*model.User, error) { panic("unexpected") }
func (r *pinOnlyRepo) FindByUsername(_ context.Context, _ string) (*model.User, error) {
	panic("unexpected")
}
func (r *pinOnlyRepo) FindByReferralCode(_ context.Context, _ string) (*model.User, error) {
	panic("unexpected")
}
func (r *pinOnlyRepo) Update(_ context.Context, _ *model.User) error { panic("unexpected") }
func (r *pinOnlyRepo) CreateRefreshToken(_ context.Context, _ *model.RefreshToken) error {
	panic("unexpected")
}
func (r *pinOnlyRepo) FindRefreshToken(_ context.Context, _ string) (*model.RefreshToken, error) {
	panic("unexpected")
}
func (r *pinOnlyRepo) FindRefreshTokenAny(_ context.Context, _ string) (*model.RefreshToken, error) {
	panic("unexpected")
}
func (r *pinOnlyRepo) RevokeRefreshToken(_ context.Context, _ string, _ time.Time) error {
	panic("unexpected")
}
func (r *pinOnlyRepo) RevokeRefreshFamily(_ context.Context, _ uuid.UUID, _ time.Time) error {
	panic("unexpected")
}
func (r *pinOnlyRepo) CreateOTP(_ context.Context, _ *model.OTPCode) error { panic("unexpected") }
func (r *pinOnlyRepo) InvalidatePreviousOTPs(_ context.Context, _, _ string) error {
	panic("unexpected")
}
func (r *pinOnlyRepo) FindActiveOTP(_ context.Context, _, _ string) (*model.OTPCode, error) {
	panic("unexpected")
}
func (r *pinOnlyRepo) MarkOTPUsed(_ context.Context, _ uuid.UUID, _ time.Time) error {
	panic("unexpected")
}
func (r *pinOnlyRepo) CreateAddress(_ context.Context, _ *model.Address) error { panic("unexpected") }
func (r *pinOnlyRepo) ListAddresses(_ context.Context, _ uuid.UUID) ([]model.Address, error) {
	panic("unexpected")
}
func (r *pinOnlyRepo) FindAddress(_ context.Context, _ uuid.UUID) (*model.Address, error) {
	panic("unexpected")
}
func (r *pinOnlyRepo) CreateDriverProfile(_ context.Context, _ *model.DriverProfile) error {
	panic("unexpected")
}
func (r *pinOnlyRepo) FindDriverProfile(_ context.Context, _ uuid.UUID) (*model.DriverProfile, error) {
	panic("unexpected")
}
func (r *pinOnlyRepo) UpdateDriverProfile(_ context.Context, _ *model.DriverProfile) error {
	panic("unexpected")
}
func (r *pinOnlyRepo) CreateMerchantProfile(_ context.Context, _ *model.MerchantProfile) error {
	panic("unexpected")
}
func (r *pinOnlyRepo) FindMerchantProfile(_ context.Context, _ uuid.UUID) (*model.MerchantProfile, error) {
	panic("unexpected")
}
func (r *pinOnlyRepo) UpdateMerchantProfile(_ context.Context, _ *model.MerchantProfile) error {
	panic("unexpected")
}
func (r *pinOnlyRepo) FindDriverBankAccount(_ context.Context, _ uuid.UUID) (*model.DriverBankAccount, error) {
	panic("unexpected")
}
func (r *pinOnlyRepo) UpsertDriverBankAccount(_ context.Context, _ *model.DriverBankAccount) error {
	panic("unexpected")
}

func pinHash(pin string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}
	return string(h)
}

func TestPINLocksAfterFiveFailures(t *testing.T) {
	pin := &model.PIN{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		PINHash:        pinHash("9999"),
		FailedAttempts: 4, // one more wrong attempt should lock
	}
	repo := &pinOnlyRepo{pin: pin}
	svc := &AuthService{repo: repo}

	err := svc.VerifyPIN(context.Background(), pin.UserID, "0000") // wrong
	if err == nil {
		t.Fatal("expected error on wrong PIN")
	}
	if repo.incrementCalled != 1 {
		t.Fatalf("IncrementPINFailure called %d times, want 1", repo.incrementCalled)
	}
	if repo.lastLockUntil == nil {
		t.Fatal("lock must be set after 5th failure")
	}
	if !repo.lastLockUntil.After(time.Now()) {
		t.Fatal("lock time must be in the future")
	}
}

func TestPINLockedRejectsWithoutRevealingTime(t *testing.T) {
	locked := time.Now().Add(30 * time.Minute)
	pin := &model.PIN{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		PINHash:     pinHash("1234"),
		LockedUntil: &locked,
	}
	repo := &pinOnlyRepo{pin: pin}
	svc := &AuthService{repo: repo}

	err := svc.VerifyPIN(context.Background(), pin.UserID, "1234") // correct but locked
	if err == nil {
		t.Fatal("expected error when PIN is locked")
	}
	// Must not leak the exact unlock time (no ":" from time formatting)
	if strings.Contains(err.Error(), ":") {
		t.Errorf("PIN lock error must not contain time, got: %q", err.Error())
	}
	if repo.incrementCalled != 0 {
		t.Error("IncrementPINFailure must not be called when already locked")
	}
}

func TestSuccessfulPINResetsCounter(t *testing.T) {
	pin := &model.PIN{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		PINHash:        pinHash("1234"),
		FailedAttempts: 3,
	}
	repo := &pinOnlyRepo{pin: pin}
	svc := &AuthService{repo: repo}

	if err := svc.VerifyPIN(context.Background(), pin.UserID, "1234"); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if repo.resetCalled != 1 {
		t.Fatalf("ResetPINFailures called %d times, want 1", repo.resetCalled)
	}
}

// ── Cashout bank account guard ────────────────────────────────────────────────

// noBankWalletRepo returns "not found" for FindDriverBankAccountTx and panics on everything else.
type noBankWalletRepo struct{}

func (r *noBankWalletRepo) FindDriverBankAccountTx(_ context.Context, _ *gorm.DB, _ uuid.UUID) (*model.DriverBankAccount, error) {
	return nil, errors.New("not found")
}

// Satisfy WalletRepo interface with panics for unused methods.
func (r *noBankWalletRepo) FindPaymentIntentByKey(_ context.Context, _ string) (*model.PaymentIntent, error) {
	panic("unexpected")
}
func (r *noBankWalletRepo) CreatePaymentIntent(_ context.Context, _ *model.PaymentIntent) error {
	panic("unexpected")
}
func (r *noBankWalletRepo) Transaction(_ context.Context, _ func(*gorm.DB) error) error {
	panic("unexpected")
}
func (r *noBankWalletRepo) FindWebhookEventTx(_ context.Context, _ *gorm.DB, _, _ string) (*model.WebhookEvent, error) {
	panic("unexpected")
}
func (r *noBankWalletRepo) CreateWebhookEventTx(_ context.Context, _ *gorm.DB, _ *model.WebhookEvent) error {
	panic("unexpected")
}
func (r *noBankWalletRepo) SaveWebhookEventTx(_ context.Context, _ *gorm.DB, _ *model.WebhookEvent) error {
	panic("unexpected")
}
func (r *noBankWalletRepo) LockPaymentIntentByRefTx(_ context.Context, _ *gorm.DB, _ string) (*model.PaymentIntent, error) {
	panic("unexpected")
}
func (r *noBankWalletRepo) SavePaymentIntentTx(_ context.Context, _ *gorm.DB, _ *model.PaymentIntent) error {
	panic("unexpected")
}
func (r *noBankWalletRepo) FindIdempotencyKeyTx(_ context.Context, _ *gorm.DB, _ string) (*model.IdempotencyKey, error) {
	panic("unexpected")
}
func (r *noBankWalletRepo) CreateIdempotencyKeyTx(_ context.Context, _ *gorm.DB, _ *model.IdempotencyKey) error {
	panic("unexpected")
}
func (r *noBankWalletRepo) LockWalletBalanceTx(_ context.Context, _ *gorm.DB, _ uuid.UUID) (*model.WalletBalance, error) {
	panic("unexpected")
}
func (r *noBankWalletRepo) FindCashoutByKey(_ context.Context, _ string) (*model.CashoutRequest, error) {
	panic("unexpected")
}
func (r *noBankWalletRepo) CreateCashoutTx(_ context.Context, _ *gorm.DB, _ *model.CashoutRequest) error {
	panic("unexpected")
}
func (r *noBankWalletRepo) SaveCashoutTx(_ context.Context, _ *gorm.DB, _ *model.CashoutRequest) error {
	panic("unexpected")
}
func (r *noBankWalletRepo) LockCashoutTx(_ context.Context, _ *gorm.DB, _ uuid.UUID) (*model.CashoutRequest, error) {
	panic("unexpected")
}
func (r *noBankWalletRepo) FindIdempotencyCashoutTx(_ context.Context, _ *gorm.DB, _ string) (*model.CashoutRequest, error) {
	panic("unexpected")
}
func (r *noBankWalletRepo) SumUnpaidEarnings(_ context.Context, _ *gorm.DB, _ uuid.UUID) (int64, error) {
	panic("unexpected")
}
func (r *noBankWalletRepo) ListDriversWithUnpaidEarnings(_ context.Context) ([]uuid.UUID, error) {
	panic("unexpected")
}
func (r *noBankWalletRepo) FindMerchantBankAccountTx(_ context.Context, _ *gorm.DB, _ uuid.UUID) (*model.MerchantBankAccount, error) {
	panic("unexpected")
}

func TestCashoutWithoutBankAccountReturnsError(t *testing.T) {
	svc := &WalletService{repo: &noBankWalletRepo{}}
	err := svc.EWACashout(context.Background(), uuid.New(), 100_000, "key-1")
	if !errors.Is(err, ErrBankAccountRequired) {
		t.Fatalf("expected ErrBankAccountRequired, got: %v", err)
	}
}
