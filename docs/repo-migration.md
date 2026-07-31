# Repo Migration: Remove `s.db` from Service Layer

## The Rule
Services must never call `s.db.*` directly. Every DB call goes through a repo interface.
The only acceptable `*gorm.DB` in a service is as a parameter inside a `Transaction` callback.

## The Pattern (identical every time)

**Step 1 — `repo/{name}.go`**
```go
type XxxRepo interface { /* one method per s.db.* call */ }
type xxxRepo struct{ db *gorm.DB }
func NewXxxRepo(db *gorm.DB) XxxRepo { return &xxxRepo{db: db} }
// implement every method
```

**Step 2 — Update the service**
- Replace `db *gorm.DB` field with `repo XxxRepo`
- Update constructor: drop `db *gorm.DB`, add `repo XxxRepo`
- Replace every `s.db.*` call with `s.repo.*`
- If transactions needed: add `Transaction(ctx, fn func(*gorm.DB) error) error` to the repo interface

**Step 3 — `cmd/server/main.go`**
- Add `xxxRepo := repo.NewXxxRepo(gormDB)`
- Pass it into the service constructor

**Step 4 — Verify**
```
go build ./... && go vet ./... && go test ./...
```
Must be clean before moving to the next service.

---

## Existing repos to reuse (do NOT duplicate these)

| Repo | Key methods |
|------|-------------|
| `repo.OrderRepo` | `FindByID`, `FindMerchant`, `FindAddress`, `FindByIDWithItems`, `Transaction`, `CreateTx`, `SaveTx`, `CreateEventTx` |
| `repo.UserRepo` | `FindByID`, `FindPIN`, `FindDriverProfile`, `FindMerchantProfile`, `UpdateMerchantProfile`, `UpdateDriverProfile` |
| `repo.LedgerRepo` | `LockEscrowHold`, `SaveEscrowHold` |
| `repo.DispatchRepo` | `NearbyDrivers`, `CreateOffer`, `UpsertDriverLocation`, `CreateKYCCheck`, `FindKYCCheck`, `SaveKYCCheck`, `ListPendingKYCChecks`, `CreateKYCDocument` |
| `repo.TierRepo` | `LockTier`, `SaveTier`, `GetTier` |
| `repo.MerchantRepo` | `FindByUserID`, `FindBankAccount` |

---

## Service 1 — `loyalty.go` ✦ START HERE

**Current violations in `service/loyalty.go`:**
```go
s.db.WithContext(ctx).Where("user_id = ?", userID).First(&b)          // GetBalance
s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { ... })    // Redeem
s.db.WithContext(ctx).Where(...).Order(...).Limit(...).Find(&events)   // History
// Award already takes *gorm.DB tx — keep that signature, it's called from other services' txns
```

**Create `repo/loyalty.go`:**
```go
package repo

import (
    "context"
    "github.com/google/uuid"
    "github.com/speedplus/api/internal/model"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"
)

type LoyaltyRepo interface {
    FindBalance(ctx context.Context, userID uuid.UUID) (*model.LoyaltyBalance, error)
    Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
    LockBalance(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*model.LoyaltyBalance, error)
    DeductBalanceTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID, points int) error
    ListEvents(ctx context.Context, userID uuid.UUID, limit int) ([]model.LoyaltyEvent, error)
}

type loyaltyRepo struct{ db *gorm.DB }
func NewLoyaltyRepo(db *gorm.DB) LoyaltyRepo { return &loyaltyRepo{db: db} }

func (r *loyaltyRepo) FindBalance(ctx context.Context, userID uuid.UUID) (*model.LoyaltyBalance, error) {
    var b model.LoyaltyBalance
    err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&b).Error
    return &b, err
}

func (r *loyaltyRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
    return r.db.WithContext(ctx).Transaction(fn)
}

func (r *loyaltyRepo) LockBalance(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*model.LoyaltyBalance, error) {
    var b model.LoyaltyBalance
    err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
        Where("user_id = ?", userID).First(&b).Error
    return &b, err
}

func (r *loyaltyRepo) DeductBalanceTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID, points int) error {
    return tx.WithContext(ctx).Exec(
        `UPDATE loyalty_balances SET points = points - ?, updated_at = NOW() WHERE user_id = ?`,
        points, userID,
    ).Error
}

func (r *loyaltyRepo) ListEvents(ctx context.Context, userID uuid.UUID, limit int) ([]model.LoyaltyEvent, error) {
    var events []model.LoyaltyEvent
    err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&events).Error
    return events, err
}
```

**Update `service/loyalty.go`:**
- Replace `db *gorm.DB` with `repo repo.LoyaltyRepo`
- `NewLoyaltyService(r repo.LoyaltyRepo) *LoyaltyService`
- `Award`: keep `tx *gorm.DB` parameter — it's called from inside other services' transactions. The `tx.WithContext(ctx).Create(...)` and `tx.WithContext(ctx).Exec(...)` calls inside Award are fine because the tx is passed in from outside.
- `GetBalance`: `s.repo.FindBalance(ctx, userID)` — return 0, nil on not-found error
- `Redeem`: `s.repo.Transaction(ctx, func(tx *gorm.DB) error { b, err := s.repo.LockBalance(ctx, tx, userID); ...; return s.repo.DeductBalanceTx(ctx, tx, userID, points) })`
- `History`: `s.repo.ListEvents(ctx, userID, limit)`
- Remove `gorm.io/gorm` and `gorm.io/gorm/clause` imports

**`main.go`:**
```go
loyaltyRepo := repo.NewLoyaltyRepo(gormDB)
loyaltySvc := service.NewLoyaltyService(loyaltyRepo)
```

