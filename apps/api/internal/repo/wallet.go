package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WalletRepo interface {
	FindPaymentIntentByKey(ctx context.Context, key string) (*model.PaymentIntent, error)
	CreatePaymentIntent(ctx context.Context, intent *model.PaymentIntent) error
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	FindWebhookEventTx(ctx context.Context, tx *gorm.DB, provider, eventID string) (*model.WebhookEvent, error)
	CreateWebhookEventTx(ctx context.Context, tx *gorm.DB, e *model.WebhookEvent) error
	SaveWebhookEventTx(ctx context.Context, tx *gorm.DB, e *model.WebhookEvent) error
	LockPaymentIntentByRefTx(ctx context.Context, tx *gorm.DB, ref string) (*model.PaymentIntent, error)
	SavePaymentIntentTx(ctx context.Context, tx *gorm.DB, intent *model.PaymentIntent) error
	FindIdempotencyKeyTx(ctx context.Context, tx *gorm.DB, key string) (*model.IdempotencyKey, error)
	CreateIdempotencyKeyTx(ctx context.Context, tx *gorm.DB, k *model.IdempotencyKey) error
	LockWalletBalanceTx(ctx context.Context, tx *gorm.DB, accountID uuid.UUID) (*model.WalletBalance, error)
	FindCashoutByKey(ctx context.Context, key string) (*model.CashoutRequest, error)
	CreateCashoutTx(ctx context.Context, tx *gorm.DB, c *model.CashoutRequest) error
	SaveCashoutTx(ctx context.Context, tx *gorm.DB, c *model.CashoutRequest) error
	// LockCashoutTx re-reads a cashout under SELECT ... FOR UPDATE so a reversal
	// can prove it has not already been applied. ReverseCashout runs from a
	// retrying asynq worker; without this guard a retry credits the wallet twice.
	LockCashoutTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.CashoutRequest, error)
	FindIdempotencyCashoutTx(ctx context.Context, tx *gorm.DB, key string) (*model.CashoutRequest, error)
	SumUnpaidEarnings(ctx context.Context, tx *gorm.DB, driverID uuid.UUID) (int64, error)
	ListDriversWithUnpaidEarnings(ctx context.Context) ([]uuid.UUID, error)
	FindMerchantBankAccountTx(ctx context.Context, tx *gorm.DB, merchantID uuid.UUID) (*model.MerchantBankAccount, error)
	FindDriverBankAccountTx(ctx context.Context, tx *gorm.DB, driverID uuid.UUID) (*model.DriverBankAccount, error)
}

type walletRepo struct{ db *gorm.DB }

func NewWalletRepo(db *gorm.DB) WalletRepo { return &walletRepo{db: db} }

func (r *walletRepo) FindPaymentIntentByKey(ctx context.Context, key string) (*model.PaymentIntent, error) {
	var i model.PaymentIntent
	err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&i).Error
	return &i, err
}

func (r *walletRepo) CreatePaymentIntent(ctx context.Context, intent *model.PaymentIntent) error {
	return r.db.WithContext(ctx).Create(intent).Error
}

func (r *walletRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *walletRepo) FindWebhookEventTx(ctx context.Context, tx *gorm.DB, provider, eventID string) (*model.WebhookEvent, error) {
	var e model.WebhookEvent
	err := tx.WithContext(ctx).Where("provider = ? AND event_id = ?", provider, eventID).First(&e).Error
	return &e, err
}

func (r *walletRepo) CreateWebhookEventTx(ctx context.Context, tx *gorm.DB, e *model.WebhookEvent) error {
	return tx.WithContext(ctx).Create(e).Error
}

func (r *walletRepo) SaveWebhookEventTx(ctx context.Context, tx *gorm.DB, e *model.WebhookEvent) error {
	return tx.WithContext(ctx).Save(e).Error
}

// LockPaymentIntentByRefTx locks the intent for a provider reference at ANY
// status. It deliberately does not filter status='pending': the caller must be
// able to distinguish "no intent exists yet" — a race against POST /wallet/fund,
// which MUST be retried or the customer's money is stranded — from "intent
// already settled", a genuine webhook replay that must be acked. Filtering on
// status here collapses both cases into ErrRecordNotFound and makes them
// indistinguishable. provider_ref is UNIQUE, so this still matches ≤1 row.
func (r *walletRepo) LockPaymentIntentByRefTx(ctx context.Context, tx *gorm.DB, ref string) (*model.PaymentIntent, error) {
	var i model.PaymentIntent
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("provider_ref = ?", ref).First(&i).Error
	return &i, err
}

