package service

import (
	"math"
	"testing"

	"github.com/speedplus/api/internal/model"
)

// The take-rate split is applied at settlement; if it doesn't sum to 1.0 the
// journal silently leaks (or double-pays) delivery-fee kobo.
func TestDefaultFeeTableInvariants(t *testing.T) {
	required := []string{"food", "grocery", "pharmacy", "gas", "package"}
	for _, v := range required {
		if _, ok := DefaultFeeTable[v]; !ok {
			t.Fatalf("vertical %q missing from DefaultFeeTable", v)
		}
	}

	for vertical, f := range DefaultFeeTable {
		if f.BaseFeeKobo < 0 || f.PerKmKobo < 0 || f.PerKgKobo < 0 {
			t.Errorf("%s: negative fee component: %+v", vertical, f)
		}
		if f.ServicePct < 0 || f.ServicePct > 0.5 {
			t.Errorf("%s: service pct out of range: %v", vertical, f.ServicePct)
		}
		if f.MerchantTakeRate <= 0.5 || f.MerchantTakeRate > 1 {
			t.Errorf("%s: merchant take rate out of range: %v", vertical, f.MerchantTakeRate)
		}
		if sum := f.DriverTakeRate + f.PlatformTakeRate; math.Abs(sum-1.0) > 1e-9 {
			t.Errorf("%s: driver+platform take rates = %v, want 1.0", vertical, sum)
		}
	}
}

// Gas fee table must have distance and weight components — the old flat/zero
// config is what made every gas delivery rider-negative.
func TestGasFeeTableHasDistanceAndWeight(t *testing.T) {
	gas := DefaultFeeTable["gas"]
	if gas.PerKmKobo == 0 {
		t.Error("gas PerKmKobo must be > 0: distance-blind pricing inverts rider economics")
	}
	if gas.PerKgKobo == 0 {
		t.Error("gas PerKgKobo must be > 0: weight must be priced")
	}
	if gas.PerStopKobo == 0 {
		t.Error("gas PerStopKobo must be > 0: required for batched multi-drop runs")
	}
}

// Worked example from the plan: 12.5 kg swap, 4 km.
// Delivery = 800 + 4×220 + 12.5×20 = 800 + 880 + 250 = 1930 (₦19.30 in kobo = 193000).
func TestGasFeeWorkedExample(t *testing.T) {
	gas := DefaultFeeTable["gas"]
	const distKm = 4.0
	const weightKg = 12.5
	delivery := gas.BaseFeeKobo +
		int64(distKm*float64(gas.PerKmKobo)) +
		int64(weightKg*float64(gas.PerKgKobo))
	const want = 193000 // ₦1,930 in kobo
	if delivery != want {
		t.Errorf("gas delivery kobo = %d, want %d", delivery, want)
	}
}

// vehicleClassFor must route gas orders by weight, not vertical.
func TestVehicleClassForGas(t *testing.T) {
	cases := []struct {
		kg   float64
		want model.VehicleType
	}{
		{3, model.VehicleMotorcycle},
		{6, model.VehicleMotorcycle},
		{6.1, model.VehicleCar},
		{12.5, model.VehicleCar},
		{12.6, model.VehicleVan},
		{25, model.VehicleVan},
	}
	for _, tc := range cases {
		got := vehicleClassFor("gas", tc.kg)
		if got != tc.want {
			t.Errorf("vehicleClassFor(gas, %.1f) = %v, want %v", tc.kg, got, tc.want)
		}
	}
}

// Non-gas verticals must still use the static map (motorcycle minimum).
func TestVehicleClassForNonGas(t *testing.T) {
	for _, v := range []string{"food", "grocery", "pharmacy", "package"} {
		got := vehicleClassFor(v, 100) // large weight must not affect non-gas
		if got != model.VehicleMotorcycle {
			t.Errorf("vehicleClassFor(%s, 100) = %v, want motorcycle", v, got)
		}
	}
}