---

## Service 2 — `ussd.go`

**Current violations:**
```go
s.db.WithContext(ctx).Create(&intent)                              // PaymentIntent
s.db.WithContext(ctx).Create(ui)                                   // USSDIntent
s.db.WithContext(ctx).Where("id = ? AND user_id = ?").First(&ui)  // GetIntent
```

**Create `repo/ussd.go`:**
```go
type USSDRepo interface {
    CreatePaymentIntent(ctx context.Context, intent *model.PaymentIntent) error
    CreateIntent(ctx context.Context, intent *model.USSDIntent) error
    FindIntent(ctx context.Context, id, userID uuid.UUID) (*model.USSDIntent, error)
}
```
Implement each with the corresponding `r.db.WithContext(ctx).*` call.

**Update `service/ussd.go`:**
- Replace `db *gorm.DB` with `repo repo.USSDRepo`
- `NewUSSDService(r repo.USSDRepo, monnify monnifyUSSDProvider)`
- Replace the three `s.db.*` calls with `s.repo.*`
- Remove `gorm.io/gorm` import

**`main.go`:**
```go
ussdRepo := repo.NewUSSDRepo(gormDB)
ussdSvc := service.NewUSSDService(ussdRepo, monnifyProvider)
```

---

## Service 3 — `gift_card.go`

**Current violations:**
```go
s.db.WithContext(ctx).Create(gc)                                          // Issue
s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { ... })       // Redeem
// inside tx: Clauses(Locking).Where(...).First(&gc), tx.Save(&gc)
```

**Create `repo/gift_card.go`:**
```go
type GiftCardRepo interface {
    Create(ctx context.Context, gc *model.GiftCard) error
    Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
    LockByCodeHash(ctx context.Context, tx *gorm.DB, codeHash string) (*model.GiftCard, error)
    SaveTx(ctx context.Context, tx *gorm.DB, gc *model.GiftCard) error
}
```
- `LockByCodeHash`: `tx.WithContext(ctx).Clauses(clause.Locking{Strength:"UPDATE"}).Where("code_hash = ? AND redeemed_by IS NULL", codeHash).First(&gc)`

**Update `service/gift_card.go`:**
- Replace `db *gorm.DB` with `repo repo.GiftCardRepo`
- `NewGiftCardService(r repo.GiftCardRepo, ledger *LedgerService)`
- `Issue`: `s.repo.Create(ctx, gc)`
- `Redeem`: `s.repo.Transaction(ctx, func(tx *gorm.DB) error { gc, err := s.repo.LockByCodeHash(ctx, tx, codeHash); ...; return s.repo.SaveTx(ctx, tx, gc) })`
- Remove `gorm.io/gorm` and `gorm.io/gorm/clause` imports

**`main.go`:**
```go
giftCardRepo := repo.NewGiftCardRepo(gormDB)
giftCardSvc := service.NewGiftCardService(giftCardRepo, ledgerSvc)
```

---

## Service 4 — `referral.go`

**Current violations:**
```go
s.db.WithContext(ctx).First(&referrer, referrerID)                          // Record
s.db.WithContext(ctx).First(&referee, refereeID)                            // Record
s.db.WithContext(ctx).Where("user_id = ?", referrerID).First(&tier)         // Record
s.db.WithContext(ctx).Create(&ref)                                           // Record
s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { ... })          // SettleCompletedOrder
// inside tx: Where("referee_id = ? AND reward_paid_at IS NULL").First(&ref)
// inside tx: tx.Model(&ref).Update("reward_paid_at", gorm.Expr("NOW()"))
```

**Note:** `repo.UserRepo` already has `FindByID` — inject it instead of duplicating user lookups.

**Create `repo/referral.go`:**
```go
type ReferralRepo interface {
    FindTrustTier(ctx context.Context, userID uuid.UUID) (*model.UserTrustTier, error)
    CreateReferral(ctx context.Context, r *model.Referral) error
    Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
    FindUnpaidReferral(ctx context.Context, tx *gorm.DB, refereeID uuid.UUID) (*model.Referral, error)
    MarkReferralPaidTx(ctx context.Context, tx *gorm.DB, ref *model.Referral) error
}
```
- `FindTrustTier`: `r.db.WithContext(ctx).Where("user_id = ?", userID).First(&tier)`
- `FindUnpaidReferral`: `tx.WithContext(ctx).Where("referee_id = ? AND reward_paid_at IS NULL", refereeID).First(&ref)`
- `MarkReferralPaidTx`: `tx.WithContext(ctx).Model(ref).Update("reward_paid_at", gorm.Expr("NOW()"))`

**Update `service/referral.go`:**
- Replace `db *gorm.DB` with `repo repo.ReferralRepo` and add `users repo.UserRepo`
- `NewReferralService(r repo.ReferralRepo, users repo.UserRepo, ledger *LedgerService, loyalty *LoyaltyService)`
- `Record`: replace `s.db.WithContext(ctx).First(&referrer, referrerID)` → `s.users.FindByID(ctx, referrerID)` (ignore error = unknown referrer); same for referee; tier check → `s.repo.FindTrustTier`; create → `s.repo.CreateReferral`
- `SettleCompletedOrder`: `s.repo.Transaction(ctx, func(tx *gorm.DB) error { ref, err := s.repo.FindUnpaidReferral(ctx, tx, refereeID); ...; return s.repo.MarkReferralPaidTx(ctx, tx, ref) })`
- Remove `gorm.io/gorm` import

**`main.go`:**
```go
referralRepo := repo.NewReferralRepo(gormDB)
referralSvc := service.NewReferralService(referralRepo, userRepo, ledgerSvc, loyaltySvc)
```

