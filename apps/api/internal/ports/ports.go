// Package ports declares the behaviour interfaces that cross service boundaries.
//
// Rule: a service that needs behaviour from another service imports the
// interface from this package, not the concrete type from service/.
// The concrete wiring happens only in cmd/server/main.go.
package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
)

// PINVerifier is the subset of AuthService used by WalletService and PaycodeService.
type PINVerifier interface {
	VerifyPIN(ctx context.Context, userID uuid.UUID, pin string) error
}

// TierRecorder is the subset of TierService used by PaycodeService.
type TierRecorder interface {
	RecordCompletion(ctx context.Context, userID uuid.UUID)
}

// OnboardingRunner is the subset of OnboardingService used by AuthService.
type OnboardingRunner interface {
	Run(ctx context.Context, user *model.User) error
	RunByID(ctx context.Context, userID string) error
}

// WhatsAppNotifier is the subset of whatsapp.Client used by OrderService.
// Accepting an interface here keeps service/order.go decoupled from the
// concrete HTTP client and makes the service testable without network calls.
type WhatsAppNotifier interface {
	OrderConfirmed(phone, orderID, merchantName, total string)
	RiderAssigned(phone, riderName, eta string)
	DeliveryCode(phone, code string)
	OrderDelivered(phone, orderID string)
	OrderCancelled(phone, reason, refundAmount string)
	PrescriptionReady(phone, pharmacyName string)
	SendOTP(phone, code, purpose string)
}
