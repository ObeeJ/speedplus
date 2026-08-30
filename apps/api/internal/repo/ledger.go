package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LedgerRepo interface {
	// Accounts
	FindOrCreateWallet(ctx context.Context, tx *gorm.DB, ownerID uuid.UUID) (*model.LedgerAccount, error)
	FindOrCreatePlatformAccount(ctx context.Context, tx *gorm.DB, acctType model.AccountType) (*model.LedgerAccount, error)

	// Entries — append-only
	CreateEntries(ctx context.Context, tx *gorm.DB, entries []model.LedgerEntry) error
	FindEntriesByJournal(ctx context.Context, tx *gorm.DB, journalID uuid.UUID) ([]model.LedgerEntry, error)
	// CountReversalsForJournal counts reversal entries already written against a
	// journal. Reverse() uses it to stay idempotent: reversing the same journal
	// twice would post two equal-and-opposite journals and silently double the
	// correction, moving the balance the wrong way by the original amount.
	CountReversalsForJournal(ctx context.Context, tx *gorm.DB, journalID uuid.UUID) (int64, error)

	// Balances — always SELECT FOR UPDATE inside a tx
	LockBalance(ctx context.Context, tx *gorm.DB, accountID uuid.UUID) (*model.WalletBalance, error)
	UpdateBalance(ctx context.Context, tx *gorm.DB, accountID uuid.UUID, newBalance int64) error
	GetBalance(ctx context.Context, ownerID uuid.UUID) (int64, error)
	GetTransactions(ctx context.Context, ownerID uuid.UUID, cursor *uuid.UUID, limit int) ([]model.LedgerEntry, error)

	// Escrow
	CreateEscrowHold(ctx context.Context, tx *gorm.DB, hold *model.EscrowHold) error
	LockEscrowHold(ctx context.Context, tx *gorm.DB, orderID uuid.UUID, status model.EscrowStatus) (*model.EscrowHold, error)
	SaveEscrowHold(ctx context.Context, tx *gorm.DB, hold *model.EscrowHold) error
	FindOverdueFrozenEscrows(ctx context.Context) ([]model.EscrowHold, error)
	FindEscrowHoldByOrder(ctx context.Context, orderID uuid.UUID) (*model.EscrowHold, error)

	// Cancellation rules
	FindCancellationRule(ctx context.Context, vertical, statusAtCancel string) (*model.CancellationRule, error)

	// Driver earnings
	CreateDriverEarning(ctx context.Context, tx *gorm.DB, e *model.DriverEarning) error
	SumUnpaidEarnings(ctx context.Context, driverID uuid.UUID) (int64, error)
	ListDriversWithUnpaidEarnings(ctx context.Context) ([]uuid.UUID, error)

	// Payment intents
	CreatePaymentIntent(ctx context.Context, intent *model.PaymentIntent) error
	FindPaymentIntentByKey(ctx context.Context, idempotencyKey string) (*model.PaymentIntent, error)
	LockPaymentIntentByRef(ctx context.Context, tx *gorm.DB, providerRef string) (*model.PaymentIntent, error)
	SavePaymentIntent(ctx context.Context, tx *gorm.DB, intent *model.PaymentIntent) error

	// Webhook events
	CreateWebhookEvent(ctx context.Context, tx *gorm.DB, e *model.WebhookEvent) error
	FindWebhookEvent(ctx context.Context, provider, eventID string) (*model.WebhookEvent, error)
	SaveWebhookEvent(ctx context.Context, tx *gorm.DB, e *model.WebhookEvent) error

	// Cashout
	CreateCashoutRequest(ctx context.Context, tx *gorm.DB, c *model.CashoutRequest) error
	FindCashoutByKey(ctx context.Context, key string) (*model.CashoutRequest, error)

	// Idempotency keys
	CreateIdempotencyKey(ctx context.Context, tx *gorm.DB, k *model.IdempotencyKey) error
	FindIdempotencyKey(ctx context.Context, key string) (*model.IdempotencyKey, error)

	// Reconciliation / snapshots
	GetMaterialisedBalance(ctx context.Context, accountID uuid.UUID) (int64, error)
	GetLedgerSum(ctx context.Context, accountID uuid.UUID) (int64, error)
	CreateBalanceSnapshot(ctx context.Context, snap *model.PlatformBalanceSnapshot) error

	// Non-tx platform account lookup (reconciliation / snapshot paths)
	FindPlatformAccount(ctx context.Context, acctType model.AccountType) (*model.LedgerAccount, error)
	// FindMerchantUserID resolves a merchants.id to its owning user_id inside a tx.
	FindMerchantUserID(ctx context.Context, tx *gorm.DB, merchantID uuid.UUID) (uuid.UUID, error)
}

