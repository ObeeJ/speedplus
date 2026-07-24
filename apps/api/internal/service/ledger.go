package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/observability"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

type LedgerService struct {
	db         *gorm.DB // kept for transaction boundaries only
	repo       repo.LedgerRepo
	pricing    *PricingService
	feeConfigs *FeeConfigService // nil => DefaultFeeTable fallback
}

func NewLedgerService(db *gorm.DB, r repo.LedgerRepo, pricing *PricingService) *LedgerService {
	return &LedgerService{db: db, repo: r, pricing: pricing}
}

// InjectFeeConfigs wires the FeeConfigService after construction to break the
// circular dependency: LedgerService ← FeeConfigService ← (nothing).
func (s *LedgerService) InjectFeeConfigs(fc *FeeConfigService) {
	s.feeConfigs = fc
}

// ── Core journal writer ───────────────────────────────────────────────────────

func (s *LedgerService) journal(ctx context.Context, tx *gorm.DB, entries []model.LedgerEntry) error {
	var sum int64
	for _, e := range entries {
		sum += e.AmountKobo
	}
	if sum != 0 {
		return fmt.Errorf("ledger invariant violated: journal sums to %d, must be 0", sum)
	}
	return s.repo.CreateEntries(ctx, tx, entries)
}

func (s *LedgerService) adjustBalance(ctx context.Context, tx *gorm.DB, accountID uuid.UUID, deltaKobo int64) error {
	bal, err := s.repo.LockBalance(ctx, tx, accountID)
	if err != nil {
		return err
	}
	newBal := bal.BalanceKobo + deltaKobo
	if newBal < 0 {
		return fmt.Errorf("insufficient balance: have %d kobo, need %d", bal.BalanceKobo, -deltaKobo)
	}
	return s.repo.UpdateBalance(ctx, tx, accountID, newBal)
}

// ── Account helpers (used by other services) ──────────────────────────────────

func (s *LedgerService) EnsureWallet(ctx context.Context, tx *gorm.DB, ownerID uuid.UUID) (*model.LedgerAccount, error) {
	return s.repo.FindOrCreateWallet(ctx, tx, ownerID)
}

func (s *LedgerService) platformAccount(ctx context.Context, tx *gorm.DB, acctType model.AccountType) (*model.LedgerAccount, error) {
	return s.repo.FindOrCreatePlatformAccount(ctx, tx, acctType)
}

// ── Escrow hold ───────────────────────────────────────────────────────────────

func (s *LedgerService) HoldEscrow(ctx context.Context, tx *gorm.DB, orderID, customerID uuid.UUID, amountKobo int64) error {
	customerWallet, err := s.EnsureWallet(ctx, tx, customerID)
	if err != nil {
		return err
	}
	escrowAcct, err := s.platformAccount(ctx, tx, model.AccountEscrow)
	if err != nil {
		return err
	}

	journalID := uuid.New()
	entries := []model.LedgerEntry{
		{ID: uuid.New(), JournalID: journalID, AccountID: customerWallet.ID, AmountKobo: -amountKobo, Description: "escrow hold", RefType: "order", RefID: &orderID},
		{ID: uuid.New(), JournalID: journalID, AccountID: escrowAcct.ID, AmountKobo: amountKobo, Description: "escrow hold", RefType: "order", RefID: &orderID},
	}
	if err := s.journal(ctx, tx, entries); err != nil {
		return err
	}
	if err := s.adjustBalance(ctx, tx, customerWallet.ID, -amountKobo); err != nil {
		return err
	}
	// Materialise escrow platform balance so reconciliation can assert it in O(1).
	if err := s.adjustBalance(ctx, tx, escrowAcct.ID, amountKobo); err != nil {
		return fmt.Errorf("hold escrow: adjust escrow balance: %w", err)
	}

	hold := &model.EscrowHold{
		ID:         uuid.New(),
		OrderID:    orderID,
		AccountID:  escrowAcct.ID,
		AmountKobo: amountKobo,
		Status:     model.EscrowHeld,
	}
	return s.repo.CreateEscrowHold(ctx, tx, hold)
}

