package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/observability"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

type LedgerService struct {
	repo       repo.LedgerRepo
	pricing    *PricingService
	feeConfigs *FeeConfigService // nil => DefaultFeeTable fallback
}

func NewLedgerService(r repo.LedgerRepo, pricing *PricingService) *LedgerService {
	return &LedgerService{repo: r, pricing: pricing}
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
		return fmt.Errorf("%w: have %d kobo, need %d", ErrInsufficientBalance, bal.BalanceKobo, -deltaKobo)
	}
	return s.repo.UpdateBalance(ctx, tx, accountID, newBal)
}

// ── Account helpers (used by other services) ──────────────────────────────────

func (s *LedgerService) EnsureWallet(ctx context.Context, tx *gorm.DB, ownerID uuid.UUID) (*model.LedgerAccount, error) {
	return s.repo.FindOrCreateWallet(ctx, tx, ownerID)
}

// EnsureMerchantWallet resolves a merchants.id to its owning user before
// touching the ledger. ledger_accounts.owner_id carries an FK to users(id), so
// every ledger account — merchant ones included — is keyed by User.ID.
// Passing model.Merchant.ID straight through is an FK violation, not merely a
// lookup miss.
func (s *LedgerService) EnsureMerchantWallet(ctx context.Context, tx *gorm.DB, merchantID uuid.UUID) (*model.LedgerAccount, error) {
	userID, err := s.repo.FindMerchantUserID(ctx, tx, merchantID)
	if err != nil {
		return nil, fmt.Errorf("resolve merchant %s to user: %w", merchantID, err)
	}
	return s.repo.FindOrCreateWallet(ctx, tx, userID)
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

// shortfallTolerance is the fraction of ordered weight within which no refund
// is issued. 2% on a 12.5kg cylinder = 250g tolerance.
const shortfallTolerance = 0.02

// weightProof loads the weight_photo proof row for a gas order and returns
// the measured kg. Returns 0, nil only when no weight_photo row exists yet —
// any other DB error is propagated, never swallowed. A swallowed error here
// would look identical to "no proof" and silently skip the shortfall refund.
func (s *LedgerService) weightProof(ctx context.Context, tx *gorm.DB, orderID uuid.UUID) (float64, error) {
	var proof model.ProofMedia
	err := tx.WithContext(ctx).
		Where("order_id = ? AND kind = 'weight_photo'", orderID).
		Order("captured_at DESC").
		First(&proof).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("load weight proof: %w", err)
	}
	if proof.MeasuredKg == nil {
		return 0, nil
	}
	return *proof.MeasuredKg, nil
}

// orderedWeightKg sums the ordered weight directly from order_items via tx,
// rather than trusting order.Items to be populated. Settle is called with
// orders loaded by LockOrderTx / FindActiveInTransitOrder (paycode.go), none
// of which preload Items — relying on the in-memory struct here would make
// this always read 0 and silently skip every gas shortfall calculation in
// production, while still passing tests that construct Order with Items set
// by hand.
func (s *LedgerService) orderedWeightKg(ctx context.Context, tx *gorm.DB, orderID uuid.UUID) (float64, error) {
	var orderedKg float64
	err := tx.WithContext(ctx).Model(&model.OrderItem{}).
		Where("order_id = ?", orderID).
		Select("COALESCE(SUM(weight_kg * quantity), 0)").
		Scan(&orderedKg).Error
	if err != nil {
		return 0, fmt.Errorf("load ordered weight: %w", err)
	}
	return orderedKg, nil
}

