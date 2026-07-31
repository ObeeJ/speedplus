package model

import (
	"time"

	"github.com/google/uuid"
)

// ── Phase 2: Catalog ──────────────────────────────────────────────────────────

type Merchant struct {
	ID              uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID          uuid.UUID        `gorm:"type:uuid;uniqueIndex;not null"`
	BusinessName    string           `gorm:"not null"`
	Vertical        MerchantVertical `gorm:"type:varchar(20);not null"`
	Status          MerchantStatus   `gorm:"type:varchar(20);default:'pending'"`
	Rating          float64          `gorm:"default:5.0"`
	IsOpen          bool             `gorm:"default:false"`
	Lat             float64
	Lng             float64
	FillAccuracyPct *float64 `gorm:"default:null"` // gas: avg(measured/ordered), nil until first verified fill
	FillSampleCount int      `gorm:"default:0"`
	FillStatus      string   `gorm:"not null;default:'good'"` // good|warned|probation|delisted
	// Gas plant fields (Phase 1)
	IsGasPlant      bool `gorm:"not null;default:false"` // true = filling plant (can do refill mode)
	PlantCapacityKg *int `gorm:"default:null"`           // daily fill capacity in kg
	FloatCount      *int `gorm:"default:null"`           // filled cylinders held as swap float
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Product struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	MerchantID     uuid.UUID `gorm:"type:uuid;not null;index"`
	Name           string    `gorm:"not null"`
	Description    *string
	PriceKobo      int64 `gorm:"not null"` // always kobo
	Category       string
	IsAvailable    bool       `gorm:"default:true"`
	CylinderSpecID *uuid.UUID `gorm:"type:uuid;default:null"`                           // gas products only: FK to cylinder_specs
	SearchVec      string     `gorm:"type:tsvector;index:idx_products_search,type:gin"` // populated by trigger
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type PricingQuote struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID      uuid.UUID `gorm:"type:uuid;not null"`
	MerchantID      uuid.UUID `gorm:"type:uuid;not null"`
	DistanceKm      float64
	ETAMinutes      int     // driving time from OSRM + pickup buffer
	StopCount       int     `gorm:"default:1"` // multi-drop package orders: number of dropoff stops
	WeightKg        float64 // package vertical only
	SizeCategory    string  `gorm:"type:varchar(10)"` // small|medium|large
	SubtotalKobo    int64
	DeliveryKobo    int64
	ServiceKobo     int64
	TotalKobo       int64
	WeatherAdvisory string    `gorm:"type:text"`            // informational only, no fee
	QuoteHash       string    `gorm:"uniqueIndex;not null"` // HMAC of fields — tamper-proof
	ExpiresAt       time.Time `gorm:"not null"`
	UsedAt          *time.Time
	CreatedAt       time.Time
}

// ── Phase 3: Orders ───────────────────────────────────────────────────────────

type OrderStatus string

const (
	OrderPending            OrderStatus = "pending"
	OrderConfirmed          OrderStatus = "confirmed"
	OrderPreparing          OrderStatus = "preparing"
	OrderReadyForPickup     OrderStatus = "ready_for_pickup"
	OrderDriverAssigned     OrderStatus = "driver_assigned"
	OrderAwaitingCollection OrderStatus = "awaiting_collection" // gas refill: waiting for rider to collect empty
	OrderEmptyCollected     OrderStatus = "empty_collected"     // gas refill: empty cylinder picked up
	OrderAtPlant            OrderStatus = "at_plant"            // gas refill: cylinder at filling plant
	OrderInTransit          OrderStatus = "in_transit"
	OrderDelivered          OrderStatus = "delivered"
	OrderCancelled          OrderStatus = "cancelled"
	OrderRefunded           OrderStatus = "refunded"
)

