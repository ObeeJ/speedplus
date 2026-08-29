package service

// Tests for SetUserActive (admin deactivation/reactivation).
//
// Covers:
//   - Deactivating an active user sets IsActive=false, revokes refresh tokens, writes audit log
//   - Reactivating a user sets IsActive=true, does NOT revoke tokens, writes audit log
//   - Deactivating a non-existent user returns an error without touching tokens or audit log
//
// Pure mock — no DATABASE_URL required.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

// ── mock ──────────────────────────────────────────────────────────────────────

type userActiveRepo struct {
	user              *model.User
	savedUser         *model.User
	auditLog          *model.AdminAuditLog
	revokeTokensCalled bool
}

func (r *userActiveRepo) Transaction(_ context.Context, fn func(*gorm.DB) error) error {
	return fn(nil)
}

func (r *userActiveRepo) LockUserTx(_ context.Context, _ *gorm.DB, id uuid.UUID) (*model.User, error) {
	if r.user == nil || r.user.ID != id {
		return nil, errors.New("not found")
	}
	return r.user, nil
}

func (r *userActiveRepo) SaveUserTx(_ context.Context, _ *gorm.DB, u *model.User) error {
	r.savedUser = u
	return nil
}

func (r *userActiveRepo) RevokeAllUserRefreshTokensTx(_ context.Context, _ *gorm.DB, _ uuid.UUID) error {
	r.revokeTokensCalled = true
	return nil
}

func (r *userActiveRepo) CreateAuditLogTx(_ context.Context, _ *gorm.DB, log *model.AdminAuditLog) error {
	r.auditLog = log
	return nil
}

