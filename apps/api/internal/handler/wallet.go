package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/payment"
	"github.com/speedplus/api/internal/repo"
	"github.com/speedplus/api/internal/service"
)

type walletSvc interface {
	InitiateFund(ctx context.Context, userID uuid.UUID, amountKobo int64, email, idempotencyKey, callbackURL string) (*payment.ChargeResponse, error)
	Transfer(ctx context.Context, senderID, recipientID uuid.UUID, amountKobo int64, pin, idempotencyKey string) error
	EWACashout(ctx context.Context, driverID uuid.UUID, amountKobo int64, idempotencyKey string) error
	ProcessWebhook(ctx context.Context, payload service.WebhookPayload) error
	InitiateCryptoFund(ctx context.Context, userID uuid.UUID, amountKobo int64, email, fullName, idempotencyKey, callbackURL string, bridge *payment.BridgeProvider) (*payment.ChargeResponse, error)
}

type ledgerSvc interface {
	ResolveWalletOwner(ctx context.Context, userID uuid.UUID, role string) (uuid.UUID, error)
	GetBalance(ctx context.Context, userID uuid.UUID) (int64, error)
	GetTransactions(ctx context.Context, userID uuid.UUID, cursor *uuid.UUID, limit int) ([]model.LedgerEntry, error)
}

type WalletHandler struct {
	wallet   walletSvc
	ledger   ledgerSvc
	bridge   *payment.BridgeProvider // nil when Bridge is disabled
	userRepo interface {
		FindByPhone(ctx context.Context, phone string) (*model.User, error)
		FindByUsername(ctx context.Context, username string) (*model.User, error)
		FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	}
}

func NewWalletHandler(wallet *service.WalletService, ledger *service.LedgerService, userRepo repo.UserRepo) *WalletHandler {
	return &WalletHandler{wallet: wallet, ledger: ledger, userRepo: userRepo}
}

// SetBridge wires the Bridge provider after construction (called only when BRIDGE_ENABLED=true).
func (h *WalletHandler) SetBridge(b *payment.BridgeProvider) { h.bridge = b }

func (h *WalletHandler) GetBalance(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	role := c.GetString(middleware.CtxUserRole)
	ownerID, err := h.ledger.ResolveWalletOwner(c.Request.Context(), userID, role)
	if err != nil {
		internalError(c, err)
		return
	}
	bal, err := h.ledger.GetBalance(c.Request.Context(), ownerID)
	if err != nil {
		internalError(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, successResp(gin.H{
		"balanceKobo": bal,
		"currency":    "NGN",
	}))
}

func (h *WalletHandler) GetTransactions(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	role := c.GetString(middleware.CtxUserRole)
	ownerID, err := h.ledger.ResolveWalletOwner(c.Request.Context(), userID, role)
	if err != nil {
		internalError(c, err)
		return
	}

	var cursor *uuid.UUID
	if raw := c.Query("cursor"); raw != "" {
		id, err := uuid.Parse(raw)
		if err == nil {
			cursor = &id
		}
	}
	limit := 20

	entries, err := h.ledger.GetTransactions(c.Request.Context(), ownerID, cursor, limit)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"transactions": entries}))
}

func (h *WalletHandler) Fund(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Idempotency-Key required", "Idempotency-Key"))
		return
	}

	var req struct {
		AmountKobo  int64  `json:"amountKobo" binding:"required,min=10000"`
		Email       string `json:"email" binding:"required,email"`
		CallbackURL string `json:"callbackUrl" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	resp, err := h.wallet.InitiateFund(c.Request.Context(), userID, req.AmountKobo, req.Email, idempotencyKey, req.CallbackURL)
	if err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{
		"authorizationUrl": resp.AuthorizationURL,
		"reference":        resp.Reference,
	}))
}

// FundCrypto initiates a stablecoin wallet-funding session via Bridge.xyz.
// POST /wallet/fund/crypto
func (h *WalletHandler) FundCrypto(c *gin.Context) {
	if h.bridge == nil {
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Stablecoin payments are not enabled", ""))
		return
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Idempotency-Key required", "Idempotency-Key"))
		return
	}

	var req struct {
		AmountKobo  int64  `json:"amountKobo"  binding:"required,min=10000"`
		Email       string `json:"email"       binding:"required,email"`
		FullName    string `json:"fullName"    binding:"required"`
		CallbackURL string `json:"callbackUrl" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	resp, err := h.wallet.InitiateCryptoFund(
		c.Request.Context(), userID, req.AmountKobo,
		req.Email, req.FullName, idempotencyKey, req.CallbackURL, h.bridge,
	)
	if err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{
		"authorizationUrl": resp.AuthorizationURL,
		"reference":        resp.Reference,
	}))
}

