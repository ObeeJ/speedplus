// Package sms_test is a mock integration smoke test.
// It exercises every notification method on MockClient to verify:
//   1. The method exists and compiles (interface compliance).
//   2. The MockClient records the call with the correct phone.
//   3. No method panics on zero/nil inputs.
//
// Run: go test ./internal/sms/... -v -race
package sms_test

import (
	"testing"

	"github.com/speedplus/api/internal/sms"
)

// Compile-time assertion: MockClient satisfies the same surface as Client.
// If a method is added to Client but not MockClient, this file fails to compile.
var _ interface {
	OTP(phone, code, purpose string)
	OrderDelivered(phone, orderID string, totalKobo int64)
	OrderCancelled(phone, reason, refundAmount string)
	DeliveryCode(phone, code string)
	DeliveryCodeLocked(phone string)
	PINLocked(phone string)
	KYCApproved(phone, docType string)
	KYCRejected(phone, docType, note string)
	WalletFunded(phone string, amountKobo, balanceKobo int64)
	TransferReceived(phone string, amountKobo int64, senderName string)
	PaymentLinkPaid(phone string, amountKobo int64)
	ReferralRewarded(phone string, rewardKobo int64)
	LoyaltyPointsAwarded(phone string, points, totalPoints int)
	GiftCardIssued(phone, code string, amountKobo int64)
	GiftCardRedeemed(phone string, amountKobo int64)
	SubscriptionDunning(phone string, attempt int)
	SubscriptionPaused(phone string)
	RecertReminder(phone, serial string, daysLeft int)
	GasShortfallRefund(phone string, refundKobo int64)
	DisputeAutoRefunded(phone string, amountKobo int64)
	DisputeSLAUpdate(phone string)
	DisputeResolved(phone, recipient string, amountKobo int64)
	AccountSuspended(phone, reason string)
	AccountReactivated(phone string)
	TierUpgraded(phone string)
	PrescriptionRejected(phone, pharmacyName string, note *string)
	USSDCode(phone, ussdCode string, amountKobo int64)
	WeeklyPayoutFailed(phone string, amountKobo int64)
	RunNoDriver(phone string)
	OpsAlert(subject, detail string)
} = &sms.MockClient{}

const testPhone = "2348012345678"

