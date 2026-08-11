package repo

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

type PlatformSettingRepo interface {
	GetLatest(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, updatedBy uuid.UUID, reason string) error
}

type platformSettingRepo struct{ db *gorm.DB }

func NewPlatformSettingRepo(db *gorm.DB) PlatformSettingRepo {
	return &platformSettingRepo{db: db}
}

func (r *platformSettingRepo) GetLatest(ctx context.Context, key string) (string, error) {
	var row model.PlatformSetting
	err := r.db.WithContext(ctx).
		Where("key = ?", key).
		Order("created_at DESC").
		First(&row).Error
	return row.Value, err
}

func (r *platformSettingRepo) Set(ctx context.Context, key, value string, updatedBy uuid.UUID, reason string) error {
	return r.db.WithContext(ctx).Create(&model.PlatformSetting{
		ID:        uuid.New(),
		Key:       key,
		Value:     value,
		UpdatedBy: updatedBy,
		Reason:    reason,
		CreatedAt: time.Now(),
	}).Error
}

// GetWeatherSurcharge reads enabled + amount from the DB.
// Exported so PricingService can call it without importing the service package.
func GetWeatherSurcharge(ctx context.Context, r PlatformSettingRepo) (enabled bool, amountKobo int64) {
	if v, err := r.GetLatest(ctx, "weather_surcharge_enabled"); err == nil {
		enabled, _ = strconv.ParseBool(v)
	}
	if v, err := r.GetLatest(ctx, "weather_surcharge_kobo"); err == nil {
		amountKobo, _ = strconv.ParseInt(v, 10, 64)
	}
	if amountKobo <= 0 {
		amountKobo = 20000 // ₦200 default
	}
	return enabled, amountKobo
}
