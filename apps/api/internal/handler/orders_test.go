package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/dto"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/service"
)

// ── stub ──────────────────────────────────────────────────────────────────────

type stubOrders struct {
	order      *model.Order
	orders     []model.Order
	stops      []service.OrderStopOut
	receipt    *service.ReceiptResponse
	badges     []model.DriverBadge
	createErr  error
	getErr     error
	cancelErr  error
	reviewErr  error
	stopsErr   error
	confirmErr error
	disputeErr error
	receiptErr error
	badgesErr  error
	listErr    error
	disputeStatus string
	disputeStatusErr error
}

func (s *stubOrders) Create(_ context.Context, _ service.CreateOrderInput) (*model.Order, error) {
	return s.order, s.createErr
}
func (s *stubOrders) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) (*model.Order, error) {
	return s.order, s.getErr
}
func (s *stubOrders) ListForCustomer(_ context.Context, _ uuid.UUID, _, _ string, _ *uuid.UUID, _ int) ([]model.Order, error) {
	return s.orders, s.listErr
}
func (s *stubOrders) Cancel(_ context.Context, _, _ uuid.UUID, _ string, _ string) error {
	return s.cancelErr
}
func (s *stubOrders) SubmitReview(_ context.Context, _, _ uuid.UUID, _ string, _ int, _ *string) error {
	return s.reviewErr
}
func (s *stubOrders) GetStops(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) ([]service.OrderStopOut, error) {
	return s.stops, s.stopsErr
}
func (s *stubOrders) ConfirmStop(_ context.Context, _, _ uuid.UUID, _ int, _ service.ConfirmStopInput) error {
	return s.confirmErr
}
func (s *stubOrders) RaiseDispute(_ context.Context, _, _ uuid.UUID, _ string) error {
	return s.disputeErr
}
func (s *stubOrders) GetDisputeStatus(_ context.Context, _, _ uuid.UUID) (string, error) {
	return s.disputeStatus, s.disputeStatusErr
}
func (s *stubOrders) GetReceipt(_ context.Context, _, _ uuid.UUID) (*service.ReceiptResponse, error) {
	return s.receipt, s.receiptErr
}
func (s *stubOrders) GetDriverBadges(_ context.Context, _ uuid.UUID) ([]model.DriverBadge, error) {
	return s.badges, s.badgesErr
}
func (s *stubOrders) ToResponse(_ context.Context, o *model.Order) dto.OrderResponse {
	if o == nil {
		return dto.OrderResponse{}
	}
	return dto.OrderFromModel(o)
}

type stubMerchantSvc struct {
	merchant *model.Merchant
	err      error
}

func (s *stubMerchantSvc) ResolveByUserID(_ context.Context, _ uuid.UUID) (*model.Merchant, error) {
	return s.merchant, s.err
}

// ── helpers ───────────────────────────────────────────────────────────────────

func ordersRouter(svc orderService) *gin.Engine {
	r := gin.New()
	h := &OrderHandler{orders: svc}
	r.POST("/orders", seedCtx("customer"), h.Create)
	r.GET("/orders", seedCtx("customer"), h.List)
	r.GET("/orders/:id", seedCtx("customer"), h.GetByID)
	r.GET("/orders/:id/receipt", seedCtx("customer"), h.Receipt)
	r.POST("/orders/:id/cancel", seedCtx("customer"), h.Cancel)
	r.POST("/orders/:id/review", seedCtx("customer"), h.Review)
	r.GET("/orders/:id/stops", seedCtx("driver"), h.GetStops)
	r.POST("/orders/:id/stops/confirm", seedCtx("driver"), h.ConfirmStop)
	r.POST("/orders/:id/dispute", seedCtx("customer"), h.RaiseDispute)
	r.GET("/orders/:id/dispute", seedCtx("customer"), h.GetDisputeStatus)
	r.GET("/drivers/:id/badges", h.DriverBadges)
	return r
}

