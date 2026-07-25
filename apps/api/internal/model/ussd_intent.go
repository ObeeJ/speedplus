package model

import (
	"time"

	"github.com/google/uuid"
)

type USSDIntent struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID          uuid.UUID  `gorm:"type:uuid;not null;index"`
	AmountKobo      int64      `gorm:"not null"`
	Provider        string     `gorm:"type:varchar(20);not null"`
	BankCode        string     `gorm:"type:varchar(10);not null"`
	BankName        *string    `gorm:"type:varchar(100)"`
	USSDCode        string     `gorm:"type:varchar(64);not null"`
	ProviderRef     string     `gorm:"type:varchar(100);uniqueIndex;not null"`
	PaymentIntentID *uuid.UUID `gorm:"type:uuid"`
	Status          string     `gorm:"type:varchar(20);default:'pending'"`
	ExpiresAt       time.Time  `gorm:"not null"`
	PaidAt          *time.Time
	CreatedAt       time.Time
}
