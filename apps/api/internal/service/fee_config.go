package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

// feeConfigCacheTTL bounds how stale a quote's fee config can be after an
// admin change. Signed quotes stay internally consistent either way.
const feeConfigCacheTTL = 60 * time.Second

// fuelSuggestionDeltaPct: fuel reference moves beyond this fraction trigger a
// per-km adjustment suggestion (never auto-applied).
const fuelSuggestionDeltaPct = 0.10

type cachedFees struct {
	fees      FeeConfig
	fetchedAt time.Time
}

// FeeConfigService serves per-vertical pricing from the append-only
// fee_configs table, falling back to DefaultFeeTable when no row exists.
type FeeConfigService struct {
	db    *gorm.DB
	mu    sync.RWMutex
	cache map[string]cachedFees
}

func NewFeeConfigService(db *gorm.DB) *FeeConfigService {
	return &FeeConfigService{db: db, cache: make(map[string]cachedFees)}
}

func toFees(r *model.FeeConfig) FeeConfig {
	return FeeConfig{
		BaseFeeKobo:      r.BaseFeeKobo,
		PerKmKobo:        r.PerKmKobo,
		PerKgKobo:        r.PerKgKobo,
		ServicePct:       r.ServicePct,
		MerchantTakeRate: r.MerchantTakeRate,
		DriverTakeRate:   r.DriverTakeRate,
		PlatformTakeRate: r.PlatformTakeRate,
	}
}

func defaultFees(vertical string) FeeConfig {
	if f, ok := DefaultFeeTable[vertical]; ok {
		return f
	}
	return DefaultFeeTable["package"]
}

// GetFees returns the live config for a vertical (latest effective_at <= now),
// cached for feeConfigCacheTTL. Missing rows fall back to DefaultFeeTable.
func (s *FeeConfigService) GetFees(ctx context.Context, vertical string) FeeConfig {
	s.mu.RLock()
	if c, ok := s.cache[vertical]; ok && time.Since(c.fetchedAt) < feeConfigCacheTTL {
		s.mu.RUnlock()
		return c.fees
	}
	s.mu.RUnlock()

	fees := s.GetFeesAt(ctx, vertical, time.Now())

	s.mu.Lock()
	s.cache[vertical] = cachedFees{fees: fees, fetchedAt: time.Now()}
	s.mu.Unlock()
	return fees
}

// GetFeesAt returns the config effective at a point in time — used by
// settlement to pin the split to the rates in force when the order was
// created, so mid-flight admin changes never reallocate money. Uncached.
func (s *FeeConfigService) GetFeesAt(ctx context.Context, vertical string, at time.Time) FeeConfig {
	var row model.FeeConfig
	err := s.db.WithContext(ctx).
		Where("vertical = ? AND effective_at <= ?", vertical, at).
		Order("effective_at DESC").
		First(&row).Error
	if err != nil {
		return defaultFees(vertical)
	}
	return toFees(&row)
}

// List returns the latest config row per vertical. Verticals with no DB row
// are synthesised from DefaultFeeTable (zero ID marks them as defaults).
func (s *FeeConfigService) List(ctx context.Context) ([]model.FeeConfig, error) {
	var rows []model.FeeConfig
	err := s.db.WithContext(ctx).
		Raw(`SELECT DISTINCT ON (vertical) * FROM fee_configs
		     WHERE effective_at <= NOW()
		     ORDER BY vertical, effective_at DESC`).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(rows))
	for _, r := range rows {
		seen[r.Vertical] = true
	}
	for _, v := range []string{"food", "grocery", "pharmacy", "gas", "package"} {
		if !seen[v] {
			f := DefaultFeeTable[v]
			rows = append(rows, model.FeeConfig{
				Vertical:         v,
				BaseFeeKobo:      f.BaseFeeKobo,
				PerKmKobo:        f.PerKmKobo,
				PerKgKobo:        f.PerKgKobo,
				ServicePct:       f.ServicePct,
				MerchantTakeRate: f.MerchantTakeRate,
				DriverTakeRate:   f.DriverTakeRate,
				PlatformTakeRate: f.PlatformTakeRate,
			})
		}
	}
	return rows, nil
}

// FeeConfigInput is the admin-supplied new config for one vertical.
type FeeConfigInput struct {
	Vertical         string
	BaseFeeKobo      int64
	PerKmKobo        int64
	PerKgKobo        int64
	ServicePct       float64
	MerchantTakeRate float64
	DriverTakeRate   float64
	PlatformTakeRate float64
	FuelPriceRefKobo int64
	Reason           string
}