---

## Service 5 — `proof_media.go`

**Current violations:**
```go
s.db.WithContext(ctx).First(&order, orderID)                              // mustBeAssignedDriver (×2)
s.db.WithContext(ctx).Create(media)                                       // ConfirmUpload
s.db.WithContext(ctx).Where("order_id = ?").Order(...).Find(&rows)        // GetMediaForOrder
```

**Note:** `repo.OrderRepo` already has `FindByID` — inject it instead of a new order lookup.

**Create `repo/proof_media.go`:**
```go
type ProofMediaRepo interface {
    Create(ctx context.Context, m *model.ProofMedia) error
    ListByOrder(ctx context.Context, orderID uuid.UUID) ([]model.ProofMedia, error)
}
```
- `ListByOrder`: `r.db.WithContext(ctx).Where("order_id = ?", orderID).Order("captured_at ASC").Find(&rows)`

**Update `service/proof_media.go`:**
- Replace `db *gorm.DB` with `repo repo.ProofMediaRepo` and add `orders repo.OrderRepo`
- `NewProofMediaService(r repo.ProofMediaRepo, orders repo.OrderRepo, r2 *storage.R2Client)`
- `mustBeAssignedDriver`: `s.orders.FindByID(ctx, orderID)`
- `ConfirmUpload`: `s.repo.Create(ctx, media)`
- `GetMediaForOrder`: order lookup → `s.orders.FindByID`; list → `s.repo.ListByOrder(ctx, orderID)`
- Remove `gorm.io/gorm` import

**`main.go`:**
```go
proofMediaRepo := repo.NewProofMediaRepo(gormDB)
proofMediaSvc := service.NewProofMediaService(proofMediaRepo, orderRepo, r2Client)
```

---

## Service 6 — `kyc.go`

**Current violations:**
```go
s.db.WithContext(ctx).Create(&doc)                                         // RecordDocumentUpload
s.db.WithContext(ctx).Create(&check) / Create(&failedCheck)               // SubmitCheck
s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { ... })        // AdminApprove
// inside tx: tx.First(&check, checkID), tx.Save(&check), tryActivateProfile
// tryActivateProfile: tx.First(&user), tx.Where(...).First(&dp/mp), tx.Save(&dp/mp)
s.db.WithContext(ctx).First(&check, checkID)                               // AdminReject
s.db.WithContext(ctx).Save(&check)                                         // AdminReject
s.db.WithContext(ctx).Where(...).Find(&checks)                             // QueueForAdmin
```

**Note:** `repo.DispatchRepo` already has `CreateKYCCheck`, `FindKYCCheck`, `SaveKYCCheck`, `ListPendingKYCChecks`, `CreateKYCDocument`. Use those — do NOT create a new KYCRepo. Just inject `repo.DispatchRepo` into `KYCService`.

**Also needed (not in DispatchRepo yet) — add to DispatchRepo interface + implementation:**
```go
Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
FindUserTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*model.User, error)
FindDriverProfileTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*model.DriverProfile, error)
SaveDriverProfileTx(ctx context.Context, tx *gorm.DB, dp *model.DriverProfile) error
FindMerchantProfileTx(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*model.MerchantProfile, error)
SaveMerchantProfileTx(ctx context.Context, tx *gorm.DB, mp *model.MerchantProfile) error
```

**Update `service/kyc.go`:**
- Replace `db *gorm.DB` with `repo repo.DispatchRepo`
- `NewKYCService(r repo.DispatchRepo, provider kyc.Provider)`
- `RecordDocumentUpload`: `s.repo.CreateKYCDocument(ctx, &doc)`
- `SubmitCheck`: `s.repo.CreateKYCCheck(ctx, &check)` / `s.repo.CreateKYCCheck(ctx, &failedCheck)`
- `AdminApprove`: `s.repo.Transaction(ctx, func(tx *gorm.DB) error { ... })` — inside use `s.repo.FindKYCCheck`, `s.repo.SaveKYCCheck`, then `tryActivateProfile` using the new Tx methods
- `AdminReject`: `s.repo.FindKYCCheck` + `s.repo.SaveKYCCheck`
- `QueueForAdmin`: `s.repo.ListPendingKYCChecks(ctx, page*pageSize, pageSize)`
- `tryActivateProfile`: use `s.repo.FindUserTx`, `s.repo.FindDriverProfileTx`, `s.repo.SaveDriverProfileTx`, etc.
- Remove `gorm.io/gorm` import

**`main.go`:** No change needed — `dispatchRepo` is already created. Update:
```go
kycSvc := service.NewKYCService(dispatchRepo, kycProvider)
```
Remove the `gormDB` argument.

---

## Service 7 — `fee_config.go`

**Current violations:**
```go
s.db.WithContext(ctx).Where("vertical = ? AND effective_at <= ?").Order(...).First(&row)  // GetFeesAt
s.db.WithContext(ctx).Raw(`SELECT DISTINCT ON (vertical)...`).Scan(&rows)                 // List
s.db.WithContext(ctx).Where("vertical = ? AND effective_at <= NOW()").Order(...).First(&p) // Upsert (prev lookup)
s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { tx.Create(row); tx.Create(&auditLog) }) // Upsert
```

