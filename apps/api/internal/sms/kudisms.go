// Package sms delivers transactional SMS via the Kudi SMS corporate API.
//
// Design mirrors internal/whatsapp/whatsapp.go:
//   - Zero value is inert (all sends are no-ops when disabled).
//   - All public methods are fire-and-forget via sendAsync.
//   - Credentials absent → disabled, never panics.
//   - PII rule: only last-4 of phone logged. Message bodies must never
//     contain full card numbers, tokens, or passwords.
package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	baseURL = "https://my.kudisms.net/api"
	// gateway=2 refunds charge for DND numbers (DND won't deliver).
	dndGateway = "2"
)

// Client sends SMS via Kudi SMS corporate endpoint.
// Construct with New; zero value is inert.
type Client struct {
	token    string
	senderID string
	opsPhone string // internal ops number for fraud/reconciliation alerts
	http     *http.Client
	enabled  bool
}

// New returns a configured Client. If token or senderID is empty the client
// is disabled — all Send* calls become silent no-ops.
// opsPhone is the internal number that receives fraud/ops alerts (optional).
func New(token, senderID, opsPhone string) *Client {
	return &Client{
		token:    token,
		senderID: senderID,
		opsPhone: opsPhone,
		http:     &http.Client{Timeout: 10 * time.Second},
		enabled:  token != "" && senderID != "",
	}
}

// NormalisePhone strips leading '+' and converts leading '0' to '234' for
// Nigerian numbers. Kudi expects E.164 without the plus sign.
func NormalisePhone(phone string) string {
	p := strings.TrimSpace(phone)
	if strings.HasPrefix(p, "+") {
		return p[1:]
	}
	if strings.HasPrefix(p, "0") {
		return "234" + p[1:]
	}
	return p
}

// ── Wire types ────────────────────────────────────────────────────────────────

type kudiResponse struct {
	Status    string `json:"status"`
	ErrorCode string `json:"error_code"`
	Msg       string `json:"msg"`
}

// ── Internal send ─────────────────────────────────────────────────────────────

func (c *Client) send(ctx context.Context, to, message string) error {
	params := url.Values{}
	params.Set("token", c.token)
	params.Set("senderID", c.senderID)
	params.Set("recipients", to)
	params.Set("message", message)
	params.Set("gateway", dndGateway)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+"/corporate",
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	if resp.StatusCode >= 300 {
		return fmt.Errorf("kudi http %d", resp.StatusCode)
	}

	var kr kudiResponse
	if err := json.Unmarshal(body, &kr); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if kr.ErrorCode != "000" {
		return fmt.Errorf("kudi error_code=%s msg=%s", kr.ErrorCode, kr.Msg)
	}
	return nil
}

// Send delivers an SMS synchronously. Used by the asynq worker (handleSendSMS)
// so retries are managed by the queue, not by a bare goroutine.
// Callers outside the worker should use EnqueueSMS instead.
func (c *Client) Send(ctx context.Context, to, message string) error {
	if !c.enabled {
		return nil
	}
	return c.send(ctx, to, message)
}

// sendAsync fires the SMS in a goroutine. Used only for ops alerts where
// the caller has no asynq client available (e.g. LedgerService.AlertOverdueDisputes).
// For all user-facing notifications prefer EnqueueSMS via the worker queue.
func (c *Client) sendAsync(phone, message string) {
	if !c.enabled {
		return
	}
	go func() {
		if err := c.send(context.Background(), phone, message); err != nil {
			slog.Error("sms send failed",
				"phone_suffix", safeSuffix(phone),
				"error", err,
			)
		}
	}()
}

// safeSuffix returns the last 4 digits of a phone number for log correlation
// without logging the full number (PII).
func safeSuffix(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return "****" + phone[len(phone)-4:]
}

// ── User-facing notifications ─────────────────────────────────────────────────

// OTP sends a one-time code. purpose is a human label e.g. "phone verification".
func (c *Client) OTP(phone, code, purpose string) {
	c.sendAsync(phone, fmt.Sprintf(
		"Your Fourdat %s code is: %s. Valid for 5 minutes. Do not share.",
		purpose, code,
	))
}

// OrderDelivered notifies a customer their order was delivered and wallet debited.
func (c *Client) OrderDelivered(phone, orderID string, totalKobo int64) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat: Order %s delivered. \u20a6%.0f debited from your wallet. Thank you!",
		orderID, float64(totalKobo)/100,
	))
}