// ValidTransitions is the authoritative state machine.
// 409 on any transition not listed here.
var ValidTransitions = map[OrderStatus][]OrderStatus{
	OrderPending:            {OrderConfirmed, OrderCancelled},
	OrderConfirmed:          {OrderPreparing, OrderCancelled},
	OrderPreparing:          {OrderReadyForPickup, OrderCancelled},
	OrderReadyForPickup:     {OrderDriverAssigned},
	OrderDriverAssigned:     {OrderInTransit, OrderAwaitingCollection, OrderCancelled},
	OrderAwaitingCollection: {OrderEmptyCollected, OrderCancelled},
	OrderEmptyCollected:     {OrderAtPlant, OrderCancelled},
	OrderAtPlant:            {OrderInTransit, OrderCancelled},
	OrderInTransit:          {OrderDelivered, OrderCancelled},
	OrderDelivered:          {},
	OrderCancelled:          {OrderRefunded},
	OrderRefunded:           {},
}

type Order struct {
	ID                uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID        uuid.UUID   `gorm:"type:uuid;not null;index"`
	MerchantID        uuid.UUID   `gorm:"type:uuid;not null;index"`
	DriverID          *uuid.UUID  `gorm:"type:uuid;index"`
	QuoteID           uuid.UUID   `gorm:"type:uuid;not null"`
	Vertical          string      `gorm:"not null"`
	Status            OrderStatus `gorm:"type:varchar(30);default:'pending'"`
	SubtotalKobo      int64
	DeliveryKobo      int64
	ServiceKobo       int64
	TipKobo           int64 `gorm:"default:0"`
	TotalKobo         int64
	DeliveryAddressID uuid.UUID `gorm:"type:uuid;not null"`
	// AES-GCM ciphertext (base64) — never expose these columns directly in an
	// API response. Decrypt only for an authorized reader via OrderService.
	RecipientNameEnc  *string    `gorm:"column:recipient_name_enc;type:text"`
	RecipientPhoneEnc *string    `gorm:"column:recipient_phone_enc;type:text"`
	PrescriptionID    *uuid.UUID `gorm:"type:uuid"`
	ScheduledFor      *time.Time
	EstimatedAt       *time.Time
	DeliveredAt       *time.Time
	CancelReason      *string
	PaymentMethod     string  `gorm:"type:varchar(20);default:'wallet'"` // wallet|pay_on_arrival
	DeclaredValueKobo *int64  `gorm:"default:null"`                      // sender-stated package value; nil = not declared
	TrackingRef       *string `gorm:"type:varchar(12);uniqueIndex"`      // short human-readable ref e.g. SPX-A3K9
	// Gas-specific fields (Phase 1)
	GasMode        *string    `gorm:"type:text;default:null"` // swap|refill|new_cylinder
	CylinderID     *uuid.UUID `gorm:"type:uuid;default:null"` // refill mode: customer's registered cylinder
	IdempotencyKey string     `gorm:"uniqueIndex;not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time

	Items  []OrderItem  `gorm:"foreignKey:OrderID"`
	Events []OrderEvent `gorm:"foreignKey:OrderID"`
	Stops  []OrderStop  `gorm:"foreignKey:OrderID"`
}

type OrderItem struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID          uuid.UUID  `gorm:"type:uuid;not null;index"`
	ProductID        uuid.UUID  `gorm:"type:uuid;not null"`
	Name             string     `gorm:"not null"`
	Quantity         int        `gorm:"not null"`
	UnitPriceKobo    int64      `gorm:"not null"`
	TotalKobo        int64      `gorm:"not null"`
	WeightKg         float64    `gorm:"default:0"`              // package vertical only
	SizeCategory     string     `gorm:"type:varchar(10)"`       // small|medium|large
	CylinderSpecID   *uuid.UUID `gorm:"type:uuid;default:null"` // gas vertical only
	Customizations   *string
	SubstitutionPref *string `gorm:"type:varchar(10)"` // allow|deny|contact
}

type OrderEvent struct {
	ID         uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID    uuid.UUID   `gorm:"type:uuid;not null;index"`
	FromStatus OrderStatus `gorm:"type:varchar(30)"`
	ToStatus   OrderStatus `gorm:"type:varchar(30);not null"`
	ActorID    uuid.UUID   `gorm:"type:uuid;not null"`
	ActorRole  string      `gorm:"not null"`
	Note       *string
	CreatedAt  time.Time
}

type OrderStop struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID   uuid.UUID `gorm:"type:uuid;not null;index"`
	Sequence  int       `gorm:"not null"`
	AddressID uuid.UUID `gorm:"type:uuid;not null"`
	// AES-GCM ciphertext (base64) — never expose directly in an API response.
	// Decrypt only for the assigned driver, on the active stop, while in transit.
	RecipientNameEnc  *string `gorm:"column:recipient_name_enc;type:text"`
	RecipientPhoneEnc *string `gorm:"column:recipient_phone_enc;type:text"`
	Notes             *string `gorm:"type:text"`
	QRCode            string  `gorm:"not null"`          // signed paycode per stop
	Status            string  `gorm:"default:'pending'"` // pending|confirmed|skipped
	ConfirmedAt       *time.Time
	// Gas reverse logistics — populated by ConfirmStop for gas refill orders.
	EmptyCollected      bool    `gorm:"not null;default:false"`
	EmptyCylinderSerial *string `gorm:"type:text;default:null"`
}

// CylinderCustodyEvent is an append-only audit trail of cylinder custody
// changes during a gas refill order. Insert-only — never update or delete.
type CylinderCustodyEvent struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID     uuid.UUID  `gorm:"type:uuid;not null;index"`
	StopID      *uuid.UUID `gorm:"type:uuid"`
	CylinderID  *uuid.UUID `gorm:"type:uuid"` // FK to customer_cylinders; nil until Phase 1
	Serial      *string    `gorm:"type:text"` // human-readable serial
	EventType   string     `gorm:"not null"`  // collected|at_plant|returned
	ActorID     uuid.UUID  `gorm:"type:uuid;not null"`
	CapturedLat *float64
	CapturedLng *float64
	OccurredAt  time.Time `gorm:"not null"`
	CreatedAt   time.Time
}

// CylinderSpec is the canonical catalog of LPG cylinder sizes.
type CylinderSpec struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SizeKg    float64   `gorm:"type:numeric(4,1);not null;uniqueIndex"`
	TareKg    float64   `gorm:"type:numeric(4,1);not null"`
	ValveType string    `gorm:"not null;default:'standard'"` // standard|POL|ACME
	Label     string    `gorm:"not null"`
	IsActive  bool      `gorm:"not null;default:true"`
	CreatedAt time.Time
}

// CustomerCylinder is the customer's registered LPG cylinder.
type CustomerCylinder struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID          uuid.UUID  `gorm:"type:uuid;not null;index"`
	SpecID          *uuid.UUID `gorm:"type:uuid;default:null"` // FK to cylinder_specs
	Serial          string     `gorm:"not null"`
	ManufactureYear *int16     `gorm:"default:null"`
	ValveType       *string    `gorm:"type:text;default:null"`
	TareKg          *float64   `gorm:"type:numeric(4,1);default:null"`
	LastRecertAt    *time.Time `gorm:"type:date;default:null"`    // nil = unknown
	Status          string     `gorm:"not null;default:'active'"` // active|retired|in_custody
	Notes           *string    `gorm:"type:text;default:null"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CylinderHandoverChecklist is an append-only safety checklist recorded by
// the rider at handover. Insert-only — never update or delete.
type CylinderHandoverChecklist struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID         uuid.UUID `gorm:"type:uuid;not null;index"`
	DriverID        uuid.UUID `gorm:"type:uuid;not null"`
	ValveSeated     bool      `gorm:"not null;default:false"`
	NoHiss          bool      `gorm:"not null;default:false"`
	RegulatorFitted bool      `gorm:"not null;default:false"`
	Notes           *string   `gorm:"type:text"`
	CapturedLat     *float64
	CapturedLng     *float64
	RecordedAt      time.Time `gorm:"not null"`
	CreatedAt       time.Time
}

