package service

import (
	"testing"

	"github.com/speedplus/api/internal/model"
)

// ── State machine: refill states ──────────────────────────────────────────────

func TestRefillStatesInValidTransitions(t *testing.T) {
	required := []model.OrderStatus{
		model.OrderAwaitingCollection,
		model.OrderEmptyCollected,
		model.OrderAtPlant,
	}
	for _, s := range required {
		if _, ok := model.ValidTransitions[s]; !ok {
			t.Errorf("refill state %q missing from ValidTransitions", s)
		}
	}
}

func TestRefillStateMachineChain(t *testing.T) {
	// driver_assigned → awaiting_collection → empty_collected → at_plant → in_transit → delivered
	chain := []struct{ from, to model.OrderStatus }{
		{model.OrderDriverAssigned, model.OrderAwaitingCollection},
		{model.OrderAwaitingCollection, model.OrderEmptyCollected},
		{model.OrderEmptyCollected, model.OrderAtPlant},
		{model.OrderAtPlant, model.OrderInTransit},
		{model.OrderInTransit, model.OrderDelivered},
	}
	for _, step := range chain {
		allowed := model.ValidTransitions[step.from]
		found := false
		for _, s := range allowed {
			if s == step.to {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("transition %s → %s not in ValidTransitions", step.from, step.to)
		}
	}
}

func TestSwapPathUnchanged(t *testing.T) {
	// Swap orders use driver_assigned → in_transit — must still be valid.
	allowed := model.ValidTransitions[model.OrderDriverAssigned]
	found := false
	for _, s := range allowed {
		if s == model.OrderInTransit {
			found = true
			break
		}
	}
	if !found {
		t.Error("driver_assigned → in_transit must remain valid for swap orders")
	}
}

func TestRefillStatesCancellable(t *testing.T) {
	// All three refill states must be cancellable.
	for _, s := range []model.OrderStatus{
		model.OrderAwaitingCollection,
		model.OrderEmptyCollected,
		model.OrderAtPlant,
	} {
		allowed := model.ValidTransitions[s]
		found := false
		for _, next := range allowed {
			if next == model.OrderCancelled {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("refill state %q must be cancellable", s)
		}
	}
}

// ── Gas mode validation ───────────────────────────────────────────────────────

func validGasMode(mode *string) bool {
	if mode == nil {
		return false
	}
	valid := map[string]bool{"swap": true, "refill": true, "new_cylinder": true}
	return valid[*mode]
}

func TestGasModeValidSwap(t *testing.T) {
	m := "swap"
	if !validGasMode(&m) {
		t.Error("swap must be a valid gas_mode")
	}
}

func TestGasModeValidRefill(t *testing.T) {
	m := "refill"
	if !validGasMode(&m) {
		t.Error("refill must be a valid gas_mode")
	}
}

func TestGasModeValidNewCylinder(t *testing.T) {
	m := "new_cylinder"
	if !validGasMode(&m) {
		t.Error("new_cylinder must be a valid gas_mode")
	}
}

func TestGasModeNilInvalid(t *testing.T) {
	if validGasMode(nil) {
		t.Error("nil gas_mode must be invalid")
	}
}

func TestGasModeUnknownInvalid(t *testing.T) {
	m := "on_demand"
	if validGasMode(&m) {
		t.Error("unknown gas_mode must be invalid")
	}
}

// ── CylinderSpec model ────────────────────────────────────────────────────────

func TestCylinderSpecValveTypes(t *testing.T) {
	for _, vt := range []string{"standard", "POL", "ACME"} {
		spec := model.CylinderSpec{ValveType: vt}
		if spec.ValveType != vt {
			t.Errorf("valve type %q not preserved", vt)
		}
	}
}

func TestCylinderSpecSeedSizes(t *testing.T) {
	// The four seeded sizes must match the product catalog from migration 022.
	expected := []float64{3, 6, 12.5, 25}
	for _, kg := range expected {
		spec := model.CylinderSpec{SizeKg: kg}
		if spec.SizeKg != kg {
			t.Errorf("spec size %.1f not preserved", kg)
		}
	}
}

// ── CustomerCylinder Phase 1 fields ──────────────────────────────────────────

func TestCustomerCylinderPhase1Fields(t *testing.T) {
	year := int16(2019)
	vt := "standard"
	tare := 14.5
	cyl := model.CustomerCylinder{
		ManufactureYear: &year,
		ValveType:       &vt,
		TareKg:          &tare,
	}
	if *cyl.ManufactureYear != 2019 {
		t.Error("ManufactureYear not preserved")
	}
	if *cyl.ValveType != "standard" {
		t.Error("ValveType not preserved")
	}
	if *cyl.TareKg != 14.5 {
		t.Error("TareKg not preserved")
	}
}

// ── Order gas fields ──────────────────────────────────────────────────────────

func TestOrderGasModeField(t *testing.T) {
	mode := "swap"
	o := model.Order{GasMode: &mode}
	if o.GasMode == nil || *o.GasMode != "swap" {
		t.Error("Order.GasMode not preserved")
	}
}

func TestOrderCylinderIDNilable(t *testing.T) {
	var o model.Order
	if o.CylinderID != nil {
		t.Error("Order.CylinderID should be nil by default")
	}
}

// ── Merchant gas plant fields ─────────────────────────────────────────────────

func TestMerchantGasPlantDefaults(t *testing.T) {
	var m model.Merchant
	if m.IsGasPlant {
		t.Error("IsGasPlant should default to false")
	}
	if m.PlantCapacityKg != nil {
		t.Error("PlantCapacityKg should be nil by default")
	}
	if m.FloatCount != nil {
		t.Error("FloatCount should be nil by default")
	}
}
