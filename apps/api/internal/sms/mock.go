package sms

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// MockClient records sent messages for use in tests.
// All methods are safe for concurrent use.
type MockClient struct {
	mu   sync.Mutex
	sent []MockMessage
}

// MockMessage records a single sent SMS.
type MockMessage struct {
	Phone   string
	Message string
}

func (m *MockClient) record(phone, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, MockMessage{Phone: phone, Message: message})
}

// Sent returns a copy of all recorded messages.
func (m *MockClient) Sent() []MockMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]MockMessage, len(m.sent))
	copy(out, m.sent)
	return out
}

// Reset clears recorded messages.
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = nil
}

// Implement the same surface as Client so tests can swap in MockClient.

func (m *MockClient) OTP(phone, code, purpose string)                          { m.record(phone, "OTP:"+code) }
func (m *MockClient) OrderDelivered(phone, orderID string, totalKobo int64)    { m.record(phone, "OrderDelivered:"+orderID) }
func (m *MockClient) OrderCancelled(phone, reason, refund string)              { m.record(phone, "OrderCancelled:"+reason) }
func (m *MockClient) DeliveryCode(phone, code string)                          { m.record(phone, "DeliveryCode:"+code) }
func (m *MockClient) DeliveryCodeLocked(phone string)                          { m.record(phone, "DeliveryCodeLocked") }
func (m *MockClient) PINLocked(phone string)                                   { m.record(phone, "PINLocked") }
func (m *MockClient) KYCApproved(phone, docType string)                        { m.record(phone, "KYCApproved:"+docType) }
func (m *MockClient) KYCRejected(phone, docType, note string)                  { m.record(phone, "KYCRejected:"+docType) }
func (m *MockClient) WalletFunded(phone string, amount, balance int64)         { m.record(phone, "WalletFunded") }
func (m *MockClient) TransferReceived(phone string, amount int64, sender string) { m.record(phone, "TransferReceived:"+sender) }
func (m *MockClient) PaymentLinkPaid(phone string, amount int64)               { m.record(phone, "PaymentLinkPaid") }
func (m *MockClient) ReferralRewarded(phone string, reward int64)              { m.record(phone, "ReferralRewarded") }
func (m *MockClient) LoyaltyPointsAwarded(phone string, pts, total int)        { m.record(phone, "LoyaltyPointsAwarded") }
func (m *MockClient) GiftCardIssued(phone, code string, amount int64)          { m.record(phone, "GiftCardIssued:"+code) }
func (m *MockClient) GiftCardRedeemed(phone string, amount int64)              { m.record(phone, "GiftCardRedeemed") }
func (m *MockClient) SubscriptionDunning(phone string, attempt int)            { m.record(phone, "SubscriptionDunning") }
func (m *MockClient) SubscriptionPaused(phone string)                          { m.record(phone, "SubscriptionPaused") }
func (m *MockClient) RecertReminder(phone, serial string, days int)            { m.record(phone, "RecertReminder:"+serial) }
func (m *MockClient) GasShortfallRefund(phone string, refund int64)            { m.record(phone, "GasShortfallRefund") }
func (m *MockClient) DisputeAutoRefunded(phone string, amount int64)           { m.record(phone, "DisputeAutoRefunded") }
func (m *MockClient) DisputeSLAUpdate(phone string)                            { m.record(phone, "DisputeSLAUpdate") }
func (m *MockClient) DisputeResolved(phone, recipient string, amount int64)    { m.record(phone, "DisputeResolved:"+recipient) }
func (m *MockClient) AccountSuspended(phone, reason string)                    { m.record(phone, "AccountSuspended") }
func (m *MockClient) AccountReactivated(phone string)                          { m.record(phone, "AccountReactivated") }
func (m *MockClient) TierUpgraded(phone string)                                { m.record(phone, "TierUpgraded") }
func (m *MockClient) PrescriptionRejected(phone, pharmacy string, note *string) { m.record(phone, "PrescriptionRejected") }
func (m *MockClient) USSDCode(phone, code string, amount int64)                { m.record(phone, "USSDCode:"+code) }
func (m *MockClient) WeeklyPayoutFailed(phone string, amount int64)            { m.record(phone, "WeeklyPayoutFailed") }
func (m *MockClient) RunNoDriver(phone string)                                 { m.record(phone, "RunNoDriver") }
func (m *MockClient) OpsAlert(subject, detail string)                          { m.record("ops", subject+":"+detail) }

// Send satisfies the worker's smsSender interface.
func (m *MockClient) Send(_ context.Context, phone, message string) error {
	m.record(phone, message)
	return nil
}

// PhoneByID satisfies the worker's phoneResolver interface.
// Returns a deterministic fake phone so worker tests don't need a real DB.
func (m *MockClient) PhoneByID(_ context.Context, userID uuid.UUID) (string, error) {
	return "2348000000000", nil
}
