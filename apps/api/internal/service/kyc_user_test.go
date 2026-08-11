package service

// Tests for GetUserKYC (admin AML trace + user self-status).
//
// Covers:
//   - Returns all checks for a user ordered by the repo (most recent first)
//   - Returns empty slice (not error) when user has no checks
//   - Repo error is propagated
//
// Pure mock — no DATABASE_URL required.

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

// ── mock ──────────────────────────────────────────────────────────────────────

type kycUserRepo struct {
	checks []model.KYCCheck
	err    error
	// record which userID was queried
	queriedUserID uuid.UUID
}

func (r *kycUserRepo) ListKYCChecksByUser(_ context.Context, userID uuid.UUID) ([]model.KYCCheck, error) {
	r.queriedUserID = userID
	return r.checks, r.err
}

// Satisfy remaining DispatchRepo interface with panics.
func (r *kycUserRepo) UpsertDriverLocation(_ context.Context, _ uuid.UUID, _, _, _ float64) error {
	panic("unexpected")
}
func (r *kycUserRepo) CreateOffer(_ context.Context, _ *model.DeliveryOffer) error {
	panic("unexpected")
}
func (r *kycUserRepo) AtomicAcceptOffer(_ context.Context, _, _ uuid.UUID) (bool, error) {
	panic("unexpected")
}
func (r *kycUserRepo) ExpireStaleOffers(_ context.Context) error { panic("unexpected") }
func (r *kycUserRepo) AssignDriverToOrder(_ context.Context, _ *gorm.DB, _, _ uuid.UUID) error {
	panic("unexpected")
}
func (r *kycUserRepo) NearbyDrivers(_ context.Context, _, _, _ float64, _ model.VehicleType, _ bool, _ int) ([]repo.NearbyDriver, error) {
	panic("unexpected")
}
func (r *kycUserRepo) CreateKYCCheck(_ context.Context, _ *model.KYCCheck) error {
	panic("unexpected")
}
func (r *kycUserRepo) FindKYCCheck(_ context.Context, _ uuid.UUID) (*model.KYCCheck, error) {
	panic("unexpected")
}
func (r *kycUserRepo) SaveKYCCheck(_ context.Context, _ *model.KYCCheck) error { panic("unexpected") }
func (r *kycUserRepo) ListPendingKYCChecks(_ context.Context, _, _ int) ([]model.KYCCheck, error) {
	panic("unexpected")
}
func (r *kycUserRepo) CreateKYCDocument(_ context.Context, _ *model.KYCDocument) error {
	panic("unexpected")
}
func (r *kycUserRepo) Transaction(_ context.Context, _ func(*gorm.DB) error) error {
	panic("unexpected")
}
func (r *kycUserRepo) FindKYCCheckTx(_ context.Context, _ *gorm.DB, _ uuid.UUID) (*model.KYCCheck, error) {
	panic("unexpected")
}
func (r *kycUserRepo) SaveKYCCheckTx(_ context.Context, _ *gorm.DB, _ *model.KYCCheck) error {
	panic("unexpected")
}
func (r *kycUserRepo) FindUserTx(_ context.Context, _ *gorm.DB, _ uuid.UUID) (*model.User, error) {
	panic("unexpected")
}
func (r *kycUserRepo) FindDriverProfileTx(_ context.Context, _ *gorm.DB, _ uuid.UUID) (*model.DriverProfile, error) {
	panic("unexpected")
}
func (r *kycUserRepo) SaveDriverProfileTx(_ context.Context, _ *gorm.DB, _ *model.DriverProfile) error {
	panic("unexpected")
}
func (r *kycUserRepo) FindMerchantProfileTx(_ context.Context, _ *gorm.DB, _ uuid.UUID) (*model.MerchantProfile, error) {
	panic("unexpected")
}
func (r *kycUserRepo) SaveMerchantProfileTx(_ context.Context, _ *gorm.DB, _ *model.MerchantProfile) error {
	panic("unexpected")
}
func (r *kycUserRepo) AcceptOfferTx(_ context.Context, _ *gorm.DB, _, _ uuid.UUID) (int64, error) {
	panic("unexpected")
}
func (r *kycUserRepo) FindOfferTx(_ context.Context, _ *gorm.DB, _ uuid.UUID) (*model.DeliveryOffer, error) {
	panic("unexpected")
}
func (r *kycUserRepo) SetDriverID(_ context.Context, _ *gorm.DB, _, _ uuid.UUID) error {
	panic("unexpected")
}
func (r *kycUserRepo) UpdateOfferStatus(_ context.Context, _, _ uuid.UUID, _ string) error {
	panic("unexpected")
}
func (r *kycUserRepo) LockOrderTx(_ context.Context, _ *gorm.DB, _ uuid.UUID) (*model.Order, error) {
	panic("unexpected")
}
func (r *kycUserRepo) SaveOrderTx(_ context.Context, _ *gorm.DB, _ *model.Order) error {
	panic("unexpected")
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestGetUserKYC_ReturnsAllChecksForUser(t *testing.T) {
	userID := uuid.New()
	checks := []model.KYCCheck{
		{ID: uuid.New(), UserID: userID, DocType: model.DocNIN, Status: model.KYCApproved},
		{ID: uuid.New(), UserID: userID, DocType: model.DocBVN, Status: model.KYCRejected},
		{ID: uuid.New(), UserID: userID, DocType: model.DocLiveness, Status: model.KYCPending},
	}
	r := &kycUserRepo{checks: checks}
	svc := &KYCService{repo: r}

	got, err := svc.GetUserKYC(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("got %d checks, want 3", len(got))
	}
	if r.queriedUserID != userID {
		t.Error("repo was queried with wrong userID")
	}
}

func TestGetUserKYC_EmptySliceWhenNoChecks(t *testing.T) {
	r := &kycUserRepo{checks: []model.KYCCheck{}}
	svc := &KYCService{repo: r}

	got, err := svc.GetUserKYC(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d checks, want 0", len(got))
	}
}

func TestGetUserKYC_PropagatesRepoError(t *testing.T) {
	repoErr := errors.New("db unavailable")
	r := &kycUserRepo{err: repoErr}
	svc := &KYCService{repo: r}

	_, err := svc.GetUserKYC(context.Background(), uuid.New())
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error to propagate, got: %v", err)
	}
}

func TestGetUserKYC_BusinessRegCheckIncluded(t *testing.T) {
	userID := uuid.New()
	r := &kycUserRepo{checks: []model.KYCCheck{
		{ID: uuid.New(), UserID: userID, DocType: model.DocBusinessReg, Status: model.KYCApproved},
	}}
	svc := &KYCService{repo: r}

	got, err := svc.GetUserKYC(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].DocType != model.DocBusinessReg {
		t.Errorf("docType = %q, want %q", got[0].DocType, model.DocBusinessReg)
	}
}