// Satisfy remaining AdminRepo interface with panics.
func (r *userActiveRepo) ListMerchantProfiles(_ context.Context, _ string, _ *uuid.UUID, _ int) ([]model.MerchantProfile, error) {
	panic("unexpected")
}
func (r *userActiveRepo) LockMerchantProfileTx(_ context.Context, _ *gorm.DB, _ uuid.UUID) (*model.MerchantProfile, error) {
	panic("unexpected")
}
func (r *userActiveRepo) SaveMerchantProfileTx(_ context.Context, _ *gorm.DB, _ *model.MerchantProfile) error {
	panic("unexpected")
}
func (r *userActiveRepo) UpdateMerchantStatusByUserIDTx(_ context.Context, _ *gorm.DB, _ uuid.UUID, _ model.MerchantStatus) error {
	panic("unexpected")
}
func (r *userActiveRepo) ListDriverProfiles(_ context.Context, _ string, _ *uuid.UUID, _ int) ([]model.DriverProfile, error) {
	panic("unexpected")
}
func (r *userActiveRepo) LockDriverProfileTx(_ context.Context, _ *gorm.DB, _ uuid.UUID) (*model.DriverProfile, error) {
	panic("unexpected")
}
func (r *userActiveRepo) SaveDriverProfileTx(_ context.Context, _ *gorm.DB, _ *model.DriverProfile) error {
	panic("unexpected")
}
func (r *userActiveRepo) SearchOrders(_ context.Context, _, _ string, _ *uuid.UUID, _ int) ([]model.Order, error) {
	panic("unexpected")
}
func (r *userActiveRepo) FindOrderWithEvents(_ context.Context, _ uuid.UUID) (*model.Order, error) {
	panic("unexpected")
}
func (r *userActiveRepo) FindOrderTx(_ context.Context, _ *gorm.DB, _ uuid.UUID) (*model.Order, error) {
	panic("unexpected")
}
func (r *userActiveRepo) ListCancellationRules(_ context.Context) ([]model.CancellationRule, error) {
	panic("unexpected")
}
func (r *userActiveRepo) UpsertCancellationRule(_ context.Context, _ model.CancellationRule) (*model.CancellationRule, error) {
	panic("unexpected")
}
func (r *userActiveRepo) DeleteCancellationRule(_ context.Context, _ uuid.UUID) error {
	panic("unexpected")
}
func (r *userActiveRepo) ListGasMerchants(_ context.Context, _ string, _ *uuid.UUID, _ int) ([]model.Merchant, error) {
	panic("unexpected")
}
func (r *userActiveRepo) LockMerchantTx(_ context.Context, _ *gorm.DB, _ uuid.UUID) (*model.Merchant, error) {
	panic("unexpected")
}
func (r *userActiveRepo) SaveMerchantTx(_ context.Context, _ *gorm.DB, _ *model.Merchant) error {
	panic("unexpected")
}
func (r *userActiveRepo) ListZones(_ context.Context, _ string, _ *uuid.UUID, _ int) ([]model.ServiceZone, error) {
	panic("unexpected")
}
func (r *userActiveRepo) LockZoneTx(_ context.Context, _ *gorm.DB, _ uuid.UUID) (*model.ServiceZone, error) {
	panic("unexpected")
}
func (r *userActiveRepo) SaveZoneTx(_ context.Context, _ *gorm.DB, _ *model.ServiceZone) error {
	panic("unexpected")
}
func (r *userActiveRepo) GetMetrics(_ context.Context) (*repo.OperationalMetrics, error) {
	panic("unexpected")
}
func (r *userActiveRepo) ListUsers(_ context.Context, _, _ string, _ *uuid.UUID, _ int) ([]model.User, error) {
	panic("unexpected")
}
func (r *userActiveRepo) ListRuns(_ context.Context, _ string, _ *uuid.UUID, _ int) ([]model.DeliveryRun, error) {
	panic("unexpected")
}
func (r *userActiveRepo) ListSubscriptions(_ context.Context, _ string, _ *uuid.UUID, _ int) ([]model.Subscription, error) {
	panic("unexpected")
}
func (r *userActiveRepo) ListPrescriptions(_ context.Context, _ string, _ *uuid.UUID, _ int) ([]model.Prescription, error) {
	panic("unexpected")
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestSetUserActive_DeactivatesUserAndRevokesTokens(t *testing.T) {
	userID := uuid.New()
	adminID := uuid.New()
	r := &userActiveRepo{user: &model.User{ID: userID, IsActive: true}}
	svc := &AdminService{repo: r}

	if err := svc.SetUserActive(context.Background(), userID, adminID, false, "AML investigation"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.savedUser == nil || r.savedUser.IsActive {
		t.Error("user.IsActive should be false after deactivation")
	}
	if !r.revokeTokensCalled {
		t.Error("RevokeAllUserRefreshTokensTx must be called on deactivation")
	}
	if r.auditLog == nil {
		t.Fatal("audit log must be written")
	}
	if r.auditLog.Action != "user_deactivated" {
		t.Errorf("audit action = %q, want %q", r.auditLog.Action, "user_deactivated")
	}
	if r.auditLog.AdminID != adminID {
		t.Error("audit log must record the acting admin ID")
	}
	if r.auditLog.Reason != "AML investigation" {
		t.Errorf("audit reason = %q, want %q", r.auditLog.Reason, "AML investigation")
	}
}

func TestSetUserActive_ReactivatesUserWithoutRevokingTokens(t *testing.T) {
	userID := uuid.New()
	r := &userActiveRepo{user: &model.User{ID: userID, IsActive: false}}
	svc := &AdminService{repo: r}

	if err := svc.SetUserActive(context.Background(), userID, uuid.New(), true, "investigation cleared"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.savedUser == nil || !r.savedUser.IsActive {
		t.Error("user.IsActive should be true after reactivation")
	}
	if r.revokeTokensCalled {
		t.Error("RevokeAllUserRefreshTokensTx must NOT be called on reactivation")
	}
	if r.auditLog == nil || r.auditLog.Action != "user_activated" {
		t.Errorf("audit action = %q, want %q", r.auditLog.Action, "user_activated")
	}
}

func TestSetUserActive_UnknownUserReturnsError(t *testing.T) {
	r := &userActiveRepo{user: nil} // no user in store
	svc := &AdminService{repo: r}

	err := svc.SetUserActive(context.Background(), uuid.New(), uuid.New(), false, "test")
	if err == nil {
		t.Fatal("expected error for unknown user, got nil")
	}
	if r.revokeTokensCalled {
		t.Error("tokens must not be revoked when user is not found")
	}
	if r.auditLog != nil {
		t.Error("audit log must not be written when user is not found")
	}
}

func TestSetUserActive_AuditLogTimestampIsRecent(t *testing.T) {
	userID := uuid.New()
	before := time.Now()
	r := &userActiveRepo{user: &model.User{ID: userID, IsActive: true}}
	svc := &AdminService{repo: r}

	svc.SetUserActive(context.Background(), userID, uuid.New(), false, "test") //nolint:errcheck

	if r.auditLog.CreatedAt.Before(before) {
		t.Error("audit log CreatedAt must be set to current time")
	}
}
