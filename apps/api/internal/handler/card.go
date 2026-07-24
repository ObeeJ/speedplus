package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/service"
	"gorm.io/gorm"
)

type CardHandler struct {
	paycode *service.PaycodeService
	auth    *service.AuthService
	db      *gorm.DB
}

func NewCardHandler(paycode *service.PaycodeService, auth *service.AuthService, db *gorm.DB) *CardHandler {
	return &CardHandler{paycode: paycode, auth: auth, db: db}
}

// ScanCard is called by the rider's app when they scan the customer's SpeedPlus card.
//
// Flow:
//  1. Rider submits card QR payload
//  2. Backend verifies HMAC signature → extracts customerID
//  3. Backend finds the active in_transit order for that customer
//  4. Rider hands phone to customer
//  5. Customer enters their 4-digit wallet PIN
//  6. PIN verified → escrow released → order marked delivered
//
// The PIN step is the customer's explicit consent at the door.
// It replaces the order-QR paycode flow for offline/card scenarios.
func (h *CardHandler) ScanCard(c *gin.Context) {
	var req struct {
		CardPayload string `json:"cardPayload" binding:"required"`
		PIN         string `json:"pin"         binding:"required,len=4"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	driverID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))

	if err := h.paycode.ConfirmByCard(c.Request.Context(), req.CardPayload, driverID, req.PIN); err != nil {
		switch err {
		case service.ErrPaycodeInvalid:
			c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid card", "cardPayload"))
		case service.ErrNotAssigned:
			c.JSON(http.StatusForbidden, errResp("FORBIDDEN", "You are not the assigned driver for this order", ""))
		case service.ErrAlreadyConfirmed:
			c.JSON(http.StatusConflict, errResp("CONFLICT", "Order already confirmed", ""))
		default:
			internalError(c, err)
		}
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{"message": "delivery confirmed"}))
}

// GetCard returns the user's digital SpeedPlus card payload.
// The frontend renders this as a QR code on the wallet tab.
// Can be screenshotted for offline use.
func (h *CardHandler) GetCard(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))

	var card model.UserCard
	if err := h.db.WithContext(c.Request.Context()).
		Where("user_id = ? AND is_active = true", userID).
		First(&card).Error; err != nil {
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Card not yet generated. Please try again shortly.", ""))
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{
		"payload":   card.Payload,
		"createdAt": card.CreatedAt,
	}))
}

// GetVirtualAccount returns the user's dedicated virtual account details.
// Shown on the onboarding screen and wallet tab.
func (h *CardHandler) GetVirtualAccount(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))

	var va model.VirtualAccount
	if err := h.db.WithContext(c.Request.Context()).
		Where("user_id = ? AND is_active = true", userID).
		First(&va).Error; err != nil {
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Virtual account not yet created. Please try again shortly.", ""))
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{
		"accountNumber": va.AccountNumber,
		"bankName":      va.BankName,
		"bankCode":      va.BankCode,
		"provider":      va.Provider,
	}))
}

// GetTrustTier returns the user's current trust tier and progress.
// Powers the gamified progress screen on the frontend.
func (h *CardHandler) GetTrustTier(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))

	var tier model.UserTrustTier
	if err := h.db.WithContext(c.Request.Context()).
		Where("user_id = ?", userID).
		First(&tier).Error; err != nil {
		// No row yet — return Tier 0 defaults
		c.JSON(http.StatusOK, successResp(tierView(&model.UserTrustTier{UserID: userID})))
		return
	}

	c.JSON(http.StatusOK, successResp(tierView(&tier)))
}

func tierView(t *model.UserTrustTier) gin.H {
	ordersToNext := 0
	tierName := "New Member"
	nextTierName := "Regular"
	canPayOnArrival := false

	switch t.Tier {
	case model.TierNew:
		ordersToNext = 3 - t.CompletedOrders
		if ordersToNext < 0 {
			ordersToNext = 0
		}
	case model.TierRegular:
		tierName = "Regular"
		nextTierName = ""
		canPayOnArrival = true
		ordersToNext = 0
	}

	return gin.H{
		"tier":            t.Tier,
		"tierName":        tierName,
		"completedOrders": t.CompletedOrders,
		"ordersToNext":    ordersToNext,
		"nextTierName":    nextTierName,
		"canPayOnArrival": canPayOnArrival,
		"frozen":          t.Frozen,
	}
}
