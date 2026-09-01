package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
)

// ── LoyaltyHandler ────────────────────────────────────────────────────────────

// stubLoyalty satisfies loyaltyService for route-existence tests.
type stubLoyalty struct{}

func (s *stubLoyalty) GetBalance(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (s *stubLoyalty) History(_ context.Context, _ uuid.UUID, _ int) ([]model.LoyaltyEvent, error) {
	return nil, nil
}

func loyaltyValidationRouter() *gin.Engine {
	r := gin.New()
	h := &LoyaltyHandler{loyalty: &stubLoyalty{}}
	r.GET("/loyalty", seedCtx("customer"), h.GetBalance)
	r.GET("/loyalty/history", seedCtx("customer"), h.GetHistory)
	return r
}

// Loyalty handler has no validation-only paths (both reach service immediately).
// We verify the routes are registered and return non-404.
func TestLoyaltyHandler_RoutesExist(t *testing.T) {
	r := loyaltyValidationRouter()
	for _, path := range []string{"/loyalty", "/loyalty/history"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code == http.StatusNotFound {
			t.Errorf("route %s not registered (got 404)", path)
		}
	}
}

// ── GiftCardHandler ───────────────────────────────────────────────────────────

func giftCardValidationRouter() *gin.Engine {
	r := gin.New()
	h := &GiftCardHandler{svc: nil}
	r.POST("/gift-cards", seedCtx("customer"), h.Issue)
	r.POST("/gift-cards/redeem", seedCtx("customer"), h.Redeem)
	return r
}

func TestGiftCardHandler_Validation(t *testing.T) {
	r := giftCardValidationRouter()

	t.Run("Issue missing amountKobo returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/gift-cards", bytes.NewReader([]byte(`{}`))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("Redeem missing code returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/gift-cards/redeem", bytes.NewReader([]byte(`{}`))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

// ── SubscriptionHandler ───────────────────────────────────────────────────────

func subscriptionValidationRouter() *gin.Engine {
	r := gin.New()
	h := &SubscriptionHandler{svc: nil}
	r.POST("/subscriptions", seedCtx("customer"), h.Create)
	r.GET("/gas/price-index", h.GetLPGPrice)
	r.POST("/admin/gas/price-index", seedCtx("admin"), h.RecordLPGPrice)
	return r
}

func TestSubscriptionHandler_Validation(t *testing.T) {
	r := subscriptionValidationRouter()

	t.Run("Create missing merchantId returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{
			"addressId": uuid.NewString(), "vertical": "gas",
			"cadence": "weekly", "paymentMethod": "wallet",
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("Create invalid cadence returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{
			"merchantId": uuid.NewString(), "addressId": uuid.NewString(),
			"vertical": "gas", "cadence": "daily", "paymentMethod": "wallet",
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("RecordLPGPrice missing region returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"pricePerKgKobo": 120000, "source": "PPPRA"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/gas/price-index", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

// ── GasHandler ────────────────────────────────────────────────────────────────

func gasValidationRouter() *gin.Engine {
	r := gin.New()
	h := &GasHandler{gas: nil}
	r.POST("/cylinders", seedCtx("customer"), h.RegisterCylinder)
	r.POST("/cylinders/:id/retire", seedCtx("customer"), h.RetireCylinder)
	return r
}

func TestGasHandler_Validation(t *testing.T) {
	r := gasValidationRouter()

	t.Run("RegisterCylinder missing serial returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/cylinders", bytes.NewReader([]byte(`{}`))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("RegisterCylinder invalid specId returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"serial": "SN001", "specId": "bad-uuid"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/cylinders", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("RegisterCylinder invalid lastRecertAt returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"serial": "SN001", "lastRecertAt": "not-a-date"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/cylinders", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("RetireCylinder invalid UUID returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/cylinders/bad-id/retire", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

// ── RunHandler ────────────────────────────────────────────────────────────────

func runValidationRouter() *gin.Engine {
	r := gin.New()
	h := &RunHandler{runs: nil}
	r.GET("/runs/:id", seedCtx("driver"), h.GetRun)
	return r
}

func TestRunHandler_Validation(t *testing.T) {
	r := runValidationRouter()

	t.Run("invalid run UUID returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/runs/not-a-uuid", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

// ── PaycodeHandler ────────────────────────────────────────────────────────────

func paycodeValidationRouter() *gin.Engine {
	r := gin.New()
	h := &PaycodeHandler{paycodes: nil}
	r.POST("/paycodes/generate", seedCtx("customer"), h.Generate)
	r.POST("/paycodes/resolve", seedCtx("driver"), h.Resolve)
	r.POST("/paycodes/confirm-code", seedCtx("driver"), h.ConfirmByCode)
	r.POST("/paycodes/:id/confirm", seedCtx("driver"), h.Confirm)
	return r
}

func TestPaycodeHandler_Validation(t *testing.T) {
	r := paycodeValidationRouter()

	t.Run("Generate missing orderId returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/paycodes/generate", bytes.NewReader([]byte(`{}`))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("Generate invalid orderId UUID returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"orderId": "bad-id"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/paycodes/generate", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("Resolve missing payload returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/paycodes/resolve", bytes.NewReader([]byte(`{}`))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("ConfirmByCode missing orderId returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"code": "123456"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/paycodes/confirm-code", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("ConfirmByCode invalid orderId UUID returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"orderId": "bad-id", "code": "123456"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/paycodes/confirm-code", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("Confirm invalid paycodeId UUID returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/paycodes/bad-id/confirm", bytes.NewReader([]byte(`{}`))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

// ── ProofMediaHandler ─────────────────────────────────────────────────────────

func proofMediaValidationRouter() *gin.Engine {
	r := gin.New()
	h := &ProofMediaHandler{media: nil}
	r.POST("/orders/:id/proof/presign", seedCtx("driver"), h.PresignUpload)
	r.POST("/orders/:id/proof/confirm", seedCtx("driver"), h.ConfirmUpload)
	r.GET("/orders/:id/proof", seedCtx("customer"), h.GetMedia)
	return r
}

func TestProofMediaHandler_Validation(t *testing.T) {
	r := proofMediaValidationRouter()

	t.Run("PresignUpload invalid order UUID returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"kind": "weight_photo", "contentType": "image/jpeg"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/orders/bad-id/proof/presign", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("PresignUpload missing kind returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"contentType": "image/jpeg"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/orders/"+uuid.NewString()+"/proof/presign", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("PresignUpload invalid stopId returns 400", func(t *testing.T) {
		stopID := "bad-stop-id"
		b, _ := json.Marshal(map[string]any{"kind": "weight_photo", "contentType": "image/jpeg", "stopId": stopID})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/orders/"+uuid.NewString()+"/proof/presign", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("ConfirmUpload invalid order UUID returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"kind": "weight_photo", "key": "k1", "sha256": "abc"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/orders/bad-id/proof/confirm", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("ConfirmUpload missing key returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"kind": "weight_photo", "sha256": "abc"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/orders/"+uuid.NewString()+"/proof/confirm", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("GetMedia invalid order UUID returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders/bad-id/proof", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}
