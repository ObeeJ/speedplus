package model

import (
	"time"

	"github.com/google/uuid"
)

// PaymentLink is a shareable payment request.
// The creator requests money; the payer can be a Fourdat user (wallet debit)
// or a guest (Paystack card charge — no account required).
//
// URL: fourdat.com/pay/{slug}
type PaymentLink struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatorID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	Slug        string     `gorm:"uniqueIndex;not null"` // 12-char random slug
	AmountKobo  int64      `gorm:"not null"`
	Note        *string    // "For jollof rice 😂"
	Status      string     `gorm:"default:'pending'"` // pending|paid|expired|cancelled
	PaidByID    *uuid.UUID `gorm:"type:uuid"`         // NULL for guest payments
	PaidByEmail *string    // guest payer email (from Paystack)
	ProviderRef *string    // Paystack reference for guest payment
	PaidAt      *time.Time
	ExpiresAt   time.Time  `gorm:"not null"`
	CreatedAt   time.Time
}