type Prescription struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID      uuid.UUID  `gorm:"type:uuid;not null;index"`
	MerchantID      *uuid.UUID `gorm:"type:uuid;not null"` // pharmacy the Rx was submitted to; required as of migration 034
	R2Key           string     `gorm:"not null"`
	Status          string     `gorm:"default:'pending'"` // pending|approved|rejected|consumed|expired
	ReviewerID      *uuid.UUID `gorm:"type:uuid"`         // pharmacist user ID
	ReviewNote      *string
	ExpiresAt       *time.Time `gorm:"type:timestamptz"` // set on approval; sweep flips approved→expired
	ConsumedOrderID *uuid.UUID `gorm:"type:uuid"`         // set atomically when an order consumes this Rx
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ── Phase 4: Dispatch ─────────────────────────────────────────────────────────

type DriverLocation struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DriverID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	// PostGIS point — raw SQL for GIST index; GORM stores as WKT string
	Location  string `gorm:"type:geometry(Point,4326);not null"`
	Heading   *float64
	UpdatedAt time.Time `gorm:"index"`
}

type DeliveryOffer struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	DriverID  *uuid.UUID `gorm:"type:uuid;index"`   // NULL until accepted
	Status    string     `gorm:"default:'pending'"` // pending|accepted|rejected|expired
	ExpiresAt time.Time  `gorm:"not null"`
	CreatedAt time.Time
}