// FuelSuggestion is returned when the fuel reference price moved >10%:
// a proposed per-km rate the admin may apply with a follow-up save.
type FuelSuggestion struct {
	PrevFuelKobo       int64 `json:"prevFuelKobo"`
	NewFuelKobo        int64 `json:"newFuelKobo"`
	CurrentPerKmKobo   int64 `json:"currentPerKmKobo"`
	SuggestedPerKmKobo int64 `json:"suggestedPerKmKobo"`
}

func (in FeeConfigInput) validate() error {
	if _, ok := DefaultFeeTable[in.Vertical]; !ok {
		return fmt.Errorf("unknown vertical %q", in.Vertical)
	}
	if in.BaseFeeKobo < 0 || in.PerKmKobo < 0 || in.PerKgKobo < 0 || in.FuelPriceRefKobo < 0 {
		return errors.New("fee amounts must be >= 0")
	}
	if in.ServicePct < 0 || in.ServicePct > 0.5 {
		return errors.New("service pct must be between 0 and 0.5")
	}
	if in.MerchantTakeRate <= 0.5 || in.MerchantTakeRate > 1 {
		return errors.New("merchant take rate must be in (0.5, 1]")
	}
	if math.Abs(in.DriverTakeRate+in.PlatformTakeRate-1.0) > 1e-9 {
		return errors.New("driver + platform take rates must sum to 1.0")
	}
	if in.Reason == "" {
		return errors.New("reason is required")
	}
	return nil
}

// Upsert appends a new config version for the vertical (insert-only — the DB
// rules silently no-op updates, so this must never use Save), writes an admin
// audit log in the same transaction, and invalidates the cache. Returns the
// new row plus a fuel suggestion when the fuel reference moved >10%.
func (s *FeeConfigService) Upsert(ctx context.Context, adminID uuid.UUID, in FeeConfigInput) (*model.FeeConfig, *FuelSuggestion, error) {
	if err := in.validate(); err != nil {
		return nil, nil, err
	}

	var prev *model.FeeConfig
	var p model.FeeConfig
	if err := s.db.WithContext(ctx).
		Where("vertical = ? AND effective_at <= NOW()", in.Vertical).
		Order("effective_at DESC").
		First(&p).Error; err == nil {
		prev = &p
	}

	row := &model.FeeConfig{
		ID:               uuid.New(),
		Vertical:         in.Vertical,
		BaseFeeKobo:      in.BaseFeeKobo,
		PerKmKobo:        in.PerKmKobo,
		PerKgKobo:        in.PerKgKobo,
		ServicePct:       in.ServicePct,
		MerchantTakeRate: in.MerchantTakeRate,
		DriverTakeRate:   in.DriverTakeRate,
		PlatformTakeRate: in.PlatformTakeRate,
		FuelPriceRefKobo: in.FuelPriceRefKobo,
		EffectiveAt:      time.Now(),
		UpdatedBy:        adminID,
		Reason:           in.Reason,
		CreatedAt:        time.Now(),
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		return tx.Create(&model.AdminAuditLog{
			ID:         uuid.New(),
			AdminID:    adminID,
			Action:     "fee_config_change",
			TargetType: "fee_config",
			TargetID:   row.ID,
			Reason:     in.Reason,
			CreatedAt:  time.Now(),
		}).Error
	})
	if err != nil {
		return nil, nil, err
	}

	s.mu.Lock()
	delete(s.cache, in.Vertical)
	s.mu.Unlock()

	return row, s.fuelSuggestion(prev, row), nil
}

func (s *FeeConfigService) fuelSuggestion(prev, cur *model.FeeConfig) *FuelSuggestion {
	if prev == nil || prev.FuelPriceRefKobo == 0 || cur.FuelPriceRefKobo == 0 {
		return nil
	}
	delta := math.Abs(float64(cur.FuelPriceRefKobo-prev.FuelPriceRefKobo)) / float64(prev.FuelPriceRefKobo)
	if delta <= fuelSuggestionDeltaPct {
		return nil
	}
	suggested := int64(math.Round(float64(cur.PerKmKobo) * float64(cur.FuelPriceRefKobo) / float64(prev.FuelPriceRefKobo)))
	return &FuelSuggestion{
		PrevFuelKobo:       prev.FuelPriceRefKobo,
		NewFuelKobo:        cur.FuelPriceRefKobo,
		CurrentPerKmKobo:   cur.PerKmKobo,
		SuggestedPerKmKobo: suggested,
	}
}
