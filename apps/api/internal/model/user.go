package model

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleCustomer UserRole = "customer"
	RoleDriver   UserRole = "driver"
	RoleMerchant UserRole = "merchant"
	RoleAdmin    UserRole = "admin"
)

type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Role         UserRole   `gorm:"type:varchar(20);not null"`
	FirstName    string     `gorm:"not null"`
	LastName     string     `gorm:"not null"`
	Phone        string     `gorm:"uniqueIndex;not null"`
	Email        *string    `gorm:"uniqueIndex"`
	Username     *string    `gorm:"uniqueIndex"` // optional, for P2P transfers
	ReferralCode string     `gorm:"uniqueIndex;not null"` // e.g. MUSA500 — generated on registration
	AvatarURL    *string
	PasswordHash string     `gorm:"not null"` // argon2id
	IsVerified   bool       `gorm:"default:false"`
	IsActive     bool       `gorm:"default:true"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ── Trust tier ────────────────────────────────────────────────────────────────

type TrustTier int

const (
	TierNew     TrustTier = 0 // prepaid only, card allowed
	TierRegular TrustTier = 1 // 3 completed orders, 0 fraud flags → pay-on-arrival unlocked
)

type UserTrustTier struct {
	UserID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Tier            TrustTier `gorm:"not null;default:0"`
	CompletedOrders int       `gorm:"not null;default:0"`
	FraudFlags      int       `gorm:"not null;default:0"`
	Frozen          bool      `gorm:"not null;default:false"` // permanent Tier 0 lock
	UpdatedAt       time.Time
}

// ── Virtual account (DVA) ─────────────────────────────────────────────────────

type VirtualAccount struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID        uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	AccountNumber string    `gorm:"not null"`
	BankName      string    `gorm:"not null"`
	BankCode      string    `gorm:"not null"`
	Provider      string    `gorm:"not null"` // "monnify"
	ProviderRef   string    `gorm:"not null"` // Monnify accountReference
	IsActive      bool      `gorm:"default:true"`
	CreatedAt     time.Time
}

// ── SpeedPlus card (identity QR) ──────────────────────────────────────────────
// The QR payload is: spd.card.v1.{userID}.{HMAC-SHA256}
// It is static — it never changes. It is an identity QR, not a payment QR.
// The rider scans it → backend finds the active in_transit order for that user.

type UserCard struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	Payload   string    `gorm:"not null"` // spd.card.v1.{userID}.{sig}
	IsActive  bool      `gorm:"default:true"`
	CreatedAt time.Time
}

// RefreshToken represents a single rotation token.
// FamilyID groups all tokens from the same login chain.
// On detected reuse of a revoked token, the entire family is revoked.
type RefreshToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	FamilyID  uuid.UUID  `gorm:"type:uuid;not null;index"` // migration 008 adds this column
	TokenHash string     `gorm:"uniqueIndex;not null"` // SHA-256 of raw token
	ExpiresAt time.Time  `gorm:"not null"`
	RevokedAt *time.Time
	CreatedAt time.Time
}

type OTPCode struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Phone     string    `gorm:"not null;index"`
	CodeHash  string    `gorm:"not null"` // bcrypt of 6-digit code
	Purpose   string    `gorm:"not null"` // "verify_phone" | "login" | "reset_pin"
	ExpiresAt time.Time `gorm:"not null"`
	UsedAt    *time.Time
	CreatedAt time.Time
}

type PIN struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	PINHash   string    `gorm:"not null"` // bcrypt
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Address struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID               uuid.UUID `gorm:"type:uuid;not null;index"`
	Label                *string
	Street               string    `gorm:"not null"`
	City                 string    `gorm:"not null"`
	State                string    `gorm:"not null"`
	Country              string    `gorm:"default:'Nigeria'"`
	Lat                  float64   `gorm:"not null"`
	Lng                  float64   `gorm:"not null"`
	DeliveryInstructions *string
	IsDefault            bool      `gorm:"default:false"`
	CreatedAt            time.Time
}

type VehicleType string

const (
	VehicleBicycle    VehicleType = "bicycle"
	VehicleMotorcycle VehicleType = "motorcycle"
	VehicleCar        VehicleType = "car"
	VehicleVan        VehicleType = "van"
)

type DriverStatus string

const (
	DriverPending     DriverStatus = "pending"
	DriverUnderReview DriverStatus = "under_review"
	DriverApproved    DriverStatus = "approved"
	DriverSuspended   DriverStatus = "suspended"
)

type DriverProfile struct {
	ID               uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID           uuid.UUID    `gorm:"type:uuid;uniqueIndex;not null"`
	Status           DriverStatus `gorm:"type:varchar(20);default:'pending'"`
	VehicleType      VehicleType  `gorm:"type:varchar(20);not null"`
	VehiclePlate     string       `gorm:"not null"`
	Rating           float64      `gorm:"default:5.0"`
	TotalDeliveries  int          `gorm:"default:0"`
	IsOnline         bool         `gorm:"default:false"`
	HazmatCertified  bool         `gorm:"not null;default:false"` // required for 25kg+ gas runs
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type MerchantVertical string

const (
	VerticalGas      MerchantVertical = "gas"
	VerticalGrocery  MerchantVertical = "grocery"
	VerticalFood     MerchantVertical = "food"
	VerticalPharmacy MerchantVertical = "pharmacy"
	VerticalPackage  MerchantVertical = "package"
)

type MerchantStatus string

const (
	MerchantPending   MerchantStatus = "pending"
	MerchantActive    MerchantStatus = "active"
	MerchantSuspended MerchantStatus = "suspended"
)

type MerchantProfile struct {
	ID             uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID         uuid.UUID        `gorm:"type:uuid;uniqueIndex;not null"`
	BusinessName   string           `gorm:"not null"`
	Vertical       MerchantVertical `gorm:"type:varchar(20);not null"`
	Status         MerchantStatus   `gorm:"type:varchar(20);default:'pending'"`
	Rating         float64          `gorm:"default:5.0"`
	IsOpen         bool             `gorm:"default:false"`
	LicenceNumber  *string          // PCN for pharmacy
	AddressID      *uuid.UUID       `gorm:"type:uuid"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type KYCDocType string

const (
	DocNIN             KYCDocType = "nin"
	DocBVN             KYCDocType = "bvn"
	DocDriversLicence  KYCDocType = "drivers_licence"
	DocLiveness        KYCDocType = "liveness"
	DocBusinessReg     KYCDocType = "business_reg"
)

type KYCStatus string

const (
	KYCPending  KYCStatus = "pending"
	KYCReview   KYCStatus = "under_review"
	KYCApproved KYCStatus = "approved"
	KYCRejected KYCStatus = "rejected"
)

type KYCDocument struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null;index"`
	DocType     KYCDocType `gorm:"type:varchar(30);not null"`
	R2Key       string     `gorm:"not null"` // private R2 object key
	UploadedAt  time.Time
}

type KYCCheck struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID          uuid.UUID  `gorm:"type:uuid;not null;index"`
	DocType         KYCDocType `gorm:"type:varchar(30);not null"`
	Status          KYCStatus  `gorm:"type:varchar(20);default:'pending'"`
	ProviderRef     *string    // Prembly reference ID
	ProviderPayload *string    `gorm:"type:jsonb"` // raw provider response
	ReviewedBy      *uuid.UUID `gorm:"type:uuid"` // admin user ID
	ReviewNote      *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
