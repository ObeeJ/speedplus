package payment_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/speedplus/api/internal/payment"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func mockServer(handler http.HandlerFunc) (*httptest.Server, string) {
	srv := httptest.NewServer(handler)
	return srv, srv.URL
}

func jsonResp(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// ── Paystack ──────────────────────────────────────────────────────────────────

func TestPaystackInitiateCharge(t *testing.T) {
	srv, _ := mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/transaction/initialize" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		jsonResp(w, map[string]interface{}{
			"status": true,
			"data": map[string]string{
				"authorization_url": "https://checkout.paystack.com/abc",
				"reference":         "ref-001",
				"access_code":       "acc-001",
			},
		})
	})
	defer srv.Close()

	p := payment.NewPaystackWithURL("sk_test_key", srv.URL)
	resp, err := p.InitiateCharge(context.Background(), payment.ChargeRequest{
		AmountKobo:  500000,
		Email:       "user@example.com",
		Reference:   "ref-001",
		CallbackURL: "https://app.fourdat.com/wallet?funded=1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.AuthorizationURL != "https://checkout.paystack.com/abc" {
		t.Errorf("got %s", resp.AuthorizationURL)
	}
}

func TestPaystackVerifyTransaction(t *testing.T) {
	srv, _ := mockServer(func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, map[string]interface{}{
			"status": true,
			"data": map[string]interface{}{
				"reference": "ref-001",
				"amount":    500000,
				"status":    "success",
				"paid_at":   "2026-07-29T00:00:00Z",
			},
		})
	})
	defer srv.Close()

	p := payment.NewPaystackWithURL("sk_test_key", srv.URL)
	resp, err := p.VerifyTransaction(context.Background(), "ref-001")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "success" {
		t.Errorf("status = %s, want success", resp.Status)
	}
	if resp.AmountKobo != 500000 {
		t.Errorf("amountKobo = %d, want 500000", resp.AmountKobo)
	}
}

func TestPaystackVerifyWebhookSignature(t *testing.T) {
	p := payment.NewPaystack("mysecret")
	// Valid: HMAC-SHA512 of body with key "mysecret"
	body := []byte(`{"event":"charge.success"}`)
	sig := payment.PaystackHMAC(body, "mysecret")
	if !p.VerifyWebhookSignature(body, sig) {
		t.Error("valid signature rejected")
	}
	if p.VerifyWebhookSignature(body, "badsig") {
		t.Error("invalid signature accepted")
	}
}

// ── Flutterwave ───────────────────────────────────────────────────────────────

func TestFlutterwaveInitiateCharge(t *testing.T) {
	srv, _ := mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/payments" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Verify amount is sent as naira float, not kobo int
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["amount"].(float64) != 5000.0 {
			t.Errorf("amount = %v, want 5000.0 naira", body["amount"])
		}
		jsonResp(w, map[string]interface{}{
			"status": "success",
			"data":   map[string]string{"link": "https://checkout.flutterwave.com/xyz"},
		})
	})
	defer srv.Close()

	f := payment.NewFlutterwaveWithURL("flw_secret", "static-hash", srv.URL)
	resp, err := f.InitiateCharge(context.Background(), payment.ChargeRequest{
		AmountKobo:  500000,
		Email:       "user@example.com",
		Reference:   "ref-002",
		CallbackURL: "https://app.fourdat.com/wallet?funded=1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.AuthorizationURL != "https://checkout.flutterwave.com/xyz" {
		t.Errorf("got %s", resp.AuthorizationURL)
	}
}

func TestFlutterwaveVerifyTransaction_Pending(t *testing.T) {
	srv, _ := mockServer(func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"tx_ref": "ref-002",
				"amount": 5000.0,
				"status": "pending",
			},
		})
	})
	defer srv.Close()

	f := payment.NewFlutterwaveWithURL("flw_secret", "static-hash", srv.URL)
	resp, err := f.VerifyTransaction(context.Background(), "ref-002")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "pending" {
		t.Errorf("status = %s, want pending", resp.Status)
	}
}

func TestFlutterwaveVerifyWebhookSignature(t *testing.T) {
	// Flutterwave uses a static secret hash, NOT an HMAC of the body.
	f := payment.NewFlutterwave("flw_secret", "my-static-hash")
	body := []byte(`{"event":"charge.completed"}`)

	if !f.VerifyWebhookSignature(body, "my-static-hash") {
		t.Error("valid static hash rejected")
	}
	if f.VerifyWebhookSignature(body, "wrong-hash") {
		t.Error("invalid static hash accepted")
	}
	// Body content must NOT affect the result (it's not an HMAC)
	if !f.VerifyWebhookSignature([]byte(`different body`), "my-static-hash") {
		t.Error("static hash should be body-independent")
	}
}

