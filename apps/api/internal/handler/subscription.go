package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/service"
)

type SubscriptionHandler struct {
	svc *service.SubscriptionService
}

func NewSubscriptionHandler(svc *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{svc: svc}
}

func (h *SubscriptionHandler) Create(c *gin.Context) {
	var req struct {
		MerchantID    string `json:"merchantId" binding:"required"`
		AddressID     string `json:"addressId" binding:"required"`
		Vertical      string `json:"vertical" binding:"required"`
		Cadence       string `json:"cadence" binding:"required,oneof=weekly biweekly monthly"`
		PaymentMethod string `json:"paymentMethod" binding:"required,oneof=wallet"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	customerID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	merchantID, _ := uuid.Parse(req.MerchantID)
	addressID, _ := uuid.Parse(req.AddressID)

	sub, err := h.svc.Create(c.Request.Context(), customerID, merchantID, addressID, req.Vertical, req.Cadence, req.PaymentMethod)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, successResp(gin.H{"subscription": sub}))
}

func (h *SubscriptionHandler) Pause(c *gin.Context) {
	subID, _ := uuid.Parse(c.Param("id"))
	customerID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.svc.Pause(c.Request.Context(), customerID, subID); err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"message": "subscription paused"}))
}

func (h *SubscriptionHandler) Cancel(c *gin.Context) {
	subID, _ := uuid.Parse(c.Param("id"))
	customerID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.svc.Cancel(c.Request.Context(), customerID, subID); err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"message": "subscription cancelled"}))
}
