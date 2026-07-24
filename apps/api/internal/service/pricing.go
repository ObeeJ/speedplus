package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/config"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

// SizeCategory classifies package dimensions.
type SizeCategory string

const (
	SizeSmall  SizeCategory = "small"  // fits in a backpack
	SizeMedium SizeCategory = "medium" // shoebox-sized
	SizeLarge  SizeCategory = "large"  // requires van
)

// FeeConfig holds per-vertical delivery fee parameters.
type FeeConfig struct {
	BaseFeeKobo      int64   // flat base
	PerKmKobo        int64   // per km
	PerKgKobo        int64   // per kg of weight (package vertical)
	ServicePct       float64 // platform service fee % of subtotal
	MerchantTakeRate float64 // merchant share % (e.g. 0.92 = 8% commission)
	DriverTakeRate   float64 // driver share % of delivery fee
	PlatformTakeRate float64 // platform share % of delivery fee
}

// sizeSurchargeKobo is a flat surcharge added on top of weight for large packages.
var sizeSurchargeKobo = map[SizeCategory]int64{
	SizeSmall:  0,
	SizeMedium: 15000, // ₦150
	SizeLarge:  40000, // ₦400
}

// DefaultFeeTable — 2026 rates per vertical, calibrated so a rider nets a
// living wage after fuel/maintenance at ₦1,400/L (dead-km included).
// Merchant 8% commission. Gas is a flat fee (per-km 0): fixed cylinder runs.
// Fallback only once fee_configs rows exist — live rates come from FeeConfigService.
var DefaultFeeTable = map[string]FeeConfig{
	"food":     {BaseFeeKobo: 90000, PerKmKobo: 15000, ServicePct: 0.05, MerchantTakeRate: 0.92, DriverTakeRate: 0.80, PlatformTakeRate: 0.20},
	"grocery":  {BaseFeeKobo: 90000, PerKmKobo: 15000, ServicePct: 0.05, MerchantTakeRate: 0.92, DriverTakeRate: 0.80, PlatformTakeRate: 0.20},
	"pharmacy": {BaseFeeKobo: 100000, PerKmKobo: 15000, ServicePct: 0.05, MerchantTakeRate: 0.92, DriverTakeRate: 0.80, PlatformTakeRate: 0.20},
	"gas":      {BaseFeeKobo: 150000, PerKmKobo: 0, ServicePct: 0.03, MerchantTakeRate: 0.92, DriverTakeRate: 0.80, PlatformTakeRate: 0.20},
	"package":  {BaseFeeKobo: 90000, PerKmKobo: 17000, PerKgKobo: 7000, ServicePct: 0.04, MerchantTakeRate: 0.92, DriverTakeRate: 0.80, PlatformTakeRate: 0.20},
}

type PricingService struct {
	db         *gorm.DB
	cfg        *config.Config
	osrmURL    string
	httpClient *http.Client
	feeConfigs *FeeConfigService // nil => DefaultFeeTable fallback
}

func NewPricingService(db *gorm.DB, cfg *config.Config, osrmURL string, feeConfigs *FeeConfigService) *PricingService {
	return &PricingService{
		db:         db,
		cfg:        cfg,
		osrmURL:    osrmURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		feeConfigs: feeConfigs,
	}
}

type QuoteRequest struct {
	CustomerID   uuid.UUID
	MerchantID   uuid.UUID
	Vertical     string
	SubtotalKobo int64
	OriginLat    float64
	OriginLng    float64
	DestLat      float64
	DestLng      float64
	// Package-only fields
	WeightKg     float64
	SizeCategory SizeCategory
}

