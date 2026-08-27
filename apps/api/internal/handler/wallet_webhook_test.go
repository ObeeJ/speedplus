package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/payment"
	"github.com/speedplus/api/internal/service"
)

// ── stubs ─────────────────────────────────────────────────────────────────────

// captureWallet records every WebhookPayload handed to ProcessWebhook so a test
// can assert on the derived dedup key. wallet_test.go's stubWallet discards it.
type captureWallet struct {
	got []service.WebhookPayload
}

func (c *captureWallet) ProcessWebhook(_ context.Context, p service.WebhookPayload) error {
	c.got = append(c.got, p)
	return nil
}
func (c *captureWallet) Transfer(_ context.Context, _, _ uuid.UUID, _ int64, _, _ string) error {
	return nil
}
func (c *captureWallet) InitiateFund(_ context.Context, _ uuid.UUID, _ int64, _, _, _ string) (*payment.ChargeResponse, error) {
	return nil, nil
}
func (c *captureWallet) EWACashout(_ context.Context, _ uuid.UUID, _ int64, _ string) error {
	return nil
}
func (c *captureWallet) InitiateCryptoFund(_ context.Context, _ uuid.UUID, _ int64, _, _, _, _ string, _ *payment.BridgeProvider) (*payment.ChargeResponse, error) {
	return nil, nil
}

var _ walletSvc = (*captureWallet)(nil)

// stubVerifier accepts every signature — signature verification has its own
// coverage in the payment package. This suite is about the dedup key.
type stubVerifier struct{ name string }

func (s *stubVerifier) VerifyWebhookSignature(_ []byte, _ string) bool { return true }
func (s *stubVerifier) Name() string                                   { return s.name }

func webhookRouter(w walletSvc, provider, sigHeader string) *gin.Engine {
	r := gin.New()
	h := &WalletHandler{wallet: w}
	r.POST("/hook", h.handleWebhook(&stubVerifier{name: provider}, sigHeader, "charge.success"))
	return r
}

func postRaw(r *gin.Engine, sigHeader, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/hook", bytes.NewReader([]byte(body)))
	req.Header.Set(sigHeader, "any-signature")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ── the regression this suite exists for ──────────────────────────────────────

// Paystack and Flutterwave send data.id as a JSON NUMBER. encoding/json decodes
// that into an interface{} as float64, so the original
// `payload.Data.ID.(string)` assertion always failed and every event got
// EventID "". Combined with UNIQUE(provider, event_id) on webhook_events, the
// first stored event then shadowed every later one at the replay guard in
// WalletService.ProcessWebhook — acking real payments with 200 OK while never
// crediting the customer.
func TestHandleWebhook_EventIDNormalisation(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		header   string
		body     string
		wantID   string
	}{
		{
			name:     "paystack numeric data.id",
			provider: "paystack",
			header:   "x-paystack-signature",
			body:     `{"event":"charge.success","data":{"id":302961,"reference":"qTPrJoy9Bx","status":"success"}}`,
			wantID:   "302961",
		},
		{
			name:     "flutterwave numeric data.id",
			provider: "flutterwave",
			header:   "verif-hash",
			body:     `{"event":"charge.completed","data":{"id":285959875,"tx_ref":"fw_ref_1","status":"successful"}}`,
			wantID:   "285959875",
		},
		{
			name:     "string data.id is preserved",
			provider: "paystack",
			header:   "x-paystack-signature",
			body:     `{"event":"charge.success","data":{"id":"evt_abc123","reference":"ref_1"}}`,
			wantID:   "evt_abc123",
		},
		{
			name:     "absent data.id falls back to reference",
			provider: "paystack",
			header:   "x-paystack-signature",
			body:     `{"event":"charge.success","data":{"reference":"ref_no_id"}}`,
			wantID:   "ref_no_id",
		},
		{
			name:     "absent data.id falls back to tx_ref",
			provider: "flutterwave",
			header:   "verif-hash",
			body:     `{"event":"charge.completed","data":{"tx_ref":"fw_no_id"}}`,
			wantID:   "fw_no_id",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cw := &captureWallet{}
			r := webhookRouter(cw, tc.provider, tc.header)

			if w := postRaw(r, tc.header, tc.body); w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
			}
			if len(cw.got) != 1 {
				t.Fatalf("ProcessWebhook called %d times, want 1", len(cw.got))
			}
			got := cw.got[0]
			if got.EventID == "" {
				t.Fatal("EventID is empty — every event would collide on UNIQUE(provider, event_id) and be silently dropped")
			}
			if got.EventID != tc.wantID {
				t.Errorf("EventID = %q, want %q", got.EventID, tc.wantID)
			}
			if got.Provider != tc.provider {
				t.Errorf("Provider = %q, want %q", got.Provider, tc.provider)
			}
		})
	}
}

// Two genuinely different payments must produce two different dedup keys.
// Before the fix both were "", so the second was discarded with a 200 OK and
// the customer was never credited.
func TestHandleWebhook_DistinctPaymentsGetDistinctEventIDs(t *testing.T) {
	cw := &captureWallet{}
	r := webhookRouter(cw, "paystack", "x-paystack-signature")

	postRaw(r, "x-paystack-signature", `{"event":"charge.success","data":{"id":302961,"reference":"ref_one"}}`)
	postRaw(r, "x-paystack-signature", `{"event":"charge.success","data":{"id":302962,"reference":"ref_two"}}`)

	if len(cw.got) != 2 {
		t.Fatalf("ProcessWebhook called %d times, want 2", len(cw.got))
	}
	first, second := cw.got[0].EventID, cw.got[1].EventID
	if first == "" || second == "" {
		t.Fatalf("empty EventID: first=%q second=%q", first, second)
	}
	if first == second {
		t.Fatalf("both payments derived EventID %q — the second would be dropped as a replay", first)
	}
}

// Monnify uses a separate handler that already passed a real string ID. This
// guards it against regressing into the same shape.
func TestHandleMonnifyWebhook_UsesTransactionReference(t *testing.T) {
	cw := &captureWallet{}
	r := gin.New()
	h := &WalletHandler{wallet: cw}
	r.POST("/hook", h.HandleMonnifyWebhook(&stubVerifier{name: "monnify"}))

	body := `{"eventType":"SUCCESSFUL_TRANSACTION","eventData":{"transactionReference":"MNFY|123","paymentReference":"pay_1"}}`
	req := httptest.NewRequest(http.MethodPost, "/hook", bytes.NewReader([]byte(body)))
	req.Header.Set("monnify-signature", "any")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if len(cw.got) != 1 {
		t.Fatalf("ProcessWebhook called %d times, want 1", len(cw.got))
	}
	if cw.got[0].EventID != "MNFY|123" {
		t.Errorf("EventID = %q, want MNFY|123", cw.got[0].EventID)
	}
	if cw.got[0].EventType != "charge.success" {
		t.Errorf("EventType = %q, want charge.success (mapped)", cw.got[0].EventType)
	}
}
