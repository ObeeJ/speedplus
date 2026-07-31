package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestMonnify wires a MonnifyProvider against a fake HTTP server.
func newTestMonnify(t *testing.T, mux *http.ServeMux) *MonnifyProvider {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	p := NewMonnify("test-api-key", "test-secret", "CONTRACT123", "WALLET001")
	p.baseURL = srv.URL
	return p
}

// authMux registers the /api/v1/auth/login stub on mux and returns it for chaining.
func authMux(mux *http.ServeMux) *http.ServeMux {
	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"requestSuccessful": true,
			"responseBody": map[string]interface{}{
				"accessToken": "mock-jwt",
				"expiresIn":   3600,
			},
		})
	})
	return mux
}

// ── Auth ──────────────────────────────────────────────────────────────────────

func TestMonnify_accessToken_cached(t *testing.T) {
	calls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewEncoder(w).Encode(map[string]interface{}{
			"requestSuccessful": true,
			"responseBody":      map[string]interface{}{"accessToken": "tok", "expiresIn": 3600},
		})
	})
	p := newTestMonnify(t, mux)

	ctx := context.Background()
	p.accessToken(ctx) //nolint
	p.accessToken(ctx) //nolint

	if calls != 1 {
		t.Fatalf("expected 1 auth call (cached), got %d", calls)
	}
}

func TestMonnify_accessToken_refreshes_when_expired(t *testing.T) {
	calls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewEncoder(w).Encode(map[string]interface{}{
			"requestSuccessful": true,
			"responseBody":      map[string]interface{}{"accessToken": "tok", "expiresIn": 3600},
		})
	})
	p := newTestMonnify(t, mux)

	// Force token to appear expired
	p.token = "old"
	p.tokenExp = time.Now().Add(-1 * time.Second)

	ctx := context.Background()
	tok, err := p.accessToken(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if tok != "tok" {
		t.Fatalf("expected refreshed token, got %q", tok)
	}
	if calls != 1 {
		t.Fatalf("expected 1 refresh call, got %d", calls)
	}
}

func TestMonnify_accessToken_error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"requestSuccessful": false,
			"responseMessage":   "invalid credentials",
		})
	})
	p := newTestMonnify(t, mux)
	_, err := p.accessToken(context.Background())
	if err == nil {
		t.Fatal("expected error for failed auth")
	}
}

// ── InitiateCharge ────────────────────────────────────────────────────────────

func TestMonnify_InitiateCharge_success(t *testing.T) {
	mux := authMux(http.NewServeMux())
	mux.HandleFunc("/api/v1/merchant/transactions/init-transaction", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		// amount must be in Naira (kobo/100)
		if body["amount"].(float64) != 100.0 {
			http.Error(w, "wrong amount", http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"requestSuccessful": true,
			"responseBody": map[string]interface{}{
				"transactionReference": "MNFY|001",
				"paymentReference":     "ref-001",
				"checkoutUrl":          "https://sandbox.sdk.monnify.com/checkout/MNFY|001",
			},
		})
	})

	p := newTestMonnify(t, mux)
	resp, err := p.InitiateCharge(context.Background(), ChargeRequest{
		AmountKobo:  10000, // ₦100
		Email:       "user@example.com",
		Reference:   "ref-001",
		CallbackURL: "https://app.example.com/callback",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.AuthorizationURL == "" {
		t.Fatal("expected checkout URL")
	}
	if resp.Reference != "MNFY|001" {
		t.Fatalf("unexpected reference: %s", resp.Reference)
	}
}

func TestMonnify_InitiateCharge_provider_error(t *testing.T) {
	mux := authMux(http.NewServeMux())
	mux.HandleFunc("/api/v1/merchant/transactions/init-transaction", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"requestSuccessful": false,
			"responseMessage":   "invalid contract code",
		})
	})
	p := newTestMonnify(t, mux)
	_, err := p.InitiateCharge(context.Background(), ChargeRequest{AmountKobo: 10000, Email: "u@e.com", Reference: "r"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ── VerifyTransaction ─────────────────────────────────────────────────────────

func TestMonnify_VerifyTransaction_paid(t *testing.T) {
	mux := authMux(http.NewServeMux())
	mux.HandleFunc("/api/v2/transactions/MNFY%7C001", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"requestSuccessful": true,
			"responseBody": map[string]interface{}{
				"transactionReference": "MNFY|001",
				"amountPaid":           "100.00",
				"paymentStatus":        "PAID",
				"paidOn":               "29/07/2026 12:00:00 PM",
			},
		})
	})

	p := newTestMonnify(t, mux)
	vr, err := p.VerifyTransaction(context.Background(), "MNFY|001")
	if err != nil {
		t.Fatal(err)
	}
	if vr.Status != "success" {
		t.Fatalf("expected success, got %s", vr.Status)
	}
	if vr.AmountKobo != 10000 {
		t.Fatalf("expected 10000 kobo, got %d", vr.AmountKobo)
	}
	if vr.PaidAt == nil {
		t.Fatal("expected PaidAt to be set")
	}
}