// ── Settlement ────────────────────────────────────────────────────────────────

func (s *LedgerService) Settle(ctx context.Context, tx *gorm.DB, order *model.Order, paycodeEventID uuid.UUID) error {
	// Pin the split to the fee config in force when the order was created —
	// an admin rate change mid-delivery must never reallocate money in flight.
	var fees FeeConfig
	if s.feeConfigs != nil {
		fees = s.feeConfigs.GetFeesAt(ctx, order.Vertical, order.CreatedAt)
	} else {
		fees = defaultFees(order.Vertical)
	}

	hold, err := s.repo.LockEscrowHold(ctx, tx, order.ID, model.EscrowHeld)
	if err != nil {
		return fmt.Errorf("escrow hold not found: %w", err)
	}

	merchantShareKobo := int64(float64(order.SubtotalKobo) * fees.MerchantTakeRate)
	driverEarningKobo := int64(float64(order.DeliveryKobo) * fees.DriverTakeRate)
	platformTotal := (order.SubtotalKobo - merchantShareKobo) + (order.DeliveryKobo - driverEarningKobo) + order.ServiceKobo

	if order.DriverID == nil {
		return fmt.Errorf("cannot settle: no driver assigned")
	}

	// FIX #3: propagate all account/wallet lookup errors — no more _, _ discards.
	escrowAcct, err := s.platformAccount(ctx, tx, model.AccountEscrow)
	if err != nil {
		return fmt.Errorf("settle: escrow account: %w", err)
	}
	revenueAcct, err := s.platformAccount(ctx, tx, model.AccountRevenue)
	if err != nil {
		return fmt.Errorf("settle: revenue account: %w", err)
	}
	merchantWallet, err := s.EnsureWallet(ctx, tx, order.MerchantID)
	if err != nil {
		return fmt.Errorf("settle: merchant wallet: %w", err)
	}
	driverWallet, err := s.EnsureWallet(ctx, tx, *order.DriverID)
	if err != nil {
		return fmt.Errorf("settle: driver wallet: %w", err)
	}

	// FIX #1: base escrow debit covers only subtotal+delivery+service.
	// The tip is held in escrow too (HoldEscrow used TotalKobo = base+tip),
	// but we debit it in a separate journal line below so the invariant holds
	// for both the base block and the tip block independently.
	baseHoldKobo := order.SubtotalKobo + order.DeliveryKobo + order.ServiceKobo

	journalID := uuid.New()
	entries := []model.LedgerEntry{
		{ID: uuid.New(), JournalID: journalID, AccountID: escrowAcct.ID, AmountKobo: -baseHoldKobo, Description: "settlement debit escrow", RefType: "order", RefID: &order.ID},
		{ID: uuid.New(), JournalID: journalID, AccountID: merchantWallet.ID, AmountKobo: merchantShareKobo, Description: "merchant settlement", RefType: "order", RefID: &order.ID},
		{ID: uuid.New(), JournalID: journalID, AccountID: driverWallet.ID, AmountKobo: driverEarningKobo, Description: "driver delivery fee", RefType: "order", RefID: &order.ID},
		{ID: uuid.New(), JournalID: journalID, AccountID: revenueAcct.ID, AmountKobo: platformTotal, Description: "platform fee", RefType: "order", RefID: &order.ID},
	}
	// Tip block: separate escrow debit + driver credit — sums to zero on its own.
	if order.TipKobo > 0 {
		entries = append(entries,
			model.LedgerEntry{ID: uuid.New(), JournalID: journalID, AccountID: escrowAcct.ID, AmountKobo: -order.TipKobo, Description: "tip debit escrow", RefType: "order", RefID: &order.ID},
			model.LedgerEntry{ID: uuid.New(), JournalID: journalID, AccountID: driverWallet.ID, AmountKobo: order.TipKobo, Description: "tip 100% driver", RefType: "order", RefID: &order.ID},
		)
	}

	if err := s.journal(ctx, tx, entries); err != nil {
		return err
	}
	// Materialise escrow balance: debit the full hold (base + tip).
	if err := s.adjustBalance(ctx, tx, escrowAcct.ID, -(baseHoldKobo + order.TipKobo)); err != nil {
		return fmt.Errorf("settle: adjust escrow balance: %w", err)
	}
	if err := s.adjustBalance(ctx, tx, merchantWallet.ID, merchantShareKobo); err != nil {
		return fmt.Errorf("settle: adjust merchant balance: %w", err)
	}
	if err := s.adjustBalance(ctx, tx, driverWallet.ID, driverEarningKobo+order.TipKobo); err != nil {
		return fmt.Errorf("settle: adjust driver balance: %w", err)
	}

	if err := s.repo.CreateDriverEarning(ctx, tx, &model.DriverEarning{
		ID:         uuid.New(),
		DriverID:   *order.DriverID,
		OrderID:    order.ID,
		AmountKobo: driverEarningKobo,
		TipKobo:    order.TipKobo,
	}); err != nil {
		observability.CaptureError(ctx, err, "settle: create driver earning", "order_id", order.ID.String())
		// non-fatal: earning row missing is recoverable; settlement itself must not roll back
	}

	now := time.Now()
	hold.Status = model.EscrowReleased
	hold.ReleasedAt = &now
	hold.ReleasedBy = &paycodeEventID
	return s.repo.SaveEscrowHold(ctx, tx, hold)
}