func (r *walletRepo) SavePaymentIntentTx(ctx context.Context, tx *gorm.DB, intent *model.PaymentIntent) error {
	return tx.WithContext(ctx).Save(intent).Error
}

func (r *walletRepo) FindIdempotencyKeyTx(ctx context.Context, tx *gorm.DB, key string) (*model.IdempotencyKey, error) {
	var k model.IdempotencyKey
	err := tx.WithContext(ctx).Where("key = ?", key).First(&k).Error
	return &k, err
}

func (r *walletRepo) CreateIdempotencyKeyTx(ctx context.Context, tx *gorm.DB, k *model.IdempotencyKey) error {
	return tx.WithContext(ctx).Create(k).Error
}

func (r *walletRepo) LockWalletBalanceTx(ctx context.Context, tx *gorm.DB, accountID uuid.UUID) (*model.WalletBalance, error) {
	var b model.WalletBalance
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("account_id = ?", accountID).First(&b).Error
	return &b, err
}

func (r *walletRepo) FindCashoutByKey(ctx context.Context, key string) (*model.CashoutRequest, error) {
	var c model.CashoutRequest
	err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&c).Error
	return &c, err
}

func (r *walletRepo) CreateCashoutTx(ctx context.Context, tx *gorm.DB, c *model.CashoutRequest) error {
	return tx.WithContext(ctx).Create(c).Error
}

func (r *walletRepo) LockCashoutTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.CashoutRequest, error) {
	var c model.CashoutRequest
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&c).Error
	return &c, err
}

func (r *walletRepo) SaveCashoutTx(ctx context.Context, tx *gorm.DB, c *model.CashoutRequest) error {
	return tx.WithContext(ctx).Save(c).Error
}

func (r *walletRepo) FindIdempotencyCashoutTx(ctx context.Context, tx *gorm.DB, key string) (*model.CashoutRequest, error) {
	var c model.CashoutRequest
	err := tx.WithContext(ctx).Where("idempotency_key = ?", key).First(&c).Error
	return &c, err
}

func (r *walletRepo) SumUnpaidEarnings(ctx context.Context, tx *gorm.DB, driverID uuid.UUID) (int64, error) {
	var total int64
	db := r.db.WithContext(ctx)
	if tx != nil {
		db = tx.WithContext(ctx)
	}
	// FIX #7: propagate the DB error — a scan failure must not look like "zero earnings".
	err := db.Model(&model.DriverEarning{}).
		Where("driver_id = ? AND paid_out_at IS NULL", driverID).
		Select("COALESCE(SUM(amount_kobo + tip_kobo), 0)").Scan(&total).Error
	return total, err
}

func (r *walletRepo) ListDriversWithUnpaidEarnings(ctx context.Context) ([]uuid.UUID, error) {
	var drivers []uuid.UUID
	// FIX #8: propagate the DB error — a query failure must not return an empty slice.
	err := r.db.WithContext(ctx).Model(&model.DriverEarning{}).
		Where("paid_out_at IS NULL").
		Distinct("driver_id").
		Pluck("driver_id", &drivers).Error
	return drivers, err
}

func (r *walletRepo) FindMerchantBankAccountTx(ctx context.Context, tx *gorm.DB, merchantID uuid.UUID) (*model.MerchantBankAccount, error) {
	var acct model.MerchantBankAccount
	err := tx.WithContext(ctx).Where("merchant_id = ? AND is_verified = true", merchantID).First(&acct).Error
	return &acct, err
}

func (r *walletRepo) FindDriverBankAccountTx(ctx context.Context, tx *gorm.DB, driverID uuid.UUID) (*model.DriverBankAccount, error) {
	db := r.db.WithContext(ctx)
	if tx != nil {
		db = tx.WithContext(ctx)
	}
	var acct model.DriverBankAccount
	err := db.Where("driver_id = ? AND is_verified = true", driverID).First(&acct).Error
	return &acct, err
}