// OrderCancelled notifies a customer their order was cancelled.
func (c *Client) OrderCancelled(phone, reason, refundAmount string) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat: Your order was cancelled (%s). Refund: %s.",
		reason, refundAmount,
	))
}

// DeliveryCode sends the 6-digit delivery confirmation code.
func (c *Client) DeliveryCode(phone, code string) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat delivery code: %s. Share this with your rider to confirm delivery.",
		code,
	))
}

// DeliveryCodeLocked tells the customer their delivery code is locked.
func (c *Client) DeliveryCodeLocked(phone string) {
	c.sendAsync(phone, "Fourdat: Your delivery code has been locked after too many failed attempts. Show your Fourdat card to the rider instead.")
}

// PINLocked tells the customer their wallet PIN is temporarily locked.
func (c *Client) PINLocked(phone string) {
	c.sendAsync(phone, "Fourdat: Your wallet PIN has been locked for 30 minutes due to failed attempts. Try again later or contact support.")
}

// KYCApproved notifies a user their KYC document was approved.
func (c *Client) KYCApproved(phone, docType string) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat: Your %s verification has been approved. You can now access all features.",
		docType,
	))
}

// KYCRejected notifies a user their KYC document was rejected.
func (c *Client) KYCRejected(phone, docType, note string) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat: Your %s verification was not approved. Reason: %s. Please resubmit.",
		docType, note,
	))
}

// WalletFunded notifies a user their wallet was credited.
func (c *Client) WalletFunded(phone string, amountKobo, balanceKobo int64) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat: \u20a6%.0f added to your wallet. New balance: \u20a6%.0f.",
		float64(amountKobo)/100, float64(balanceKobo)/100,
	))
}

// TransferReceived notifies a user they received a wallet transfer.
func (c *Client) TransferReceived(phone string, amountKobo int64, senderName string) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat: You received \u20a6%.0f from %s.",
		float64(amountKobo)/100, senderName,
	))
}

// PaymentLinkPaid notifies a payment link creator they received money.
func (c *Client) PaymentLinkPaid(phone string, amountKobo int64) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat: Your payment link was paid. \u20a6%.0f added to your wallet.",
		float64(amountKobo)/100,
	))
}

// ReferralRewarded notifies a referrer they earned a reward.
func (c *Client) ReferralRewarded(phone string, rewardKobo int64) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat: You earned \u20a6%.0f for referring a friend! Keep sharing your code.",
		float64(rewardKobo)/100,
	))
}

// LoyaltyPointsAwarded notifies a user they earned loyalty points.
func (c *Client) LoyaltyPointsAwarded(phone string, points, totalPoints int) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat: You earned %d loyalty points. Total: %d pts.",
		points, totalPoints,
	))
}

// GiftCardIssued sends the gift card code to the issuer.
// Security: code is sent only to the issuer's own phone — never to a third party.
func (c *Client) GiftCardIssued(phone, code string, amountKobo int64) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat Gift Card: \u20a6%.0f. Code: %s. Share this code with the recipient.",
		float64(amountKobo)/100, code,
	))
}

// GiftCardRedeemed notifies a user their gift card was redeemed.
func (c *Client) GiftCardRedeemed(phone string, amountKobo int64) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat: Gift card redeemed. \u20a6%.0f added to your wallet.",
		float64(amountKobo)/100,
	))
}

// SubscriptionDunning notifies a customer their subscription charge failed.
func (c *Client) SubscriptionDunning(phone string, attempt int) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat: We couldn't process your auto-refill (attempt %d/3). Please top up your wallet to avoid pausing.",
		attempt,
	))
}

// SubscriptionPaused notifies a customer their subscription was paused.
func (c *Client) SubscriptionPaused(phone string) {
	c.sendAsync(phone, "Fourdat: Your gas auto-refill has been paused after 3 failed charges. Top up your wallet and reactivate in the app.")
}

// RecertReminder notifies a customer their cylinder is due for recertification.
func (c *Client) RecertReminder(phone, serial string, daysLeft int) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat: Your cylinder (SN: %s) is due for recertification in %d days. Visit a certified centre to avoid service interruption.",
		serial, daysLeft,
	))
}

