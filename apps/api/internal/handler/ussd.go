package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/service"
)

type USSDHandler struct {
	ussd *service.USSDService
}

func NewUSSDHandler(ussd *service.USSDService) *USSDHandler {
	return &USSDHandler{ussd: ussd}
}

func (h *USSDHandler) Banks(c *gin.Context) {
	c.JSON(http.StatusOK, successResp(gin.H{"banks": h.ussd.SupportedBanks()}))
}

func (h *USSDHandler) Initiate(c *gin.Context) {
	var req struct {
		BankCode   string `json:"bankCode" binding:"required"`
		AmountKobo int64  `json:"amountKobo" binding:"required,min=100"`
		Email      string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	intent, err := h.ussd.InitiateUSSDFund(c.Request.Context(), userID, req.BankCode, req.AmountKobo, req.Email)
	if err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{
		"id":          intent.ID,
		"ussdCode":    intent.USSDCode,
		"bankName":    intent.BankName,
		"amountKobo":  intent.AmountKobo,
		"expiresAt":   intent.ExpiresAt,
		"status":      intent.Status,
	}))
}

func (h *USSDHandler) Status(c *gin.Context) {
	intentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid intent ID", "id"))
		return
	}

	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	intent, err := h.ussd.GetIntent(c.Request.Context(), userID, intentID)
	if err != nil {
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Intent not found", ""))
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{
		"id":         intent.ID,
		"status":     intent.Status,
		"ussdCode":   intent.USSDCode,
		"amountKobo": intent.AmountKobo,
		"paidAt":     intent.PaidAt,
		"expiresAt":  intent.ExpiresAt,
	}))
}