// ── Cancellation refund engine ────────────────────────────────────────────────

func (s *LedgerService) ProcessCancellationRefund(ctx context.Context, tx *gorm.DB, order *model.Order) error {
	rule, err := s.repo.FindCancellationRule(ctx, order.Vertical, string(order.Status))
	if err != nil || rule.FullRefund {
		return s.fullRefund(ctx, tx, order)
	}

	hold, err := s.repo.LockEscrowHold(ctx, tx, order.ID, model.EscrowHeld)
	if err != nil {
		return fmt.Errorf("escrow hold not found for refund")
	}

	merchantComp := rule.MerchantCompKobo
	if rule.MerchantCompPct > 0 {
		merchantComp = int64(float64(order.SubtotalKobo) * rule.MerchantCompPct)
	}
	var riderComp int64
	if rule.RiderCompPctOfDelivery > 0 && order.DriverID != nil {
		riderComp = int64(float64(order.DeliveryKobo) * rule.RiderCompPctOfDelivery)
	}
	refundKobo := hold.AmountKobo - merchantComp - riderComp
	if refundKobo < 0 {
		refundKobo = 0
	}

	escrowAcct, err := s.platformAccount(ctx, tx, model.AccountEscrow)
	if err != nil {
		return fmt.Errorf("refund: escrow account: %w", err)
	}
	customerWallet, err := s.EnsureWallet(ctx, tx, order.CustomerID)
	if err != nil {
		return fmt.Errorf("refund: customer wallet: %w", err)
	}

	journalID := uuid.New()
	entries := []model.LedgerEntry{
		{ID: uuid.New(), JournalID: journalID, AccountID: escrowAcct.ID, AmountKobo: -hold.AmountKobo, Description: "cancellation refund debit escrow", RefType: "order", RefID: &order.ID},
		{ID: uuid.New(), JournalID: journalID, AccountID: customerWallet.ID, AmountKobo: refundKobo, Description: "cancellation refund to customer", RefType: "order", RefID: &order.ID},
	}
	if merchantComp > 0 {
		mw, err := s.EnsureWallet(ctx, tx, order.MerchantID)
		if err != nil {
			return fmt.Errorf("refund: merchant wallet: %w", err)
		}
		entries = append(entries, model.LedgerEntry{ID: uuid.New(), JournalID: journalID, AccountID: mw.ID, AmountKobo: merchantComp, Description: "merchant prep compensation", RefType: "order", RefID: &order.ID})
		if err := s.adjustBalance(ctx, tx, mw.ID, merchantComp); err != nil {
			return fmt.Errorf("refund: adjust merchant balance: %w", err)
		}
	}
	if riderComp > 0 && order.DriverID != nil {
		dw, err := s.EnsureWallet(ctx, tx, *order.DriverID)
		if err != nil {
			return fmt.Errorf("refund: driver wallet: %w", err)
		}
		entries = append(entries, model.LedgerEntry{ID: uuid.New(), JournalID: journalID, AccountID: dw.ID, AmountKobo: riderComp, Description: "rider distance compensation", RefType: "order", RefID: &order.ID})
		if err := s.adjustBalance(ctx, tx, dw.ID, riderComp); err != nil {
			return fmt.Errorf("refund: adjust rider balance: %w", err)
		}
	}

	if err := s.journal(ctx, tx, entries); err != nil {
		return err
	}
	if err := s.adjustBalance(ctx, tx, escrowAcct.ID, -hold.AmountKobo); err != nil {
		return fmt.Errorf("refund: adjust escrow balance: %w", err)
	}
	if err := s.adjustBalance(ctx, tx, customerWallet.ID, refundKobo); err != nil {
		return fmt.Errorf("refund: adjust customer balance: %w", err)
	}

	now := time.Now()
	hold.Status = model.EscrowReversed
	hold.ReleasedAt = &now
	return s.repo.SaveEscrowHold(ctx, tx, hold)
}