type ledgerRepo struct{ db *gorm.DB }

func NewLedgerRepo(db *gorm.DB) LedgerRepo { return &ledgerRepo{db: db} }

func (r *ledgerRepo) FindOrCreateWallet(ctx context.Context, tx *gorm.DB, ownerID uuid.UUID) (*model.LedgerAccount, error) {
	var acct model.LedgerAccount
	// Attrs (not a positional struct) supplies create-only values: GORM treats
	// a struct passed to FirstOrCreate as extra *query conditions*, so the
	// freshly generated ID would join the WHERE clause, never match the
	// existing wallet, and the insert would then violate the
	// (owner_id, type) unique index.
	err := tx.WithContext(ctx).
		Where("owner_id = ? AND type = ?", ownerID, model.AccountWallet).
		Attrs(model.LedgerAccount{ID: uuid.New(), OwnerID: &ownerID, Type: model.AccountWallet}).
		FirstOrCreate(&acct).Error
	if err != nil {
		return nil, err
	}
	// Ensure balance row
	var bal model.WalletBalance
	if err := tx.WithContext(ctx).
		Where("account_id = ?", acct.ID).
		Attrs(model.WalletBalance{AccountID: acct.ID, BalanceKobo: 0}).
		FirstOrCreate(&bal).Error; err != nil {
		return nil, fmt.Errorf("ensure wallet balance: %w", err)
	}
	return &acct, nil
}

func (r *ledgerRepo) FindOrCreatePlatformAccount(ctx context.Context, tx *gorm.DB, acctType model.AccountType) (*model.LedgerAccount, error) {
	var acct model.LedgerAccount
	err := tx.WithContext(ctx).
		Where("owner_id IS NULL AND type = ?", acctType).
		Attrs(model.LedgerAccount{ID: uuid.New(), Type: acctType}).
		FirstOrCreate(&acct).Error
	if err != nil {
		return nil, err
	}
	// Platform accounts need a balance row too — adjustBalance locks it and
	// fails with "record not found" otherwise, which would break every escrow
	// hold and settlement on an environment where it was never pre-seeded.
	var bal model.WalletBalance
	if err := tx.WithContext(ctx).
		Where("account_id = ?", acct.ID).
		Attrs(model.WalletBalance{AccountID: acct.ID, BalanceKobo: 0}).
		FirstOrCreate(&bal).Error; err != nil {
		return nil, fmt.Errorf("ensure platform balance: %w", err)
	}
	return &acct, err
}

func (r *ledgerRepo) CreateEntries(ctx context.Context, tx *gorm.DB, entries []model.LedgerEntry) error {
	return tx.WithContext(ctx).Create(&entries).Error
}

func (r *ledgerRepo) CountReversalsForJournal(ctx context.Context, tx *gorm.DB, journalID uuid.UUID) (int64, error) {
	var count int64
	db := r.db
	if tx != nil {
		db = tx
	}
	err := db.WithContext(ctx).
		Model(&model.LedgerEntry{}).
		Where("ref_type = ? AND ref_id = ?", "reversal", journalID).
		Count(&count).Error
	return count, err
}

func (r *ledgerRepo) FindEntriesByJournal(ctx context.Context, tx *gorm.DB, journalID uuid.UUID) ([]model.LedgerEntry, error) {
	var entries []model.LedgerEntry
	err := tx.WithContext(ctx).Where("journal_id = ?", journalID).Find(&entries).Error
	return entries, err
}

func (r *ledgerRepo) LockBalance(ctx context.Context, tx *gorm.DB, accountID uuid.UUID) (*model.WalletBalance, error) {
	var bal model.WalletBalance
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("account_id = ?", accountID).
		First(&bal).Error
	return &bal, err
}

func (r *ledgerRepo) UpdateBalance(ctx context.Context, tx *gorm.DB, accountID uuid.UUID, newBalance int64) error {
	return tx.WithContext(ctx).
		Model(&model.WalletBalance{}).
		Where("account_id = ?", accountID).
		Updates(map[string]interface{}{"balance_kobo": newBalance, "updated_at": time.Now()}).Error
}

