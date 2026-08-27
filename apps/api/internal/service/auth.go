package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/speedplus/api/internal/config"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/ports"
	"github.com/speedplus/api/internal/repo"
	"github.com/speedplus/api/internal/whatsapp"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrPhoneExists        = errors.New("phone already registered")
	ErrTokenExpired       = errors.New("token expired")
	ErrTokenInvalid       = errors.New("token invalid")
	ErrOTPInvalid         = errors.New("otp invalid or expired")
	ErrVehicleRequired    = errors.New("vehicleType and vehiclePlate are required to register as a driver")
	ErrVehicleTypeInvalid = errors.New("invalid vehicleType")
)

type Claims struct {
	UserID     string `json:"uid"`
	Role       string `json:"role"`
	IsVerified bool   `json:"verified"`
	jwt.RegisteredClaims
}

type AuthService struct {
	repo       repo.UserRepo
	cfg        *config.Config
	onboarding ports.OnboardingRunner
	email      emailSender
	wa         ports.WhatsAppNotifier
	queue      *asynq.Client
	referrals  referralRecorder
	enqueueJob func(userID string) error // injected by main.go to avoid import cycle
}

// referralRecorder is the subset of ReferralService used by AuthService.
type referralRecorder interface {
	Record(ctx context.Context, refereeID, referrerID uuid.UUID) error
}

// emailSender is the subset of email.Client used by AuthService.
type emailSender interface {
	SendWelcome(ctx context.Context, toEmail, firstName string)
	SendOTP(ctx context.Context, toEmail, firstName, code, purpose string)
}

func NewAuthService(r repo.UserRepo, cfg *config.Config, onboarding ports.OnboardingRunner, email emailSender) *AuthService {
	return &AuthService{repo: r, cfg: cfg, onboarding: onboarding, email: email}
}

// InjectWhatsApp wires the WhatsApp notifier for OTP delivery to phone-only users.
func (s *AuthService) InjectWhatsApp(wa ports.WhatsAppNotifier) {
	s.wa = wa
}

// InjectReferrals wires the ReferralService after construction to avoid import cycles.
func (s *AuthService) InjectReferrals(r referralRecorder) {
	s.referrals = r
}

// InjectQueue wires the asynq client and the enqueue function.
// The enqueue function is passed from main.go to avoid a service→worker import cycle.
func (s *AuthService) InjectQueue(q *asynq.Client, enqueue func(userID string) error) {
	s.queue = q
	s.enqueueJob = enqueue
}

// ── Registration ──────────────────────────────────────────────────────────────