func TestMonnify_VerifyTransaction_overpaid_maps_to_success(t *testing.T) {
	mux := authMux(http.NewServeMux())
	mux.HandleFunc("/api/v2/transactions/MNFY%7C002", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"requestSuccessful": true,
			"responseBody": map[string]interface{}{
				"transactionReference": "MNFY|002",
				"amountPaid":           "200.00",
				"paymentStatus":        "OVERPAID",
			},
		})
	})
	p := newTestMonnify(t, mux)
	vr, err := p.VerifyTransaction(context.Background(), "MNFY|002")
	if err != nil {
		t.Fatal(err)
	}
	if vr.Status != "success" {
		t.Fatalf("OVERPAID should map to success, got %s", vr.Status)
	}
}

func TestMonnify_VerifyTransaction_pending(t *testing.T) {
	mux := authMux(http.NewServeMux())
	mux.HandleFunc("/api/v2/transactions/MNFY%7C003", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"requestSuccessful": true,
			"responseBody": map[string]interface{}{
				"transactionReference": "MNFY|003",
				"amountPaid":           "0.00",
				"paymentStatus":        "PENDING",
			},
		})
	})
	p := newTestMonnify(t, mux)
	vr, err := p.VerifyTransaction(context.Background(), "MNFY|003")
	if err != nil {
		t.Fatal(err)
	}
	if vr.Status != "pending" {
		t.Fatalf("expected pending, got %s", vr.Status)
	}
}

// ── InitiateTransfer ──────────────────────────────────────────────────────────

