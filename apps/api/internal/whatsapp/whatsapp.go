// Package whatsapp sends order lifecycle notifications via Meta Cloud API.
// Cost: $0.0067/message (Nigeria utility rate, effective July 2026).
// Only the first outbound template per order is charged; all subsequent
// messages inside an open customer-service window (24 h after customer reply)
// are free per Meta's July 2025 pricing update.
package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const apiBase = "https://graph.facebook.com/v21.0"

// Client sends WhatsApp template messages via Meta Cloud API.
// Construct with New; zero value is inert (all sends are no-ops when disabled).
type Client struct {
	phoneNumberID string
	token         string
	http          *http.Client
	enabled       bool
}

// New returns a configured Client. If either credential is empty the client
// is disabled — all Send* calls become silent no-ops so the API boots cleanly
// in dev without WhatsApp credentials.
func New(phoneNumberID, token string) *Client {
	return &Client{
		phoneNumberID: phoneNumberID,
		token:         token,
		http:          &http.Client{Timeout: 10 * time.Second},
		enabled:       phoneNumberID != "" && token != "",
	}
}

// NormalisePhone strips a leading '+' so the number is in the E.164 format
// Meta expects without the plus sign (e.g. "2348012345678").
// Exported so callers can normalise before passing to any method.
func NormalisePhone(phone string) string {
	if len(phone) > 0 && phone[0] == '+' {
		return phone[1:]
	}
	return phone
}

// ── Wire types ────────────────────────────────────────────────────────────────

type message struct {
	MessagingProduct string `json:"messaging_product"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Template         *tmpl  `json:"template"`
}

type tmpl struct {
	Name       string      `json:"name"`
	Language   langCode    `json:"language"`
	Components []component `json:"components,omitempty"`
}

type langCode struct {
	Code string `json:"code"`
}

type component struct {
	Type       string  `json:"type"`
	Parameters []param `json:"parameters"`
}

type param struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// metaErrorResponse is the subset of Meta's error envelope we care about.
// Full shape: {"error":{"message":"...","type":"...","code":190,"fbtrace_id":"..."}}
// We log only code + type — never message, which can echo back token details.
type metaErrorResponse struct {
	Error struct {
		Code int    `json:"code"`
		Type string `json:"type"`
	} `json:"error"`
}

// ── Internal send ─────────────────────────────────────────────────────────────

func bodyParams(values ...string) []component {
	if len(values) == 0 {
		return nil
	}
	params := make([]param, len(values))
	for i, v := range values {
		params[i] = param{Type: "text", Text: v}
	}
	return []component{{Type: "body", Parameters: params}}
}

func (c *Client) send(ctx context.Context, to, templateName string, params ...string) error {
	payload := message{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "template",
		Template: &tmpl{
			Name:       templateName,
			Language:   langCode{Code: "en"},
			Components: bodyParams(params...),
		},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/%s/messages", apiBase, c.phoneNumberID),
		bytes.NewReader(b),
	)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		// Decode only code+type — never log the full body or message field,
		// which Meta can populate with token details on auth failures.
		var errBody metaErrorResponse
		json.NewDecoder(resp.Body).Decode(&errBody)
		return fmt.Errorf("status %d: meta_code=%d type=%s",
			resp.StatusCode, errBody.Error.Code, errBody.Error.Type)
	}
	return nil
}

// sendAsync fires the notification in a goroutine and logs failures.
// WhatsApp notifications are best-effort — a send failure must never
// roll back an order or block the API response.
// ctx is intentionally not propagated: the caller's request context will be
// cancelled before the goroutine completes; we use Background instead.
func (c *Client) sendAsync(phone, templateName string, params ...string) {
	if !c.enabled {
		return
	}
	go func() {
		if err := c.send(context.Background(), phone, templateName, params...); err != nil {
			slog.Error("whatsapp send failed",
				"template", templateName,
				"error", err,
			)
		}
	}()
}

// ── Public notification methods ───────────────────────────────────────────────
// phone must be E.164 without '+': use NormalisePhone before calling.
// Template names must match exactly what is approved in Meta Business Suite.

// OrderConfirmed — the one paid message per order ($0.0067).
// Opens the customer-service window; all subsequent sends in this order are free
// once the customer replies.
// Template vars: {{1}} orderID  {{2}} merchantName  {{3}} total (e.g. ₦6,750)
func (c *Client) OrderConfirmed(phone, orderID, merchantName, total string) {
	c.sendAsync(phone, "order_confirmed", orderID, merchantName, total)
}

// RiderAssigned — driver has accepted the offer.
// Template vars: {{1}} riderName  {{2}} eta (e.g. "15 mins")
func (c *Client) RiderAssigned(phone, riderName, eta string) {
	c.sendAsync(phone, "rider_assigned", riderName, eta)
}

// DeliveryCode — order is in_transit; rider has picked up.
// Template vars: {{1}} 6-digit code
func (c *Client) DeliveryCode(phone, code string) {
	c.sendAsync(phone, "delivery_code", code)
}

// OrderDelivered — code verified, escrow settled.
// Template vars: {{1}} orderID
func (c *Client) OrderDelivered(phone, orderID string) {
	c.sendAsync(phone, "order_delivered", orderID)
}

// OrderCancelled — order cancelled at any stage.
// Template vars: {{1}} reason  {{2}} refund amount or "no charge"
func (c *Client) OrderCancelled(phone, reason, refundAmount string) {
	c.sendAsync(phone, "order_cancelled", reason, refundAmount)
}

// PrescriptionReady — pharmacy has approved the prescription.
// Called from CatalogService.ReviewPrescription when approve=true.
// Template vars: {{1}} pharmacyName
func (c *Client) PrescriptionReady(phone, pharmacyName string) {
	c.sendAsync(phone, "prescription_ready", pharmacyName)
}

// SendOTP — delivers a one-time code to the user's WhatsApp number.
// Used as the primary OTP channel for phone-only users (no email on file).
// Template vars: {{1}} 6-digit code  {{2}} purpose label (e.g. "login")
func (c *Client) SendOTP(phone, code, purpose string) {
	c.sendAsync(phone, "otp", code, purpose)
}