func (s *LedgerService) fullRefund(ctx context.Context, tx *gorm.DB, order *model.Order) error {
	hold, err := s.repo.LockEscrowHold(ctx, tx, order.ID, model.EscrowHeld)
	if err != nil {
		// FIX #5: propagate the error — returning nil here silently skips the refund.
		return fmt.Errorf("full refund: escrow hold not found: %w", err)
	}
	escrowAcct, err := s.platformAccount(ctx, tx, model.AccountEscrow)
	if err != nil {
		return fmt.Errorf("full refund: escrow account: %w", err)
	}
	customerWallet, err := s.EnsureWallet(ctx, tx, order.CustomerID)
	if err != nil {
		return fmt.Errorf("full refund: customer wallet: %w", err)
	}

	journalID := uuid.New()
	entries := []model.LedgerEntry{
		{ID: uuid.New(), JournalID: journalID, AccountID: escrowAcct.ID, AmountKobo: -hold.AmountKobo, Description: "full refund debit escrow", RefType: "order", RefID: &order.ID},
		{ID: uuid.New(), JournalID: journalID, AccountID: customerWallet.ID, AmountKobo: hold.AmountKobo, Description: "full refund to customer wallet", RefType: "order", RefID: &order.ID},
	}
	if err := s.journal(ctx, tx, entries); err != nil {
		return err
	}
	if err := s.adjustBalance(ctx, tx, escrowAcct.ID, -hold.AmountKobo); err != nil {
		return fmt.Errorf("full refund: adjust escrow balance: %w", err)
	}
	if err := s.adjustBalance(ctx, tx, customerWallet.ID, hold.AmountKobo); err != nil {
		return fmt.Errorf("full refund: adjust customer balance: %w", err)
	}

	now := time.Now()
	hold.Status = model.EscrowReversed
	hold.ReleasedAt = &now
	return s.repo.SaveEscrowHold(ctx, tx, hold)
}

// ── Wallet credit (pay-in) ────────────────────────────────────────────────────

// CreditWallet journals an inbound payment from a provider.
// FIX #4: debit AccountProviderClearing (asset in transit), NOT AccountRevenue.
// Revenue is only recognised at settlement time (platform fee entry in Settle).
func (s *LedgerService) CreditWallet(ctx context.Context, tx *gorm.DB, userID uuid.UUID, amountKobo int64, refType string, refID *uuid.UUID) error {
	wallet, err := s.EnsureWallet(ctx, tx, userID)
	if err != nil {
		return err
	}
	clearingAcct, err := s.platformAccount(ctx, tx, model.AccountProviderClearing)
	if err != nil {
		return fmt.Errorf("credit wallet: clearing account: %w", err)
	}

	journalID := uuid.New()
	entries := []model.LedgerEntry{
		{ID: uuid.New(), JournalID: journalID, AccountID: clearingAcct.ID, AmountKobo: -amountKobo, Description: "wallet fund debit provider clearing", RefType: refType, RefID: refID},
		{ID: uuid.New(), JournalID: journalID, AccountID: wallet.ID, AmountKobo: amountKobo, Description: "wallet fund credit", RefType: refType, RefID: refID},
	}
	if err := s.journal(ctx, tx, entries); err != nil {
		return err
	}
	if err := s.adjustBalance(ctx, tx, wallet.ID, amountKobo); err != nil {
		return fmt.Errorf("credit wallet: adjust balance: %w", err)
	}
	return nil
}

