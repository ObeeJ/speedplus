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
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/payment"
	"github.com/speedplus/api/internal/service"
)

func init() { gin.SetMode(gin.TestMode) }

// ── stubs ─────────────────────────────────────────────────────────────────────

type stubWallet struct {
	transferErr error
	fundResp    *payment.ChargeResponse
	fundErr     error
}

func (s *stubWallet) Transfer(_ context.Context, _, _ uuid.UUID, _ int64, _, _ string) error {
	return s.transferErr
}
func (s *stubWallet) InitiateFund(_ context.Context, _ uuid.UUID, _ int64, _, _, _ string) (*payment.ChargeResponse, error) {
	return s.fundResp, s.fundErr
}
func (s *stubWallet) EWACashout(_ context.Context, _ uuid.UUID, _ int64, _ string) error {
	return nil
}
func (s *stubWallet) ProcessWebhook(_ context.Context, _ service.WebhookPayload) error { return nil }
func (s *stubWallet) InitiateCryptoFund(_ context.Context, _ uuid.UUID, _ int64, _, _, _, _ string, _ *payment.BridgeProvider) (*payment.ChargeResponse, error) {
	return nil, nil
}

type stubLedger struct {
	balance int64
	balErr  error
}

func (s *stubLedger) ResolveWalletOwner(_ context.Context, id uuid.UUID, _ string) (uuid.UUID, error) {
	return id, nil
}
func (s *stubLedger) GetBalance(_ context.Context, _ uuid.UUID) (int64, error) {
	return s.balance, s.balErr
}
func (s *stubLedger) GetTransactions(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ int) ([]model.LedgerEntry, error) {
	return nil, nil
}

type stubUserRepo struct {
	user *model.User
	err  error
}

func (s *stubUserRepo) FindByPhone(_ context.Context, _ string) (*model.User, error) {
	return s.user, s.err
}
func (s *stubUserRepo) FindByUsername(_ context.Context, _ string) (*model.User, error) {
	return s.user, s.err
}
func (s *stubUserRepo) FindByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return s.user, s.err
}

// ── helpers ───────────────────────────────────────────────────────────────────

func walletRouter(w walletSvc, l ledgerSvc, u *stubUserRepo) *gin.Engine {
	r := gin.New()
	h := &WalletHandler{wallet: w, ledger: l, userRepo: u}
	r.GET("/balance", seedCtx("customer"), h.GetBalance)
	r.POST("/fund", seedCtx("customer"), h.Fund)
	r.POST("/transfer", seedCtx("customer"), h.Transfer)
	return r
}

func seedCtx(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.CtxUserID, uuid.NewString())
		c.Set(middleware.CtxUserRole, role)
		c.Set(middleware.CtxIsVerified, true)
	}
}

func post(r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b)))
	return w
}

func get(r *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

// ── GetBalance ────────────────────────────────────────────────────────────────

func TestWalletHandler_GetBalance(t *testing.T) {
	t.Run("returns balance", func(t *testing.T) {
		r := walletRouter(&stubWallet{}, &stubLedger{balance: 5_000_000}, &stubUserRepo{})
		w := get(r, "/balance")
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]any)
		if data["balanceKobo"].(float64) != 5_000_000 {
			t.Errorf("balanceKobo = %v, want 5000000", data["balanceKobo"])
		}
	})

	t.Run("ledger error returns 500", func(t *testing.T) {
		r := walletRouter(&stubWallet{}, &stubLedger{balErr: errors.New("db down")}, &stubUserRepo{})
		w := get(r, "/balance")
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", w.Code)
		}
	})

	t.Run("Cache-Control header is set", func(t *testing.T) {
		r := walletRouter(&stubWallet{}, &stubLedger{}, &stubUserRepo{})
		w := get(r, "/balance")
		if w.Header().Get("Cache-Control") != "private, no-store" {
			t.Errorf("Cache-Control = %q, want private, no-store", w.Header().Get("Cache-Control"))
		}
	})
}

// ── Fund ──────────────────────────────────────────────────────────────────────

