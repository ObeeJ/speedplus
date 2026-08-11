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
		MerchantID     string  `json:"merchantId"     binding:"required"`
		AddressID      string  `json:"addressId"      binding:"required"`
		Vertical       string  `json:"vertical"       binding:"required"`
		Cadence        string  `json:"cadence"        binding:"required,oneof=weekly biweekly monthly"`
		PaymentMethod  string  `json:"paymentMethod"  binding:"required,oneof=wallet"`
		GasMode        *string `json:"gasMode"`        // swap|refill|new_cylinder
		CylinderSpecID *string `json:"cylinderSpecId"` // UUID of the cylinder spec
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	customerID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	merchantID, _ := uuid.Parse(req.MerchantID)
	addressID, _ := uuid.Parse(req.AddressID)

	var cylinderSpecID *uuid.UUID
	if req.CylinderSpecID != nil {
		id, err := uuid.Parse(*req.CylinderSpecID)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid cylinderSpecId", "cylinderSpecId"))
			return
		}
		cylinderSpecID = &id
	}

	sub, err := h.svc.Create(c.Request.Context(), customerID, merchantID, addressID,
		req.Vertical, req.Cadence, req.PaymentMethod, req.GasMode, cylinderSpecID)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, successResp(gin.H{"subscription": sub}))
}

// List — GET /subscriptions. Returns only the caller's own subscriptions; the
// customer id comes from the JWT, never from the request.
func (h *SubscriptionHandler) List(c *gin.Context) {
	customerID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	subs, err := h.svc.ListMine(c.Request.Context(), customerID)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"subscriptions": subs}))
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

// GetLPGPrice — GET /gas/price-index?region=Lagos
func (h *SubscriptionHandler) GetLPGPrice(c *gin.Context) {
	region := c.DefaultQuery("region", "Lagos")
	entry, err := h.svc.GetLiveLPGPrice(c.Request.Context(), region)
	if err != nil {
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "No LPG price found for region", "region"))
		return
	}
	c.JSON(http.StatusOK, successResp(entry))
}

// RecordLPGPrice — POST /admin/gas/price-index (admin only)
func (h *SubscriptionHandler) RecordLPGPrice(c *gin.Context) {
	var req struct {
		Region         string `json:"region"         binding:"required"`
		PricePerKgKobo int64  `json:"pricePerKgKobo" binding:"required,min=1"`
		Source         string `json:"source"         binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	adminID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	row, suggestion, err := h.svc.RecordLPGPrice(c.Request.Context(), adminID, req.Region, req.PricePerKgKobo, req.Source)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, successResp(gin.H{"entry": row, "suggestion": suggestion}))
}
