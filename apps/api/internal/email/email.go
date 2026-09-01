package email

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	sendbyte "github.com/Sendbyte-Africa/sendbyte-go"
	"github.com/speedplus/api/internal/config"
)

// Client wraps the SendByte SDK.
// All Send* methods are fire-and-forget safe: they log errors but never panic
// and never block the caller's transaction.
type Client struct {
	sb      *sendbyte.Client
	from    string
	enabled bool
}

// New initialises the SendByte client from config.
// Returns a no-op client when SENDBYTE_API_KEY is empty so the server starts
// cleanly in environments where email is not configured.
func New(cfg *config.Config) (*Client, error) {
	if cfg.SendbyteAPIKey == "" {
		slog.Warn("SENDBYTE_API_KEY not set — email sending disabled")
		return &Client{enabled: false, from: cfg.SendbyteFromAddr}, nil
	}
	sb, err := sendbyte.New(cfg.SendbyteAPIKey)
	if err != nil {
		return nil, fmt.Errorf("sendbyte init: %w", err)
	}
	return &Client{sb: sb, from: cfg.SendbyteFromAddr, enabled: true}, nil
}

// ── Transactional emails ──────────────────────────────────────────────────────

func (c *Client) SendWelcome(ctx context.Context, toEmail, firstName string) {
	c.send(ctx, sendbyte.SendEmailRequest{
		From:    c.from,
		To:      []string{toEmail},
		Subject: "Welcome to Fourdat",
		HTML: fmt.Sprintf(`<p>Hi %s,</p>
<p>Your Fourdat account is ready. Your wallet and virtual account number are waiting in the app.</p>
<p>The Fourdat Team</p>`, firstName),
		Text:           fmt.Sprintf("Hi %s,\n\nYour Fourdat account is ready.\n\nThe Fourdat Team", firstName),
		IdempotencyKey: "welcome-" + toEmail,
	})
}

func (c *Client) SendOTP(ctx context.Context, toEmail, firstName, code, purpose string) {
	subject := "Your Fourdat verification code"
	switch purpose {
	case "login":
		subject = "Your Fourdat login code"
	case "reset_pin":
		subject = "Your Fourdat PIN reset code"
	}
	c.send(ctx, sendbyte.SendEmailRequest{
		From:    c.from,
		To:      []string{toEmail},
		Subject: subject,
		HTML: fmt.Sprintf(`<p>Hi %s,</p>
<p>Your code is: <strong style="font-size:24px;letter-spacing:6px;">%s</strong></p>
<p>Expires in 5 minutes. Do not share it.</p>`, firstName, code),
		Text:           fmt.Sprintf("Hi %s,\n\nYour code: %s\n\nExpires in 5 minutes.", firstName, code),
		IdempotencyKey: "otp-" + toEmail + "-" + code,
	})
}

func (c *Client) SendOrderConfirmed(ctx context.Context, toEmail, firstName, orderID, merchantName string, totalKobo int64) {
	c.send(ctx, sendbyte.SendEmailRequest{
		From:    c.from,
		To:      []string{toEmail},
		Subject: fmt.Sprintf("Order from %s confirmed", merchantName),
		HTML: fmt.Sprintf(`<p>Hi %s,</p>
<p>Your order from <strong>%s</strong> is confirmed. Total: <strong>%s</strong></p>
<p>Your payment is protected and only released when you receive your order.</p>`,
			firstName, merchantName, formatKobo(totalKobo)),
		Text:           fmt.Sprintf("Hi %s,\n\nOrder from %s confirmed. Total: %s\nPayment protected until delivery.", firstName, merchantName, formatKobo(totalKobo)),
		IdempotencyKey: "order-confirmed-" + orderID,
	})
}

func (c *Client) SendDeliveryCode(ctx context.Context, toEmail, firstName, code, orderID string) {
	c.send(ctx, sendbyte.SendEmailRequest{
		From:    c.from,
		To:      []string{toEmail},
		Subject: "Your Fourdat delivery code",
		HTML: fmt.Sprintf(`<p>Hi %s,</p>
<p>Your rider is on the way. Share this code with whoever is collecting your package:</p>
<p style="font-size:36px;font-weight:bold;letter-spacing:10px;text-align:center;">%s</p>
<p>The code expires in 2 hours. Do not share it until the rider arrives.</p>`,
			firstName, code),
		Text:           fmt.Sprintf("Hi %s,\n\nYour delivery code: %s\n\nShare with whoever is collecting. Expires in 2 hours.", firstName, code),
		IdempotencyKey: "delivery-code-" + orderID,
	})
}

func (c *Client) SendOrderDelivered(ctx context.Context, toEmail, firstName, orderID, merchantName string, totalKobo int64) {
	c.send(ctx, sendbyte.SendEmailRequest{
		From:    c.from,
		To:      []string{toEmail},
		Subject: fmt.Sprintf("Order from %s delivered", merchantName),
		HTML: fmt.Sprintf(`<p>Hi %s,</p>
<p>Your order from <strong>%s</strong> has been delivered. <strong>%s</strong> released to merchant and rider.</p>
<p>Rate your experience in the app.</p>`,
			firstName, merchantName, formatKobo(totalKobo)),
		Text:           fmt.Sprintf("Hi %s,\n\nOrder from %s delivered. %s released. Rate your experience in the app.", firstName, merchantName, formatKobo(totalKobo)),
		IdempotencyKey: "order-delivered-" + orderID,
	})
}