func TestFlutterwaveInitiateTransfer_BadRecipientCode(t *testing.T) {
	f := payment.NewFlutterwave("flw_secret", "hash")
	_, err := f.InitiateTransfer(context.Background(), payment.TransferRequest{
		RecipientCode: "044-no-colon", // wrong format
		AmountKobo:    100000,
	})
	if err == nil || !strings.Contains(err.Error(), "bankCode:accountNumber") {
		t.Errorf("expected format error, got %v", err)
	}
}

func TestFlutterwaveInitiateTransfer(t *testing.T) {
	srv, _ := mockServer(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["account_bank"] != "044" {
			t.Errorf("account_bank = %v, want 044", body["account_bank"])
		}
		if body["account_number"] != "0690000040" {
			t.Errorf("account_number = %v, want 0690000040", body["account_number"])
		}
		if body["amount"].(float64) != 1000.0 {
			t.Errorf("amount = %v, want 1000.0 naira", body["amount"])
		}
		jsonResp(w, map[string]interface{}{
			"status": "success",
			"data":   map[string]interface{}{"id": 1, "status": "NEW"},
		})
	})
	defer srv.Close()

	f := payment.NewFlutterwaveWithURL("flw_secret", "hash", srv.URL)
	resp, err := f.InitiateTransfer(context.Background(), payment.TransferRequest{
		RecipientCode: "044:0690000040",
		AmountKobo:    100000,
		Reference:     "trf-001",
		Reason:        "payout",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "NEW" {
		t.Errorf("status = %s, want NEW", resp.Status)
	}
}

// ── Bridge ────────────────────────────────────────────────────────────────────

func TestBridgeEnsureWallet(t *testing.T) {
	srv, _ := mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wallets" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Api-Key") != "bridge-key" {
			t.Error("missing Api-Key header")
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["external_id"] != "user-uuid-123" {
			t.Errorf("external_id = %v", body["external_id"])
		}
		jsonResp(w, map[string]string{"id": "wallet-abc"})
	})
	defer srv.Close()

	b := payment.NewBridgeWithURL("bridge-key", srv.URL)
	id, err := b.EnsureWallet(context.Background(), "user-uuid-123", "user@example.com", "John Doe")
	if err != nil {
		t.Fatal(err)
	}
	if id != "wallet-abc" {
		t.Errorf("wallet id = %s, want wallet-abc", id)
	}
}

func TestBridgeInitiateCharge(t *testing.T) {
	srv, _ := mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orchestration/payment_sessions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["external_reference"] != "ref-bridge-001" {
			t.Errorf("external_reference = %v", body["external_reference"])
		}
		jsonResp(w, map[string]string{
			"id":           "session-xyz",
			"checkout_url": "https://bridge.xyz/pay/session-xyz",
		})
	})
	defer srv.Close()

	b := payment.NewBridgeWithURL("bridge-key", srv.URL)
	resp, err := b.InitiateCharge(context.Background(), payment.ChargeRequest{
		AmountKobo:  500000,
		Email:       "user@example.com",
		Reference:   "ref-bridge-001",
		CallbackURL: "https://app.fourdat.com/wallet?funded=1",
		Metadata:    map[string]string{"bridge_wallet_id": "wallet-abc"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.AuthorizationURL != "https://bridge.xyz/pay/session-xyz" {
		t.Errorf("got %s", resp.AuthorizationURL)
	}
	if resp.AccessCode != "session-xyz" {
		t.Errorf("AccessCode (session id) = %s, want session-xyz", resp.AccessCode)
	}
}

func TestBridgeVerifyTransaction(t *testing.T) {
	cases := []struct {
		state      string
		wantStatus string
	}{
		{"payment_processed", "success"},
		{"funds_received", "success"},
		{"pending", "pending"},
		{"failed", "failed"},
		{"cancelled", "failed"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.state, func(t *testing.T) {
			srv, _ := mockServer(func(w http.ResponseWriter, r *http.Request) {
				jsonResp(w, map[string]string{"state": tc.state})
			})
			defer srv.Close()

			b := payment.NewBridgeWithURL("bridge-key", srv.URL)
			resp, err := b.VerifyTransaction(context.Background(), "ref-bridge-001")
			if err != nil {
				t.Fatal(err)
			}
			if resp.Status != tc.wantStatus {
				t.Errorf("state %s: status = %s, want %s", tc.state, resp.Status, tc.wantStatus)
			}
		})
	}
}

func TestBridgeVerifyWebhookSignature(t *testing.T) {
	b := payment.NewBridge("bridge-key")
	body := []byte(`{"event_type":"payment.payment_processed"}`)

	if !b.VerifyWebhookSignature(body, "bridge-key") {
		t.Error("valid api key rejected")
	}
	if b.VerifyWebhookSignature(body, "wrong-key") {
		t.Error("invalid api key accepted")
	}
}

func TestBridgeTransferNotSupported(t *testing.T) {
	b := payment.NewBridge("bridge-key")
	_, err := b.InitiateTransfer(context.Background(), payment.TransferRequest{})
	if err == nil {
		t.Error("expected error for unsupported transfer")
	}
}
