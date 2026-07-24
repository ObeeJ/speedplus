package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/service"
)

type GiftCardHandler struct {
	svc *service.GiftCardService
}

func NewGiftCardHandler(svc *service.GiftCardService) *GiftCardHandler {
	return &GiftCardHandler{svc: svc}
}

func (h *GiftCardHandler) Issue(c *gin.Context) {
	var req struct {
		AmountKobo int64 `json:"amountKobo" binding:"required,min=100"`
		ExpiryDays int   `json:"expiryDays"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	issuerID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	code, gc, err := h.svc.Issue(c.Request.Context(), issuerID, req.AmountKobo, req.ExpiryDays)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, successResp(gin.H{
		"code":       code,
		"amountKobo": gc.AmountKobo,
		"expiresAt":  gc.ExpiresAt,
	}))
}

func (h *GiftCardHandler) Redeem(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.svc.Redeem(c.Request.Context(), userID, req.Code); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errResp("REDEEM_FAILED", err.Error(), "code"))
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"message": "gift card redeemed"}))
}
