package service

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

const weatherSettingCacheTTL = 60 * time.Second

type weatherCache struct {
	enabled   bool
	kobo      int64
	fetchedAt time.Time
}

// WeatherSurchargeService reads and writes the two platform_settings rows
// that control the weather surcharge. Cached for 60 s — same discipline as
// fee configs so an admin change takes effect on new quotes within ~1 minute.
type WeatherSurchargeService struct {
	repo      repo.PlatformSettingRepo
	feeRepo   repo.FeeConfigRepo // for audit log
	mu        sync.RWMutex
	cached    *weatherCache
}

func NewWeatherSurchargeService(r repo.PlatformSettingRepo, feeRepo repo.FeeConfigRepo) *WeatherSurchargeService {
	return &WeatherSurchargeService{repo: r, feeRepo: feeRepo}
}

// Get returns the live enabled flag and surcharge amount.
func (s *WeatherSurchargeService) Get(ctx context.Context) (enabled bool, amountKobo int64) {
	s.mu.RLock()
	if s.cached != nil && time.Since(s.cached.fetchedAt) < weatherSettingCacheTTL {
		e, k := s.cached.enabled, s.cached.kobo
		s.mu.RUnlock()
		return e, k
	}
	s.mu.RUnlock()

	enabled, amountKobo = repo.GetWeatherSurcharge(ctx, s.repo)

	s.mu.Lock()
	s.cached = &weatherCache{enabled: enabled, kobo: amountKobo, fetchedAt: time.Now()}
	s.mu.Unlock()
	return enabled, amountKobo
}

// Set writes both settings in a transaction and audit-logs the change.
func (s *WeatherSurchargeService) Set(ctx context.Context, adminID uuid.UUID, enabled bool, amountKobo int64, reason string) error {
	if reason == "" {
		return fmt.Errorf("reason is required")
	}
	if amountKobo < 0 {
		return fmt.Errorf("weather_surcharge_kobo must be >= 0")
	}

	err := s.feeRepo.Transaction(ctx, func(tx *gorm.DB) error {
		now := time.Now()
		rows := []model.PlatformSetting{
			{ID: uuid.New(), Key: "weather_surcharge_enabled", Value: strconv.FormatBool(enabled), UpdatedBy: adminID, Reason: reason, CreatedAt: now},
			{ID: uuid.New(), Key: "weather_surcharge_kobo", Value: strconv.FormatInt(amountKobo, 10), UpdatedBy: adminID, Reason: reason, CreatedAt: now},
		}
		for i := range rows {
			if err := tx.WithContext(ctx).Create(&rows[i]).Error; err != nil {
				return err
			}
		}
		return tx.WithContext(ctx).Create(&model.AdminAuditLog{
			ID:         uuid.New(),
			AdminID:    adminID,
			Action:     "weather_surcharge_change",
			TargetType: "platform_setting",
			TargetID:   uuid.Nil,
			Reason:     reason,
			CreatedAt:  now,
		}).Error
	})
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.cached = nil // invalidate so next quote reads fresh
	s.mu.Unlock()
	return nil
}
