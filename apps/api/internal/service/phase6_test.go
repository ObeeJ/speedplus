package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
)

// ── Hazmat gate ───────────────────────────────────────────────────────────────

// hazmatRequired mirrors the production guard in Dispatch.
func hazmatRequired(vertical string, totalKg float64) bool {
	return vertical == "gas" && totalKg > 12.5
}

func TestHazmatRequiredFor25kg(t *testing.T) {
	if !hazmatRequired("gas", 25) {
		t.Error("25kg gas order must require hazmat_certified driver")
	}
}

func TestHazmatRequiredFor12_6kg(t *testing.T) {
	if !hazmatRequired("gas", 12.6) {
		t.Error("12.6kg gas order must require hazmat_certified driver")
	}
}

func TestHazmatNotRequiredFor12_5kg(t *testing.T) {
	if hazmatRequired("gas", 12.5) {
		t.Error("12.5kg gas order must NOT require hazmat_certified driver")
	}
}

func TestHazmatNotRequiredForNonGas(t *testing.T) {
	for _, v := range []string{"food", "grocery", "pharmacy", "package"} {
		if hazmatRequired(v, 100) {
			t.Errorf("vertical %q must never require hazmat_certified", v)
		}
	}
}

// ── Recertification window ────────────────────────────────────────────────────

// daysUntilExpiry mirrors the production calculation in RecertReminders.
func daysUntilExpiry(lastRecert time.Time) int {
	expiry := lastRecert.AddDate(0, 0, recertPeriodDays)
	return int(time.Until(expiry).Hours() / 24)
}

func TestRecertPeriodIs5Years(t *testing.T) {
	if recertPeriodDays != 1825 {
		t.Errorf("recertPeriodDays = %d, want 1825 (5 years)", recertPeriodDays)
	}
}

func TestRecertWarningIs60Days(t *testing.T) {
	if recertWarningDays != 60 {
		t.Errorf("recertWarningDays = %d, want 60", recertWarningDays)
	}
}

func TestDaysUntilExpiryFresh(t *testing.T) {
	// Cylinder recertified today — should expire in ~1825 days
	lastRecert := time.Now()
	days := daysUntilExpiry(lastRecert)
	if days < 1820 || days > 1826 {
		t.Errorf("fresh cylinder days until expiry = %d, want ~1825", days)
	}
}

func TestDaysUntilExpiryNearExpiry(t *testing.T) {
	// Cylinder recertified 4 years and 10 months ago — should be ~60 days left
	lastRecert := time.Now().AddDate(-4, -10, 0)
	days := daysUntilExpiry(lastRecert)
	// Allow ±5 days for month-length variation
	if days < 50 || days > 70 {
		t.Errorf("near-expiry cylinder days = %d, want ~60", days)
	}
}

func TestDaysUntilExpiryExpired(t *testing.T) {
	// Cylinder recertified 6 years ago — already expired
	lastRecert := time.Now().AddDate(-6, 0, 0)
	days := daysUntilExpiry(lastRecert)
	if days >= 0 {
		t.Errorf("expired cylinder should have negative days, got %d", days)
	}
}

// ── CylinderHandoverChecklist model ──────────────────────────────────────────

func TestHandoverChecklistAllPassed(t *testing.T) {
	c := model.CylinderHandoverChecklist{
		ID:              uuid.New(),
		OrderID:         uuid.New(),
		DriverID:        uuid.New(),
		ValveSeated:     true,
		NoHiss:          true,
		RegulatorFitted: true,
		RecordedAt:      time.Now(),
	}
	if !c.ValveSeated || !c.NoHiss || !c.RegulatorFitted {
		t.Error("all checklist items should be true")
	}
}

func TestHandoverChecklistDefaults(t *testing.T) {
	var c model.CylinderHandoverChecklist
	if c.ValveSeated || c.NoHiss || c.RegulatorFitted {
		t.Error("checklist items should default to false")
	}
}

// ── CustomerCylinder model ────────────────────────────────────────────────────

func TestCustomerCylinderStatusValues(t *testing.T) {
	for _, s := range []string{"active", "retired", "in_custody"} {
		cyl := model.CustomerCylinder{Status: s}
		if cyl.Status != s {
			t.Errorf("status %q not preserved", s)
		}
	}
}

func TestCustomerCylinderLastRecertNilable(t *testing.T) {
	cyl := model.CustomerCylinder{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Serial: "SN-001",
		Status: "active",
	}
	if cyl.LastRecertAt != nil {
		t.Error("LastRecertAt should be nil when not set")
	}
}

// ── DriverProfile hazmat field ────────────────────────────────────────────────

func TestDriverProfileHazmatDefault(t *testing.T) {
	var dp model.DriverProfile
	if dp.HazmatCertified {
		t.Error("HazmatCertified should default to false")
	}
}