**Create `repo/fee_config.go`:**
```go
type FeeConfigRepo interface {
    FindLatestByVertical(ctx context.Context, vertical string, at time.Time) (*model.FeeConfig, error)
    ListLatestPerVertical(ctx context.Context) ([]model.FeeConfig, error)
    Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
    CreateTx(ctx context.Context, tx *gorm.DB, row *model.FeeConfig) error
    CreateAuditLogTx(ctx context.Context, tx *gorm.DB, log *model.AdminAuditLog) error
}
```
- `FindLatestByVertical`: `r.db.WithContext(ctx).Where("vertical = ? AND effective_at <= ?", vertical, at).Order("effective_at DESC").First(&row)`
- `ListLatestPerVertical`: the `DISTINCT ON` raw query
- `CreateTx` / `CreateAuditLogTx`: `tx.WithContext(ctx).Create(...)`

**Update `service/fee_config.go`:**
- Replace `db *gorm.DB` with `repo repo.FeeConfigRepo` (keep `mu sync.RWMutex` and `cache map[string]cachedFees`)
- `NewFeeConfigService(r repo.FeeConfigRepo) *FeeConfigService`
- `GetFeesAt`: `s.repo.FindLatestByVertical(ctx, vertical, at)` — on error return `defaultFees(vertical)`
- `List`: `s.repo.ListLatestPerVertical(ctx)` then append defaults for missing verticals
- `Upsert`: prev lookup → `s.repo.FindLatestByVertical(ctx, in.Vertical, time.Now())`; transaction → `s.repo.Transaction(ctx, func(tx *gorm.DB) error { s.repo.CreateTx(ctx, tx, row); s.repo.CreateAuditLogTx(ctx, tx, &auditLog) })`
- Remove `gorm.io/gorm` import

**`main.go`:**
```go
feeConfigRepo := repo.NewFeeConfigRepo(gormDB)
feeConfigSvc := service.NewFeeConfigService(feeConfigRepo)
```

---

## Service 8 — `payment_link.go`

**Current violations:**
```go
s.db.WithContext(ctx).Create(pl)                                                    // Create
s.db.WithContext(ctx).Where("slug = ? AND status = 'pending'").First(&pl)          // Get
s.db.WithContext(ctx).Model(&pl).Update("status", "expired")                       // Get (side-effect)
s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { ... })                 // PayByWallet
// inside tx: Where("key = ?").First(&existing), Clauses(Locking).Where(...).First(&pl),
//            tx.Model(&pl).Update("status","expired"), tx.Save(&pl), tx.Create(&IdempotencyKey)
s.db.WithContext(ctx).Create(&intent)                                               // InitiateGuestPayment
s.db.WithContext(ctx).Model(pl).Updates(...)                                        // InitiateGuestPayment
```

**Create `repo/payment_link.go`:**
```go
type PaymentLinkRepo interface {
    Create(ctx context.Context, pl *model.PaymentLink) error
    FindPendingBySlug(ctx context.Context, slug string) (*model.PaymentLink, error)
    ExpireLink(ctx context.Context, id uuid.UUID) error
    Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
    FindIdempotencyKey(ctx context.Context, tx *gorm.DB, key string) (*model.IdempotencyKey, error)
    LockPendingBySlugTx(ctx context.Context, tx *gorm.DB, slug string) (*model.PaymentLink, error)
    ExpireLinkTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) error
    SaveLinkTx(ctx context.Context, tx *gorm.DB, pl *model.PaymentLink) error
    CreateIdempotencyKeyTx(ctx context.Context, tx *gorm.DB, k *model.IdempotencyKey) error
    CreatePaymentIntent(ctx context.Context, intent *model.PaymentIntent) error
    UpdateLinkProviderRef(ctx context.Context, id uuid.UUID, ref, email string) error
}
```

**Update `service/payment_link.go`:**
- Replace `db *gorm.DB` with `repo repo.PaymentLinkRepo`
- `NewPaymentLinkService(r repo.PaymentLinkRepo, ledger *LedgerService, provider payment.Provider, email linkEmailSender, users repo.UserRepo)`
- Map each `s.db.*` call to the corresponding `s.repo.*` method
- The `PayByWallet` transaction body still uses `s.ledger.EnsureWallet`, `s.ledger.journal`, `s.ledger.adjustBalance` — those are fine, they take a `*gorm.DB tx`
- Remove `gorm.io/gorm` and `gorm.io/gorm/clause` imports

**`main.go`:**
```go
paymentLinkRepo := repo.NewPaymentLinkRepo(gormDB)
paymentLinkSvc := service.NewPaymentLinkService(paymentLinkRepo, ledgerSvc, paystackProvider, emailClient, userRepo)
```

---

## Service 9 — `pricing.go`

**No new repo needed.** `repo.OrderRepo` already has `CreateQuote` (via `CreateTx`... actually check: it has `CreateQuote(ctx, q)` directly). Looking at the interface: yes, `CreateQuote`, `FindQuote` (as `FindByID` on quotes — actually `FindQuote(ctx, id)`), and `MarkQuoteUsed` are all on `OrderRepo`.

**Update `service/pricing.go`:**
- Replace `db *gorm.DB` with `orders repo.OrderRepo`
- `NewPricingService(orders repo.OrderRepo, cfg *config.Config, osrmURL string, feeConfigs *FeeConfigService)`
- `Quote`: `s.orders.CreateQuote(ctx, quote)` instead of `s.db.WithContext(ctx).Create(quote)`
- `ValidateQuote`: `s.orders.FindQuote(ctx, quoteID)` instead of `s.db.WithContext(ctx).First(&q, quoteID)`
- `MarkQuoteUsed`: `s.orders.MarkQuoteUsed(ctx, quoteID)` instead of `s.db.WithContext(ctx).Model(...).Update(...)`
- `QuoteMultiStop`: same as `Quote` — `s.orders.CreateQuote(ctx, quote)`
- Remove `gorm.io/gorm` import

