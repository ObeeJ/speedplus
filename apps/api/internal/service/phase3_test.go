package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
)

// ── ConfirmStopInput construction ─────────────────────────────────────────────

func TestConfirmStopInputGasWithCollection(t *testing.T) {
	serial := "SN-12345"
	lat, lng := 6.5244, 3.3792
	in := ConfirmStopInput{
		Code:                "123456",
		EmptyCollected:      true,
		EmptyCylinderSerial: &serial,
		CapturedLat:         &lat,
		CapturedLng:         &lng,
	}
	if !in.EmptyCollected {
		t.Error("EmptyCollected should be true")
	}
	if in.EmptyCylinderSerial == nil || *in.EmptyCylinderSerial != serial {
		t.Errorf("EmptyCylinderSerial = %v, want %q", in.EmptyCylinderSerial, serial)
	}
}

func TestConfirmStopInputNonGas(t *testing.T) {
	// Non-gas stops must work with just a code — no collection fields required.
	in := ConfirmStopInput{Code: "654321"}
	if in.EmptyCollected {
		t.Error("EmptyCollected should default to false for non-gas")
	}
	if in.EmptyCylinderSerial != nil {
		t.Error("EmptyCylinderSerial should be nil for non-gas")
	}
}

// ── CylinderCustodyEvent model ────────────────────────────────────────────────

func TestCylinderCustodyEventTypes(t *testing.T) {
	validTypes := []string{"collected", "at_plant", "returned"}
	for _, et := range validTypes {
		e := model.CylinderCustodyEvent{
			ID:        uuid.New(),
			OrderID:   uuid.New(),
			EventType: et,
			ActorID:   uuid.New(),
			OccurredAt: time.Now(),
		}
		if e.EventType != et {
			t.Errorf("event type %q not preserved", et)
		}
	}
}

func TestCylinderCustodyEventOptionalFields(t *testing.T) {
	// cylinder_id and serial are both optional — a custody event is valid
	// without them (Phase 1 adds the FK; Phase 3 ships without it).
	e := model.CylinderCustodyEvent{
		ID:         uuid.New(),
		OrderID:    uuid.New(),
		EventType:  "collected",
		ActorID:    uuid.New(),
		OccurredAt: time.Now(),
	}
	if e.CylinderID != nil {
		t.Error("CylinderID should be nil when not set")
	}
	if e.Serial != nil {
		t.Error("Serial should be nil when not set")
	}
}

// ── OrderStop empty collection fields ────────────────────────────────────────

func TestOrderStopEmptyCollectedDefault(t *testing.T) {
	// Zero-value OrderStop must have EmptyCollected=false — the DB default
	// matches and no gas order should accidentally appear collected.
	var stop model.OrderStop
	if stop.EmptyCollected {
		t.Error("OrderStop.EmptyCollected zero value must be false")
	}
	if stop.EmptyCylinderSerial != nil {
		t.Error("OrderStop.EmptyCylinderSerial zero value must be nil")
	}
}

func TestOrderStopOutEmptyCollectionFields(t *testing.T) {
	serial := "SN-99999"
	now := time.Now()
	out := OrderStopOut{
		ID:                  uuid.New(),
		OrderID:             uuid.New(),
		Sequence:            1,
		AddressID:           uuid.New(),
		Status:              "confirmed",
		ConfirmedAt:         &now,
		EmptyCollected:      true,
		EmptyCylinderSerial: &serial,
	}
	if !out.EmptyCollected {
		t.Error("OrderStopOut.EmptyCollected should be true")
	}
	if out.EmptyCylinderSerial == nil || *out.EmptyCylinderSerial != serial {
		t.Errorf("OrderStopOut.EmptyCylinderSerial = %v, want %q", out.EmptyCylinderSerial, serial)
	}
}

// ── Gas-only guard ────────────────────────────────────────────────────────────

// custodyEventShouldWrite mirrors the production guard: only write a custody
// event when vertical == "gas" AND EmptyCollected is true.
func custodyEventShouldWrite(vertical string, in ConfirmStopInput) bool {
	return vertical == "gas" && in.EmptyCollected
}

func TestCustodyEventOnlyForGas(t *testing.T) {
	gasIn := ConfirmStopInput{Code: "111111", EmptyCollected: true}
	if !custodyEventShouldWrite("gas", gasIn) {
		t.Error("should write custody event for gas + EmptyCollected=true")
	}
	for _, v := range []string{"food", "grocery", "pharmacy", "package"} {
		if custodyEventShouldWrite(v, gasIn) {
			t.Errorf("must NOT write custody event for vertical %q", v)
		}
	}
}

func TestCustodyEventNotWrittenWhenNotCollected(t *testing.T) {
	in := ConfirmStopInput{Code: "222222", EmptyCollected: false}
	if custodyEventShouldWrite("gas", in) {
		t.Error("must NOT write custody event when EmptyCollected=false")
	}
}
