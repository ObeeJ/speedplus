package payment

// Bridge.xyz stablecoin payment provider.
//
// Docs:
//   - Wallets:       https://apidocs.bridge.xyz/api-reference/bridge-wallets/
//   - Orchestration: https://apidocs.bridge.xyz/platform/orchestration/overview
//   - Introduction:  https://apidocs.bridge.xyz/api-reference/introduction/introduction
//
// Flow:
//  1. On first use, create a Bridge wallet for the user (idempotent by externalId).
//  2. Create an Orchestration payment session — Bridge returns a hosted checkout URL.
//  3. User completes payment in stablecoin (USDC/USDT on supported chains).
//  4. Bridge calls our webhook when the payment settles; we credit the wallet.

import (
	"bytes"
	"context"
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const bridgeBaseURL = "https://api.bridge.xyz/v0"

// BridgeProvider implements Provider for stablecoin payments via Bridge.xyz.
// It does NOT implement DVAProvider — Bridge wallets are crypto-native, not NGN bank accounts.
type BridgeProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewBridge(apiKey string) *BridgeProvider {
	return &BridgeProvider{
		apiKey:     apiKey,
		baseURL:    bridgeBaseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// NewBridgeWithURL is for testing — allows injecting a mock server URL.
func NewBridgeWithURL(apiKey, baseURL string) *BridgeProvider {
	b := NewBridge(apiKey)
	b.baseURL = baseURL
	return b
}

func (b *BridgeProvider) Name() string { return "bridge" }

// EnsureWallet creates a Bridge wallet for the user if one doesn't exist yet.
// externalID should be the Fourdat user UUID (stable, unique).
// Returns the Bridge wallet ID.
func (b *BridgeProvider) EnsureWallet(ctx context.Context, externalID, email, fullName string) (string, error) {
	body := map[string]interface{}{
		"external_id": externalID,
		"email":       email,
		"full_name":   fullName,
		"type":        "individual",
	}
	var resp struct {
		ID    string `json:"id"`
		Error string `json:"error"`
	}
	if err := b.post(ctx, "/wallets", body, &resp); err != nil {
		return "", fmt.Errorf("bridge ensure wallet: %w", err)
	}
	if resp.Error != "" {
		return "", fmt.Errorf("bridge ensure wallet: %s", resp.Error)
	}
	return resp.ID, nil
}

// InitiateCharge creates a Bridge Orchestration payment session.
// The user pays in stablecoin; Bridge converts and settles in NGN equivalent.
// req.Metadata["bridge_wallet_id"] must be set by the caller.
func (b *BridgeProvider) InitiateCharge(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
	walletID, _ := req.Metadata["bridge_wallet_id"]

	// Amount in USD cents (Bridge uses USD as the quote currency).
	// We convert from kobo: 1 NGN ≈ 0.00065 USD — caller should pass USD cents directly
	// via Metadata["amount_usd_cents"] when available; fall back to kobo/1538 otherwise.
	amountUSDCents := int64(0)
	if v, ok := req.Metadata["amount_usd_cents"]; ok {
		fmt.Sscanf(v, "%d", &amountUSDCents)
	}
	if amountUSDCents == 0 {
		// rough fallback: 1 USD ≈ 1538 NGN (kobo/100/1538*100)
		amountUSDCents = req.AmountKobo / 1538
		if amountUSDCents < 1 {
			amountUSDCents = 1
		}
	}

	body := map[string]interface{}{
		"source": map[string]interface{}{
			"payment_rail": "crypto",
			"currency":     "usdc",
		},
		"destination": map[string]interface{}{
			"payment_rail": "wallet",
			"currency":     "ngn",
			"wallet_id":    walletID,
		},
		"amount":       fmt.Sprintf("%d", amountUSDCents),
		"on_behalf_of": req.Metadata["bridge_wallet_id"],
		"developer_fee": map[string]interface{}{
			"amount":   "0",
			"currency": "usd",
		},
		"external_reference": req.Reference,
		"redirect_url":       req.CallbackURL,
	}

	var resp struct {
		ID          string `json:"id"`
		CheckoutURL string `json:"checkout_url"`
		Error       string `json:"error"`
	}
	if err := b.post(ctx, "/orchestration/payment_sessions", body, &resp); err != nil {
		return nil, fmt.Errorf("bridge initiate charge: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("bridge initiate charge: %s", resp.Error)
	}

	return &ChargeResponse{
		AuthorizationURL: resp.CheckoutURL,
		Reference:        req.Reference,
		AccessCode:       resp.ID, // Bridge session ID stored as AccessCode
	}, nil
}

// VerifyTransaction checks the status of a Bridge payment session.
func (b *BridgeProvider) VerifyTransaction(ctx context.Context, ref string) (*VerifyResponse, error) {
	var resp struct {
		State  string `json:"state"` // "pending" | "payment_processed" | "funds_received" | "failed"
		Amount string `json:"amount"`
		Error  string `json:"error"`
	}
	if err := b.get(ctx, "/orchestration/payment_sessions/"+ref, &resp); err != nil {
		return nil, fmt.Errorf("bridge verify: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("bridge verify: %s", resp.Error)
	}

	status := "pending"
	switch resp.State {
	case "payment_processed", "funds_received":
		status = "success"
	case "failed", "cancelled":
		status = "failed"
	}

	return &VerifyResponse{
		Reference: ref,
		Status:    status,
	}, nil
}

// InitiateTransfer is not supported for Bridge (crypto-native, no NGN bank transfer).
func (b *BridgeProvider) InitiateTransfer(_ context.Context, _ TransferRequest) (*TransferResponse, error) {
	return nil, fmt.Errorf("bridge: bank transfers not supported; use wallet-to-wallet")
}

// VerifyWebhookSignature validates Bridge webhook authenticity.
// Bridge sends an "Api-Key" header matching your API key.
// Uses hmac.Equal for constant-time comparison to prevent timing attacks.
func (b *BridgeProvider) VerifyWebhookSignature(_ []byte, signature string) bool {
	return hmac.Equal([]byte(signature), []byte(b.apiKey))
}

func (b *BridgeProvider) post(ctx context.Context, path string, body interface{}, out interface{}) error {
	bs, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+path, bytes.NewReader(bs))
	req.Header.Set("Api-Key", b.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

func (b *BridgeProvider) get(ctx context.Context, path string, out interface{}) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, b.baseURL+path, nil)
	req.Header.Set("Api-Key", b.apiKey)
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}
