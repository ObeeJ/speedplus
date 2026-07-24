package service

import (
	"math"
	"testing"
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