// ServiceZone is a named PostGIS polygon with a weekly delivery schedule.
// active_days is a bitmask matching Go's time.Weekday() (Sunday=0..Saturday=6),
// computed elsewhere as 1 << now.Weekday() — NOT ISO 8601 (Monday=1) numbering.
// window_start/end are minutes-since-midnight UTC. Nigeria is fixed UTC+1 with
// no DST, so an 08:00-17:00 WAT business window is stored as 420-960 UTC.
type ServiceZone struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name         string    `gorm:"not null"`
	Boundary     string    `gorm:"type:geometry(Polygon,4326);not null"`
	ActiveDays   int16     `gorm:"not null;default:127"`
	WindowStart  int16     `gorm:"not null;default:420"`        // 08:00 WAT = 07:00 UTC
	WindowEnd    int16     `gorm:"not null;default:960"`        // 17:00 WAT = 16:00 UTC
	LaunchStatus string    `gorm:"not null;default:'piloting'"` // piloting|live|paused
	IsActive     bool      `gorm:"not null;default:true"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// DeliveryRun is one rider, one zone, one window, N orders.
// optimized_sequence is the OSRM-ordered array of order IDs as JSON.
type DeliveryRun struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ZoneID            uuid.UUID  `gorm:"type:uuid;not null;index"`
	DriverID          *uuid.UUID `gorm:"type:uuid;index"`
	WindowStart       time.Time  `gorm:"not null"`
	WindowEnd         time.Time  `gorm:"not null"`
	Status            string     `gorm:"not null;default:'assembling'"` // assembling|dispatched|in_progress|completed|cancelled
	OptimizedSequence string     `gorm:"type:jsonb;default:'[]'"`
	TotalDistanceKm   float64    `gorm:"default:0"`
	CreatedAt         time.Time
	UpdatedAt         time.Time

	Orders []RunOrder `gorm:"foreignKey:RunID"`
}

// RunOrder is the join between a delivery run and an order, with sequence.
type RunOrder struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RunID     uuid.UUID `gorm:"type:uuid;not null;index"`
	OrderID   uuid.UUID `gorm:"type:uuid;not null;index"`
	Sequence  int       `gorm:"not null"`
	CreatedAt time.Time
}

// ── Phase 5: Wallet & Ledger ──────────────────────────────────────────────────

type AccountType string

const (
	AccountWallet           AccountType = "wallet"
	AccountEscrow           AccountType = "escrow"
	AccountRevenue          AccountType = "revenue"
	AccountEarnings         AccountType = "earnings"
	AccountLiability        AccountType = "liability"         // gift cards, loyalty
	AccountProviderClearing AccountType = "provider_clearing" // asset: funds in transit from payment provider
)

type LedgerAccount struct {
	ID        uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID   *uuid.UUID  `gorm:"type:uuid;index"` // NULL for platform accounts
	Type      AccountType `gorm:"type:varchar(20);not null"`
	Currency  string      `gorm:"default:'NGN'"`
	CreatedAt time.Time
}

type LedgerEntry struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	JournalID   uuid.UUID  `gorm:"type:uuid;not null;index"` // groups debit+credit pair
	AccountID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	AmountKobo  int64      `gorm:"not null"` // positive=credit, negative=debit
	Description string     `gorm:"not null"`
	RefType     string     // "order"|"withdrawal"|"cashout"|"tip"|"refund"
	RefID       *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time
}

type WalletBalance struct {
	AccountID   uuid.UUID `gorm:"type:uuid;primaryKey"`
	BalanceKobo int64     `gorm:"not null;default:0"`
	UpdatedAt   time.Time
}

type PaymentIntent struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index"`
	AmountKobo     int64     `gorm:"not null"`
	Currency       string    `gorm:"default:'NGN'"`
	Provider       string    `gorm:"not null"` // paystack|monnify|flutterwave
	ProviderRef    *string   `gorm:"uniqueIndex"`
	Status         string    `gorm:"default:'pending'"` // pending|success|failed
	IdempotencyKey string    `gorm:"uniqueIndex;not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ProviderTransaction struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Provider    string    `gorm:"not null"`
	ProviderRef string    `gorm:"uniqueIndex;not null"`
	AmountKobo  int64     `gorm:"not null"`
	Status      string    `gorm:"not null"`
	RawPayload  string    `gorm:"type:jsonb"`
	CreatedAt   time.Time
}

type WebhookEvent struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Provider    string    `gorm:"not null"`
	EventID     string    `gorm:"uniqueIndex:idx_webhook_dedupe;not null"` // provider event ID
	EventType   string    `gorm:"not null"`
	Payload     string    `gorm:"type:jsonb;not null"`
	ProcessedAt *time.Time
	CreatedAt   time.Time
}

type IdempotencyKey struct {
	Key        string    `gorm:"primaryKey"`
	UserID     uuid.UUID `gorm:"type:uuid;not null"`
	StatusCode int
	Response   string `gorm:"type:jsonb"`
	CreatedAt  time.Time
	ExpiresAt  time.Time `gorm:"index"`
}

// PlatformBalanceSnapshot is a daily point-in-time balance for aggregate-only
// platform accounts (revenue, provider_clearing). Written by the asynq cron job.
type PlatformBalanceSnapshot struct {
	ID           uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AccountType  AccountType `gorm:"type:varchar(30);not null;index"`
	BalanceKobo  int64       `gorm:"not null"`
	SnapshotDate time.Time   `gorm:"not null;index"` // truncated to midnight UTC
	CreatedAt    time.Time
}

// ── Phase 6: Escrow, Paycodes, Earnings, Splits ───────────────────────────────

type EscrowStatus string

const (
	EscrowHeld     EscrowStatus = "held"
	EscrowReleased EscrowStatus = "released"
	EscrowFrozen   EscrowStatus = "frozen"
	EscrowReversed EscrowStatus = "reversed"
)

type EscrowHold struct {
	ID           uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID      uuid.UUID    `gorm:"type:uuid;uniqueIndex;not null"`
	AccountID    uuid.UUID    `gorm:"type:uuid;not null"` // escrow ledger account
	AmountKobo   int64        `gorm:"not null"`
	Status       EscrowStatus `gorm:"type:varchar(20);default:'held'"`
	ReleasedAt   *time.Time
	ReleasedBy   *uuid.UUID `gorm:"type:uuid"` // paycode confirm event ID
	FrozenAt     *time.Time
	FrozenReason *string
	// Dual-admin approval for manual release
	ApprovalOne *uuid.UUID `gorm:"type:uuid"`
	ApprovalTwo *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Paycode struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID     uuid.UUID  `gorm:"type:uuid;not null;index"`
	Nonce       string     `gorm:"uniqueIndex;not null"`
	Payload     string     `gorm:"not null"` // spq.v1.{orderId}.{nonce}.{exp}.{HMAC}
	ExpiresAt   time.Time  `gorm:"not null"`
	ScannedBy   *uuid.UUID `gorm:"type:uuid"` // driver ID
	ConfirmedAt *time.Time
	CreatedAt   time.Time
}

// DeliveryCode is a 6-digit one-time code sent to the customer when the rider
// is dispatched (order transitions to in_transit).
// The customer shares this code with whoever is collecting the order.
// The rider enters it on their app to confirm delivery and release escrow.
// The SpeedPlus card is the fallback when the customer has no phone signal.
type DeliveryCode struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID   uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"` // one active code per order
	CodeHash  string    `gorm:"not null"`                       // bcrypt of 6-digit code
	Attempts  int       `gorm:"not null;default:0"`             // max 5 before lockout
	ExpiresAt time.Time `gorm:"not null"`
	UsedAt    *time.Time
	CreatedAt time.Time

	// GPS evidence captured at successful confirmation. Flag-only — a far or
	// missing location never blocks settlement, it queues the order for review.
	ConfirmLat       *float64
	ConfirmLng       *float64
	ConfirmDistanceM *float64
	LocationFlagged  bool `gorm:"not null;default:false"`
}