// RegisterInput carries role-specific fields alongside the base registration
// details. VehicleType/VehiclePlate are required and validated only for
// role == model.RoleDriver.
type RegisterInput struct {
	FirstName    string
	LastName     string
	Phone        string
	Password     string
	Role         model.UserRole
	ReferralCode string
	VehicleType  string
	VehiclePlate string
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*model.User, string, string, error) {
	firstName, lastName, phone, password, role, referralCode := in.FirstName, in.LastName, in.Phone, in.Password, in.Role, in.ReferralCode
	if _, err := s.repo.FindByPhone(ctx, phone); err == nil {
		return nil, "", "", ErrPhoneExists
	}

	var vehicleType model.VehicleType
	if role == model.RoleDriver {
		if in.VehicleType == "" || in.VehiclePlate == "" {
			return nil, "", "", ErrVehicleRequired
		}
		vehicleType = model.VehicleType(in.VehicleType)
		switch vehicleType {
		case model.VehicleBicycle, model.VehicleMotorcycle, model.VehicleCar, model.VehicleVan:
		default:
			return nil, "", "", ErrVehicleTypeInvalid
		}
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, "", "", fmt.Errorf("hash password: %w", err)
	}

	ownerCode, err := generateReferralCode()
	if err != nil {
		return nil, "", "", fmt.Errorf("referral code: %w", err)
	}

	user := &model.User{
		ID:           uuid.New(),
		Role:         role,
		FirstName:    firstName,
		LastName:     lastName,
		Phone:        phone,
		PasswordHash: hash,
		ReferralCode: ownerCode,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, "", "", fmt.Errorf("create user: %w", err)
	}

	// Driver profile is created synchronously (local DB write, no external
	// calls) so dispatch/KYC can see the row immediately — unlike DVA/card
	// onboarding below, this must not be lost to a best-effort async job.
	if role == model.RoleDriver {
		if err := s.repo.CreateDriverProfile(ctx, &model.DriverProfile{
			ID:           uuid.New(),
			UserID:       user.ID,
			Status:       model.DriverPending,
			VehicleType:  vehicleType,
			VehiclePlate: in.VehiclePlate,
		}); err != nil {
			return nil, "", "", fmt.Errorf("create driver profile: %w", err)
		}
	}

	// Wire referral link if a valid code was supplied.
	// Synchronous: this is a local DB write with no external calls.
	// A goroutine here loses the referral silently on any failure with no
	// retry path, and runs outside the caller's context.
	if referralCode != "" && s.referrals != nil {
		if referrer, err := s.repo.FindByReferralCode(ctx, referralCode); err == nil {
			if recErr := s.referrals.Record(ctx, user.ID, referrer.ID); recErr != nil {
				slog.WarnContext(ctx, "referral record failed", "error", recErr, "user_id", user.ID)
			}
		}
	}

	// Onboarding: DVA + card + trust tier.
	// Enqueued as a retryable asynq job so failures are visible and retried.
	// Falls back to direct goroutine only if the queue is not yet wired (tests).
	if s.enqueueJob != nil {
		_ = s.enqueueJob(user.ID.String())
	} else {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("onboarding goroutine panic", "panic", r)
				}
			}()
			_ = s.onboarding.Run(context.Background(), user)
		}()
	}

	// Welcome email — best-effort, non-blocking.
	if user.Email != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("welcome email goroutine panic", "panic", r)
				}
			}()
			s.email.SendWelcome(context.Background(), *user.Email, user.FirstName)
		}()
	}

	access, refresh, err := s.issueTokenPair(ctx, user)
	return user, access, refresh, err
}

// ── Login ─────────────────────────────────────────────────────────────────────

func (s *AuthService) Login(ctx context.Context, phone, password string) (*model.User, string, string, error) {
	user, err := s.repo.FindByPhone(ctx, phone)
	if err != nil {
		bcrypt.CompareHashAndPassword([]byte("$2a$12$dummy"), []byte(password)) //nolint
		return nil, "", "", ErrInvalidCredentials
	}

	if !verifyPassword(password, user.PasswordHash) {
		return nil, "", "", ErrInvalidCredentials
	}

	access, refresh, err := s.issueTokenPair(ctx, user)
	return user, access, refresh, err
}

func (s *AuthService) Refresh(ctx context.Context, rawRefresh string) (string, string, error) {
	tokenHash := hashToken(rawRefresh)

	// Look up including revoked tokens to detect reuse attacks
	rt, err := s.repo.FindRefreshTokenAny(ctx, tokenHash)
	if err != nil {
		return "", "", ErrTokenInvalid
	}

	// Reuse detection: token found but already revoked → revoke entire family
	if rt.RevokedAt != nil {
		_ = s.repo.RevokeRefreshFamily(ctx, rt.FamilyID, time.Now())
		return "", "", ErrTokenInvalid
	}

	// Token expired
	if rt.ExpiresAt.Before(time.Now()) {
		return "", "", ErrTokenExpired
	}

	// Rotate: revoke the presented token, issue a new one in the same family
	if err := s.repo.RevokeRefreshToken(ctx, tokenHash, time.Now()); err != nil {
		return "", "", fmt.Errorf("refresh: revoke token: %w", err)
	}

	user, err := s.repo.FindByID(ctx, rt.UserID)
	if err != nil {
		return "", "", ErrUserNotFound
	}

	return s.issueTokenPairWithFamily(ctx, user, rt.FamilyID)
}

// ── Logout ────────────────────────────────────────────────────────────────────