func makeOrder() *model.Order {
	return &model.Order{ID: uuid.New(), Status: model.OrderPending}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestOrderHandler_Create(t *testing.T) {
	validBody := map[string]any{
		"merchantId":        uuid.NewString(),
		"quoteId":           uuid.NewString(),
		"vertical":          "food",
		"deliveryAddressId": uuid.NewString(),
		"items": []map[string]any{
			{"productId": uuid.NewString(), "quantity": 1},
		},
	}

	t.Run("missing Idempotency-Key returns 400", func(t *testing.T) {
		r := ordersRouter(&stubOrders{order: makeOrder()})
		b, _ := json.Marshal(validBody)
		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("success returns 201 with order", func(t *testing.T) {
		r := ordersRouter(&stubOrders{order: makeOrder()})
		b, _ := json.Marshal(validBody)
		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(b))
		req.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
		}
	})

	t.Run("ErrQuoteInvalid returns 422", func(t *testing.T) {
		r := ordersRouter(&stubOrders{createErr: service.ErrQuoteInvalid})
		b, _ := json.Marshal(validBody)
		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(b))
		req.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", w.Code)
		}
	})

	t.Run("ErrMerchantClosed returns 409", func(t *testing.T) {
		r := ordersRouter(&stubOrders{createErr: service.ErrMerchantClosed})
		b, _ := json.Marshal(validBody)
		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(b))
		req.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", w.Code)
		}
	})

	t.Run("ErrRxRequired returns 422 PRESCRIPTION_REQUIRED", func(t *testing.T) {
		r := ordersRouter(&stubOrders{createErr: service.ErrRxRequired})
		b, _ := json.Marshal(validBody)
		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(b))
		req.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", w.Code)
		}
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["error"].(map[string]any)["code"] != "PRESCRIPTION_REQUIRED" {
			t.Errorf("expected PRESCRIPTION_REQUIRED code")
		}
	})

	t.Run("ErrInsufficientBalance returns 402", func(t *testing.T) {
		r := ordersRouter(&stubOrders{createErr: service.ErrInsufficientBalance})
		b, _ := json.Marshal(validBody)
		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(b))
		req.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusPaymentRequired {
			t.Fatalf("status = %d, want 402", w.Code)
		}
	})

	t.Run("missing items returns 400", func(t *testing.T) {
		r := ordersRouter(&stubOrders{})
		b, _ := json.Marshal(map[string]any{
			"merchantId": uuid.NewString(), "quoteId": uuid.NewString(),
			"vertical": "food", "deliveryAddressId": uuid.NewString(),
		})
		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(b))
		req.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestOrderHandler_GetByID(t *testing.T) {
	t.Run("success returns 200", func(t *testing.T) {
		r := ordersRouter(&stubOrders{order: makeOrder()})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders/"+uuid.NewString(), nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := ordersRouter(&stubOrders{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders/not-a-uuid", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		r := ordersRouter(&stubOrders{getErr: errors.New("not found")})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders/"+uuid.NewString(), nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestOrderHandler_List(t *testing.T) {
	t.Run("returns orders array", func(t *testing.T) {
		r := ordersRouter(&stubOrders{orders: []model.Order{*makeOrder()}})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("service error returns 500", func(t *testing.T) {
		r := ordersRouter(&stubOrders{listErr: errors.New("db error")})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders", nil))
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", w.Code)
		}
	})
}

// ── Cancel ────────────────────────────────────────────────────────────────────

func TestOrderHandler_Cancel(t *testing.T) {
	t.Run("success returns 200 with message", func(t *testing.T) {
		r := ordersRouter(&stubOrders{})
		b, _ := json.Marshal(map[string]any{"reason": "changed mind"})
		req := httptest.NewRequest(http.MethodPost, "/orders/"+uuid.NewString()+"/cancel", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
	})

	t.Run("ErrIllegalTransition returns 409", func(t *testing.T) {
		r := ordersRouter(&stubOrders{cancelErr: service.ErrIllegalTransition})
		b, _ := json.Marshal(map[string]any{"reason": "too late"})
		req := httptest.NewRequest(http.MethodPost, "/orders/"+uuid.NewString()+"/cancel", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", w.Code)
		}
	})

	t.Run("ErrOrderNotFound returns 404", func(t *testing.T) {
		r := ordersRouter(&stubOrders{cancelErr: service.ErrOrderNotFound})
		b, _ := json.Marshal(map[string]any{"reason": "x"})
		req := httptest.NewRequest(http.MethodPost, "/orders/"+uuid.NewString()+"/cancel", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("missing reason returns 400", func(t *testing.T) {
		r := ordersRouter(&stubOrders{})
		req := httptest.NewRequest(http.MethodPost, "/orders/"+uuid.NewString()+"/cancel", bytes.NewReader([]byte(`{}`)))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

// ── Review ────────────────────────────────────────────────────────────────────

func TestOrderHandler_Review(t *testing.T) {
	t.Run("missing Idempotency-Key returns 400", func(t *testing.T) {
		r := ordersRouter(&stubOrders{})
		b, _ := json.Marshal(map[string]any{"revieweeType": "driver", "rating": 5})
		req := httptest.NewRequest(http.MethodPost, "/orders/"+uuid.NewString()+"/review", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("success returns 201", func(t *testing.T) {
		r := ordersRouter(&stubOrders{})
		b, _ := json.Marshal(map[string]any{"revieweeType": "driver", "rating": 5})
		req := httptest.NewRequest(http.MethodPost, "/orders/"+uuid.NewString()+"/review", bytes.NewReader(b))
		req.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
		}
	})

	t.Run("review error returns 422", func(t *testing.T) {
		r := ordersRouter(&stubOrders{reviewErr: errors.New("already reviewed")})
		b, _ := json.Marshal(map[string]any{"revieweeType": "driver", "rating": 4})
		req := httptest.NewRequest(http.MethodPost, "/orders/"+uuid.NewString()+"/review", bytes.NewReader(b))
		req.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", w.Code)
		}
	})
}

// ── GetStops ──────────────────────────────────────────────────────────────────

func TestOrderHandler_GetStops(t *testing.T) {
	t.Run("success returns stops", func(t *testing.T) {
		r := ordersRouter(&stubOrders{
			order: makeOrder(),
			stops: []service.OrderStopOut{{Sequence: 1, AddressID: uuid.New(), Status: "pending"}},
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders/"+uuid.NewString()+"/stops", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("ownership failure returns 403", func(t *testing.T) {
		r := ordersRouter(&stubOrders{getErr: errors.New("forbidden")})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders/"+uuid.NewString()+"/stops", nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})
}

// ── ConfirmStop ───────────────────────────────────────────────────────────────

func TestOrderHandler_ConfirmStop(t *testing.T) {
	t.Run("success returns 200", func(t *testing.T) {
		r := ordersRouter(&stubOrders{})
		b, _ := json.Marshal(map[string]any{"sequence": 1, "code": "ABC123"})
		req := httptest.NewRequest(http.MethodPost, "/orders/"+uuid.NewString()+"/stops/confirm", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
	})

	t.Run("wrong code returns 422", func(t *testing.T) {
		r := ordersRouter(&stubOrders{confirmErr: errors.New("invalid code")})
		b, _ := json.Marshal(map[string]any{"sequence": 1, "code": "WRONG1"})
		req := httptest.NewRequest(http.MethodPost, "/orders/"+uuid.NewString()+"/stops/confirm", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", w.Code)
		}
	})
}

// ── RaiseDispute ──────────────────────────────────────────────────────────────

func TestOrderHandler_RaiseDispute(t *testing.T) {
	t.Run("success returns 201", func(t *testing.T) {
		r := ordersRouter(&stubOrders{})
		b, _ := json.Marshal(map[string]any{"reason": "item missing"})
		req := httptest.NewRequest(http.MethodPost, "/orders/"+uuid.NewString()+"/dispute", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
		}
	})

	t.Run("ErrOrderNotFound returns 404", func(t *testing.T) {
		r := ordersRouter(&stubOrders{disputeErr: service.ErrOrderNotFound})
		b, _ := json.Marshal(map[string]any{"reason": "x"})
		req := httptest.NewRequest(http.MethodPost, "/orders/"+uuid.NewString()+"/dispute", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("ErrOrderForbidden returns 403", func(t *testing.T) {
		r := ordersRouter(&stubOrders{disputeErr: service.ErrOrderForbidden})
		b, _ := json.Marshal(map[string]any{"reason": "x"})
		req := httptest.NewRequest(http.MethodPost, "/orders/"+uuid.NewString()+"/dispute", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})
}

// ── GetDisputeStatus ──────────────────────────────────────────────────────────

func TestOrderHandler_GetDisputeStatus(t *testing.T) {
	t.Run("success returns status", func(t *testing.T) {
		r := ordersRouter(&stubOrders{disputeStatus: "open"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders/"+uuid.NewString()+"/dispute", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["data"].(map[string]any)["status"] != "open" {
			t.Errorf("expected status=open")
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		r := ordersRouter(&stubOrders{disputeStatusErr: service.ErrOrderNotFound})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders/"+uuid.NewString()+"/dispute", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})
}

// ── Receipt ───────────────────────────────────────────────────────────────────

func TestOrderHandler_Receipt(t *testing.T) {
	t.Run("success returns receipt", func(t *testing.T) {
		r := ordersRouter(&stubOrders{receipt: &service.ReceiptResponse{OrderID: uuid.NewString(), Vertical: "food"}})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders/"+uuid.NewString()+"/receipt", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("ErrOrderForbidden returns 403", func(t *testing.T) {
		r := ordersRouter(&stubOrders{receiptErr: service.ErrOrderForbidden})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders/"+uuid.NewString()+"/receipt", nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		r := ordersRouter(&stubOrders{receiptErr: errors.New("not found")})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders/"+uuid.NewString()+"/receipt", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})
}

// ── DriverBadges ──────────────────────────────────────────────────────────────

func TestOrderHandler_DriverBadges(t *testing.T) {
	t.Run("success returns badges", func(t *testing.T) {
		r := ordersRouter(&stubOrders{badges: []model.DriverBadge{{BadgeType: "speed_demon"}}})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/drivers/"+uuid.NewString()+"/badges", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("invalid driver UUID returns 400", func(t *testing.T) {
		r := ordersRouter(&stubOrders{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/drivers/not-a-uuid/badges", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}