type DriverEarning struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DriverID   uuid.UUID `gorm:"type:uuid;not null;index"`
	OrderID    uuid.UUID `gorm:"type:uuid;not null;index"`
	AmountKobo int64     `gorm:"not null"`
	TipKobo    int64     `gorm:"default:0"`
	PaidOutAt  *time.Time
	CreatedAt  time.Time
}

type CashoutRequest struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DriverID       uuid.UUID  `gorm:"type:uuid;not null;index"`
	MerchantID     *uuid.UUID `gorm:"type:uuid;index"`           // set for merchant withdrawals
	ActorType      string     `gorm:"not null;default:'driver'"` // driver|merchant
	AmountKobo     int64      `gorm:"not null"`
	FeeKobo        int64      `gorm:"not null"`
	Status         string     `gorm:"default:'pending'"` // pending|processing|paid|failed
	ProviderRef    *string
	IdempotencyKey string `gorm:"uniqueIndex;not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// MerchantBankAccount stores the verified bank account a merchant withdraws to.
// account_name is resolved via the payment provider's account-resolution API
// before saving — never trusted from the request body.
type MerchantBankAccount struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	MerchantID    uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	BankCode      string    `gorm:"not null"`
	BankName      string    `gorm:"not null"`
	AccountNumber string    `gorm:"not null"`
	AccountName   string    `gorm:"not null"` // provider-resolved, not user-supplied
	Provider      string    `gorm:"not null;default:'paystack'"`
	IsVerified    bool      `gorm:"not null;default:true"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type OrderSplit struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID        uuid.UUID `gorm:"type:uuid;not null;index"`
	ParticipantID  uuid.UUID `gorm:"type:uuid;not null"`
	ShareKobo      int64     `gorm:"not null"`
	Status         string    `gorm:"default:'pending'"` // pending|funded|refunded
	FundedAt       *time.Time
	RefundedAt     *time.Time
	IdempotencyKey string `gorm:"uniqueIndex;not null"`
	CreatedAt      time.Time
}

