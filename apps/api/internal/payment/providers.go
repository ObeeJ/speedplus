package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DVARequest is the input for creating a dedicated virtual account.
type DVARequest struct {
	UserID    string // used as accountReference
	FullName  string
	Email     string
}

// DVAResponse is the created virtual account details.
type DVAResponse struct {
	AccountNumber string
	BankName      string
	BankCode      string
	ProviderRef   string // accountReference echoed back
}

// DVAProvider is implemented by providers that support dedicated virtual accounts.
type DVAProvider interface {
	CreateReservedAccount(ctx context.Context, req DVARequest) (*DVAResponse, error)
}

// Provider is the interface all payment backends must satisfy.
type Provider interface {
	Name() string
	InitiateCharge(ctx context.Context, req ChargeRequest) (*ChargeResponse, error)
	VerifyTransaction(ctx context.Context, ref string) (*VerifyResponse, error)
	InitiateTransfer(ctx context.Context, req TransferRequest) (*TransferResponse, error)
	VerifyWebhookSignature(payload []byte, signature string) bool
}

type ChargeRequest struct {
	AmountKobo     int64
	Email          string
	Reference      string
	CallbackURL    string
	Metadata       map[string]string
}

type ChargeResponse struct {
	AuthorizationURL string
	Reference        string
	AccessCode       string
}

type VerifyResponse struct {
	Reference  string
	AmountKobo int64
	Status     string // "success" | "failed" | "pending"
	PaidAt     *time.Time
}

type TransferRequest struct {
	AmountKobo    int64
	RecipientCode string
	Reference     string
	Reason        string
}

type TransferResponse struct {
	TransferCode string
	Status       string
}

// ── Paystack ──────────────────────────────────────────────────────────────────

type PaystackProvider struct {
	secretKey  string
	httpClient *http.Client
}