func (r *ledgerRepo) GetBalance(ctx context.Context, ownerID uuid.UUID) (int64, error) {
	var acct model.LedgerAccount
	if err := r.db.WithContext(ctx).
		Where("owner_id = ? AND type = ?", ownerID, model.AccountWallet).
		First(&acct).Error; err != nil {
		return 0, nil
	}
	var bal model.WalletBalance
	if err := r.db.WithContext(ctx).Where("account_id = ?", acct.ID).First(&bal).Error; err != nil {
		return 0, nil
	}
	return bal.BalanceKobo, nil
}

func (r *ledgerRepo) GetTransactions(ctx context.Context, ownerID uuid.UUID, cursor *uuid.UUID, limit int) ([]model.LedgerEntry, error) {
	var acct model.LedgerAccount
	if err := r.db.WithContext(ctx).
		Where("owner_id = ? AND type = ?", ownerID, model.AccountWallet).
		First(&acct).Error; err != nil {
		return nil, nil
	}
	q := r.db.WithContext(ctx).
		Where("account_id = ?", acct.ID).
		Order("created_at DESC, id DESC").
		Limit(limit)
	if cursor != nil {
		var pivot model.LedgerEntry
		if err := r.db.WithContext(ctx).First(&pivot, cursor).Error; err == nil {
			q = q.Where("(created_at, id) < (?, ?)", pivot.CreatedAt, pivot.ID)
		}
	}
	var entries []model.LedgerEntry
	return entries, q.Find(&entries).Error
}

func (r *ledgerRepo) CreateEscrowHold(ctx context.Context, tx *gorm.DB, hold *model.EscrowHold) error {
	return tx.WithContext(ctx).Create(hold).Error
}

func (r *ledgerRepo) LockEscrowHold(ctx context.Context, tx *gorm.DB, orderID uuid.UUID, status model.EscrowStatus) (*model.EscrowHold, error) {
	var hold model.EscrowHold
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("order_id = ? AND status = ?", orderID, status).
		First(&hold).Error
	return &hold, err
}

func (r *ledgerRepo) SaveEscrowHold(ctx context.Context, tx *gorm.DB, hold *model.EscrowHold) error {
	return tx.WithContext(ctx).Save(hold).Error
}

func (r *ledgerRepo) FindOverdueFrozenEscrows(ctx context.Context) ([]model.EscrowHold, error) {
	var holds []model.EscrowHold
	err := r.db.WithContext(ctx).
		Where("status = ? AND frozen_sla_deadline IS NOT NULL AND frozen_sla_deadline < NOW()", model.EscrowFrozen).
		Find(&holds).Error
	return holds, err
}

func (r *ledgerRepo) FindEscrowHoldByOrder(ctx context.Context, orderID uuid.UUID) (*model.EscrowHold, error) {
	var hold model.EscrowHold
	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("created_at DESC").
		First(&hold).Error
	return &hold, err
}

func (r *ledgerRepo) FindCancellationRule(ctx context.Context, vertical, statusAtCancel string) (*model.CancellationRule, error) {
	var rule model.CancellationRule
	err := r.db.WithContext(ctx).
		Where("(vertical = ? OR vertical = '*') AND order_status_at_cancel = ?", vertical, statusAtCancel).
		Order("CASE WHEN vertical = '*' THEN 1 ELSE 0 END").
		First(&rule).Error
	return &rule, err
}

func (r *ledgerRepo) CreateDriverEarning(ctx context.Context, tx *gorm.DB, e *model.DriverEarning) error {
	return tx.WithContext(ctx).Create(e).Error
}

func (r *ledgerRepo) SumUnpaidEarnings(ctx context.Context, driverID uuid.UUID) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.DriverEarning{}).
		Where("driver_id = ? AND paid_out_at IS NULL", driverID).
		Select("COALESCE(SUM(amount_kobo + tip_kobo), 0)").Scan(&total).Error
	return total, err
}

func (r *ledgerRepo) ListDriversWithUnpaidEarnings(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).Model(&model.DriverEarning{}).
		Where("paid_out_at IS NULL").
		Distinct("driver_id").
		Pluck("driver_id", &ids).Error
	return ids, err
}

func (r *ledgerRepo) CreatePaymentIntent(ctx context.Context, intent *model.PaymentIntent) error {
	return r.db.WithContext(ctx).Create(intent).Error
}