**`main.go`:**
```go
// orderRepo already exists
pricingSvc := service.NewPricingService(orderRepo, cfg, cfg.OSRMURL, feeConfigSvc)
```
Remove the `gormDB` argument.

---

## Service 10 — `tier.go`

**Current violations:**
```go
s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { ... })  // RecordCompletion, RecordFraudFlag, RecordPayOnArrivalFailure
s.db.WithContext(ctx).Model(&model.Order{}).Where(...).Count(&active) // CanUsePayOnArrival
```

**`repo.TierRepo` already exists** with `LockTier`, `SaveTier`, `GetTier`. Add `Transaction` to it:

In `repo/tier.go`, add to the interface:
```go
Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
CountActivePODOrders(ctx context.Context, userID uuid.UUID) (int64, error)
```
And implement:
```go
func (r *tierRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
    return r.db.WithContext(ctx).Transaction(fn)
}
func (r *tierRepo) CountActivePODOrders(ctx context.Context, userID uuid.UUID) (int64, error) {
    var count int64
    err := r.db.WithContext(ctx).Model(&model.Order{}).
        Where("customer_id = ? AND payment_method = 'pay_on_arrival' AND status NOT IN ('delivered','cancelled','refunded')", userID).
        Count(&count).Error
    return count, err
}
```

**Update `service/tier.go`:**
- Remove `db *gorm.DB` field (keep `repo repo.TierRepo`)
- `NewTierService(r repo.TierRepo) *TierService` — drop `db *gorm.DB`
- `RecordCompletion`, `RecordFraudFlag`, `RecordPayOnArrivalFailure`: replace `s.db.WithContext(ctx).Transaction(...)` → `s.repo.Transaction(ctx, ...)`
- `CanUsePayOnArrival`: replace `s.db.WithContext(ctx).Model(&model.Order{})...Count(...)` → `s.repo.CountActivePODOrders(ctx, userID)`
- Remove `gorm.io/gorm` import

**`main.go`:**
```go
// tierRepo already exists
tierSvc := service.NewTierService(tierRepo)  // drop gormDB argument
```

---

## Service 11 — `subscription.go`

**Current violations:**
```go
s.db.WithContext(ctx).Create(sub)                                          // Create
s.db.WithContext(ctx).Model(&model.Subscription{}).Where(...).Update(...)  // Pause, Cancel
s.db.WithContext(ctx).Where("status = 'active' AND next_charge_at <= NOW()").Find(&subs) // ProcessDue
s.db.WithContext(ctx).Model(&sub).Updates(...)                             // ProcessDue (dunning)
s.db.WithContext(ctx).Model(&sub).Update("status", "paused")              // ProcessDue (dunning)
s.db.WithContext(ctx).Model(&sub).Updates(...)                             // ProcessDue (success)
// chargeOne:
s.db.WithContext(ctx).Where(...).Order(...).First(&prod)                   // product lookup
s.db.WithContext(ctx).First(&product, productID)                           // product lookup
s.db.WithContext(ctx).First(&merchant, sub.MerchantID)                    // merchant lookup
s.db.WithContext(ctx).First(&addr, sub.AddressID)                         // address lookup
// UpdateBurnRates:
s.db.WithContext(ctx).Raw(`...`).Scan(&rows)                               // burn rate query
s.db.WithContext(ctx).Model(&model.Subscription{}).Where(...).Updates(...) // burn rate update
// LPG:
s.db.WithContext(ctx).Where("region = ? AND effective_at <= NOW()").Order(...).First(&row) // GetLiveLPGPrice
s.db.WithContext(ctx).Where("region = ? AND effective_at <= NOW()").Order(...).First(&prev) // RecordLPGPrice
s.db.WithContext(ctx).Create(row)                                          // RecordLPGPrice
```

**Note:** `repo.OrderRepo` already has `FindMerchant` and `FindAddress`. Inject it.

**Create `repo/subscription.go`:**
```go
type SubscriptionRepo interface {
    Create(ctx context.Context, sub *model.Subscription) error
    PauseByCustomer(ctx context.Context, subID, customerID uuid.UUID) error
    CancelByCustomer(ctx context.Context, subID, customerID uuid.UUID) error
    ListDue(ctx context.Context) ([]model.Subscription, error)
    UpdateDunning(ctx context.Context, subID uuid.UUID, dunningCount int, nextCharge time.Time) error
    PauseByID(ctx context.Context, subID uuid.UUID) error
    UpdateNextCharge(ctx context.Context, subID uuid.UUID, nextCharge time.Time) error
    FindCheapestAvailableProduct(ctx context.Context, merchantID uuid.UUID) (*model.Product, error)
    FindProduct(ctx context.Context, id uuid.UUID) (*model.Product, error)
    BurnRateStats(ctx context.Context) ([]SubscriptionBurnRow, error)
    UpdateBurnRate(ctx context.Context, customerID string, avgDays float64, predictedRunout time.Time) error
    GetLiveLPGPrice(ctx context.Context, region string) (*model.LPGPriceIndex, error)
    GetPrevLPGPrice(ctx context.Context, region string) (*model.LPGPriceIndex, error)
    CreateLPGPrice(ctx context.Context, row *model.LPGPriceIndex) error
}

type SubscriptionBurnRow struct {
    CustomerID          string
    AvgDaysBetweenFills float64
    LastDeliveredAt     time.Time
}
```
Implement each method with the corresponding `r.db.WithContext(ctx).*` call. The `BurnRateStats` method wraps the raw SQL query from `UpdateBurnRates`.

