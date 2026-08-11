package service

// Tests for the WhatsApp OTP routing added in the notification audit:
//   - users with an email get OTP via email (WhatsApp not called)
//   - phone-only users (no email) get OTP via WhatsApp
//   - unknown phone (FindByPhone error) sends nothing and still returns the code

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/config"
	"github.com/speedplus/api/internal/model"
)

// ── stubs ─────────────────────────────────────────────────────────────────────

type stubEmailSender struct {
	mu   sync.Mutex
	otps []string // codes received
}

func (s *stubEmailSender) SendWelcome(_ context.Context, _, _ string)                    {}
func (s *stubEmailSender) SendOTP(_ context.Context, _, _, code, _ string) {
	s.mu.Lock()
	s.otps = append(s.otps, code)
	s.mu.Unlock()
}

type stubWA struct {
	mu   sync.Mutex
	otps []string
}

func (w *stubWA) OrderConfirmed(_, _, _, _ string)      {}
func (w *stubWA) RiderAssigned(_, _, _ string)          {}
func (w *stubWA) DeliveryCode(_, _ string)              {}
func (w *stubWA) OrderDelivered(_, _ string)            {}
func (w *stubWA) OrderCancelled(_, _, _ string)         {}
func (w *stubWA) PrescriptionReady(_, _ string)         {}
func (w *stubWA) SendOTP(_, code, _ string) {
	w.mu.Lock()
	w.otps = append(w.otps, code)
	w.mu.Unlock()
}

// stubUserRepo satisfies repo.UserRepo minimally for RequestOTP.
type stubUserRepo struct {
	user *model.User // nil → FindByPhone returns error
	otps []*model.OTPCode
}