func (h *WalletHandler) Transfer(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Idempotency-Key required", "Idempotency-Key"))
		return
	}

	var req struct {
		// Recipient can be a UUID, @username, or phone number.
		// Exactly one must be provided.
		RecipientID string `json:"recipientId"`
		Username    string `json:"username"`
		Phone       string `json:"phone"`
		AmountKobo  int64  `json:"amountKobo" binding:"required,min=10000"`
		PIN         string `json:"pin"        binding:"required,len=4"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	// Resolve recipient to a UUID
	var recipient *model.User
	var err error
	ctx := c.Request.Context()

	switch {
	case req.RecipientID != "":
		rid, parseErr := uuid.Parse(req.RecipientID)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid recipient ID", "recipientId"))
			return
		}
		recipient, err = h.userRepo.FindByID(ctx, rid)
	case req.Username != "":
		username := strings.TrimPrefix(req.Username, "@")
		recipient, err = h.userRepo.FindByUsername(ctx, username)
	case req.Phone != "":
		recipient, err = h.userRepo.FindByPhone(ctx, req.Phone)
	default:
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Provide recipientId, username, or phone", ""))
		return
	}

	if err != nil {
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Recipient not found", ""))
		return
	}

	senderID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if senderID == recipient.ID {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Cannot transfer to yourself", ""))
		return
	}

	if err := h.wallet.Transfer(ctx, senderID, recipient.ID, req.AmountKobo, req.PIN, idempotencyKey); err != nil {
		switch {
		case errors.Is(err, service.ErrInsufficientBalance):
			c.JSON(http.StatusUnprocessableEntity, errResp("INSUFFICIENT_FUNDS", "Insufficient balance for this transfer", ""))
		case errors.Is(err, service.ErrBankAccountRequired):
			c.JSON(http.StatusUnprocessableEntity, errResp("BANK_ACCOUNT_REQUIRED", err.Error(), ""))
		default:
			slog.ErrorContext(ctx, "wallet transfer failed", "error", err, "sender", senderID)
			c.JSON(http.StatusUnprocessableEntity, errResp("TRANSFER_FAILED", "Transfer could not be completed. Please try again.", ""))
		}
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{
		"message":       "transfer successful",
		"recipientName": recipient.FirstName + " " + recipient.LastName,
	}))
}

func (h *WalletHandler) EWACashout(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Idempotency-Key required", "Idempotency-Key"))
		return
	}

	var req struct {
		AmountKobo int64 `json:"amountKobo" binding:"required,min=10000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	driverID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.wallet.EWACashout(c.Request.Context(), driverID, req.AmountKobo, idempotencyKey); err != nil {
		switch {
		case errors.Is(err, service.ErrBankAccountRequired):
			c.JSON(http.StatusUnprocessableEntity, errResp("BANK_ACCOUNT_REQUIRED", err.Error(), ""))
		case errors.Is(err, service.ErrInsufficientBalance):
			c.JSON(http.StatusUnprocessableEntity, errResp("INSUFFICIENT_FUNDS", "Insufficient earned balance for cashout", ""))
		default:
			slog.ErrorContext(c.Request.Context(), "EWA cashout failed", "error", err, "driver", driverID)
			c.JSON(http.StatusUnprocessableEntity, errResp("CASHOUT_FAILED", "Cashout could not be completed. Please try again.", ""))
		}
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{"message": "cashout initiated"}))
}

// ── Webhook endpoints ─────────────────────────────────────────────────────────

type webhookDeps struct {
	wallet    *service.WalletService
	providers map[string]webhookVerifier
}

type webhookVerifier interface {
	VerifyWebhookSignature(payload []byte, signature string) bool
	Name() string
}

func (h *WalletHandler) HandlePaystackWebhook(verifier webhookVerifier) gin.HandlerFunc {
	return h.handleWebhook(verifier, "x-paystack-signature", "charge.success")
}

func (h *WalletHandler) HandleFlutterwaveWebhook(verifier webhookVerifier) gin.HandlerFunc {
	return h.handleWebhook(verifier, "verif-hash", "charge.completed")
}

