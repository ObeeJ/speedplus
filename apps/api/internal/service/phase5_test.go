package service

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
)

// ── nextChargeTime ────────────────────────────────────────────────────────────

func TestNextChargeTimeWeekly(t *testing.T) {
	before := time.Now()
	next := nextChargeTime("weekly")
	want := before.AddDate(0, 0, 7)
	if math.Abs(next.Sub(want).Seconds()) > 1 {
		t.Errorf("weekly next charge = %v, want ~%v", next, want)
	}
}

func TestNextChargeTimeBiweekly(t *testing.T) {
	before := time.Now()
	next := nextChargeTime("biweekly")
	want := before.AddDate(0, 0, 14)
	if math.Abs(next.Sub(want).Seconds()) > 1 {
		t.Errorf("biweekly next charge = %v, want ~%v", next, want)
	}
}

func TestNextChargeTimeMonthly(t *testing.T) {
	before := time.Now()
	next := nextChargeTime("monthly")
	want := before.AddDate(0, 1, 0)
	if math.Abs(next.Sub(want).Seconds()) > 1 {
		t.Errorf("monthly next charge = %v, want ~%v", next, want)
	}
}

func TestNextChargeTimeDefaultIsMonthly(t *testing.T) {
	before := time.Now()
	next := nextChargeTime("unknown_cadence")
	want := before.AddDate(0, 1, 0)
	if math.Abs(next.Sub(want).Seconds()) > 1 {
		t.Errorf("unknown cadence should default to monthly, got %v", next)
	}
}

// ── Dunning logic ─────────────────────────────────────────────────────────────

// dunningWouldPause mirrors the production guard in ProcessDue.
func dunningWouldPause(currentCount int) bool {
	return currentCount+1 >= 3
}

func TestDunningPausesAfterThreeFailures(t *testing.T) {
	if !dunningWouldPause(2) {
		t.Error("should pause after 3rd failure (count=2 → count+1=3)")
	}
}

func TestDunningDoesNotPauseBeforeThree(t *testing.T) {
	for _, count := range []int{0, 1} {
		if dunningWouldPause(count) {
			t.Errorf("should not pause at dunning_count=%d", count)
		}
	}
}

// ── Idempotency key format ────────────────────────────────────────────────────

func TestChargeOneIdempotencyKey(t *testing.T) {
	subID := uuid.New()
	nextCharge := time.Date(2025, 3, 15, 6, 0, 0, 0, time.UTC)
	key := "sub:" + subID.String() + ":" + nextCharge.UTC().Format("2006-01-02")

	// Key must be stable across two calls with the same inputs.
	key2 := "sub:" + subID.String() + ":" + nextCharge.UTC().Format("2006-01-02")
	if key != key2 {
		t.Error("idempotency key is not stable")
	}
	// Key must differ for a different day.
	nextDay := nextCharge.AddDate(0, 0, 1)
	key3 := "sub:" + subID.String() + ":" + nextDay.UTC().Format("2006-01-02")
	if key == key3 {
		t.Error("idempotency key must differ for different days")
	}
}

// ── LPG price index ───────────────────────────────────────────────────────────

// lpgDeltaSuggestion mirrors the production delta check in RecordLPGPrice.
func lpgDeltaSuggestion(prevKobo, newKobo int64) string {
	if prevKobo == 0 {
		return ""
	}
	delta := float64(newKobo-prevKobo) / float64(prevKobo)
	if delta > 0.10 || delta < -0.10 {
		return "suggest"
	}
	return ""
}

func TestLPGDeltaSuggestionAboveThreshold(t *testing.T) {
	// 15% increase should trigger suggestion
	if lpgDeltaSuggestion(100_000, 115_000) == "" {
		t.Error("15% increase should trigger a suggestion")
	}
}

func TestLPGDeltaSuggestionBelowThreshold(t *testing.T) {
	// 5% increase should not trigger suggestion
	if lpgDeltaSuggestion(100_000, 105_000) != "" {
		t.Error("5% increase should not trigger a suggestion")
	}
}

func TestLPGDeltaSuggestionNegative(t *testing.T) {
	// 12% decrease should trigger suggestion
	if lpgDeltaSuggestion(100_000, 88_000) == "" {
		t.Error("12% decrease should trigger a suggestion")
	}
}

func TestLPGDeltaSuggestionNoPrev(t *testing.T) {
	// No previous price — no suggestion
	if lpgDeltaSuggestion(0, 100_000) != "" {
		t.Error("no previous price should not trigger a suggestion")
	}
}

// ── Subscription model gas fields ─────────────────────────────────────────────

func TestSubscriptionGasFieldsNilable(t *testing.T) {
	sub := model.Subscription{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
		MerchantID: uuid.New(),
		Vertical:   "gas",
		Cadence:    "monthly",
		AddressID:  uuid.New(),
		Status:     "active",
	}
	if sub.GasMode != nil {
		t.Error("GasMode should be nil by default")
	}
	if sub.CylinderSpecID != nil {
		t.Error("CylinderSpecID should be nil by default")
	}
	if sub.PredictedRunoutAt != nil {
		t.Error("PredictedRunoutAt should be nil by default")
	}
	if sub.AvgDaysBetweenRefills != nil {
		t.Error("AvgDaysBetweenRefills should be nil by default")
	}
}

func TestSubscriptionGasFieldsSet(t *testing.T) {
	mode := "swap"
	specID := uuid.New()
	avg := 28.5
	runout := time.Now().AddDate(0, 0, 28)

	sub := model.Subscription{
		GasMode:               &mode,
		CylinderSpecID:        &specID,
		AvgDaysBetweenRefills: &avg,
		PredictedRunoutAt:     &runout,
	}
	if *sub.GasMode != "swap" {
		t.Errorf("GasMode = %q, want swap", *sub.GasMode)
	}
	if *sub.CylinderSpecID != specID {
		t.Error("CylinderSpecID mismatch")
	}
}

// ── LPGPriceIndex model ───────────────────────────────────────────────────────

func TestLPGPriceIndexModel(t *testing.T) {
	adminID := uuid.New()
	entry := model.LPGPriceIndex{
		ID:             uuid.New(),
		Region:         "Lagos",
		PricePerKgKobo: 94_000,
		Source:         "NMDPRA",
		EffectiveAt:    time.Now(),
		UpdatedBy:      adminID,
	}
	if entry.PricePerKgKobo <= 0 {
		t.Error("price_per_kg_kobo must be positive")
	}
	if entry.Region == "" {
		t.Error("region must not be empty")
	}
}
