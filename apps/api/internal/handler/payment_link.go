package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/service"
)

type PaymentLinkHandler struct {
	links *service.PaymentLinkService
}

func NewPaymentLinkHandler(links *service.PaymentLinkService) *PaymentLinkHandler {
	return &PaymentLinkHandler{links: links}
}

// Create — authenticated user creates a payment link.
// POST /api/v1/payment-links
func (h *PaymentLinkHandler) Create(c *gin.Context) {
	var req struct {
		AmountKobo int64   `json:"amountKobo" binding:"required,min=100"`
		Note       *string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	creatorID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	link, err := h.links.Create(c.Request.Context(), creatorID, req.AmountKobo, req.Note)
	if err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, successResp(gin.H{
		"slug":       link.Slug,
		"url":        "https://speedplus.app/pay/" + link.Slug,
		"amountKobo": link.AmountKobo,
		"note":       link.Note,
		"expiresAt":  link.ExpiresAt,
	}))
}

// GetBySlug — public endpoint, no auth required.
// Returns link details so the payment page can render amount + note.
// GET /pay/:slug  (public)
func (h *PaymentLinkHandler) GetBySlug(c *gin.Context) {
	slug := c.Param("slug")
	link, err := h.links.Get(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", err.Error(), ""))
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{
		"amountKobo": link.AmountKobo,
		"note":       link.Note,
		"expiresAt":  link.ExpiresAt,
	}))
}

// PayByWallet — authenticated SpeedPlus user pays from their wallet.
// POST /api/v1/payment-links/:slug/pay
func (h *PaymentLinkHandler) PayByWallet(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Idempotency-Key required", "Idempotency-Key"))
		return
	}

	slug := c.Param("slug")
	payerID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))

	if err := h.links.PayByWallet(c.Request.Context(), slug, payerID, idempotencyKey); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", err.Error(), ""))
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{"message": "payment successful"}))
}

// InitiateGuestPayment — no auth, guest pays by card via Paystack.
// POST /pay/:slug/guest
//
// The standard Idempotency middleware cannot be used here: it scopes its Redis
// key by CtxUserID, which is empty for an unauthenticated caller, so every
// guest would share one dedup namespace and could see each other's payment
// responses. Idempotency is enforced in the service instead, against the
// idempotency_keys table (the same mechanism Transfer/Cashout use).
func (h *PaymentLinkHandler) InitiateGuestPayment(c *gin.Context) {
	slug := c.Param("slug")

	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Idempotency-Key header is required", "Idempotency-Key"))
		return
	}

	var req struct {
		Email       string `json:"email"       binding:"required,email"`
		CallbackURL string `json:"callbackUrl" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	resp, err := h.links.InitiateGuestPayment(c.Request.Context(), slug, req.Email, req.CallbackURL, idempotencyKey)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", err.Error(), ""))
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{
		"authorizationUrl": resp.AuthorizationURL,
		"reference":        resp.Reference,
	}))
}