func TestWalletHandler_Fund(t *testing.T) {
	validBody := map[string]any{
		"amountKobo":  50_000,
		"email":       "user@example.com",
		"callbackUrl": "https://example.com/callback",
	}

	t.Run("missing Idempotency-Key returns 400", func(t *testing.T) {
		r := walletRouter(&stubWallet{}, &stubLedger{}, &stubUserRepo{})
		b, _ := json.Marshal(validBody)
		req := httptest.NewRequest(http.MethodPost, "/fund", bytes.NewReader(b))
		// no Idempotency-Key header
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("valid request returns authorization URL", func(t *testing.T) {
		stub := &stubWallet{fundResp: &payment.ChargeResponse{AuthorizationURL: "https://pay.example.com", Reference: "ref123"}}
		r := walletRouter(stub, &stubLedger{}, &stubUserRepo{})
		b, _ := json.Marshal(validBody)
		req := httptest.NewRequest(http.MethodPost, "/fund", bytes.NewReader(b))
		req.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]any)
		if data["authorizationUrl"] != "https://pay.example.com" {
			t.Errorf("authorizationUrl = %v", data["authorizationUrl"])
		}
	})

	t.Run("amount below minimum returns 400", func(t *testing.T) {
		r := walletRouter(&stubWallet{}, &stubLedger{}, &stubUserRepo{})
		b, _ := json.Marshal(map[string]any{"amountKobo": 100, "email": "x@x.com", "callbackUrl": "https://x.com"})
		req := httptest.NewRequest(http.MethodPost, "/fund", bytes.NewReader(b))
		req.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

// ── Transfer ──────────────────────────────────────────────────────────────────

func TestWalletHandler_Transfer(t *testing.T) {
	recipientID := uuid.New()
	recipient := &model.User{ID: recipientID, FirstName: "Ada", LastName: "Obi"}

	validBody := map[string]any{
		"recipientId": recipientID.String(),
		"amountKobo":  50_000,
		"pin":         "1234",
	}

	t.Run("missing Idempotency-Key returns 400", func(t *testing.T) {
		r := walletRouter(&stubWallet{}, &stubLedger{}, &stubUserRepo{user: recipient})
		w := post(r, "/transfer", validBody)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("recipient not found returns 404", func(t *testing.T) {
		r := walletRouter(&stubWallet{}, &stubLedger{}, &stubUserRepo{err: errors.New("not found")})
		b, _ := json.Marshal(validBody)
		req := httptest.NewRequest(http.MethodPost, "/transfer", bytes.NewReader(b))
		req.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})

	t.Run("insufficient balance returns 422 with INSUFFICIENT_FUNDS code", func(t *testing.T) {
		stub := &stubWallet{transferErr: service.ErrInsufficientBalance}
		r := walletRouter(stub, &stubLedger{}, &stubUserRepo{user: recipient})
		b, _ := json.Marshal(validBody)
		req := httptest.NewRequest(http.MethodPost, "/transfer", bytes.NewReader(b))
		req.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", w.Code)
		}
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		errBlock := resp["error"].(map[string]any)
		if errBlock["code"] != "INSUFFICIENT_FUNDS" {
			t.Errorf("code = %v, want INSUFFICIENT_FUNDS", errBlock["code"])
		}
	})

	t.Run("successful transfer returns recipient name", func(t *testing.T) {
		r := walletRouter(&stubWallet{}, &stubLedger{}, &stubUserRepo{user: recipient})
		b, _ := json.Marshal(validBody)
		req := httptest.NewRequest(http.MethodPost, "/transfer", bytes.NewReader(b))
		req.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]any)
		if data["recipientName"] != "Ada Obi" {
			t.Errorf("recipientName = %v, want Ada Obi", data["recipientName"])
		}
	})

	t.Run("no recipient identifier returns 400", func(t *testing.T) {
		r := walletRouter(&stubWallet{}, &stubLedger{}, &stubUserRepo{user: recipient})
		b, _ := json.Marshal(map[string]any{"amountKobo": 50_000, "pin": "1234"})
		req := httptest.NewRequest(http.MethodPost, "/transfer", bytes.NewReader(b))
		req.Header.Set("Idempotency-Key", uuid.NewString())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}