func (r *ledgerRepo) FindPaymentIntentByKey(ctx context.Context, key string) (*model.PaymentIntent, error) {
	var i model.PaymentIntent
	err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&i).Error
	return &i, err
}

func (r *ledgerRepo) LockPaymentIntentByRef(ctx context.Context, tx *gorm.DB, providerRef string) (*model.PaymentIntent, error) {
	var i model.PaymentIntent
	// Do NOT filter on status='pending'. The caller must distinguish
	// "no intent exists" (race — retry) from "intent already settled"
	// (genuine replay — ack). Filtering on pending collapses both into
	// ErrRecordNotFound and makes them indistinguishable, causing the
	// webhook processor to force a provider retry on every replay of a
	// settled payment, rolling back the dedup row each time and
	// eventually double-crediting the wallet.
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("provider_ref = ?", providerRef).
		First(&i).Error
	return &i, err
}

func (r *ledgerRepo) SavePaymentIntent(ctx context.Context, tx *gorm.DB, intent *model.PaymentIntent) error {
	return tx.WithContext(ctx).Save(intent).Error
}

func (r *ledgerRepo) CreateWebhookEvent(ctx context.Context, tx *gorm.DB, e *model.WebhookEvent) error {
	return tx.WithContext(ctx).Create(e).Error
}

func (r *ledgerRepo) FindWebhookEvent(ctx context.Context, provider, eventID string) (*model.WebhookEvent, error) {
	var e model.WebhookEvent
	err := r.db.WithContext(ctx).Where("provider = ? AND event_id = ?", provider, eventID).First(&e).Error
	return &e, err
}

func (r *ledgerRepo) SaveWebhookEvent(ctx context.Context, tx *gorm.DB, e *model.WebhookEvent) error {
	return tx.WithContext(ctx).Save(e).Error
}

func (r *ledgerRepo) CreateCashoutRequest(ctx context.Context, tx *gorm.DB, c *model.CashoutRequest) error {
	return tx.WithContext(ctx).Create(c).Error
}

func (r *ledgerRepo) FindCashoutByKey(ctx context.Context, key string) (*model.CashoutRequest, error) {
	var c model.CashoutRequest
	err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&c).Error
	return &c, err
}

func (r *ledgerRepo) CreateIdempotencyKey(ctx context.Context, tx *gorm.DB, k *model.IdempotencyKey) error {
	return tx.WithContext(ctx).Create(k).Error
}

func (r *ledgerRepo) FindIdempotencyKey(ctx context.Context, key string) (*model.IdempotencyKey, error) {
	var k model.IdempotencyKey
	err := r.db.WithContext(ctx).Where("key = ?", key).First(&k).Error
	return &k, err
}

func (r *ledgerRepo) GetMaterialisedBalance(ctx context.Context, accountID uuid.UUID) (int64, error) {
	var bal int64
	err := r.db.WithContext(ctx).Model(&model.WalletBalance{}).
		Where("account_id = ?", accountID).
		Select("balance_kobo").Scan(&bal).Error
	return bal, err
}

func (r *ledgerRepo) GetLedgerSum(ctx context.Context, accountID uuid.UUID) (int64, error) {
	var sum int64
	err := r.db.WithContext(ctx).Model(&model.LedgerEntry{}).
		Where("account_id = ?", accountID).
		Select("COALESCE(SUM(amount_kobo), 0)").Scan(&sum).Error
	return sum, err
}

func (r *ledgerRepo) CreateBalanceSnapshot(ctx context.Context, snap *model.PlatformBalanceSnapshot) error {
	return r.db.WithContext(ctx).Create(snap).Error
}

func (r *ledgerRepo) FindPlatformAccount(ctx context.Context, acctType model.AccountType) (*model.LedgerAccount, error) {
	var acct model.LedgerAccount
	err := r.db.WithContext(ctx).
		Where("owner_id IS NULL AND type = ?", acctType).
		First(&acct).Error
	return &acct, err
}

func (r *ledgerRepo) FindMerchantUserID(ctx context.Context, tx *gorm.DB, merchantID uuid.UUID) (uuid.UUID, error) {
	var m struct{ UserID uuid.UUID }
	err := tx.WithContext(ctx).
		Table("merchants").
		Select("user_id").
		Where("id = ?", merchantID).
		First(&m).Error
	return m.UserID, err
}
