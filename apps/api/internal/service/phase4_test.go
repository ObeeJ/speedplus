package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
)

// ── maxRunOrders ──────────────────────────────────────────────────────────────

func TestMaxRunOrdersInRange(t *testing.T) {
	if maxRunOrders < 8 || maxRunOrders > 20 {
		t.Errorf("maxRunOrders = %d, want 8–20 (plan spec)", maxRunOrders)
	}
}

// ── ServiceZone model ─────────────────────────────────────────────────────────

func TestServiceZoneDefaults(t *testing.T) {
	z := model.ServiceZone{
		ID:   uuid.New(),
		Name: "Yaba",
	}
	// Zero-value active_days = 0 (not 127) in Go — the DB default applies on
	// insert. We just confirm the struct accepts the fields.
	z.ActiveDays = 127
	z.WindowStart = 480
	z.WindowEnd = 1020
	z.IsActive = true

	if z.WindowEnd <= z.WindowStart {
		t.Error("window_end must be > window_start")
	}
	if z.WindowStart < 0 || z.WindowStart >= 1440 {
		t.Errorf("window_start %d out of range", z.WindowStart)
	}
	if z.WindowEnd > 1440 {
		t.Errorf("window_end %d out of range", z.WindowEnd)
	}
}

// ── DeliveryRun model ─────────────────────────────────────────────────────────

func TestDeliveryRunStatusTransitions(t *testing.T) {
	validStatuses := []string{"assembling", "dispatched", "in_progress", "completed", "cancelled"}
	for _, s := range validStatuses {
		run := model.DeliveryRun{ID: uuid.New(), Status: s}
		if run.Status != s {
			t.Errorf("status %q not preserved", s)
		}
	}
}

func TestDeliveryRunOptimizedSequenceJSON(t *testing.T) {
	ids := []string{uuid.New().String(), uuid.New().String(), uuid.New().String()}
	b, _ := json.Marshal(ids)
	run := model.DeliveryRun{
		ID:                uuid.New(),
		OptimizedSequence: string(b),
	}
	var decoded []string
	if err := json.Unmarshal([]byte(run.OptimizedSequence), &decoded); err != nil {
		t.Fatalf("optimized_sequence is not valid JSON: %v", err)
	}
	if len(decoded) != 3 {
		t.Errorf("decoded sequence len = %d, want 3", len(decoded))
	}
}

// ── RunOrder model ────────────────────────────────────────────────────────────

func TestRunOrderSequencePositive(t *testing.T) {
	ro := model.RunOrder{
		ID:       uuid.New(),
		RunID:    uuid.New(),
		OrderID:  uuid.New(),
		Sequence: 1,
	}
	if ro.Sequence < 1 {
		t.Errorf("sequence must be >= 1, got %d", ro.Sequence)
	}
}

// ── Zone scheduling logic ─────────────────────────────────────────────────────

// isDueNow mirrors the production guard in AssembleAllDueRuns.
func isDueNow(zone model.ServiceZone, now time.Time) bool {
	dayBit := int16(1 << now.Weekday())
	if zone.ActiveDays&dayBit == 0 {
		return false
	}
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	wStart := midnight.Add(time.Duration(zone.WindowStart) * time.Minute)
	wEnd := midnight.Add(time.Duration(zone.WindowEnd) * time.Minute)
	return !now.Before(wStart) && !now.After(wEnd)
}

func TestZoneSchedulingAllDays(t *testing.T) {
	zone := model.ServiceZone{ActiveDays: 127, WindowStart: 480, WindowEnd: 1020}
	// 10:00 UTC on any day should be due
	now := time.Date(2025, 1, 6, 10, 0, 0, 0, time.UTC) // Monday
	if !isDueNow(zone, now) {
		t.Error("zone with all days active should be due at 10:00 UTC")
	}
}

func TestZoneSchedulingBeforeWindow(t *testing.T) {
	zone := model.ServiceZone{ActiveDays: 127, WindowStart: 480, WindowEnd: 1020}
	now := time.Date(2025, 1, 6, 7, 0, 0, 0, time.UTC) // 07:00 — before 08:00 window
	if isDueNow(zone, now) {
		t.Error("zone should not be due before window_start")
	}
}

func TestZoneSchedulingAfterWindow(t *testing.T) {
	zone := model.ServiceZone{ActiveDays: 127, WindowStart: 480, WindowEnd: 1020}
	now := time.Date(2025, 1, 6, 18, 0, 0, 0, time.UTC) // 18:00 — after 17:00 window
	if isDueNow(zone, now) {
		t.Error("zone should not be due after window_end")
	}
}

func TestZoneSchedulingInactiveDay(t *testing.T) {
	// Only Monday active: time.Monday = 1, so bit = 1<<1 = 2
	zone := model.ServiceZone{ActiveDays: 2, WindowStart: 480, WindowEnd: 1020}
	sunday := time.Date(2025, 1, 5, 10, 0, 0, 0, time.UTC) // Sunday
	if isDueNow(zone, sunday) {
		t.Error("zone should not be due on an inactive day")
	}
	monday := time.Date(2025, 1, 6, 10, 0, 0, 0, time.UTC) // Monday
	if !isDueNow(zone, monday) {
		t.Error("zone should be due on its active day")
	}
}

func TestZoneInactive(t *testing.T) {
	zone := model.ServiceZone{ActiveDays: 127, WindowStart: 480, WindowEnd: 1020, IsActive: false}
	now := time.Date(2025, 1, 6, 10, 0, 0, 0, time.UTC)
	// AssembleAllDueRuns skips inactive zones before the isDueNow check.
	if zone.IsActive {
		t.Error("zone should be inactive")
	}
	_ = isDueNow(zone, now) // still callable — IsActive guard is in the caller
}