**Update `service/subscription.go`:**
- Replace `db *gorm.DB` with `repo repo.SubscriptionRepo` and add `orders repo.OrderRepo`
- `NewSubscriptionService(r repo.SubscriptionRepo, orders repo.OrderRepo, orderSvc *OrderService, ledger *LedgerService)`
- Map every `s.db.*` call to `s.repo.*` or `s.orders.*` (for merchant/address lookups)
- `chargeOne`: product lookups → `s.repo.FindCheapestAvailableProduct` / `s.repo.FindProduct`; merchant → `s.orders.FindMerchant`; address → `s.orders.FindAddress`
- Remove `gorm.io/gorm` import

**`main.go`:**
```go
subscriptionRepo := repo.NewSubscriptionRepo(gormDB)
subscriptionSvc := service.NewSubscriptionService(subscriptionRepo, orderRepo, orderSvc, ledgerSvc)
```

---

## Service 12 — `dispatch.go`

**`repo.DispatchRepo` already exists** with `NearbyDrivers`, `CreateOffer`, `UpsertDriverLocation`, `AssignDriverToOrder`. Add the missing methods:

In `repo/dispatch.go`, add to the interface:
```go
Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
AcceptOfferTx(ctx context.Context, tx *gorm.DB, offerID, driverID uuid.UUID) (rowsAffected int64, err error)
FindOfferTx(ctx context.Context, tx *gorm.DB, offerID uuid.UUID) (*model.DeliveryOffer, error)
UpdateOfferStatus(ctx context.Context, offerID uuid.UUID, status string) error
LockOrderTx(ctx context.Context, tx *gorm.DB, orderID uuid.UUID) (*model.Order, error)
SaveOrderTx(ctx context.Context, tx *gorm.DB, o *model.Order) error
UpdateRunStatus(ctx context.Context, runID uuid.UUID, driverID *uuid.UUID, status string) error
```
And implement each.

**Update `service/dispatch.go`:**
- Replace `db *gorm.DB` with `repo repo.DispatchRepo`
- `NewDispatchService(r repo.DispatchRepo) *DispatchService`
- `Dispatch`: `s.repo.NearbyDrivers(...)` for the KNN query; `s.repo.CreateOffer(ctx, &offer)` for each candidate
- `AcceptOffer`: `s.repo.Transaction(ctx, func(tx *gorm.DB) error { rowsAffected, err := s.repo.AcceptOfferTx(...); offer, err := s.repo.FindOfferTx(...); return s.repo.AssignDriverToOrder(ctx, tx, offer.OrderID, driverID) })`
- `UpdateLocation`: `s.repo.UpsertDriverLocation(ctx, driverID, lat, lng, heading)`
- `ExpireOffers`: `s.repo.UpdateOfferStatus(ctx, ...)` — or add a dedicated `ExpireStaleOffers` (already exists on DispatchRepo)
- `RejectOffer`: `s.repo.UpdateOfferStatus(ctx, offerID, "rejected")`
- `ManualAssign`: `s.repo.Transaction(ctx, func(tx *gorm.DB) error { order, err := s.repo.LockOrderTx(...); ...; return s.repo.SaveOrderTx(ctx, tx, &order) })`
- Remove `gorm.io/gorm` and `gorm.io/gorm/clause` imports

**`main.go`:**
```go
// dispatchRepo already exists
dispatchSvc := service.NewDispatchService(dispatchRepo)  // drop gormDB argument
```

---

## Service 13 — `admin.go`

**Current violations:**
```go
// ListMerchants: s.db.WithContext(ctx).Model(&model.MerchantProfile{}).Select(...).Scan(&rows)
// SetMerchantStatus: s.db.WithContext(ctx).Transaction(...) — tx.Clauses(Locking).First(&mp), tx.Save(&mp), tx.Create(&auditLog)
// ListDrivers: s.db.WithContext(ctx).Model(&model.DriverProfile{}).Select(...).Scan(&rows)
// SetDriverStatus: same pattern as SetMerchantStatus
// SearchOrders: s.db.WithContext(ctx).Model(&model.Order{}).Select(...).Scan(&rows)
// GetOrderDetail: s.db.WithContext(ctx).Preload("Items").Preload("Events").First(&order)
// FreezeEscrow: s.db.WithContext(ctx).Transaction(...) — uses s.ledger.repo.LockEscrowHold/SaveEscrowHold, tx.Create(&auditLog)
// ReleaseEscrow: s.db.WithContext(ctx).Transaction(...) — same pattern + tx.First(&order)
// ListCancellationRules: s.db.WithContext(ctx).Order(...).Find(&rules)
// UpsertCancellationRule: s.db.WithContext(ctx).Where(...).Assign(...).FirstOrCreate(&rule)
// DeleteCancellationRule: s.db.WithContext(ctx).Delete(&model.CancellationRule{}, ruleID)
```

**Note:** `FreezeEscrow` and `ReleaseEscrow` already use `s.ledger.repo.LockEscrowHold` and `s.ledger.repo.SaveEscrowHold` — those are fine. The only remaining `s.db.*` calls in those methods are `tx.Create(&auditLog)` and `tx.First(&order)`.