func TestMonnify_InitiateTransfer_success(t *testing.T) {
	mux := authMux(http.NewServeMux())
	mux.HandleFunc("/api/v2/disbursements/single", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if body["sourceAccountNumber"] != "WALLET001" {
			t.Errorf("wrong sourceAccountNumber: %v", body["sourceAccountNumber"])
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if body["destinationAccountName"] == nil || body["destinationAccountName"] == "" {
			t.Error("missing destinationAccountName")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"requestSuccessful": true,
			"responseBody": map[string]interface{}{
				"reference": "TRF-001",
				"status":    "SUCCESS",
			},
		})
	})

	p := newTestMonnify(t, mux)
	resp, err := p.InitiateTransfer(context.Background(), TransferRequest{
		AmountKobo:             50000,
		RecipientCode:          "058:0123456789",
		Reference:              "TRF-001",
		Reason:                 "payout",
		DestinationAccountName: "John Doe",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TransferCode != "TRF-001" {
		t.Fatalf("unexpected transfer code: %s", resp.TransferCode)
	}
}

func TestMonnify_InitiateTransfer_bad_recipient_format(t *testing.T) {
	mux := authMux(http.NewServeMux())
	p := newTestMonnify(t, mux)
	_, err := p.InitiateTransfer(context.Background(), TransferRequest{
		AmountKobo:    1000,
		RecipientCode: "no-colon-here", // malformed
		Reference:     "r",
	})
	if err == nil {
		t.Fatal("expected error for malformed RecipientCode")
	}
}

// ── CreateReservedAccount ─────────────────────────────────────────────────────

func TestMonnify_CreateReservedAccount_success(t *testing.T) {
	mux := authMux(http.NewServeMux())
	mux.HandleFunc("/api/v2/bank-transfer/reserved-accounts", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		banks := body["preferredBanks"].([]interface{})
		if banks[0].(string) != "50515" {
			http.Error(w, "wrong preferred bank", http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"requestSuccessful": true,
			"responseBody": map[string]interface{}{
				"accountReference": "user-uuid-001",
				"accounts": []map[string]interface{}{
					{
						"bankCode":      "50515",
						"bankName":      "Moniepoint Microfinance Bank",
						"accountNumber": "6839490147",
					},
				},
			},
		})
	})

	p := newTestMonnify(t, mux)
	dva, err := p.CreateReservedAccount(context.Background(), DVARequest{
		UserID:   "user-uuid-001",
		FullName: "Jane Doe",
		Email:    "jane@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if dva.AccountNumber != "6839490147" {
		t.Fatalf("unexpected account number: %s", dva.AccountNumber)
	}
	if dva.BankCode != "50515" {
		t.Fatalf("expected Moniepoint bank code 50515, got %s", dva.BankCode)
	}
}

// ── InitiateUSSD ──────────────────────────────────────────────────────────────

func TestMonnify_InitiateUSSD_two_step_flow(t *testing.T) {
	initCalled, paymentCalled := false, false

	mux := authMux(http.NewServeMux())
	mux.HandleFunc("/api/v1/merchant/transactions/init-transaction", func(w http.ResponseWriter, r *http.Request) {
		initCalled = true
		json.NewEncoder(w).Encode(map[string]interface{}{
			"requestSuccessful": true,
			"responseBody": map[string]interface{}{
				"transactionReference": "MNFY|USSD|001",
			},
		})
	})
	mux.HandleFunc("/api/v1/merchant/bank-transfer/init-payment", func(w http.ResponseWriter, r *http.Request) {
		paymentCalled = true
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["transactionReference"] != "MNFY|USSD|001" {
			http.Error(w, "wrong txRef", http.StatusBadRequest)
			return
		}
		if body["bankCode"] != "058" {
			http.Error(w, "wrong bankCode", http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"requestSuccessful": true,
			"responseBody": map[string]interface{}{
				"transactionReference": "MNFY|USSD|001",
				"ussdPayment":          "*737*000*10000#",
			},
		})
	})

	p := newTestMonnify(t, mux)
	resp, err := p.InitiateUSSD(context.Background(), "058", 1000000, "user@example.com", "ref-ussd-001")
	if err != nil {
		t.Fatal(err)
	}
	if !initCalled {
		t.Fatal("init-transaction was not called")
	}
	if !paymentCalled {
		t.Fatal("init-payment was not called")
	}
	if resp.USSDCode != "*737*000*10000#" {
		t.Fatalf("unexpected USSD code: %s", resp.USSDCode)
	}
	if resp.ProviderRef != "MNFY|USSD|001" {
		t.Fatalf("unexpected provider ref: %s", resp.ProviderRef)
	}
}

func TestMonnify_InitiateUSSD_init_transaction_fails(t *testing.T) {
	mux := authMux(http.NewServeMux())
	mux.HandleFunc("/api/v1/merchant/transactions/init-transaction", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"requestSuccessful": false,
			"responseMessage":   "contract not found",
		})
	})
	p := newTestMonnify(t, mux)
	_, err := p.InitiateUSSD(context.Background(), "058", 1000, "u@e.com", "ref")
	if err == nil {
		t.Fatal("expected error when init-transaction fails")
	}
}

// ── VerifyWebhookSignature ────────────────────────────────────────────────────

func TestMonnify_VerifyWebhookSignature_valid(t *testing.T) {
	secret := "my-secret"
	payload := []byte(`{"event":"SUCCESSFUL_TRANSACTION"}`)
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	p := NewMonnify("key", secret, "c", "w")
	if !p.VerifyWebhookSignature(payload, sig) {
		t.Fatal("valid signature rejected")
	}
}

func TestMonnify_VerifyWebhookSignature_invalid(t *testing.T) {
	p := NewMonnify("key", "my-secret", "c", "w")
	payload := []byte(`{"event":"SUCCESSFUL_TRANSACTION"}`)
	if p.VerifyWebhookSignature(payload, "deadbeef") {
		t.Fatal("invalid signature accepted")
	}
}
