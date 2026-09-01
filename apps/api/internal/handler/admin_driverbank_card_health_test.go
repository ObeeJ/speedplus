package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
)

// ── AdminHandler validation ───────────────────────────────────────────────────

func adminValidationRouter() *gin.Engine {
	r := gin.New()
	h := &AdminHandler{admin: nil, ledger: nil, feeConfigs: nil}
	r.POST("/admin/merchants/:id/status", seedCtx("admin"), h.SetMerchantStatus)
	r.POST("/admin/drivers/:id/status", seedCtx("admin"), h.SetDriverStatus)
	r.POST("/admin/users/:id/active", seedCtx("admin"), h.SetUserActive)
	r.GET("/admin/orders/:id", seedCtx("admin"), h.GetOrderDetail)
	r.POST("/admin/disputes/:orderId/freeze", seedCtx("admin"), h.FreezeEscrow)
	r.POST("/admin/disputes/:orderId/release", seedCtx("admin"), h.ReleaseEscrow)
	r.DELETE("/admin/settings/cancellation-rules/:id", seedCtx("admin"), h.DeleteCancellationRule)
	r.GET("/admin/ledger", seedCtx("admin"), h.GetLedger)
	r.PUT("/admin/gas/merchants/:id/fill-status", seedCtx("admin"), h.SetMerchantFillStatus)
	r.PUT("/admin/gas/zones/:id/launch-status", seedCtx("admin"), h.SetZoneLaunchStatus)
	return r
}

func TestAdminHandler_Validation(t *testing.T) {
	r := adminValidationRouter()

	t.Run("SetMerchantStatus invalid UUID returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"status": "active"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/merchants/bad-id/status", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("SetMerchantStatus invalid status value returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"status": "deleted"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/merchants/"+uuid.NewString()+"/status", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("SetDriverStatus invalid UUID returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"status": "approved"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/drivers/bad-id/status", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("SetDriverStatus invalid status value returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"status": "banned"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/drivers/"+uuid.NewString()+"/status", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("SetUserActive invalid UUID returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"active": true, "reason": "reinstated"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/users/bad-id/active", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("SetUserActive missing reason returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"active": true})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/users/"+uuid.NewString()+"/active", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("GetOrderDetail invalid UUID returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/orders/bad-id", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("FreezeEscrow invalid orderId returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"reason": "dispute"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/disputes/bad-id/freeze", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("FreezeEscrow missing reason returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/disputes/"+uuid.NewString()+"/freeze", bytes.NewReader([]byte(`{}`))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("ReleaseEscrow invalid recipient returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"recipient": "driver", "reason": "x"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/disputes/"+uuid.NewString()+"/release", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("DeleteCancellationRule invalid UUID returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/admin/settings/cancellation-rules/bad-id", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("GetLedger missing userId returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/ledger", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("SetMerchantFillStatus invalid UUID returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"status": "good", "reason": "x"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/admin/gas/merchants/bad-id/fill-status", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("SetMerchantFillStatus invalid status returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"status": "excellent", "reason": "x"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/admin/gas/merchants/"+uuid.NewString()+"/fill-status", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("SetZoneLaunchStatus invalid UUID returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"status": "live", "reason": "x"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/admin/gas/zones/bad-id/launch-status", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("SetZoneLaunchStatus invalid status returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"status": "closed", "reason": "x"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/admin/gas/zones/"+uuid.NewString()+"/launch-status", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

// ── AdminHandler weather surcharge (nil service path) ─────────────────────────

func TestAdminHandler_WeatherSurcharge_NilService(t *testing.T) {
	r := gin.New()
	h := &AdminHandler{}
	r.GET("/admin/settings/weather", seedCtx("admin"), h.GetWeatherSurcharge)
	r.PUT("/admin/settings/weather", seedCtx("admin"), h.SetWeatherSurcharge)

	t.Run("GetWeatherSurcharge with nil service returns defaults", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/settings/weather", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("SetWeatherSurcharge with nil service returns 503", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"enabled": false, "amountKobo": 0, "reason": "x"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/admin/settings/weather", bytes.NewReader(b)))
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", w.Code)
		}
	})
}

// ── AdminHandler velocity (nil service path) ──────────────────────────────────

func TestAdminHandler_Velocity_NilService(t *testing.T) {
	r := gin.New()
	h := &AdminHandler{}
	r.GET("/admin/velocity/limits", seedCtx("admin"), h.ListVelocityLimits)
	r.PUT("/admin/velocity/limits", seedCtx("admin"), h.UpsertVelocityLimit)
	r.GET("/admin/velocity/suspicious", seedCtx("admin"), h.ListSuspiciousActivity)

	for _, path := range []string{"/admin/velocity/limits", "/admin/velocity/suspicious"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("%s: status = %d, want 503", path, w.Code)
		}
	}
}

// ── DriverBankHandler validation ──────────────────────────────────────────────

func TestDriverBankHandler_maskAccount(t *testing.T) {
	cases := []struct{ in, want string }{
		{"0123456789", "****6789"},
		{"1234", "****"},
		{"123", "****"},
		{"", "****"},
	}
	for _, c := range cases {
		got := maskAccount(c.in)
		if got != c.want {
			t.Errorf("maskAccount(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ── CardHandler tierView ──────────────────────────────────────────────────────

func TestCardHandler_TierView(t *testing.T) {
	t.Run("tier 0 new member has ordersToNext and no POA", func(t *testing.T) {
		v := tierView(&model.UserTrustTier{Tier: model.TierNew, CompletedOrders: 1})
		if v["canPayOnArrival"].(bool) {
			t.Error("tier 0 should not have canPayOnArrival")
		}
		if v["ordersToNext"].(int) != 2 {
			t.Errorf("ordersToNext = %v, want 2", v["ordersToNext"])
		}
	})

	t.Run("tier 1 regular has canPayOnArrival", func(t *testing.T) {
		v := tierView(&model.UserTrustTier{Tier: model.TierRegular, CompletedOrders: 10})
		if !v["canPayOnArrival"].(bool) {
			t.Error("tier 1 should have canPayOnArrival")
		}
		if v["ordersToNext"].(int) != 0 {
			t.Errorf("ordersToNext = %v, want 0", v["ordersToNext"])
		}
	})

	t.Run("frozen flag is propagated", func(t *testing.T) {
		v := tierView(&model.UserTrustTier{Tier: model.TierNew, Frozen: true})
		if !v["frozen"].(bool) {
			t.Error("frozen should be true")
		}
	})
}

// ── HealthHandler ─────────────────────────────────────────────────────────────

func TestHealthHandler_Healthz(t *testing.T) {
	r := gin.New()
	h := &HealthHandler{}
	r.GET("/healthz", h.Healthz)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %v, want ok", resp["status"])
	}
}