func (c *Client) SendWalletFunded(ctx context.Context, toEmail, firstName string, amountKobo, newBalanceKobo int64) {
	c.send(ctx, sendbyte.SendEmailRequest{
		From:    c.from,
		To:      []string{toEmail},
		Subject: fmt.Sprintf("%s added to your Fourdat wallet", formatKobo(amountKobo)),
		HTML: fmt.Sprintf(`<p>Hi %s,</p>
<p><strong>%s</strong> added to your wallet. New balance: <strong>%s</strong></p>`,
			firstName, formatKobo(amountKobo), formatKobo(newBalanceKobo)),
		Text:           fmt.Sprintf("Hi %s,\n\n%s added to wallet. New balance: %s", firstName, formatKobo(amountKobo), formatKobo(newBalanceKobo)),
		IdempotencyKey: fmt.Sprintf("wallet-funded-%s-%d", toEmail, amountKobo),
	})
}

func (c *Client) SendPaymentLinkPaid(ctx context.Context, toEmail, firstName string, amountKobo int64, payerName, linkSlug string) {
	c.send(ctx, sendbyte.SendEmailRequest{
		From:    c.from,
		To:      []string{toEmail},
		Subject: fmt.Sprintf("%s received from %s", formatKobo(amountKobo), payerName),
		HTML: fmt.Sprintf(`<p>Hi %s,</p>
<p><strong>%s</strong> received from <strong>%s</strong> via your payment link. Money is in your wallet.</p>`,
			firstName, formatKobo(amountKobo), payerName),
		Text:           fmt.Sprintf("Hi %s,\n\n%s received from %s. Money is in your wallet.", firstName, formatKobo(amountKobo), payerName),
		IdempotencyKey: "payment-link-paid-" + linkSlug,
	})
}

func (c *Client) SendTransferReceived(ctx context.Context, toEmail, firstName string, amountKobo int64, senderName string) {
	c.send(ctx, sendbyte.SendEmailRequest{
		From:    c.from,
		To:      []string{toEmail},
		Subject: fmt.Sprintf("%s received from %s", formatKobo(amountKobo), senderName),
		HTML: fmt.Sprintf(`<p>Hi %s,</p>
<p><strong>%s</strong> sent to your wallet by <strong>%s</strong>.</p>`,
			firstName, formatKobo(amountKobo), senderName),
		Text:           fmt.Sprintf("Hi %s,\n\n%s received from %s.", firstName, formatKobo(amountKobo), senderName),
		IdempotencyKey: fmt.Sprintf("transfer-received-%s-%s-%d", toEmail, senderName, amountKobo),
	})
}

func (c *Client) SendKYCApproved(ctx context.Context, toEmail, firstName, docType string) {
	c.send(ctx, sendbyte.SendEmailRequest{
		From:    c.from,
		To:      []string{toEmail},
		Subject: "Your Fourdat verification is approved",
		HTML: fmt.Sprintf(`<p>Hi %s,</p>
<p>Your <strong>%s</strong> verification has been approved. Your account is now fully active.</p>
<p>The Fourdat Team</p>`, firstName, docType),
		Text:           fmt.Sprintf("Hi %s,\n\nYour %s verification has been approved. Your account is now fully active.\n\nThe Fourdat Team", firstName, docType),
		IdempotencyKey: fmt.Sprintf("kyc-approved-%s-%s", toEmail, docType),
	})
}

func (c *Client) SendKYCRejected(ctx context.Context, toEmail, firstName, docType, note string) {
	c.send(ctx, sendbyte.SendEmailRequest{
		From:    c.from,
		To:      []string{toEmail},
		Subject: "Action required: Fourdat verification",
		HTML: fmt.Sprintf(`<p>Hi %s,</p>
<p>Your <strong>%s</strong> verification could not be approved.</p>
<p>Reason: %s</p>
<p>Please re-submit with a valid document or contact support.</p>
<p>The Fourdat Team</p>`, firstName, docType, note),
		Text:           fmt.Sprintf("Hi %s,\n\nYour %s verification could not be approved.\nReason: %s\n\nPlease re-submit or contact support.\n\nThe Fourdat Team", firstName, docType, note),
		IdempotencyKey: fmt.Sprintf("kyc-rejected-%s-%s", toEmail, docType),
	})
}

// ── Internal ──────────────────────────────────────────────────────────────────

func (c *Client) send(ctx context.Context, req sendbyte.SendEmailRequest) {
	if !c.enabled {
		slog.Debug("email disabled — skipping", "subject", req.Subject)
		return
	}
	_, err := c.sb.Emails.Send(ctx, req)
	if err != nil {
		var apiErr *sendbyte.Error
		if errors.As(err, &apiErr) {
			slog.Error("sendbyte send failed",
				"code", apiErr.Code,
				"status", apiErr.Status,
				"message", apiErr.Message,
				"subject", req.Subject,
			)
			return
		}
		slog.Error("sendbyte send failed", "error", err, "subject", req.Subject)
	}
}

func formatKobo(kobo int64) string {
	naira := kobo / 100
	rem := kobo % 100
	if rem == 0 {
		return fmt.Sprintf("₦%s", thousands(naira))
	}
	return fmt.Sprintf("₦%s.%02d", thousands(naira), rem)
}

func thousands(n int64) string {
	s := fmt.Sprintf("%d", n)
	out := ""
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += ","
		}
		out += string(ch)
	}
	return out
}