func (r *stubUserRepo) FindByPhone(_ context.Context, _ string) (*model.User, error) {
	if r.user == nil {
		return nil, ErrUserNotFound
	}
	return r.user, nil
}
func (r *stubUserRepo) CreateOTP(_ context.Context, o *model.OTPCode) error {
	r.otps = append(r.otps, o)
	return nil
}
func (r *stubUserRepo) InvalidatePreviousOTPs(_ context.Context, _, _ string) error { return nil }
func (r *stubUserRepo) Create(_ context.Context, _ *model.User) error               { panic("unexpected") }
func (r *stubUserRepo) FindByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	panic("unexpected")
}
func (r *stubUserRepo) FindByUsername(_ context.Context, _ string) (*model.User, error) {
	panic("unexpected")
}
func (r *stubUserRepo) FindByReferralCode(_ context.Context, _ string) (*model.User, error) {
	panic("unexpected")
}
func (r *stubUserRepo) Update(_ context.Context, _ *model.User) error { panic("unexpected") }
func (r *stubUserRepo) CreateRefreshToken(_ context.Context, _ *model.RefreshToken) error {
	panic("unexpected")
}
func (r *stubUserRepo) FindRefreshToken(_ context.Context, _ string) (*model.RefreshToken, error) {
	panic("unexpected")
}
func (r *stubUserRepo) FindRefreshTokenAny(_ context.Context, _ string) (*model.RefreshToken, error) {
	panic("unexpected")
}
func (r *stubUserRepo) RevokeRefreshToken(_ context.Context, _ string, _ time.Time) error {
	panic("unexpected")
}
func (r *stubUserRepo) RevokeRefreshFamily(_ context.Context, _ uuid.UUID, _ time.Time) error {
	panic("unexpected")
}
func (r *stubUserRepo) FindActiveOTP(_ context.Context, _, _ string) (*model.OTPCode, error) {
	panic("unexpected")
}
func (r *stubUserRepo) MarkOTPUsed(_ context.Context, _ uuid.UUID, _ time.Time) error {
	panic("unexpected")
}
func (r *stubUserRepo) UpsertPIN(_ context.Context, _ uuid.UUID, _ string) error { panic("unexpected") }
func (r *stubUserRepo) FindPIN(_ context.Context, _ uuid.UUID) (*model.PIN, error) {
	panic("unexpected")
}
func (r *stubUserRepo) IncrementPINFailure(_ context.Context, _ uuid.UUID, _ *time.Time) error {
	panic("unexpected")
}
func (r *stubUserRepo) ResetPINFailures(_ context.Context, _ uuid.UUID) error { panic("unexpected") }
func (r *stubUserRepo) FindDriverBankAccount(_ context.Context, _ uuid.UUID) (*model.DriverBankAccount, error) {
	panic("unexpected")
}
func (r *stubUserRepo) UpsertDriverBankAccount(_ context.Context, _ *model.DriverBankAccount) error {
	panic("unexpected")
}
func (r *stubUserRepo) CreateAddress(_ context.Context, _ *model.Address) error { panic("unexpected") }
func (r *stubUserRepo) ListAddresses(_ context.Context, _ uuid.UUID) ([]model.Address, error) {
	panic("unexpected")
}
func (r *stubUserRepo) FindAddress(_ context.Context, _ uuid.UUID) (*model.Address, error) {
	panic("unexpected")
}
func (r *stubUserRepo) CreateDriverProfile(_ context.Context, _ *model.DriverProfile) error {
	panic("unexpected")
}
func (r *stubUserRepo) FindDriverProfile(_ context.Context, _ uuid.UUID) (*model.DriverProfile, error) {
	panic("unexpected")
}
func (r *stubUserRepo) UpdateDriverProfile(_ context.Context, _ *model.DriverProfile) error {
	panic("unexpected")
}
func (r *stubUserRepo) CreateMerchantProfile(_ context.Context, _ *model.MerchantProfile) error {
	panic("unexpected")
}
func (r *stubUserRepo) FindMerchantProfile(_ context.Context, _ uuid.UUID) (*model.MerchantProfile, error) {
	panic("unexpected")
}
func (r *stubUserRepo) UpdateMerchantProfile(_ context.Context, _ *model.MerchantProfile) error {
	panic("unexpected")
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newAuthForOTPTest(ur *stubUserRepo, em *stubEmailSender, wa *stubWA) *AuthService {
	cfg := &config.Config{JWTSecret: "test-secret-at-least-32-bytes-long!!", JWTAccessTTLMin: 15, JWTRefreshTTLDays: 30}
	svc := NewAuthService(ur, cfg, nil, em)
	if wa != nil {
		svc.InjectWhatsApp(wa)
	}
	return svc
}

func waitFor(t *testing.T, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met within 500ms")
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestRequestOTP_EmailUserGetsEmail(t *testing.T) {
	email := "user@example.com"
	ur := &stubUserRepo{user: &model.User{
		ID: uuid.New(), Phone: "+2348001234567", Email: &email, FirstName: "Ada",
	}}
	em := &stubEmailSender{}
	wa := &stubWA{}
	svc := newAuthForOTPTest(ur, em, wa)

	code, err := svc.RequestOTP(context.Background(), "+2348001234567", "phone_verification")
	if err != nil {
		t.Fatalf("RequestOTP: %v", err)
	}
	if code == "" {
		t.Fatal("expected non-empty code")
	}

	waitFor(t, func() bool { em.mu.Lock(); defer em.mu.Unlock(); return len(em.otps) > 0 })

	em.mu.Lock()
	waCalls := len(wa.otps)
	em.mu.Unlock()
	wa.mu.Lock()
	defer wa.mu.Unlock()
	if waCalls != 0 {
		t.Errorf("WhatsApp called %d times, want 0 for email user", waCalls)
	}
}

func TestRequestOTP_PhoneOnlyUserGetsWhatsApp(t *testing.T) {
	ur := &stubUserRepo{user: &model.User{
		ID: uuid.New(), Phone: "+2348009999999", Email: nil, FirstName: "Bola",
	}}
	em := &stubEmailSender{}
	wa := &stubWA{}
	svc := newAuthForOTPTest(ur, em, wa)

	_, err := svc.RequestOTP(context.Background(), "+2348009999999", "phone_verification")
	if err != nil {
		t.Fatalf("RequestOTP: %v", err)
	}

	waitFor(t, func() bool { wa.mu.Lock(); defer wa.mu.Unlock(); return len(wa.otps) > 0 })

	em.mu.Lock()
	emailCalls := len(em.otps)
	em.mu.Unlock()
	if emailCalls != 0 {
		t.Errorf("email called %d times, want 0 for phone-only user", emailCalls)
	}
}

func TestRequestOTP_UnknownPhoneStillReturnsCode(t *testing.T) {
	// FindByPhone returns error — no notification sent, but code is still returned
	// so the caller can't enumerate registered phones.
	ur := &stubUserRepo{user: nil}
	em := &stubEmailSender{}
	wa := &stubWA{}
	svc := newAuthForOTPTest(ur, em, wa)

	code, err := svc.RequestOTP(context.Background(), "+2348000000000", "phone_verification")
	if err != nil {
		t.Fatalf("RequestOTP: %v", err)
	}
	if code == "" {
		t.Fatal("expected non-empty code even for unknown phone")
	}
	// Give goroutines a moment to fire (they shouldn't).
	time.Sleep(50 * time.Millisecond)
	em.mu.Lock()
	wa.mu.Lock()
	defer em.mu.Unlock()
	defer wa.mu.Unlock()
	if len(em.otps) != 0 || len(wa.otps) != 0 {
		t.Errorf("unexpected notification for unknown phone: email=%d wa=%d", len(em.otps), len(wa.otps))
	}
}

func TestRequestOTP_NoWAInjected_PhoneOnlyUserSilent(t *testing.T) {
	// wa == nil — phone-only user gets nothing, no panic.
	ur := &stubUserRepo{user: &model.User{
		ID: uuid.New(), Phone: "+2348001111111", Email: nil, FirstName: "Chidi",
	}}
	em := &stubEmailSender{}
	svc := newAuthForOTPTest(ur, em, nil) // no WA injected

	_, err := svc.RequestOTP(context.Background(), "+2348001111111", "phone_verification")
	if err != nil {
		t.Fatalf("RequestOTP: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	em.mu.Lock()
	defer em.mu.Unlock()
	if len(em.otps) != 0 {
		t.Errorf("email called for phone-only user with no WA injected")
	}
}
