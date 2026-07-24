package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrDeliveryCodeInvalid  = errors.New("delivery code invalid or expired")
	ErrDeliveryCodeLocked   = errors.New("too many failed attempts — contact support")
	ErrDeliveryCodeUsed     = errors.New("delivery code already used")
)

const deliveryCodeMaxAttempts = 5

// DeliveryCodeService manages the one-time delivery confirmation codes.
//
// Primary flow (customer has phone signal):
//  1. Order transitions to in_transit → GenerateAndSend() called
//  2. Customer receives 6-digit code via SMS/push
//  3. Customer shares code with whoever is collecting
//  4. Rider enters code on their app → Verify() called
//  5. On success: caller (PaycodeService) proceeds with settlement
//
// Fallback flow (customer has no phone signal):
//  - Rider uses PaycodeService.ConfirmByCard() instead
//  - Customer shows SpeedPlus card + enters 4-digit wallet PIN
type DeliveryCodeService struct {
	db   *gorm.DB // transaction boundary only
	repo repo.DeliveryCodeRepo
}

func NewDeliveryCodeService(db *gorm.DB, r repo.DeliveryCodeRepo) *DeliveryCodeService {
	return &DeliveryCodeService{db: db, repo: r}
}

// Generate creates a new 6-digit delivery code for the order and returns the
// plaintext code. The caller is responsible for sending it to the customer
// (via SMS or push notification).
// Called when order transitions to in_transit.
func (s *DeliveryCodeService) Generate(ctx context.Context, orderID uuid.UUID) (string, error) {
	code, err := generateSixDigit()
	if err != nil {
		return "", fmt.Errorf("delivery code: generate: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(code), 12)
	if err != nil {
		return "", fmt.Errorf("delivery code: hash: %w", err)
	}

	dc := &model.DeliveryCode{
		ID:        uuid.New(),
		OrderID:   orderID,
		CodeHash:  string(hash),
		ExpiresAt: time.Now().Add(2 * time.Hour), // 2h — covers long deliveries
	}
	if err := s.repo.Upsert(ctx, dc); err != nil {
		return "", fmt.Errorf("delivery code: upsert: %w", err)
	}

	return code, nil
}

// Verify checks the submitted code against the stored hash.
// Returns nil on success. The caller must call settlement inside the same
// transaction — do NOT call Verify and then open a new transaction.
//
// On success the code is marked used inside the provided transaction.
// On wrong code the attempt counter is incremented.
// After 5 failed attempts the code is permanently locked.
func (s *DeliveryCodeService) Verify(ctx context.Context, tx *gorm.DB, orderID uuid.UUID, submitted string) error {
	dc, err := s.repo.LockByOrder(ctx, tx, orderID)
	if err != nil {
		return ErrDeliveryCodeInvalid
	}

	if dc.Attempts >= deliveryCodeMaxAttempts {
		return ErrDeliveryCodeLocked
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dc.CodeHash), []byte(submitted)); err != nil {
		// Wrong code — increment attempts, do not mark used
		_ = s.repo.IncrementAttempts(ctx, tx, dc.ID)
		remaining := deliveryCodeMaxAttempts - dc.Attempts - 1
		if remaining <= 0 {
			return ErrDeliveryCodeLocked
		}
		return fmt.Errorf("incorrect delivery code (%d attempt(s) remaining)", remaining)
	}

	// Correct — mark used
	return s.repo.MarkUsed(ctx, tx, dc.ID, time.Now())
}

func generateSixDigit() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
