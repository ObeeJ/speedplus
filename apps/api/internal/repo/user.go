package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

// UserRepo is the interface services depend on — never *gorm.DB directly.
type UserRepo interface {
	Create(ctx context.Context, u *model.User) error
	FindByPhone(ctx context.Context, phone string) (*model.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	FindByReferralCode(ctx context.Context, code string) (*model.User, error)
	Update(ctx context.Context, u *model.User) error

	CreateRefreshToken(ctx context.Context, rt *model.RefreshToken) error
	FindRefreshToken(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	FindRefreshTokenAny(ctx context.Context, tokenHash string) (*model.RefreshToken, error) // includes revoked
	RevokeRefreshToken(ctx context.Context, tokenHash string, at time.Time) error
	RevokeRefreshFamily(ctx context.Context, familyID uuid.UUID, at time.Time) error

	CreateOTP(ctx context.Context, otp *model.OTPCode) error
	InvalidatePreviousOTPs(ctx context.Context, phone, purpose string) error
	FindActiveOTP(ctx context.Context, phone, purpose string) (*model.OTPCode, error)
	MarkOTPUsed(ctx context.Context, id uuid.UUID, at time.Time) error

	UpsertPIN(ctx context.Context, userID uuid.UUID, pinHash string) error
	FindPIN(ctx context.Context, userID uuid.UUID) (*model.PIN, error)

	CreateAddress(ctx context.Context, a *model.Address) error
	ListAddresses(ctx context.Context, userID uuid.UUID) ([]model.Address, error)
	FindAddress(ctx context.Context, id uuid.UUID) (*model.Address, error)

	CreateDriverProfile(ctx context.Context, dp *model.DriverProfile) error
	FindDriverProfile(ctx context.Context, userID uuid.UUID) (*model.DriverProfile, error)
	UpdateDriverProfile(ctx context.Context, dp *model.DriverProfile) error

	CreateMerchantProfile(ctx context.Context, mp *model.MerchantProfile) error
	FindMerchantProfile(ctx context.Context, userID uuid.UUID) (*model.MerchantProfile, error)
	UpdateMerchantProfile(ctx context.Context, mp *model.MerchantProfile) error
}

type userRepo struct{ db *gorm.DB }

func NewUserRepo(db *gorm.DB) UserRepo { return &userRepo{db: db} }

func (r *userRepo) Create(ctx context.Context, u *model.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *userRepo) FindByPhone(ctx context.Context, phone string) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).Where("phone = ? AND is_active = true", phone).First(&u).Error
	return &u, err
}

func (r *userRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).First(&u, id).Error
	return &u, err
}

func (r *userRepo) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).Where("username = ? AND is_active = true", username).First(&u).Error
	return &u, err
}

func (r *userRepo) FindByReferralCode(ctx context.Context, code string) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).Where("referral_code = ? AND is_active = true", code).First(&u).Error
	return &u, err
}

func (r *userRepo) Update(ctx context.Context, u *model.User) error {
	return r.db.WithContext(ctx).Save(u).Error
}

func (r *userRepo) CreateRefreshToken(ctx context.Context, rt *model.RefreshToken) error {
	return r.db.WithContext(ctx).Create(rt).Error
}

func (r *userRepo) FindRefreshToken(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > NOW()", tokenHash).
		First(&rt).Error
	return &rt, err
}

func (r *userRepo) FindRefreshTokenAny(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	err := r.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&rt).Error
	return &rt, err
}

func (r *userRepo) RevokeRefreshToken(ctx context.Context, tokenHash string, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("token_hash = ?", tokenHash).
		Update("revoked_at", at).Error
}

func (r *userRepo) RevokeRefreshFamily(ctx context.Context, familyID uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", at).Error
}

func (r *userRepo) CreateOTP(ctx context.Context, otp *model.OTPCode) error {
	return r.db.WithContext(ctx).Create(otp).Error
}

func (r *userRepo) InvalidatePreviousOTPs(ctx context.Context, phone, purpose string) error {
	return r.db.WithContext(ctx).
		Model(&model.OTPCode{}).
		Where("phone = ? AND purpose = ? AND used_at IS NULL", phone, purpose).
		Update("used_at", time.Now()).Error
}

func (r *userRepo) FindActiveOTP(ctx context.Context, phone, purpose string) (*model.OTPCode, error) {
	var otp model.OTPCode
	err := r.db.WithContext(ctx).
		Where("phone = ? AND purpose = ? AND used_at IS NULL AND expires_at > NOW()", phone, purpose).
		Order("created_at DESC").
		First(&otp).Error
	return &otp, err
}

func (r *userRepo) MarkOTPUsed(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).Model(&model.OTPCode{}).Where("id = ?", id).Update("used_at", at).Error
}

func (r *userRepo) UpsertPIN(ctx context.Context, userID uuid.UUID, pinHash string) error {
	return r.db.WithContext(ctx).
		Where(model.PIN{UserID: userID}).
		Assign(model.PIN{PINHash: pinHash}).
		FirstOrCreate(&model.PIN{}).Error
}

func (r *userRepo) FindPIN(ctx context.Context, userID uuid.UUID) (*model.PIN, error) {
	var p model.PIN
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error
	return &p, err
}

func (r *userRepo) CreateAddress(ctx context.Context, a *model.Address) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *userRepo) ListAddresses(ctx context.Context, userID uuid.UUID) ([]model.Address, error) {
	var addrs []model.Address
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("is_default DESC, created_at ASC").Find(&addrs).Error
	return addrs, err
}

func (r *userRepo) FindAddress(ctx context.Context, id uuid.UUID) (*model.Address, error) {
	var a model.Address
	err := r.db.WithContext(ctx).First(&a, id).Error
	return &a, err
}

func (r *userRepo) CreateDriverProfile(ctx context.Context, dp *model.DriverProfile) error {
	return r.db.WithContext(ctx).Create(dp).Error
}

func (r *userRepo) FindDriverProfile(ctx context.Context, userID uuid.UUID) (*model.DriverProfile, error) {
	var dp model.DriverProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&dp).Error
	return &dp, err
}

func (r *userRepo) UpdateDriverProfile(ctx context.Context, dp *model.DriverProfile) error {
	return r.db.WithContext(ctx).Save(dp).Error
}

func (r *userRepo) CreateMerchantProfile(ctx context.Context, mp *model.MerchantProfile) error {
	return r.db.WithContext(ctx).Create(mp).Error
}

func (r *userRepo) FindMerchantProfile(ctx context.Context, userID uuid.UUID) (*model.MerchantProfile, error) {
	var mp model.MerchantProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&mp).Error
	return &mp, err
}

func (r *userRepo) UpdateMerchantProfile(ctx context.Context, mp *model.MerchantProfile) error {
	return r.db.WithContext(ctx).Save(mp).Error
}