// liveLPGPriceKobo returns the current LPG price per kg for a region, or 0 if
// no index row exists yet. Used to price gas shortfall refunds in new_cylinder
// mode, where SubtotalKobo includes the cylinder body and is not a reliable
// ₦/kg basis.
func (s *LedgerService) liveLPGPriceKobo(ctx context.Context, tx *gorm.DB, region string) (int64, error) {
	var row model.LPGPriceIndex
	err := tx.WithContext(ctx).
		Where("region = ? AND effective_at <= NOW()", region).
		Order("effective_at DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("load lpg price: %w", err)
	}
	return row.PricePerKgKobo, nil
}

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

	// Gas orders must never settle without a weight photo — this is the
	// trust wedge the whole gas vertical rests on. Enforced here, inside
	// Settle, so every current and future confirmation path (QR, code, card)
	// is covered by construction rather than by remembering to add a check
	// at each new call site. Fetched once and reused below for the shortfall
	// calculation, so the guard and the calculation can never read different
	// data — "has a weight photo" and "has a measured_kg to price against"
	// are the same fact by construction, not two separately-maintained checks.
	var gasMeasuredKg float64
	if order.Vertical == "gas" {
		var err error
		gasMeasuredKg, err = s.weightProof(ctx, tx, order.ID)
		if err != nil {
			return fmt.Errorf("settle: weight proof: %w", err)
		}
		if gasMeasuredKg <= 0 {
			return fmt.Errorf("gas order requires a weight photo before settlement")
		}
	}

	merchantShareKobo := int64(float64(order.SubtotalKobo) * fees.MerchantTakeRate)
	driverEarningKobo := int64(float64(order.DeliveryKobo) * fees.DriverTakeRate)

	// Gas shortfall refund: if measured < ordered beyond tolerance, debit the
	// shortfall from the merchant's share and credit it to the customer —
	// within this same balanced journal.
	var shortfallRefundKobo int64
	if order.Vertical == "gas" {
		orderedKg, err := s.orderedWeightKg(ctx, tx, order.ID)
		if err != nil {
			return fmt.Errorf("settle: %w", err)
		}
		if orderedKg > 0 {
			measuredKg := gasMeasuredKg
			shortKg := orderedKg - measuredKg
			if shortKg > orderedKg*shortfallTolerance {
				// price per kg = subtotal / orderedKg. In new_cylinder mode
				// the subtotal includes the cylinder body, not just the
				// gas fill, so subtotal/kg overstates ₦/kg — price off the
				// LPG index instead when one is available.
				pricePerKg := float64(order.SubtotalKobo) / orderedKg
				if order.GasMode != nil && *order.GasMode == "new_cylinder" {
					lpgPriceKobo, err := s.liveLPGPriceKobo(ctx, tx, "Lagos")
					if err != nil {
						return fmt.Errorf("settle: lpg price: %w", err)
					}
					if lpgPriceKobo > 0 {
						pricePerKg = float64(lpgPriceKobo)
					}
				}
				shortfallRefundKobo = int64(shortKg * pricePerKg)
				// Cap at merchant's share — never go negative
				if shortfallRefundKobo > merchantShareKobo {
					shortfallRefundKobo = merchantShareKobo
				}
				merchantShareKobo -= shortfallRefundKobo
			}
		}
	}

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
	merchantWallet, err := s.EnsureMerchantWallet(ctx, tx, order.MerchantID)
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
	// Shortfall refund: credit customer from the already-reduced merchant share.
	// The escrow debit above covers the full baseHoldKobo; the shortfall is
	// redistributed within the journal so the zero-sum invariant still holds.
	if shortfallRefundKobo > 0 {
		customerWallet, err := s.EnsureWallet(ctx, tx, order.CustomerID)
		if err != nil {
			return fmt.Errorf("settle: customer wallet for shortfall: %w", err)
		}
		entries = append(entries,
			model.LedgerEntry{ID: uuid.New(), JournalID: journalID, AccountID: revenueAcct.ID, AmountKobo: -shortfallRefundKobo, Description: "gas shortfall debit platform", RefType: "order", RefID: &order.ID},
			model.LedgerEntry{ID: uuid.New(), JournalID: journalID, AccountID: customerWallet.ID, AmountKobo: shortfallRefundKobo, Description: "gas shortfall refund to customer", RefType: "order", RefID: &order.ID},
		)
		if err := s.adjustBalance(ctx, tx, customerWallet.ID, shortfallRefundKobo); err != nil {
			return fmt.Errorf("settle: adjust customer balance for shortfall: %w", err)
		}
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
		mw, err := s.EnsureMerchantWallet(ctx, tx, order.MerchantID)
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
	escrowAcct, err := s.repo.FindPlatformAccount(ctx, model.AccountEscrow)
	if err != nil {
		return 0, fmt.Errorf("reconcile: escrow account: %w", err)
	}
	materialised, err := s.repo.GetMaterialisedBalance(ctx, escrowAcct.ID)
	if err != nil {
		return 0, fmt.Errorf("reconcile: read materialised balance: %w", err)
	}
	ledgerSum, err := s.repo.GetLedgerSum(ctx, escrowAcct.ID)
	if err != nil {
		return 0, fmt.Errorf("reconcile: ledger sum: %w", err)
	}
	return materialised - ledgerSum, nil
}

// SnapshotPlatformBalances writes a daily balance snapshot for revenue and
// provider_clearing accounts. These are aggregate-only (no materialised row),
// so the snapshot is the only fast historical view.
func (s *LedgerService) SnapshotPlatformBalances(ctx context.Context) error {
	for _, acctType := range []model.AccountType{model.AccountRevenue, model.AccountProviderClearing} {
		acct, err := s.repo.FindPlatformAccount(ctx, acctType)
		if err != nil {
			return fmt.Errorf("snapshot: %s account: %w", acctType, err)
		}
		sum, err := s.repo.GetLedgerSum(ctx, acct.ID)
		if err != nil {
			return fmt.Errorf("snapshot: sum %s: %w", acctType, err)
		}
		snap := model.PlatformBalanceSnapshot{
			ID:           uuid.New(),
			AccountType:  acctType,
			BalanceKobo:  sum,
			SnapshotDate: time.Now().UTC().Truncate(24 * time.Hour),
		}
		if err := s.repo.CreateBalanceSnapshot(ctx, &snap); err != nil {
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

// ResolveWalletOwner maps the authenticated caller to the ledger account ID
// their wallet lives under. That is always their own User.ID: ledger_accounts
// .owner_id has an FK to users(id), so the ledger is uniformly user-keyed for
// every role. Merchant settlement reaches the same account by resolving
// merchants.id -> user_id first (see EnsureMerchantWallet), so no translation
// is needed on the read path.
//
// It is kept as a named seam rather than inlined because the merchant identity
// split (model.Merchant.ID for the business profile vs User.ID for login) is
// easy to get backwards; routing every balance read through here keeps the
// rule stated in one place.
func (s *LedgerService) ResolveWalletOwner(ctx context.Context, userID uuid.UUID, role string) (uuid.UUID, error) {
	return userID, nil
}