// ── Reconciliation & snapshots ───────────────────────────────────────────────

// ReconcileEscrow asserts that the materialised escrow wallet_balance equals
// the sum of all ledger entries for the escrow account.
// Returns the drift in kobo (0 = clean). Never auto-corrects.
func (s *LedgerService) ReconcileEscrow(ctx context.Context) (int64, error) {
	escrowAcct, err := s.repo.FindOrCreatePlatformAccount(ctx, s.db, model.AccountEscrow)
	if err != nil {
		return 0, fmt.Errorf("reconcile: escrow account: %w", err)
	}

	// Materialised balance
	matBal, err := s.repo.GetBalance(ctx, *escrowAcct.OwnerID)
	if err != nil {
		// Platform accounts have nil OwnerID — query by account ID directly
	}
	var materialised int64
	if err := s.db.WithContext(ctx).
		Model(&model.WalletBalance{}).
		Where("account_id = ?", escrowAcct.ID).
		Select("balance_kobo").
		Scan(&materialised).Error; err != nil {
		return 0, fmt.Errorf("reconcile: read materialised balance: %w", err)
	}
	_ = matBal

	// Ledger sum
	var ledgerSum int64
	if err := s.db.WithContext(ctx).
		Model(&model.LedgerEntry{}).
		Where("account_id = ?", escrowAcct.ID).
		Select("COALESCE(SUM(amount_kobo), 0)").
		Scan(&ledgerSum).Error; err != nil {
		return 0, fmt.Errorf("reconcile: ledger sum: %w", err)
	}

	return materialised - ledgerSum, nil
}

// SnapshotPlatformBalances writes a daily balance snapshot for revenue and
// provider_clearing accounts. These are aggregate-only (no materialised row),
// so the snapshot is the only fast historical view.
func (s *LedgerService) SnapshotPlatformBalances(ctx context.Context) error {
	for _, acctType := range []model.AccountType{model.AccountRevenue, model.AccountProviderClearing} {
		acct, err := s.repo.FindOrCreatePlatformAccount(ctx, s.db, acctType)
		if err != nil {
			return fmt.Errorf("snapshot: %s account: %w", acctType, err)
		}

		var sum int64
		if err := s.db.WithContext(ctx).
			Model(&model.LedgerEntry{}).
			Where("account_id = ?", acct.ID).
			Select("COALESCE(SUM(amount_kobo), 0)").
			Scan(&sum).Error; err != nil {
			return fmt.Errorf("snapshot: sum %s: %w", acctType, err)
		}

		snap := model.PlatformBalanceSnapshot{
			ID:           uuid.New(),
			AccountType:  acctType,
			BalanceKobo:  sum,
			SnapshotDate: time.Now().UTC().Truncate(24 * time.Hour),
		}
		if err := s.db.WithContext(ctx).Create(&snap).Error; err != nil {
			return fmt.Errorf("snapshot: create %s: %w", acctType, err)
		}
	}
	return nil
}

// ── Queries ───────────────────────────────────────────────────────────────────

func (s *LedgerService) GetBalance(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.repo.GetBalance(ctx, userID)
}

func (s *LedgerService) GetTransactions(ctx context.Context, userID uuid.UUID, cursor *uuid.UUID, limit int) ([]model.LedgerEntry, error) {
	return s.repo.GetTransactions(ctx, userID, cursor, limit)
}
