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

// catalogValidationRouter wires only the validation-testable paths.
// Calls that reach the service will panic (nil pointer) — that's fine because
// these tests only exercise the handler's own guard clauses.
func catalogValidationRouter() *gin.Engine {
	r := gin.New()
	h := &CatalogHandler{catalog: nil}
	r.GET("/merchants/:id", h.GetMerchant)
	r.GET("/products", h.ListProducts)
	r.GET("/products/:id", h.GetProduct)
	r.GET("/products/search", h.SearchProducts)
	r.POST("/prescriptions/presign", seedCtx("customer"), h.PresignPrescriptionUpload)
	r.POST("/prescriptions", seedCtx("customer"), h.CreatePrescription)
	r.GET("/prescriptions/:id", seedCtx("customer"), h.GetPrescription)
	return r
}

func TestCatalogHandler_Validation(t *testing.T) {
	r := catalogValidationRouter()

	t.Run("GetMerchant invalid UUID returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/merchants/not-a-uuid", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("ListProducts missing merchantId returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/products", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("ListProducts invalid merchantId returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/products?merchantId=bad", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("GetProduct invalid UUID returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/products/not-a-uuid", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("SearchProducts missing q returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/products/search", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("PresignPrescriptionUpload missing contentType returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/prescriptions/presign", bytes.NewReader([]byte(`{}`))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("CreatePrescription missing r2Key returns 400", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"merchantId": uuid.NewString()})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/prescriptions", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("CreatePrescription r2Key wrong owner returns 403", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{
			"r2Key":      "prescriptions/other-user-id/file.jpg",
			"merchantId": uuid.NewString(),
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/prescriptions", bytes.NewReader(b)))
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("GetPrescription invalid UUID returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/prescriptions/not-a-uuid", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}