func (s *AuthService) Logout(ctx context.Context, rawRefresh string) error {
	return s.repo.RevokeRefreshToken(ctx, hashToken(rawRefresh), time.Now())
}

// ── Access token validation ───────────────────────────────────────────────────

func (s *AuthService) ValidateAccessToken(raw string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(raw, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrTokenInvalid
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

// ── OTP ───────────────────────────────────────────────────────────────────────

// validOTPPurpose returns true for the exhaustive set of allowed OTP purposes.
// Kept as a function over a map so the set cannot be mutated at runtime.
// purpose drives privilege changes (phone_verification marks user verified)
// so it must never be free-text from the caller.
func validOTPPurpose(purpose string) bool {
	return purpose == "phone_verification" || purpose == "password_reset"
}

func (s *AuthService) RequestOTP(ctx context.Context, phone, purpose string) (string, error) {
	if !validOTPPurpose(purpose) {
		return "", ErrOTPInvalid
	}
	code, err := generateOTP()
	if err != nil {
		return "", err
	}

	codeHash, err := bcrypt.GenerateFromPassword([]byte(code), 12)
	if err != nil {
		return "", err
	}

	s.repo.InvalidatePreviousOTPs(ctx, phone, purpose)

	otp := model.OTPCode{
		ID:        uuid.New(),
		Phone:     phone,
		CodeHash:  string(codeHash),
		Purpose:   purpose,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	if err := s.repo.CreateOTP(ctx, &otp); err != nil {
		return "", err
	}
	// OTP delivery: email when available, WhatsApp otherwise.
	// Phone-only users (no email on file) receive the code via WhatsApp.
	if u, err := s.repo.FindByPhone(ctx, phone); err == nil {
		if u.Email != nil {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("OTP email goroutine panic", "panic", r)
					}
				}()
				s.email.SendOTP(context.Background(), *u.Email, u.FirstName, code, purpose)
			}()
		} else if s.wa != nil {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("OTP whatsapp goroutine panic", "panic", r)
					}
				}()
				s.wa.SendOTP(whatsapp.NormalisePhone(u.Phone), code, purpose)
			}()
		}
	}
	return code, nil
}

// VerifyOTP checks the code and, for purpose == "phone_verification", marks
// the user verified and returns a fresh token pair. RequireVerified reads the
// claim from the JWT at issue time (no DB hit), so a token issued before
// verification would still read unverified — the caller must swap in these
// tokens for verification to take effect immediately.
func (s *AuthService) VerifyOTP(ctx context.Context, phone, code, purpose string) (string, string, error) {
	otp, err := s.repo.FindActiveOTP(ctx, phone, purpose)
	if err != nil {
		return "", "", ErrOTPInvalid
	}
	if bcrypt.CompareHashAndPassword([]byte(otp.CodeHash), []byte(code)) != nil {
		return "", "", ErrOTPInvalid
	}
	s.repo.MarkOTPUsed(ctx, otp.ID, time.Now())

	user, err := s.repo.FindByPhone(ctx, phone)
	if err != nil {
		return "", "", ErrUserNotFound
	}
	if purpose == "phone_verification" && !user.IsVerified {
		user.IsVerified = true
		if err := s.repo.Update(ctx, user); err != nil {
			return "", "", err
		}
	}
	return s.issueTokenPair(ctx, user)
}

// ── Password reset ────────────────────────────────────────────────────────────

// ResetPassword verifies a "password_reset" OTP then updates the user's
// password hash. The OTP is consumed (marked used) on success.
func (s *AuthService) ResetPassword(ctx context.Context, phone, otp, newPassword string) error {
	otpRecord, err := s.repo.FindActiveOTP(ctx, phone, "password_reset")
	if err != nil {
		return ErrOTPInvalid
	}
	if bcrypt.CompareHashAndPassword([]byte(otpRecord.CodeHash), []byte(otp)) != nil {
		return ErrOTPInvalid
	}
	s.repo.MarkOTPUsed(ctx, otpRecord.ID, time.Now())

	user, err := s.repo.FindByPhone(ctx, phone)
	if err != nil {
		return ErrUserNotFound
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	user.PasswordHash = hash
	return s.repo.Update(ctx, user)
}

// ── PIN ───────────────────────────────────────────────────────────────────────

func (s *AuthService) SetPIN(ctx context.Context, userID uuid.UUID, pin string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), 12)
	if err != nil {
		return err
	}
	return s.repo.UpsertPIN(ctx, userID, string(hash))
}

