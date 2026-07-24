package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/payment"
	"gorm.io/gorm"
)

type monnifyUSSDProvider interface {
	Name() string
	InitiateUSSD(ctx context.Context, bankCode string, amountKobo int64, email, ref string) (*payment.USSDResponse, error)
}

type USSDService struct {
	db      *gorm.DB
	monnify monnifyUSSDProvider
}

func NewUSSDService(db *gorm.DB, monnify monnifyUSSDProvider) *USSDService {
	return &USSDService{db: db, monnify: monnify}
}

func (s *USSDService) SupportedBanks() []map[string]string {
	return []map[string]string{
		{"code": "058", "name": "GTBank"},
		{"code": "057", "name": "Zenith Bank"},
		{"code": "033", "name": "UBA"},
		{"code": "011", "name": "First Bank"},
		{"code": "044", "name": "Access Bank"},
		{"code": "035", "name": "Wema Bank"},
		{"code": "032", "name": "Union Bank"},
	}
}

// InitiateUSSDFund creates a USSDIntent and returns the shortcode to display.
// Confirmation arrives via the existing SUCCESSFUL_TRANSACTION Monnify webhook.
func (s *USSDService) InitiateUSSDFund(ctx context.Context, userID uuid.UUID, bankCode string, amountKobo int64, email string) (*model.USSDIntent, error) {
	ref := uuid.NewString()

	resp, err := s.monnify.InitiateUSSD(ctx, bankCode, amountKobo, email, ref)
	if err != nil {
		return nil, fmt.Errorf("ussd initiate: %w", err)
	}

	intent := model.PaymentIntent{
		ID:          uuid.New(),
		UserID:      userID,
		AmountKobo:  amountKobo,
		Provider:    s.monnify.Name(),
		Status:      "pending",
		ProviderRef: &resp.ProviderRef,
	}
	if err := s.db.WithContext(ctx).Create(&intent).Error; err != nil {
		return nil, err
	}

	bankName := bankCode
	for _, b := range s.SupportedBanks() {
		if b["code"] == bankCode {
			bankName = b["name"]
			break
		}
	}

	ui := &model.USSDIntent{
		ID:              uuid.New(),
		UserID:          userID,
		AmountKobo:      amountKobo,
		Provider:        s.monnify.Name(),
		BankCode:        bankCode,
		BankName:        &bankName,
		USSDCode:        resp.USSDCode,
		ProviderRef:     resp.ProviderRef,
		PaymentIntentID: &intent.ID,
		Status:          "pending",
		ExpiresAt:       time.Now().Add(30 * time.Minute),
	}
	if err := s.db.WithContext(ctx).Create(ui).Error; err != nil {
		return nil, err
	}
	return ui, nil
}

func (s *USSDService) GetIntent(ctx context.Context, userID, intentID uuid.UUID) (*model.USSDIntent, error) {
	var ui model.USSDIntent
	if err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", intentID, userID).
		First(&ui).Error; err != nil {
		return nil, err
	}
	return &ui, nil
}