func (s *PricingService) Quote(ctx context.Context, req QuoteRequest) (*model.PricingQuote, error) {
	distKm, etaMinutes, err := s.osrmRoute(ctx, req.OriginLat, req.OriginLng, req.DestLat, req.DestLng)
	if err != nil {
		return nil, fmt.Errorf("osrm: %w", err)
	}

	var fees FeeConfig
	if s.feeConfigs != nil {
		fees = s.feeConfigs.GetFees(ctx, req.Vertical)
	} else {
		fees = defaultFees(req.Vertical)
	}

	weatherAdvisory := s.weatherAdvisory(ctx, req.DestLat, req.DestLng)

	deliveryKobo := fees.BaseFeeKobo + int64(distKm*float64(fees.PerKmKobo))
	// Weather surcharge disabled — advisory shown to all parties but no fee charged.

	// Weight + size surcharge for package vertical
	if req.Vertical == "package" {
		deliveryKobo += int64(req.WeightKg * float64(fees.PerKgKobo))
		deliveryKobo += sizeSurchargeKobo[req.SizeCategory]
	}

	serviceKobo := int64(float64(req.SubtotalKobo) * fees.ServicePct)
	totalKobo := req.SubtotalKobo + deliveryKobo + serviceKobo

	quote := &model.PricingQuote{
		ID:              uuid.New(),
		CustomerID:      req.CustomerID,
		MerchantID:      req.MerchantID,
		DistanceKm:      distKm,
		ETAMinutes:      etaMinutes,
		WeightKg:        req.WeightKg,
		SizeCategory:    string(req.SizeCategory),
		SubtotalKobo:    req.SubtotalKobo,
		DeliveryKobo:    deliveryKobo,
		ServiceKobo:     serviceKobo,
		TotalKobo:       totalKobo,
		WeatherAdvisory: weatherAdvisory,
		ExpiresAt:       time.Now().Add(10 * time.Minute),
	}
	quote.QuoteHash = s.signQuote(quote)

	if err := s.db.WithContext(ctx).Create(quote).Error; err != nil {
		return nil, err
	}
	return quote, nil
}

// ValidateQuote checks the quote is unexpired, unused, and untampered.
func (s *PricingService) ValidateQuote(ctx context.Context, quoteID uuid.UUID, subtotalKobo int64) (*model.PricingQuote, error) {
	var q model.PricingQuote
	if err := s.db.WithContext(ctx).First(&q, quoteID).Error; err != nil {
		return nil, fmt.Errorf("quote not found")
	}
	if q.UsedAt != nil {
		return nil, fmt.Errorf("quote already used")
	}
	if time.Now().After(q.ExpiresAt) {
		return nil, fmt.Errorf("quote expired")
	}
	if q.SubtotalKobo != subtotalKobo {
		return nil, fmt.Errorf("quote subtotal mismatch")
	}
	expected := s.signQuote(&q)
	if expected != q.QuoteHash {
		return nil, fmt.Errorf("quote tampered")
	}
	return &q, nil
}

func (s *PricingService) MarkQuoteUsed(ctx context.Context, quoteID uuid.UUID) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&model.PricingQuote{}).
		Where("id = ?", quoteID).
		Update("used_at", now).Error
}

func (s *PricingService) signQuote(q *model.PricingQuote) string {
	payload := fmt.Sprintf("%s:%s:%s:%d:%d:%d:%d",
		q.ID, q.CustomerID, q.MerchantID,
		q.SubtotalKobo, q.DeliveryKobo, q.ServiceKobo, q.TotalKobo,
	)
	mac := hmac.New(sha256.New, []byte(s.cfg.JWTSecret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// osrmRoute returns road distance (km) and travel duration (minutes) from OSRM.
func (s *PricingService) osrmRoute(ctx context.Context, oLat, oLng, dLat, dLng float64) (distKm float64, etaMinutes int, err error) {
	url := fmt.Sprintf("%s/route/v1/driving/%f,%f;%f,%f?overview=false",
		s.osrmURL, oLng, oLat, dLng, dLat)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Routes []struct {
			Distance float64 `json:"distance"` // metres
			Duration float64 `json:"duration"` // seconds
		} `json:"routes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Routes) == 0 {
		return 0, 0, fmt.Errorf("osrm no route")
	}
	distKm = result.Routes[0].Distance / 1000.0
	etaMinutes = int(result.Routes[0].Duration/60) + 5 // +5 min pickup buffer
	return distKm, etaMinutes, nil
}

// weatherAdvisory calls Open-Meteo for rain/heat flag.
// Returns a human-readable advisory string (empty = clear conditions).
// No fee is charged — this is informational only for all parties.
func (s *PricingService) weatherAdvisory(ctx context.Context, lat, lng float64) string {
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current_weather=true", lat, lng)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		CurrentWeather struct {
			Weathercode int     `json:"weathercode"`
			Temperature float64 `json:"temperature"`
		} `json:"current_weather"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}

	code := result.CurrentWeather.Weathercode
	temp := result.CurrentWeather.Temperature
	switch {
	case code >= 95 && code <= 99:
		return "Thunderstorm expected — allow extra time for delivery"
	case code >= 71 && code <= 77:
		return "Heavy rain expected — rider may take longer than usual"
	case code >= 51 && code <= 67:
		return "Light rain in the area — rider is on the way"
	case temp > 38:
		return "Extreme heat today — rider may need a short break"
	default:
		return ""
	}
}
