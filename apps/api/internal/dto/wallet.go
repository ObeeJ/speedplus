package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
)

// ── Wallet ────────────────────────────────────────────────────────────────────

type WalletResponse struct {
	BalanceKobo int64  `json:"balanceKobo"`
	Currency    string `json:"currency"`
}

type TransactionResponse struct {
	ID          uuid.UUID  `json:"id"`
	JournalID   uuid.UUID  `json:"journalId"`
	AmountKobo  int64      `json:"amountKobo"`
	Description string     `json:"description"`
	RefType     string     `json:"refType,omitempty"`
	RefID       *uuid.UUID `json:"refId,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type TransactionListResponse struct {
	Transactions []TransactionResponse `json:"transactions"`
	NextCursor   *uuid.UUID            `json:"nextCursor,omitempty"`
}

func TransactionFromModel(e model.LedgerEntry) TransactionResponse {
	return TransactionResponse{
		ID:          e.ID,
		JournalID:   e.JournalID,
		AmountKobo:  e.AmountKobo,
		Description: e.Description,
		RefType:     e.RefType,
		RefID:       e.RefID,
		CreatedAt:   e.CreatedAt,
	}
}

// ── Fund ──────────────────────────────────────────────────────────────────────

type FundWalletRequest struct {
	AmountKobo  int64  `json:"amountKobo"  binding:"required,min=100"`
	Email       string `json:"email"       binding:"required,email"`
	CallbackURL string `json:"callbackUrl" binding:"required"`
}

type FundWalletResponse struct {
	AuthorizationURL string `json:"authorizationUrl"`
	Reference        string `json:"reference"`
}

// ── Transfer ──────────────────────────────────────────────────────────────────

type TransferRequest struct {
	RecipientID string `json:"recipientId" binding:"required"`
	AmountKobo  int64  `json:"amountKobo"  binding:"required,min=100"`
	PIN         string `json:"pin"         binding:"required,len=6"`
}

// ── Withdrawal ────────────────────────────────────────────────────────────────

type WithdrawRequest struct {
	AmountKobo    int64  `json:"amountKobo"    binding:"required,min=100"`
	BankCode      string `json:"bankCode"      binding:"required"`
	AccountNumber string `json:"accountNumber" binding:"required"`
	PIN           string `json:"pin"           binding:"required,len=6"`
}

// ── EWA Cashout ───────────────────────────────────────────────────────────────

type CashoutRequest struct {
	AmountKobo int64 `json:"amountKobo" binding:"required,min=100"`
}

type CashoutResponse struct {
	Message    string `json:"message"`
	FeeKobo    int64  `json:"feeKobo"`
	NetKobo    int64  `json:"netKobo"`
}

// ── Paycodes ──────────────────────────────────────────────────────────────────

type GeneratePaycodeRequest struct {
	OrderID string `json:"orderId" binding:"required"`
}

type PaycodeResponse struct {
	ID        uuid.UUID `json:"id"`
	Payload   string    `json:"payload"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type ResolvePaycodeRequest struct {
	Payload string `json:"payload" binding:"required"`
}

type ResolvePaycodeResponse struct {
	PaycodeID  uuid.UUID `json:"paycodeId"`
	OrderID    uuid.UUID `json:"orderId"`
	TotalKobo  int64     `json:"totalKobo"`
	Status     string    `json:"status"`
	MerchantID uuid.UUID `json:"merchantId"`
	CustomerID uuid.UUID `json:"customerId"`
}

type ConfirmPaycodeRequest struct {
	PIN *string `json:"pin"`
}
