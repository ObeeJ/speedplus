package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

// MerchantService is the self-service surface for an authenticated merchant
// managing their own business — as distinct from AdminService, which acts on
// merchants from the operator side.
type MerchantService struct {
	db *gorm.DB
}

func NewMerchantService(db *gorm.DB) *MerchantService {
	return &MerchantService{db: db}
}

// ResolveByUserID maps a merchant's login User.ID to their catalog/order
// -facing model.Merchant.ID. See LedgerService.ResolveWalletOwner for why
// this indirection exists — Order/Product/ledger rows key off Merchant.ID,
// not the login user ID.
func (s *MerchantService) ResolveByUserID(ctx context.Context, userID uuid.UUID) (*model.Merchant, error) {
	var m model.Merchant
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error; err != nil {
		return nil, fmt.Errorf("merchant profile not found: %w", err)
	}
	return &m, nil
}

// GetProfile returns the merchant's business row plus their onboarding/KYC
// status row (a separate table populated by the KYC approval flow — see
// KYCService.tryActivateProfile).
func (s *MerchantService) GetProfile(ctx context.Context, userID uuid.UUID) (*model.Merchant, *model.MerchantProfile, error) {
	merchant, err := s.ResolveByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	var profile model.MerchantProfile
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return merchant, nil, nil // KYC profile not yet created — not an error, just incomplete onboarding
	}
	return merchant, &profile, nil
}

// SetOpen toggles whether the merchant is currently accepting orders.
// Dispatch/catalog visibility both key off this flag.
func (s *MerchantService) SetOpen(ctx context.Context, userID uuid.UUID, isOpen bool) error {
	merchant, err := s.ResolveByUserID(ctx, userID)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&model.Merchant{}).
		Where("id = ?", merchant.ID).
		Update("is_open", isOpen).Error
}

// ── Bank account management ─────────────────────────────────────────────────────────────────

// GetBankAccount returns the merchant's saved withdrawal bank account, if any.
func (s *MerchantService) GetBankAccount(ctx context.Context, userID uuid.UUID) (*model.MerchantBankAccount, error) {
	merchant, err := s.ResolveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	var acct model.MerchantBankAccount
	if err := s.db.WithContext(ctx).Where("merchant_id = ?", merchant.ID).First(&acct).Error; err != nil {
		return nil, nil // no account saved yet — not an error
	}
	return &acct, nil
}

// SaveBankAccount upserts the merchant's withdrawal bank account.
// accountName must be pre-resolved by the caller via the payment provider's
// account-resolution API — this function trusts it as already verified.
func (s *MerchantService) SaveBankAccount(ctx context.Context, userID uuid.UUID, bankCode, bankName, accountNumber, accountName string) (*model.MerchantBankAccount, error) {
	merchant, err := s.ResolveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	acct := model.MerchantBankAccount{
		MerchantID:    merchant.ID,
		BankCode:      bankCode,
		BankName:      bankName,
		AccountNumber: accountNumber,
		AccountName:   accountName,
		IsVerified:    true,
	}
	if err := s.db.WithContext(ctx).
		Where("merchant_id = ?", merchant.ID).
		Assign(acct).
		FirstOrCreate(&acct).Error; err != nil {
		return nil, err
	}
	return &acct, nil
}