// ── Phase 7: Growth ───────────────────────────────────────────────────────────

type Referral struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ReferrerID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	RefereeID    uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	RewardPaidAt *time.Time // only after referee's first completed paid order
	CreatedAt    time.Time
}

type GiftCard struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CodeHash   string     `gorm:"uniqueIndex;not null"` // SHA-256 of code
	AmountKobo int64      `gorm:"not null"`
	IssuerID   uuid.UUID  `gorm:"type:uuid;not null"`
	RedeemedBy *uuid.UUID `gorm:"type:uuid"`
	RedeemedAt *time.Time
	ExpiresAt  *time.Time
	CreatedAt  time.Time
}

type LoyaltyEvent struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	EventType string     `gorm:"not null"` // daily_login|order_placed|order_completed|referral|review|streak
	Points    int        `gorm:"not null"`
	RefID     *uuid.UUID `gorm:"type:uuid"`
	CreatedAt time.Time
}

type LoyaltyBalance struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	Points    int       `gorm:"not null;default:0"`
	UpdatedAt time.Time
}

type Subscription struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID    uuid.UUID `gorm:"type:uuid;not null;index"`
	MerchantID    uuid.UUID `gorm:"type:uuid;not null"`
	Vertical      string    `gorm:"not null"` // gas
	Cadence       string    `gorm:"not null"` // weekly|biweekly|monthly
	AddressID     uuid.UUID `gorm:"type:uuid;not null"`
	PaymentMethod string    `gorm:"not null"`               // wallet|card
	Status        string    `gorm:"default:'active'"`       // active|paused|cancelled
	PausedReason  *string   `gorm:"type:text;default:null"` // customer|dunning|system — nil for never-paused rows
	NextChargeAt  time.Time
	DunningCount  int `gorm:"default:0"`
	// Gas-specific fields (Phase 5)
	CylinderSpecID        *uuid.UUID `gorm:"type:uuid;default:null"`
	GasMode               *string    `gorm:"type:text;default:null"` // swap|refill|new_cylinder
	PredictedRunoutAt     *time.Time `gorm:"default:null"`
	AvgDaysBetweenRefills *float64   `gorm:"default:null"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// LPGPriceIndex is an append-only log of LPG price observations per region.
// The row with the greatest effective_at <= now is the live price.
// Insert-only — the DB rules no-op UPDATE/DELETE.
type LPGPriceIndex struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Region         string    `gorm:"not null;default:'Lagos'"`
	PricePerKgKobo int64     `gorm:"not null"`
	Source         string    `gorm:"not null"`
	EffectiveAt    time.Time `gorm:"not null;index"`
	UpdatedBy      uuid.UUID `gorm:"type:uuid;not null"`
	CreatedAt      time.Time
}

func (LPGPriceIndex) TableName() string { return "lpg_price_index" }

// ── Phase 8: Community ────────────────────────────────────────────────────────

type Campaign struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatorID       uuid.UUID `gorm:"type:uuid;not null"`
	Title           string    `gorm:"not null"`
	Description     string
	GoalKobo        int64     `gorm:"not null"`
	EscrowAccountID uuid.UUID `gorm:"type:uuid;not null"`
	Status          string    `gorm:"default:'active'"` // active|completed|cancelled
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CampaignContribution struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CampaignID    uuid.UUID `gorm:"type:uuid;not null;index"`
	ContributorID uuid.UUID `gorm:"type:uuid;not null"`
	AmountKobo    int64     `gorm:"not null"`
	JournalID     uuid.UUID `gorm:"type:uuid;not null"`
	CreatedAt     time.Time
}

type CateringRFQ struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID  uuid.UUID `gorm:"type:uuid;not null;index"`
	Description string    `gorm:"not null"`
	EventDate   time.Time
	Status      string `gorm:"default:'open'"` // open|quoted|accepted|cancelled
	CreatedAt   time.Time
}

type CateringQuote struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RFQID      uuid.UUID `gorm:"type:uuid;not null;index"`
	MerchantID uuid.UUID `gorm:"type:uuid;not null"`
	AmountKobo int64     `gorm:"not null"`
	Note       *string
	Status     string `gorm:"default:'pending'"` // pending|accepted|rejected
	CreatedAt  time.Time
}

// CancellationRule — config table keyed by order status at cancel time.
type CancellationRule struct {
	ID                     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Vertical               string    `gorm:"not null"` // food|gas|grocery|pharmacy|package|*
	OrderStatusAtCancel    string    `gorm:"not null"`
	MerchantCompKobo       int64     `gorm:"default:0"` // fixed comp or 0 = use percentage
	MerchantCompPct        float64   `gorm:"default:0"` // % of subtotal
	RiderCompPctOfDelivery float64   `gorm:"default:0"` // % of delivery fee per km ridden
	FullRefund             bool      `gorm:"default:false"`
}

// AdminAuditLog records every admin action that mutates user/order/escrow state.
// Append-only — never update or delete rows.
type AdminAuditLog struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AdminID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Action     string    `gorm:"not null"` // merchant_status_change|driver_status_change|escrow_freeze|escrow_release|fee_config_change
	TargetType string    `gorm:"not null"` // merchant_profile|driver_profile|escrow_hold|fee_config
	TargetID   uuid.UUID `gorm:"type:uuid;not null"`
	Reason     string    `gorm:"not null"`
	CreatedAt  time.Time `gorm:"index"`
}

// ProofMedia is chain-of-custody evidence for a delivery. r2_key points at a
// private R2 object (never a public URL); sha256 lets a dispute prove the file
// wasn't swapped after capture. Insert-only — never update/delete once written
// (see migration 018's DB rules).
// kind: pickup_photo|pickup_video|dropoff_photo|dropoff_video|weight_photo
// weight_photo is gas-specific: measured_kg is the scale reading at the door.
type ProofMedia struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID     uuid.UUID  `gorm:"type:uuid;not null;index"`
	StopID      *uuid.UUID `gorm:"type:uuid;index"` // nil for single-drop orders
	Kind        string     `gorm:"not null"`
	R2Key       string     `gorm:"not null"`
	SHA256      string     `gorm:"not null"`
	SealSerial  *string
	MeasuredKg  *float64 `gorm:"default:null"` // weight_photo only: scale reading in kg
	CapturedLat *float64
	CapturedLng *float64
	CapturedAt  time.Time `gorm:"not null"`
	CapturedBy  uuid.UUID `gorm:"type:uuid;not null"` // the driver
	CreatedAt   time.Time
}

// OrderReview is a post-delivery rating + comment left by the customer.
// One review per order per reviewee_type — enforced by DB unique constraint.
type OrderReview struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID      uuid.UUID `gorm:"type:uuid;not null;index"`
	ReviewerID   uuid.UUID `gorm:"type:uuid;not null"`
	RevieweeID   uuid.UUID `gorm:"type:uuid;not null;index"`
	RevieweeType string    `gorm:"type:varchar(10);not null"` // driver|merchant
	Rating       int       `gorm:"not null"`                  // 1-5
	Comment      *string   `gorm:"type:text"`
	CreatedAt    time.Time
}

// DriverBadge is a milestone badge awarded to a driver by the platform.
type DriverBadge struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DriverID          uuid.UUID `gorm:"type:uuid;not null;index"`
	BadgeType         string    `gorm:"type:varchar(30);not null"` // first_delivery|10_deliveries|50_deliveries|100_deliveries|top_rated|zero_complaints
	OrderCountAtAward int       `gorm:"not null;default:0"`
	AwardedAt         time.Time `gorm:"not null"`
}

// FeeConfig is a versioned, append-only pricing configuration row per vertical.
// The row with the greatest effective_at <= now is live; settlement pins to the
// row effective at order creation. Insert-only — the DB rules no-op UPDATE/DELETE.
type FeeConfig struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Vertical         string    `gorm:"not null"                                      json:"vertical"` // food|grocery|pharmacy|gas|package
	BaseFeeKobo      int64     `gorm:"not null"                                      json:"baseFeeKobo"`
	PerKmKobo        int64     `gorm:"not null;default:0"                            json:"perKmKobo"`
	PerKgKobo        int64     `gorm:"not null;default:0"                            json:"perKgKobo"`
	PerStopKobo      int64     `gorm:"not null;default:0"                            json:"perStopKobo"`
	ServicePct       float64   `gorm:"not null"                                      json:"servicePct"`
	MerchantTakeRate float64   `gorm:"not null"                                      json:"merchantTakeRate"`
	DriverTakeRate   float64   `gorm:"not null"                                      json:"driverTakeRate"`
	PlatformTakeRate float64   `gorm:"not null"                                      json:"platformTakeRate"`
	FuelPriceRefKobo int64     `gorm:"not null;default:0"                            json:"fuelPriceRefKobo"`
	EffectiveAt      time.Time `gorm:"not null;index"                                json:"effectiveAt"`
	UpdatedBy        uuid.UUID `gorm:"type:uuid;not null"                            json:"updatedBy"`
	Reason           string    `gorm:"not null;default:''"                           json:"reason"`
	CreatedAt        time.Time `json:"createdAt"`
}