**Create `repo/admin.go`:**
```go
type AdminRepo interface {
    ListMerchantProfiles(ctx context.Context, status string, offset, limit int) ([]MerchantRow, error)
    Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
    LockMerchantProfileTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.MerchantProfile, error)
    SaveMerchantProfileTx(ctx context.Context, tx *gorm.DB, mp *model.MerchantProfile) error
    CreateAuditLogTx(ctx context.Context, tx *gorm.DB, log *model.AdminAuditLog) error
    ListDriverProfiles(ctx context.Context, status string, offset, limit int) ([]DriverRow, error)
    LockDriverProfileTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.DriverProfile, error)
    SaveDriverProfileTx(ctx context.Context, tx *gorm.DB, dp *model.DriverProfile) error
    SearchOrders(ctx context.Context, q, status string, offset, limit int) ([]OrderSummary, error)
    FindOrderWithEvents(ctx context.Context, id uuid.UUID) (*model.Order, error)
    FindOrderTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.Order, error)
    ListCancellationRules(ctx context.Context) ([]model.CancellationRule, error)
    UpsertCancellationRule(ctx context.Context, rule model.CancellationRule) (*model.CancellationRule, error)
    DeleteCancellationRule(ctx context.Context, id uuid.UUID) error
}
```
Note: `MerchantRow`, `DriverRow`, `OrderSummary` are types defined in `service/admin.go` — move them to a shared location or keep them in the service and have the repo return `[]model.MerchantProfile` etc. and let the service map them. **Simplest approach: keep the mapping in the service, have the repo return the model types.**

Revised interface (simpler):
```go
type AdminRepo interface {
    ListMerchantProfiles(ctx context.Context, status string, offset, limit int) ([]model.MerchantProfile, error)
    Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
    LockMerchantProfileTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.MerchantProfile, error)
    SaveMerchantProfileTx(ctx context.Context, tx *gorm.DB, mp *model.MerchantProfile) error
    CreateAuditLogTx(ctx context.Context, tx *gorm.DB, log *model.AdminAuditLog) error
    ListDriverProfiles(ctx context.Context, status string, offset, limit int) ([]model.DriverProfile, error)
    LockDriverProfileTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.DriverProfile, error)
    SaveDriverProfileTx(ctx context.Context, tx *gorm.DB, dp *model.DriverProfile) error
    SearchOrders(ctx context.Context, q, status string, offset, limit int) ([]model.Order, error)
    FindOrderWithEvents(ctx context.Context, id uuid.UUID) (*model.Order, error)
    FindOrderTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.Order, error)
    ListCancellationRules(ctx context.Context) ([]model.CancellationRule, error)
    UpsertCancellationRule(ctx context.Context, rule model.CancellationRule) (*model.CancellationRule, error)
    DeleteCancellationRule(ctx context.Context, id uuid.UUID) error
}
```

**Update `service/admin.go`:**
- Replace `db *gorm.DB` with `repo repo.AdminRepo`
- `NewAdminService(r repo.AdminRepo, ledger *LedgerService)`
- Map every `s.db.*` call to `s.repo.*`
- `FreezeEscrow` / `ReleaseEscrow`: replace `s.db.WithContext(ctx).Transaction(...)` → `s.repo.Transaction(ctx, ...)`; inside use `s.repo.CreateAuditLogTx` and `s.repo.FindOrderTx`; the `s.ledger.repo.*` calls stay as-is
- Remove `gorm.io/gorm` and `gorm.io/gorm/clause` imports

**`main.go`:**
```go
adminRepo := repo.NewAdminRepo(gormDB)
adminSvc := service.NewAdminService(adminRepo, ledgerSvc)
```

---

## Service 14 — `wallet.go`

**Current violations (partial — read the full file):**
```go
s.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&existing)  // InitiateFund
s.db.WithContext(ctx).Create(&intent)                                       // InitiateFund
s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { ... })         // ProcessWebhook
// inside tx: Where("provider = ? AND event_id = ?").First(&existing), Create(&webhookEvent),
//            Where("provider_ref = ?").First(&intent), Model(&intent).Update(...),
//            EnsureWallet, CreditWallet, adjustBalance
// Transfer, EWACashout, GetBalance, GetTransactions — read the full file for the rest
```

**Create `repo/wallet.go`:**
```go
type WalletRepo interface {
    FindPaymentIntentByKey(ctx context.Context, key string) (*model.PaymentIntent, error)
    CreatePaymentIntent(ctx context.Context, intent *model.PaymentIntent) error
    Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
    FindWebhookEvent(ctx context.Context, tx *gorm.DB, provider, eventID string) (*model.WebhookEvent, error)
    CreateWebhookEventTx(ctx context.Context, tx *gorm.DB, e *model.WebhookEvent) error
    FindPaymentIntentByRefTx(ctx context.Context, tx *gorm.DB, ref string) (*model.PaymentIntent, error)
    UpdatePaymentIntentStatusTx(ctx context.Context, tx *gorm.DB, id uuid.UUID, status string) error
    // Add more as you read through the full wallet.go
}
```
Read `wallet.go` fully before writing the interface — there are more `s.db.*` calls beyond line 100.

**Update `service/wallet.go`:**
- Replace `db *gorm.DB` with `repo repo.WalletRepo`
- `NewWalletService(r repo.WalletRepo, ledger *LedgerService, pins ports.PINVerifier, provider payment.Provider, email walletEmailSender, users repo.UserRepo)`
- Map every `s.db.*` call to `s.repo.*`
- Remove `gorm.io/gorm` and `gorm.io/gorm/clause` imports

**`main.go`:**
```go
walletRepo := repo.NewWalletRepo(gormDB)
walletSvc := service.NewWalletService(walletRepo, ledgerSvc, authSvc, paystackProvider, emailClient, userRepo)
```

---

## Service 15 — `paycode.go`

**Current violations (partial — read the full file):**
```go
s.db.WithContext(ctx).First(&order, orderID)  // Generate — use s.orderRepo.FindByID instead
// More s.db.* calls throughout — read the full file
```

**Note:** `PaycodeService` already has `orderRepo repo.OrderRepo` injected. Use it for order lookups.