// HandleBridgeWebhook validates Bridge.xyz webhook events.
// Bridge sends the API key in the "Api-Key" header.
// Successful payment event: "payment.payment_processed".
func (h *WalletHandler) HandleBridgeWebhook(verifier webhookVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		sig := c.GetHeader("Api-Key")
		if !verifier.VerifyWebhookSignature(body, sig) {
			c.Status(http.StatusUnauthorized)
			return
		}

		// Bridge webhook shape:
		// { "event_type": "payment.payment_processed",
		//   "object_id": "<session_id>",
		//   "data": { "external_reference": "<our ref>", "state": "payment_processed" } }
		var payload struct {
			EventType string `json:"event_type"`
			ObjectID  string `json:"object_id"`
			Data      struct {
				ExternalReference string `json:"external_reference"`
				State             string `json:"state"`
			} `json:"data"`
		}
		if err := parseJSON(body, &payload); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		eventType := payload.EventType
		if eventType == "payment.payment_processed" || eventType == "payment.funds_received" {
			eventType = "charge.success"
		}

		wp := service.WebhookPayload{
			Provider:  verifier.Name(),
			EventID:   payload.ObjectID,
			EventType: eventType,
			Reference: payload.Data.ExternalReference,
			RawBody:   body,
		}

		if err := h.wallet.ProcessWebhook(c.Request.Context(), wp); err != nil {
			if errors.Is(err, service.ErrWebhookRetryable) {
				c.Status(http.StatusServiceUnavailable)
				return
			}
			c.Status(http.StatusOK)
			return
		}
		c.Status(http.StatusOK)
	}
}

// HandleMonnifyWebhook validates the monnify-signature header (HMAC-SHA512 of body
// keyed with the Monnify secret key) then routes to the shared webhook processor.
// Monnify event for a successful charge: "SUCCESSFUL_TRANSACTION".
// The payment reference lives at responseBody.paymentReference.
func (h *WalletHandler) HandleMonnifyWebhook(verifier webhookVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		sig := c.GetHeader("monnify-signature")
		if !verifier.VerifyWebhookSignature(body, sig) {
			c.Status(http.StatusUnauthorized)
			return
		}

		// Monnify webhook shape:
		// { "eventType": "SUCCESSFUL_TRANSACTION",
		//   "eventData": { "transactionReference": "MNFY|...",
		//                  "paymentReference": "merchant-ref",
		//                  "amountPaid": 100.0, ... } }
		var payload struct {
			EventType string `json:"eventType"`
			EventData struct {
				TransactionReference string `json:"transactionReference"`
				PaymentReference     string `json:"paymentReference"`
			} `json:"eventData"`
		}
		if err := parseJSON(body, &payload); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		// Map Monnify event type to the internal convention used by ProcessWebhook.
		// We use the transactionReference as the canonical reference for verify calls.
		eventType := payload.EventType
		if eventType == "SUCCESSFUL_TRANSACTION" {
			eventType = "charge.success"
		}

		wp := service.WebhookPayload{
			Provider:  verifier.Name(),
			EventID:   payload.EventData.TransactionReference,
			EventType: eventType,
			Reference: payload.EventData.TransactionReference,
			RawBody:   body,
		}

		if err := h.wallet.ProcessWebhook(c.Request.Context(), wp); err != nil {
			if errors.Is(err, service.ErrWebhookRetryable) {
				c.Status(http.StatusServiceUnavailable) // 503 — provider retries
				return
			}
			c.Status(http.StatusOK) // non-retryable, stop retries
			return
		}
		c.Status(http.StatusOK)
	}
}

func (h *WalletHandler) handleWebhook(verifier webhookVerifier, sigHeader, successEvent string) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		sig := c.GetHeader(sigHeader)
		if !verifier.VerifyWebhookSignature(body, sig) {
			c.Status(http.StatusUnauthorized)
			return
		}

		// Parse minimal fields needed for processing
		var payload struct {
			Event string `json:"event"`
			Data  struct {
				ID        interface{} `json:"id"`
				Reference string      `json:"reference"`
				TxRef     string      `json:"tx_ref"`
			} `json:"data"`
		}
		if err := parseJSON(body, &payload); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		ref := payload.Data.Reference
		if ref == "" {
			ref = payload.Data.TxRef
		}

		eventID := ""
		if id, ok := payload.Data.ID.(string); ok {
			eventID = id
		}

		wp := service.WebhookPayload{
			Provider:  verifier.Name(),
			EventID:   eventID,
			EventType: payload.Event,
			Reference: ref,
			RawBody:   body,
		}

		if err := h.wallet.ProcessWebhook(c.Request.Context(), wp); err != nil {
			if errors.Is(err, service.ErrWebhookRetryable) {
				c.Status(http.StatusServiceUnavailable)
				return
			}
			// Return 200 to prevent provider retries for non-retryable errors
			c.Status(http.StatusOK)
			return
		}

		c.Status(http.StatusOK)
	}
}

func parseJSON(b []byte, v interface{}) error {
	return json.Unmarshal(b, v)
}