const pinMaxAttempts = 5
const pinLockDuration = 30 * time.Minute

func (s *AuthService) VerifyPIN(ctx context.Context, userID uuid.UUID, pin string) error {
	p, err := s.repo.FindPIN(ctx, userID)
	if err != nil {
		return errors.New("pin not set")
	}
	if p.LockedUntil != nil && time.Now().Before(*p.LockedUntil) {
		return errors.New("PIN temporarily locked. Try again later.")
	}
	if bcrypt.CompareHashAndPassword([]byte(p.PINHash), []byte(pin)) != nil {
		var lockUntil *time.Time
		if p.FailedAttempts+1 >= pinMaxAttempts {
			t := time.Now().Add(pinLockDuration)
			lockUntil = &t
		}
		if err := s.repo.IncrementPINFailure(ctx, userID, lockUntil); err != nil {
			slog.ErrorContext(ctx, "IncrementPINFailure failed", "error", err, "user_id", userID)
		}
		if lockUntil != nil {
			return errors.New("PIN temporarily locked. Try again later.")
		}
		return errors.New("invalid pin")
	}
	// Success: reset failure counter.
	if err := s.repo.ResetPINFailures(ctx, userID); err != nil {
		slog.ErrorContext(ctx, "ResetPINFailures failed", "error", err, "user_id", userID)
	}
	return nil
}

// ── Internals ─────────────────────────────────────────────────────────────────

func (s *AuthService) issueTokenPair(ctx context.Context, user *model.User) (string, string, error) {
	return s.issueTokenPairWithFamily(ctx, user, uuid.New())
}

func (s *AuthService) issueTokenPairWithFamily(ctx context.Context, user *model.User, familyID uuid.UUID) (string, string, error) {
	now := time.Now()

	accessClaims := Claims{
		UserID:     user.ID.String(),
		Role:       string(user.Role),
		IsVerified: user.IsVerified,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.cfg.JWTAccessTTLMin) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID.String(),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).
		SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", "", err
	}

	rawRefresh := make([]byte, 32)
	if _, err := rand.Read(rawRefresh); err != nil {
		return "", "", err
	}
	rawRefreshHex := hex.EncodeToString(rawRefresh)

	rt := model.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		FamilyID:  familyID,
		TokenHash: hashToken(rawRefreshHex),
		ExpiresAt: now.Add(time.Duration(s.cfg.JWTRefreshTTLDays) * 24 * time.Hour),
	}
	if err := s.repo.CreateRefreshToken(ctx, &rt); err != nil {
		return "", "", err
	}

	return accessToken, rawRefreshHex, nil
}

// hashPassword uses Argon2id (memory=64MB, time=1, threads=4).
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

func verifyPassword(password, encoded string) bool {
	// fmt.Sscanf with %s reads until whitespace, not until '$', so it swallows
	// both salt and hash into saltHex. Split on '$' directly instead.
	// Format: $argon2id$v=19$m=65536,t=1,p=4$<saltHex>$<hashHex>
	parts := strings.Split(encoded, "$")
	// parts: ["", "argon2id", "v=19", "m=65536,t=1,p=4", saltHex, hashHex]
	if len(parts) != 6 {
		return false
	}
	saltHex := parts[4]
	hashHex := parts[5]
	salt, _ := hex.DecodeString(saltHex)
	expected, _ := hex.DecodeString(hashHex)
	actual := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// generateReferralCode produces a 7-char alphanumeric code (e.g. MUSA500).
// Collision probability at 1M users: ~0.01% — acceptable; DB unique index is the hard guard.
func generateReferralCode() (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 7)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		b[i] = chars[n.Int64()]
	}
	return string(b), nil
}
