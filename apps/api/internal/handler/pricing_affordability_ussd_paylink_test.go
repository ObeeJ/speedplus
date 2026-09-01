package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ── PricingHandler ────────────────────────────────────────────────────────────

func pricingValidationRouter() *gin.Engine {
	r := gin.New()
	h := &PricingHandler{pricing: nil}
	r.POST("/quotes", seedCtx("customer"), h.Quote)
	r.POST("/quotes/multistop", seedCtx("customer"), h.QuoteMultiStop)
	return r
}

func TestPricingHandler_Validation(t *testing.T) {
	r := pricingValidationRouter()

	t.Run("Quote missing required fields returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/quotes", bytes.NewReader([]byte(`{}`))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("Quote invalid merchantId returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{
			"merchantId": "bad-id", "vertical": "food",
			"subtotalKobo": 100, "originLat": 6.5, "originLng": 3.3,
			"destLat": 6.4, "destLng": 3.4,
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/quotes", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("QuoteMultiStop invalid merchantId returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{
			"merchantId": "bad-id", "vertical": "package",
			"subtotalKobo": 100, "originLat": 6.5, "originLng": 3.3,
			"stops": []map[string]any{{"lat": 6.4, "lng": 3.4}},
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/quotes/multistop", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

// ── AffordabilityHandler ──────────────────────────────────────────────────────

func affordabilityValidationRouter() *gin.Engine {
	r := gin.New()
	h := &AffordabilityHandler{svc: nil}
	r.GET("/wallet/affordability", seedCtx("customer"), h.GetAffordability)
	return r
}

func TestAffordabilityHandler_Validation(t *testing.T) {
	r := affordabilityValidationRouter()

	t.Run("missing lat and lng returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/wallet/affordability", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("missing lng returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/wallet/affordability?lat=6.5", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("non-numeric lat returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/wallet/affordability?lat=abc&lng=3.3", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("non-numeric lng returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/wallet/affordability?lat=6.5&lng=xyz", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

// ── USSDHandler ───────────────────────────────────────────────────────────────

func ussdValidationRouter() *gin.Engine {
	r := gin.New()
	h := &USSDHandler{ussd: nil}
	r.POST("/wallet/ussd/initiate", seedCtx("customer"), h.Initiate)
	r.GET("/wallet/ussd/intents/:id", seedCtx("customer"), h.Status)
	return r
}

func TestUSSDHandler_Validation(t *testing.T) {
	r := ussdValidationRouter()

	t.Run("Initiate missing bankCode returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"amountKobo": 50000, "email": "a@b.com"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/wallet/ussd/initiate", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("Initiate missing email returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"bankCode": "058", "amountKobo": 50000})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/wallet/ussd/initiate", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("Status invalid intent UUID returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/wallet/ussd/intents/bad-id", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

// ── PaymentLinkHandler ────────────────────────────────────────────────────────

func paymentLinkValidationRouter() *gin.Engine {
	r := gin.New()
	h := &PaymentLinkHandler{links: nil}
	r.POST("/payment-links", seedCtx("customer"), h.Create)
	r.POST("/payment-links/:slug/pay", seedCtx("customer"), h.PayByWallet)
	r.POST("/pay/:slug/guest", h.InitiateGuestPayment)
	return r
}

func TestPaymentLinkHandler_Validation(t *testing.T) {
	r := paymentLinkValidationRouter()

	t.Run("Create missing amountKobo returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/payment-links", bytes.NewReader([]byte(`{}`))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("PayByWallet missing Idempotency-Key returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/payment-links/abc123/pay", bytes.NewReader([]byte(`{}`))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("InitiateGuestPayment missing Idempotency-Key returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"email": "g@test.com", "callbackUrl": "https://cb"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/pay/abc123/guest", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("InitiateGuestPayment missing email returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"callbackUrl": "https://cb"})
		req := httptest.NewRequest(http.MethodPost, "/pay/abc123/guest", bytes.NewReader(b))
		req.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}