**Create `repo/paycode.go`:**
```go
type PaycodeRepo interface {
    Create(ctx context.Context, pc *model.Paycode) error
    FindByNonce(ctx context.Context, nonce string) (*model.Paycode, error)
    FindByID(ctx context.Context, id uuid.UUID) (*model.Paycode, error)
    Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
    // Add more as you read through the full paycode.go
}
```
Read `paycode.go` fully before writing the interface.

**Update `service/paycode.go`:**
- Replace `db *gorm.DB` with `repo repo.PaycodeRepo`
- `NewPaycodeService(r repo.PaycodeRepo, cfg *config.Config, ledger *LedgerService, orders *OrderService, tier ports.TierRecorder, email paycodeEmailSender, users repo.UserRepo, orderRepo repo.OrderRepo, deliveryCodes *DeliveryCodeService, referrals *ReferralService)`
- Replace `s.db.WithContext(ctx).First(&order, ...)` → `s.orderRepo.FindByID(ctx, orderID)`
- Replace all other `s.db.*` calls with `s.repo.*`
- Remove `gorm.io/gorm` and `gorm.io/gorm/clause` imports

**`main.go`:**
```go
paycodeRepo := repo.NewPaycodeRepo(gormDB)
paycodeSvc := service.NewPaycodeService(paycodeRepo, cfg, ledgerSvc, orderSvc, tierSvc, emailClient, userRepo, orderRepo, deliveryCodeSvc, referralSvc)
```

---

## Service 16 — `run.go`

**Current violations:**
```go
s.db.WithContext(ctx).First(&zone, zoneID)                                    // AssembleRun
s.db.WithContext(ctx).Raw(`SELECT o.id...`).Scan(&rows)                       // AssembleRun (orders in zone)
s.db.WithContext(ctx).Raw(`SELECT lat, lng FROM merchants...`).Row().Scan(...)// AssembleRun (merchant location)
s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { ... })            // AssembleRun (persist run)
// inside tx: tx.Create(run), tx.Create(&model.RunOrder{...})
s.db.WithContext(ctx).Preload("Orders").First(&run, runID)                    // DispatchRun
s.db.WithContext(ctx).First(&firstOrder, ...)                                 // DispatchRun — use orderRepo
s.db.WithContext(ctx).First(&addr, ...)                                       // DispatchRun — use orderRepo
s.db.WithContext(ctx).Model(&model.DeliveryRun{}).Where(...).Updates(...)     // DispatchRun
s.db.WithContext(ctx).Where("is_active = true").Find(&zones)                  // AssembleAllDueRuns
s.db.WithContext(ctx).Model(&model.DeliveryRun{}).Where(...).Count(&count)    // AssembleAllDueRuns
```

**Create `repo/run.go`:**
```go
type RunRepo interface {
    FindZone(ctx context.Context, id uuid.UUID) (*model.ServiceZone, error)
    FindOrdersInZoneWindow(ctx context.Context, boundary string, windowStart, windowEnd time.Time, limit int) ([]RunOrderRow, error)
    FindGasMerchantLocation(ctx context.Context, merchantID uuid.UUID) (lat, lng float64, err error)
    Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
    CreateRunTx(ctx context.Context, tx *gorm.DB, run *model.DeliveryRun) error
    CreateRunOrderTx(ctx context.Context, tx *gorm.DB, ro *model.RunOrder) error
    FindRunWithOrders(ctx context.Context, id uuid.UUID) (*model.DeliveryRun, error)
    UpdateRunStatus(ctx context.Context, runID uuid.UUID, driverID uuid.UUID, status string) error
    ListActiveZones(ctx context.Context) ([]model.ServiceZone, error)
    CountRunsForZoneWindow(ctx context.Context, zoneID uuid.UUID, windowStart time.Time) (int64, error)
}

type RunOrderRow struct {
    OrderID uuid.UUID
    Lat     float64
    Lng     float64
}
```

**Update `service/run.go`:**
- Replace `db *gorm.DB` with `repo repo.RunRepo` and add `orders repo.OrderRepo`
- `NewRunService(r repo.RunRepo, orders repo.OrderRepo, pricing *PricingService, dispatch *DispatchService)`
- `AssembleRun`: zone → `s.repo.FindZone`; orders query → `s.repo.FindOrdersInZoneWindow`; merchant location → `s.repo.FindGasMerchantLocation`; persist → `s.repo.Transaction` with `CreateRunTx`/`CreateRunOrderTx`
- `DispatchRun`: run → `s.repo.FindRunWithOrders`; first order → `s.orders.FindByID`; address → `s.orders.FindAddress`; update → `s.repo.UpdateRunStatus`
- `AssembleAllDueRuns`: zones → `s.repo.ListActiveZones`; count → `s.repo.CountRunsForZoneWindow`
- Remove `gorm.io/gorm` import

**`main.go`:**
```go
runRepo := repo.NewRunRepo(gormDB)
runSvc := service.NewRunService(runRepo, orderRepo, pricingSvc, dispatchSvc)
```

---

## Completion Checklist

After all 16 services are done:

```bash
cd apps/api
go build ./...
go vet ./...
go test ./...
grep -r 's\.db\.' internal/service/   # must return nothing
grep -r '"gorm\.io/gorm"' internal/service/ | grep -v '_test\.go'  # only Transaction callbacks
```

The last grep may still show `gorm.io/gorm` in services that have `Transaction(ctx, func(tx *gorm.DB) error)` callbacks — that's acceptable. What must be zero: any `s.db.WithContext` or `s.db.Model` or `s.db.Where` or `s.db.Raw` call.