// GasShortfallRefund notifies a customer they received a partial refund.
func (c *Client) GasShortfallRefund(phone string, refundKobo int64) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat: A shortfall was detected in your gas delivery. \u20a6%.0f has been refunded to your wallet.",
		float64(refundKobo)/100,
	))
}

// DisputeAutoRefunded notifies a customer their dispute was auto-resolved.
func (c *Client) DisputeAutoRefunded(phone string, amountKobo int64) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat: Your dispute has been resolved. \u20a6%.0f refunded to your wallet.",
		float64(amountKobo)/100,
	))
}

// DisputeSLAUpdate notifies a customer their dispute is still under review.
func (c *Client) DisputeSLAUpdate(phone string) {
	c.sendAsync(phone, "Fourdat: Your dispute is taking longer than expected. Our team is reviewing it and will update you within 24 hours.")
}

// DisputeResolved notifies a customer their dispute outcome.
// recipient must be "customer" or "merchant".
func (c *Client) DisputeResolved(phone, recipient string, amountKobo int64) {
	if recipient == "customer" {
		c.sendAsync(phone, fmt.Sprintf(
			"Fourdat: Your dispute has been resolved in your favour. \u20a6%.0f refunded to your wallet.",
			float64(amountKobo)/100,
		))
	} else {
		c.sendAsync(phone, "Fourdat: Your dispute has been reviewed. The charge has been upheld. Contact support if you have questions.")
	}
}

// AccountSuspended notifies a user their account was suspended.
func (c *Client) AccountSuspended(phone, reason string) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat: Your account has been suspended. Reason: %s. Contact support@fourdat.com.",
		reason,
	))
}

// AccountReactivated notifies a user their account was reactivated.
func (c *Client) AccountReactivated(phone string) {
	c.sendAsync(phone, "Fourdat: Your account has been reactivated. Welcome back!")
}

// TierUpgraded notifies a customer they unlocked Pay on Arrival.
func (c *Client) TierUpgraded(phone string) {
	c.sendAsync(phone, "Fourdat: You've unlocked Pay on Arrival! You can now pay at the door for orders up to \u20a610,000.")
}

// PrescriptionRejected notifies a customer their prescription was rejected.
func (c *Client) PrescriptionRejected(phone, pharmacyName string, note *string) {
	msg := fmt.Sprintf("Fourdat: Your prescription at %s was not approved.", pharmacyName)
	if note != nil && *note != "" {
		msg += " Reason: " + *note
	}
	c.sendAsync(phone, msg)
}

// USSDCode sends the USSD dial string to the user's phone.
func (c *Client) USSDCode(phone, ussdCode string, amountKobo int64) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat: Dial %s to complete your \u20a6%.0f wallet top-up. Expires in 30 minutes.",
		ussdCode, float64(amountKobo)/100,
	))
}

// WeeklyPayoutFailed notifies a driver their weekly payout failed.
func (c *Client) WeeklyPayoutFailed(phone string, amountKobo int64) {
	c.sendAsync(phone, fmt.Sprintf(
		"Fourdat: Your weekly payout of \u20a6%.0f could not be processed. Please ensure your bank account is up to date in the app.",
		float64(amountKobo)/100,
	))
}

// RunNoDriver notifies a customer their batched gas delivery is delayed.
func (c *Client) RunNoDriver(phone string) {
	c.sendAsync(phone, "Fourdat: Your scheduled gas delivery has been delayed — we're finding a driver. We'll update you shortly.")
}

// ── Ops-facing alerts (sent to internal ops phone) ────────────────────────────

// OpsAlert sends a plain-text alert to the configured ops phone number.
// Used for: fraud flags, reconciliation drift, SLA breach, suspicious activity.
// Never call this with user PII in the message body.
func (c *Client) OpsAlert(subject, detail string) {
	if c.opsPhone == "" {
		return
	}
	// Truncate to avoid exceeding SMS page limits (Kudi max 6 pages ~900 chars).
	// Use rune-safe truncation to avoid splitting multi-byte UTF-8 characters.
	c.sendAsync(c.opsPhone, fmt.Sprintf("Fourdat OPS [%s]: %s", subject, truncateRunes(detail, 200)))
}

// truncateRunes truncates s to at most max Unicode code points, appending
// "..." if truncated. Safe for multi-byte UTF-8 strings.
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}
