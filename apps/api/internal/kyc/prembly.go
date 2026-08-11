package kyc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Provider is the interface all KYC backends must satisfy.
type Provider interface {
	VerifyNIN(ctx context.Context, nin, firstName, lastName, dob string) (*Result, error)
	VerifyBVN(ctx context.Context, bvn, firstName, lastName, dob string) (*Result, error)
	VerifyDriversLicence(ctx context.Context, licenceNo, firstName, lastName, dob string) (*Result, error)
	Liveness(ctx context.Context, imageBase64 string) (*Result, error)
	VerifyCAC(ctx context.Context, rcNumber, companyName string) (*Result, error)
}

type Result struct {
	Verified bool
	// StatusOK reflects the provider's top-level request status — distinct
	// from Verified, which is the identity-match outcome. A request can be
	// StatusOK=true but Verified=false (legitimate no-match), or
	// StatusOK=false (provider-side failure: no credit, bad input, outage) —
	// callers must not conflate the two or a KYC admin can't tell "this
	// person failed verification" from "Prembly rejected the request".
	StatusOK    bool
	Message     string // human-readable reason from the provider, empty on success
	ProviderRef string
	RawPayload  []byte
}

// PremblyProvider implements Provider against Prembly's IdentityPass API.
// Docs: https://docs.prembly.com/docs
type PremblyProvider struct {
	apiKey  string
	baseURL string
	appID   string
	client  *http.Client
}

// NewPrembly constructs a Prembly client. appID is the dashboard-issued
// application identifier (Prembly console → Applications) — it is a
// credential-like value tied to your account's rate limits and billing, not
// a free-text label.
func NewPrembly(apiKey, baseURL, appID string) *PremblyProvider {
	return &PremblyProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		appID:   appID,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *PremblyProvider) VerifyNIN(ctx context.Context, nin, firstName, lastName, dob string) (*Result, error) {
	body := map[string]string{
		"number":     nin,
		"first_name": firstName,
		"last_name":  lastName,
		"dob":        dob, // YYYY-MM-DD
	}
	return p.post(ctx, "/api/v2/biometrics/merchant/data/verification/nin_wo_face", body)
}

func (p *PremblyProvider) VerifyBVN(ctx context.Context, bvn, firstName, lastName, dob string) (*Result, error) {
	body := map[string]string{
		"number":     bvn,
		"first_name": firstName,
		"last_name":  lastName,
		"dob":        dob,
	}
	return p.post(ctx, "/api/v2/biometrics/merchant/data/verification/bvn", body)
}

func (p *PremblyProvider) VerifyDriversLicence(ctx context.Context, licenceNo, firstName, lastName, dob string) (*Result, error) {
	body := map[string]string{
		"number":     licenceNo,
		"first_name": firstName,
		"last_name":  lastName,
		"dob":        dob,
	}
	return p.post(ctx, "/api/v2/biometrics/merchant/data/verification/drivers_license", body)
}

func (p *PremblyProvider) Liveness(ctx context.Context, imageBase64 string) (*Result, error) {
	body := map[string]string{"image": imageBase64}
	return p.post(ctx, "/api/v2/biometrics/merchant/data/verification/liveness", body)
}

func (p *PremblyProvider) VerifyCAC(ctx context.Context, rcNumber, companyName string) (*Result, error) {
	body := map[string]string{
		"rc_number":    rcNumber,
		"company_name": companyName,
	}
	return p.post(ctx, "/api/v2/biometrics/merchant/data/verification/cac", body)
}

func (p *PremblyProvider) post(ctx context.Context, path string, payload map[string]string) (*Result, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("app-id", p.appID)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prembly request: %w", err)
	}
	defer resp.Body.Close()

	var raw map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("prembly decode: %w", err)
	}

	rawBytes, _ := json.Marshal(raw)

	// Prembly's envelope: {"status": true|false, "detail": {...},
	// "verification": {"status": "VERIFIED"|"NOT_FOUND"|...}, "message": "..."}
	// "status" is the request-level outcome (auth/credit/input validity);
	// "verification.status" is the identity-match outcome. They must be
	// tracked separately — see Result.StatusOK doc comment.
	statusOK, _ := raw["status"].(bool)

	verified := false
	if v, ok := raw["verification"].(map[string]interface{}); ok {
		verified = v["status"] == "VERIFIED"
	}

	message := ""
	if m, ok := raw["message"].(string); ok {
		message = m
	}

	ref := ""
	if d, ok := raw["detail"].(map[string]interface{}); ok {
		if id, ok := d["id"].(string); ok {
			ref = id
		}
	}

	if !statusOK {
		return &Result{
			Verified:    false,
			StatusOK:    false,
			Message:     message,
			ProviderRef: ref,
			RawPayload:  rawBytes,
		}, fmt.Errorf("prembly request failed: %s", message)
	}

	return &Result{
		Verified:    verified,
		StatusOK:    true,
		Message:     message,
		ProviderRef: ref,
		RawPayload:  rawBytes,
	}, nil
}