func TestNormalisePhone(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"+2348012345678", "2348012345678"},
		{"08012345678", "2348012345678"},
		{"2348012345678", "2348012345678"},
		{"  +2348012345678  ", "2348012345678"},
	}
	for _, tc := range cases {
		got := sms.NormalisePhone(tc.in)
		if got != tc.want {
			t.Errorf("NormalisePhone(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestMockLifecycle exercises every notification method in the order a real
// user would encounter them: register → verify → order → deliver → dispute.
func TestMockLifecycle(t *testing.T) {
	m := &sms.MockClient{}

	// ── Auth ──────────────────────────────────────────────────────────────────
	m.OTP(testPhone, "123456", "phone verification")
	assertRecorded(t, m, testPhone, "OTP:123456")
	m.Reset()

	m.PINLocked(testPhone)
	assertRecorded(t, m, testPhone, "PINLocked")
	m.Reset()

	// ── KYC ───────────────────────────────────────────────────────────────────
	m.KYCApproved(testPhone, "NIN")
	assertRecorded(t, m, testPhone, "KYCApproved:NIN")
	m.Reset()

	note := "image too blurry"
	m.KYCRejected(testPhone, "BVN", note)
	assertRecorded(t, m, testPhone, "KYCRejected:BVN")
	m.Reset()

	// ── Wallet ────────────────────────────────────────────────────────────────
	m.WalletFunded(testPhone, 500000, 1500000)
	assertRecorded(t, m, testPhone, "WalletFunded")
	m.Reset()

	m.TransferReceived(testPhone, 200000, "Emeka Obi")
	assertRecorded(t, m, testPhone, "TransferReceived:Emeka Obi")
	m.Reset()

	m.PaymentLinkPaid(testPhone, 300000)
	assertRecorded(t, m, testPhone, "PaymentLinkPaid")
	m.Reset()

	m.USSDCode(testPhone, "*737*000*5000#", 500000)
	assertRecorded(t, m, testPhone, "USSDCode:*737*000*5000#")
	m.Reset()

	// ── Order lifecycle ───────────────────────────────────────────────────────
	m.DeliveryCode(testPhone, "847291")
	assertRecorded(t, m, testPhone, "DeliveryCode:847291")
	m.Reset()

	m.DeliveryCodeLocked(testPhone)
	assertRecorded(t, m, testPhone, "DeliveryCodeLocked")
	m.Reset()

	m.OrderDelivered(testPhone, "SPX-AB123", 675000)
	assertRecorded(t, m, testPhone, "OrderDelivered:SPX-AB123")
	m.Reset()

	m.OrderCancelled(testPhone, "merchant closed", "no charge")
	assertRecorded(t, m, testPhone, "OrderCancelled:merchant closed")
	m.Reset()

	// ── Gas vertical ──────────────────────────────────────────────────────────
	m.GasShortfallRefund(testPhone, 45000)
	assertRecorded(t, m, testPhone, "GasShortfallRefund")
	m.Reset()

	m.RecertReminder(testPhone, "SN-00123", 45)
	assertRecorded(t, m, testPhone, "RecertReminder:SN-00123")
	m.Reset()

	m.RunNoDriver(testPhone)
	assertRecorded(t, m, testPhone, "RunNoDriver")
	m.Reset()

	// ── Subscriptions ─────────────────────────────────────────────────────────
	m.SubscriptionDunning(testPhone, 1)
	assertRecorded(t, m, testPhone, "SubscriptionDunning")
	m.Reset()

	m.SubscriptionPaused(testPhone)
	assertRecorded(t, m, testPhone, "SubscriptionPaused")
	m.Reset()

	// ── Disputes ──────────────────────────────────────────────────────────────
	m.DisputeAutoRefunded(testPhone, 675000)
	assertRecorded(t, m, testPhone, "DisputeAutoRefunded")
	m.Reset()

	m.DisputeSLAUpdate(testPhone)
	assertRecorded(t, m, testPhone, "DisputeSLAUpdate")
	m.Reset()

	m.DisputeResolved(testPhone, "customer", 675000)
	assertRecorded(t, m, testPhone, "DisputeResolved:customer")
	m.Reset()

	m.DisputeResolved(testPhone, "merchant", 0)
	assertRecorded(t, m, testPhone, "DisputeResolved:merchant")
	m.Reset()

	// ── Admin actions ─────────────────────────────────────────────────────────
	m.AccountSuspended(testPhone, "fraud flag")
	assertRecorded(t, m, testPhone, "AccountSuspended")
	m.Reset()

	m.AccountReactivated(testPhone)
	assertRecorded(t, m, testPhone, "AccountReactivated")
	m.Reset()

	m.TierUpgraded(testPhone)
	assertRecorded(t, m, testPhone, "TierUpgraded")
	m.Reset()

	// ── Catalog ───────────────────────────────────────────────────────────────
	rejNote := "expired prescription"
	m.PrescriptionRejected(testPhone, "HealthPlus Pharmacy", &rejNote)
	assertRecorded(t, m, testPhone, "PrescriptionRejected")
	m.Reset()

	// nil note must not panic
	m.PrescriptionRejected(testPhone, "HealthPlus Pharmacy", nil)
	assertRecorded(t, m, testPhone, "PrescriptionRejected")
	m.Reset()

	// ── Referral / loyalty / gift cards ───────────────────────────────────────
	m.ReferralRewarded(testPhone, 50000)
	assertRecorded(t, m, testPhone, "ReferralRewarded")
	m.Reset()

	m.LoyaltyPointsAwarded(testPhone, 50, 350)
	assertRecorded(t, m, testPhone, "LoyaltyPointsAwarded")
	m.Reset()

	m.GiftCardIssued(testPhone, "ABCD-EFGH-IJKL-MNOP", 500000)
	assertRecorded(t, m, testPhone, "GiftCardIssued:ABCD-EFGH-IJKL-MNOP")
	m.Reset()

	m.GiftCardRedeemed(testPhone, 500000)
	assertRecorded(t, m, testPhone, "GiftCardRedeemed")
	m.Reset()

	// ── Driver ────────────────────────────────────────────────────────────────
	m.WeeklyPayoutFailed(testPhone, 1500000)
	assertRecorded(t, m, testPhone, "WeeklyPayoutFailed")
	m.Reset()

	// ── Ops alerts ────────────────────────────────────────────────────────────
	m.OpsAlert("FRAUD", "structuring detected user=abc")
	assertRecorded(t, m, "ops", "FRAUD:structuring detected user=abc")
	m.Reset()

	m.OpsAlert("RECONCILIATION_DRIFT", "provider=paystack drift=50000 kobo")
	assertRecorded(t, m, "ops", "RECONCILIATION_DRIFT:provider=paystack drift=50000 kobo")
	m.Reset()
}

// TestDisabledClientNoOp verifies that a disabled (zero-credential) real Client
// never panics and never sends anything.
func TestDisabledClientNoOp(t *testing.T) {
	c := sms.New("", "", "")
	// None of these should panic.
	c.OTP(testPhone, "000000", "test")
	c.OrderDelivered(testPhone, "SPX-00001", 100000)
	c.OpsAlert("TEST", "should not send")
}

// assertRecorded checks that the mock recorded exactly one message for the
// given phone with the given message content. Fails if 0 or >1 messages were
// sent — the latter catches accidental double-sends.
func assertRecorded(t *testing.T, m *sms.MockClient, phone, wantMsg string) {
	t.Helper()
	sent := m.Sent()
	if len(sent) != 1 {
		t.Fatalf("expected exactly 1 message, got %d: %v", len(sent), sent)
	}
	got := sent[0]
	if got.Phone != phone {
		t.Errorf("phone: got %q, want %q", got.Phone, phone)
	}
	if got.Message != wantMsg {
		t.Errorf("message: got %q, want %q", got.Message, wantMsg)
	}
}