func NewPaystack(secretKey string) *PaystackProvider {
	return &PaystackProvider{
		secretKey:  secretKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *PaystackProvider) Name() string { return "paystack" }

func (p *PaystackProvider) InitiateCharge(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
	body := map[string]interface{}{
		"amount":       req.AmountKobo,
		"email":        req.Email,
		"reference":    req.Reference,
		"callback_url": req.CallbackURL,
		"metadata":     req.Metadata,
		"currency":     "NGN",
	}
	var resp struct {
		Status bool `json:"status"`
		Data   struct {
			AuthorizationURL string `json:"authorization_url"`
			Reference        string `json:"reference"`
			AccessCode       string `json:"access_code"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := p.post(ctx, "/transaction/initialize", body, &resp); err != nil {
		return nil, err
	}
	if !resp.Status {
		return nil, fmt.Errorf("paystack: %s", resp.Message)
	}
	return &ChargeResponse{
		AuthorizationURL: resp.Data.AuthorizationURL,
		Reference:        resp.Data.Reference,
		AccessCode:       resp.Data.AccessCode,
	}, nil
}

func (p *PaystackProvider) VerifyTransaction(ctx context.Context, ref string) (*VerifyResponse, error) {
	var resp struct {
		Status bool `json:"status"`
		Data   struct {
			Reference string `json:"reference"`
			Amount    int64  `json:"amount"`
			Status    string `json:"status"`
			PaidAt    string `json:"paid_at"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := p.get(ctx, "/transaction/verify/"+ref, &resp); err != nil {
		return nil, err
	}
	if !resp.Status {
		return nil, fmt.Errorf("paystack verify: %s", resp.Message)
	}

	vr := &VerifyResponse{
		Reference:  resp.Data.Reference,
		AmountKobo: resp.Data.Amount,
		Status:     resp.Data.Status,
	}
	if resp.Data.PaidAt != "" {
		t, _ := time.Parse(time.RFC3339, resp.Data.PaidAt)
		vr.PaidAt = &t
	}
	return vr, nil
}

func (p *PaystackProvider) InitiateTransfer(ctx context.Context, req TransferRequest) (*TransferResponse, error) {
	body := map[string]interface{}{
		"source":    "balance",
		"amount":    req.AmountKobo,
		"recipient": req.RecipientCode,
		"reference": req.Reference,
		"reason":    req.Reason,
	}
	var resp struct {
		Status bool `json:"status"`
		Data   struct {
			TransferCode string `json:"transfer_code"`
			Status       string `json:"status"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := p.post(ctx, "/transfer", body, &resp); err != nil {
		return nil, err
	}
	if !resp.Status {
		return nil, fmt.Errorf("paystack transfer: %s", resp.Message)
	}
	return &TransferResponse{TransferCode: resp.Data.TransferCode, Status: resp.Data.Status}, nil
}

// VerifyWebhookSignature validates Paystack HMAC-SHA512 signature.
// Must be called BEFORE processing any webhook payload.
func (p *PaystackProvider) VerifyWebhookSignature(payload []byte, signature string) bool {
	mac := hmac.New(sha512.New, []byte(p.secretKey))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (p *PaystackProvider) post(ctx context.Context, path string, body interface{}, out interface{}) error {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.paystack.co"+path, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+p.secretKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

func (p *PaystackProvider) get(ctx context.Context, path string, out interface{}) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.paystack.co"+path, nil)
	req.Header.Set("Authorization", "Bearer "+p.secretKey)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

// ── Flutterwave ───────────────────────────────────────────────────────────────

type FlutterwaveProvider struct {
	secretKey  string
	verifyHash string
	httpClient *http.Client
}

func NewFlutterwave(secretKey, verifyHash string) *FlutterwaveProvider {
	return &FlutterwaveProvider{
		secretKey:  secretKey,
		verifyHash: verifyHash,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (f *FlutterwaveProvider) Name() string { return "flutterwave" }

func (f *FlutterwaveProvider) InitiateCharge(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
	body := map[string]interface{}{
		"tx_ref":       req.Reference,
		"amount":       req.AmountKobo / 100, // Flutterwave uses naira, not kobo
		"currency":     "NGN",
		"redirect_url": req.CallbackURL,
		"customer":     map[string]string{"email": req.Email},
		"meta":         req.Metadata,
	}
	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Link string `json:"link"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := f.post(ctx, "/payments", body, &resp); err != nil {
		return nil, err
	}
	if resp.Status != "success" {
		return nil, fmt.Errorf("flutterwave: %s", resp.Message)
	}
	return &ChargeResponse{AuthorizationURL: resp.Data.Link, Reference: req.Reference}, nil
}

func (f *FlutterwaveProvider) VerifyTransaction(ctx context.Context, ref string) (*VerifyResponse, error) {
	var resp struct {
		Status string `json:"status"`
		Data   struct {
			TxRef     string  `json:"tx_ref"`
			Amount    float64 `json:"amount"`
			Status    string  `json:"status"`
			CreatedAt string  `json:"created_at"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := f.get(ctx, "/transactions/verify_by_reference?tx_ref="+ref, &resp); err != nil {
		return nil, err
	}
	status := "failed"
	if resp.Data.Status == "successful" {
		status = "success"
	}
	return &VerifyResponse{
		Reference:  resp.Data.TxRef,
		AmountKobo: int64(resp.Data.Amount * 100),
		Status:     status,
	}, nil
}

func (f *FlutterwaveProvider) InitiateTransfer(ctx context.Context, req TransferRequest) (*TransferResponse, error) {
	body := map[string]interface{}{
		"account_bank":    "000", // set by caller via recipient code
		"account_number":  req.RecipientCode,
		"amount":          req.AmountKobo / 100,
		"currency":        "NGN",
		"reference":       req.Reference,
		"narration":       req.Reason,
	}
	var resp struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := f.post(ctx, "/transfers", body, &resp); err != nil {
		return nil, err
	}
	return &TransferResponse{Status: resp.Data.Status}, nil
}

// VerifyWebhookSignature validates Flutterwave verif-hash header.
func (f *FlutterwaveProvider) VerifyWebhookSignature(payload []byte, signature string) bool {
	return hmac.Equal([]byte(f.verifyHash), []byte(signature))
}

func (f *FlutterwaveProvider) post(ctx context.Context, path string, body interface{}, out interface{}) error {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.flutterwave.com/v3"+path, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+f.secretKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

func (f *FlutterwaveProvider) get(ctx context.Context, path string, out interface{}) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.flutterwave.com/v3"+path, nil)
	req.Header.Set("Authorization", "Bearer "+f.secretKey)
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

// ReadBody reads and restores the request body (needed for webhook signature verification).
func ReadBody(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

// ── Monnify ───────────────────────────────────────────────────────────────────
// Docs: https://api.monnify.com
// Auth model: short-lived JWT obtained via Basic auth — cached in-process.
// Amount unit: Naira (not kobo). All kobo values divided by 100 on the way out,
// multiplied by 100 on the way in.

type MonnifyProvider struct {
	apiKey       string
	secretKey    string
	contractCode string
	baseURL      string
	httpClient   *http.Client

	// token cache — refreshed lazily when expired; mu guards token+tokenExp
	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

func NewMonnify(apiKey, secretKey, contractCode string) *MonnifyProvider {
	return &MonnifyProvider{
		apiKey:       apiKey,
		secretKey:    secretKey,
		contractCode: contractCode,
		baseURL:      "https://api.monnify.com",
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (m *MonnifyProvider) Name() string { return "monnify" }

// accessToken returns a cached JWT, refreshing it when within 60s of expiry.
// The whole check-and-refresh runs under mu: concurrent callers on a cold or
// expired cache serialize into a single Monnify auth call.
func (m *MonnifyProvider) accessToken(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.token != "" && time.Now().Add(60*time.Second).Before(m.tokenExp) {
		return m.token, nil
	}

	creds := m.apiKey + ":" + m.secretKey
	encoded := base64.StdEncoding.EncodeToString([]byte(creds))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/api/v1/auth/login", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic "+encoded)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("monnify auth: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		RequestSuccessful bool `json:"requestSuccessful"`
		ResponseBody      struct {
			AccessToken string `json:"accessToken"`
			ExpiresIn   int    `json:"expiresIn"` // seconds
		} `json:"responseBody"`
		ResponseMessage string `json:"responseMessage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("monnify auth decode: %w", err)
	}
	if !result.RequestSuccessful {
		return "", fmt.Errorf("monnify auth failed: %s", result.ResponseMessage)
	}

	m.token = result.ResponseBody.AccessToken
	// Use expiresIn from response; default to 55 minutes if missing
	ttl := result.ResponseBody.ExpiresIn
	if ttl == 0 {
		ttl = 3300
	}
	m.tokenExp = time.Now().Add(time.Duration(ttl) * time.Second)
	return m.token, nil
}

func (m *MonnifyProvider) InitiateCharge(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
	token, err := m.accessToken(ctx)
	if err != nil {
		return nil, err
	}

	// Monnify expects amount in Naira
	amountNaira := float64(req.AmountKobo) / 100.0

	body := map[string]interface{}{
		"amount":             amountNaira,
		"customerEmail":      req.Email,
		"paymentReference":   req.Reference,
		"paymentDescription": "SpeedPlus payment",
		"currencyCode":       "NGN",
		"contractCode":       m.contractCode,
		"redirectUrl":        req.CallbackURL,
		"paymentMethods":     []string{"CARD", "ACCOUNT_TRANSFER"},
	}

	var result struct {
		RequestSuccessful bool `json:"requestSuccessful"`
		ResponseBody      struct {
			TransactionReference string `json:"transactionReference"`
			PaymentReference     string `json:"paymentReference"`
			CheckoutUrl          string `json:"checkoutUrl"`
		} `json:"responseBody"`
		ResponseMessage string `json:"responseMessage"`
	}
	if err := m.post(ctx, token, "/api/v1/merchant/transactions/init-transaction", body, &result); err != nil {
		return nil, err
	}
	if !result.RequestSuccessful {
		return nil, fmt.Errorf("monnify charge: %s", result.ResponseMessage)
	}

	return &ChargeResponse{
		AuthorizationURL: result.ResponseBody.CheckoutUrl,
		Reference:        result.ResponseBody.TransactionReference,
	}, nil
}

func (m *MonnifyProvider) VerifyTransaction(ctx context.Context, ref string) (*VerifyResponse, error) {
	token, err := m.accessToken(ctx)
	if err != nil {
		return nil, err
	}

	// transactionReference must be URL-encoded per Monnify docs
	encRef := url.QueryEscape(ref)

	var result struct {
		RequestSuccessful bool `json:"requestSuccessful"`
		ResponseBody      struct {
			TransactionReference string  `json:"transactionReference"`
			PaymentReference     string  `json:"paymentReference"`
			AmountPaid           string  `json:"amountPaid"`
			TotalPayable         string  `json:"totalPayable"`
			PaymentStatus        string  `json:"paymentStatus"` // "PAID" | "PENDING" | "FAILED" | "OVERPAID"
			PaidOn               string  `json:"paidOn"`
		} `json:"responseBody"`
		ResponseMessage string `json:"responseMessage"`
	}
	if err := m.get(ctx, token, "/api/v2/transactions/"+encRef, &result); err != nil {
		return nil, err
	}
	if !result.RequestSuccessful {
		return nil, fmt.Errorf("monnify verify: %s", result.ResponseMessage)
	}

	status := "failed"
	if result.ResponseBody.PaymentStatus == "PAID" || result.ResponseBody.PaymentStatus == "OVERPAID" {
		status = "success"
	} else if result.ResponseBody.PaymentStatus == "PENDING" {
		status = "pending"
	}

	// Parse amount — Monnify returns Naira as a string, convert to kobo
	var amountKobo int64
	if f, err := strconv.ParseFloat(result.ResponseBody.AmountPaid, 64); err == nil {
		amountKobo = int64(f * 100)
	}

	vr := &VerifyResponse{
		Reference:  result.ResponseBody.TransactionReference,
		AmountKobo: amountKobo,
		Status:     status,
	}
	if result.ResponseBody.PaidOn != "" {
		// Monnify format: "25/07/2022 11:20:20 AM"
		if t, err := time.Parse("02/01/2006 03:04:05 PM", result.ResponseBody.PaidOn); err == nil {
			vr.PaidAt = &t
		}
	}
	return vr, nil
}

func (m *MonnifyProvider) InitiateTransfer(ctx context.Context, req TransferRequest) (*TransferResponse, error) {
	token, err := m.accessToken(ctx)
	if err != nil {
		return nil, err
	}

	// RecipientCode is expected as "bankCode:accountNumber" for Monnify
	// Callers must format it this way when building a TransferRequest for Monnify.
	parts := strings.SplitN(req.RecipientCode, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("monnify transfer: RecipientCode must be 'bankCode:accountNumber'")
	}
	bankCode, accountNumber := parts[0], parts[1]

	amountNaira := float64(req.AmountKobo) / 100.0

	body := map[string]interface{}{
		"amount":                    amountNaira,
		"reference":                 req.Reference,
		"narration":                 req.Reason,
		"destinationBankCode":       bankCode,
		"destinationAccountNumber":  accountNumber,
		"currency":                  "NGN",
		"sourceAccountNumber":       m.contractCode, // Monnify uses contract code as source
	}

	var result struct {
		RequestSuccessful bool `json:"requestSuccessful"`
		ResponseBody      struct {
			Reference string `json:"reference"`
			Status    string `json:"status"`
		} `json:"responseBody"`
		ResponseMessage string `json:"responseMessage"`
	}
	if err := m.post(ctx, token, "/api/v2/disbursements/single", body, &result); err != nil {
		return nil, err
	}
	if !result.RequestSuccessful {
		return nil, fmt.Errorf("monnify transfer: %s", result.ResponseMessage)
	}

	return &TransferResponse{
		TransferCode: result.ResponseBody.Reference,
		Status:       result.ResponseBody.Status,
	}, nil
}

// CreateReservedAccount creates a dedicated virtual account (DVA) for a user.
// Called once at registration. The account number is permanent.
func (m *MonnifyProvider) CreateReservedAccount(ctx context.Context, req DVARequest) (*DVAResponse, error) {
	token, err := m.accessToken(ctx)
	if err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"accountReference":    req.UserID,
		"accountName":         "SpeedPlus / " + req.FullName,
		"currencyCode":        "NGN",
		"contractCode":        m.contractCode,
		"customerEmail":       req.Email,
		"customerName":        req.FullName,
		"getAllAvailableBanks": false,
		"preferredBanks":      []string{"035"}, // Wema Bank
	}

	var result struct {
		RequestSuccessful bool `json:"requestSuccessful"`
		ResponseBody      struct {
			AccountReference string `json:"accountReference"`
			Accounts         []struct {
				BankCode      string `json:"bankCode"`
				BankName      string `json:"bankName"`
				AccountNumber string `json:"accountNumber"`
			} `json:"accounts"`
		} `json:"responseBody"`
		ResponseMessage string `json:"responseMessage"`
	}
	if err := m.post(ctx, token, "/api/v2/bank-transfer/reserved-accounts", body, &result); err != nil {
		return nil, err
	}
	if !result.RequestSuccessful || len(result.ResponseBody.Accounts) == 0 {
		return nil, fmt.Errorf("monnify DVA: %s", result.ResponseMessage)
	}

	acct := result.ResponseBody.Accounts[0]
	return &DVAResponse{
		AccountNumber: acct.AccountNumber,
		BankName:      acct.BankName,
		BankCode:      acct.BankCode,
		ProviderRef:   result.ResponseBody.AccountReference,
	}, nil
}

// VerifyWebhookSignature validates Monnify HMAC-SHA512.
// Monnify sends the signature in the "monnify-signature" header as a hex-encoded
// HMAC-SHA512 of the raw request body, keyed with the secret key.
func (m *MonnifyProvider) VerifyWebhookSignature(payload []byte, signature string) bool {
	mac := hmac.New(sha512.New, []byte(m.secretKey))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (m *MonnifyProvider) post(ctx context.Context, token, path string, body interface{}, out interface{}) error {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+path, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

func (m *MonnifyProvider) get(ctx context.Context, token, path string, out interface{}) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, m.baseURL+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

// USSDResponse holds the USSD shortcode and Monnify transaction reference.
type USSDResponse struct {
	USSDCode    string
	ProviderRef string
}

// InitiateUSSD generates a USSD shortcode for wallet funding via Monnify.
// bankCode is the CBN bank code (e.g. "058" = GTBank, "057" = Zenith).
// Confirmation arrives via the existing SUCCESSFUL_TRANSACTION webhook.
func (m *MonnifyProvider) InitiateUSSD(ctx context.Context, bankCode string, amountKobo int64, email, ref string) (*USSDResponse, error) {
	token, err := m.accessToken(ctx)
	if err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"amount":             float64(amountKobo) / 100.0,
		"bankCode":           bankCode,
		"paymentReference":   ref,
		"customerEmail":      email,
		"paymentDescription": "SpeedPlus wallet funding",
		"contractCode":       m.contractCode,
		"currencyCode":       "NGN",
	}

	var result struct {
		RequestSuccessful bool `json:"requestSuccessful"`
		ResponseBody      struct {
			TransactionReference string `json:"transactionReference"`
			UssdPaymentCode      string `json:"ussdPaymentCode"`
		} `json:"responseBody"`
		ResponseMessage string `json:"responseMessage"`
	}
	if err := m.post(ctx, token, "/api/v1/merchant/transactions/pay-with-shortcode", body, &result); err != nil {
		return nil, err
	}
	if !result.RequestSuccessful {
		return nil, fmt.Errorf("monnify ussd: %s", result.ResponseMessage)
	}

	return &USSDResponse{
		USSDCode:    result.ResponseBody.UssdPaymentCode,
		ProviderRef: result.ResponseBody.TransactionReference,
	}, nil
}
