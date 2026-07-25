package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/service"
)

// MerchantHandler is the authenticated merchant's self-service surface:
// profile/KYC status, open/closed toggle, order queue + fulfillment
// transitions, and product catalog CRUD. All routes are RequireRole("merchant")
// and every write is scoped to the caller's own merchant.ID (see
// MerchantService.ResolveByUserID) — never trusts a merchant ID from the request.
type MerchantHandler struct {
	merchant *service.MerchantService
	orders   *service.OrderService
	catalog  *service.CatalogService
}

func NewMerchantHandler(merchant *service.MerchantService, orders *service.OrderService, catalog *service.CatalogService) *MerchantHandler {
	return &MerchantHandler{merchant: merchant, orders: orders, catalog: catalog}
}

func (h *MerchantHandler) userID(c *gin.Context) uuid.UUID {
	id, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	return id
}

// GetProfile — GET /merchant/profile
func (h *MerchantHandler) GetProfile(c *gin.Context) {
	merchant, kyc, err := h.merchant.GetProfile(c.Request.Context(), h.userID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Merchant profile not found", ""))
		return
	}
	resp := gin.H{
		"id":           merchant.ID,
		"businessName": merchant.BusinessName,
		"vertical":     merchant.Vertical,
		"status":       merchant.Status,
		"isOpen":       merchant.IsOpen,
		"rating":       merchant.Rating,
	}
	if kyc != nil {
		resp["kycStatus"] = kyc.Status
	} else {
		resp["kycStatus"] = "not_started"
	}
	c.JSON(http.StatusOK, successResp(resp))
}

// SetOpen — POST /merchant/status {isOpen: bool}
func (h *MerchantHandler) SetOpen(c *gin.Context) {
	var req struct {
		IsOpen bool `json:"isOpen"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	if err := h.merchant.SetOpen(c.Request.Context(), h.userID(c), req.IsOpen); err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"isOpen": req.IsOpen}))
}

// ListOrders — GET /merchant/orders?status=pending&cursor=...
func (h *MerchantHandler) ListOrders(c *gin.Context) {
	merchant, err := h.merchant.ResolveByUserID(c.Request.Context(), h.userID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Merchant profile not found", ""))
		return
	}
	var cursor *uuid.UUID
	if raw := c.Query("cursor"); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			cursor = &id
		}
	}
	orders, err := h.orders.ListForMerchant(c.Request.Context(), merchant.ID, c.Query("status"), cursor, 20)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"orders": orders}))
}

// TransitionOrder — POST /merchant/orders/:id/transition {to: "confirmed"|"preparing"|"ready_for_pickup"|"cancelled"}
func (h *MerchantHandler) TransitionOrder(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid order ID", "id"))
		return
	}
	var req struct {
		To   string  `json:"to" binding:"required"`
		Note *string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	merchant, err := h.merchant.ResolveByUserID(c.Request.Context(), h.userID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Merchant profile not found", ""))
		return
	}
	if err := h.orders.Transition(c.Request.Context(), orderID, merchant.ID, "merchant", model.OrderStatus(req.To), req.Note); err != nil {
		if err == service.ErrIllegalTransition {
			c.JSON(http.StatusConflict, errResp("VALIDATION_ERROR", err.Error(), ""))
			return
		}
		c.JSON(http.StatusForbidden, errResp("FORBIDDEN", "Access denied", ""))
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"message": "order updated"}))
}

// ListProducts — GET /merchant/products (includes unavailable items, unlike the public catalog)
func (h *MerchantHandler) ListProducts(c *gin.Context) {
	merchant, err := h.merchant.ResolveByUserID(c.Request.Context(), h.userID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Merchant profile not found", ""))
		return
	}
	products, err := h.catalog.ListProductsForMerchant(c.Request.Context(), merchant.ID)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"products": products}))
}

type productRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	PriceKobo   int64   `json:"priceKobo" binding:"required,min=1"`
	Category    string  `json:"category"`
	IsAvailable bool    `json:"isAvailable"`
}

// CreateProduct — POST /merchant/products
func (h *MerchantHandler) CreateProduct(c *gin.Context) {
	var req productRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	merchant, err := h.merchant.ResolveByUserID(c.Request.Context(), h.userID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Merchant profile not found", ""))
		return
	}
	product, err := h.catalog.CreateProduct(c.Request.Context(), merchant.ID, service.ProductInput{
		Name: req.Name, Description: req.Description, PriceKobo: req.PriceKobo,
		Category: req.Category, IsAvailable: req.IsAvailable,
	})
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, successResp(product))
}

// UpdateProduct — PUT /merchant/products/:id
func (h *MerchantHandler) UpdateProduct(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid product ID", "id"))
		return
	}
	var req productRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	merchant, err := h.merchant.ResolveByUserID(c.Request.Context(), h.userID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Merchant profile not found", ""))
		return
	}
	product, err := h.catalog.UpdateProduct(c.Request.Context(), merchant.ID, productID, service.ProductInput{
		Name: req.Name, Description: req.Description, PriceKobo: req.PriceKobo,
		Category: req.Category, IsAvailable: req.IsAvailable,
	})
	if err != nil {
		c.JSON(http.StatusForbidden, errResp("FORBIDDEN", "Access denied", ""))
		return
	}
	c.JSON(http.StatusOK, successResp(product))
}

// SetProductAvailability — POST /merchant/products/:id/availability {available: bool}
func (h *MerchantHandler) SetProductAvailability(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid product ID", "id"))
		return
	}
	var req struct {
		Available bool `json:"available"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	merchant, err := h.merchant.ResolveByUserID(c.Request.Context(), h.userID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Merchant profile not found", ""))
		return
	}
	if err := h.catalog.SetProductAvailability(c.Request.Context(), merchant.ID, productID, req.Available); err != nil {
		c.JSON(http.StatusForbidden, errResp("FORBIDDEN", "Access denied", ""))
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"available": req.Available}))
}
