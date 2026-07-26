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
	wallet   *service.WalletService
}

func NewMerchantHandler(merchant *service.MerchantService, orders *service.OrderService, catalog *service.CatalogService, wallet *service.WalletService) *MerchantHandler {
	return &MerchantHandler{merchant: merchant, orders: orders, catalog: catalog, wallet: wallet}
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

// GetBankAccount — GET /merchant/bank-account
func (h *MerchantHandler) GetBankAccount(c *gin.Context) {
	acct, err := h.merchant.GetBankAccount(c.Request.Context(), h.userID(c))
	if err != nil {
		internalError(c, err)
		return
	}
	if acct == nil {
		c.JSON(http.StatusOK, successResp(nil))
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{
		"bankCode":      acct.BankCode,
		"bankName":      acct.BankName,
		"accountNumber": acct.AccountNumber,
		"accountName":   acct.AccountName,
		"isVerified":    acct.IsVerified,
	}))
}

// SaveBankAccount — POST /merchant/bank-account
// The frontend must resolve the account name via Paystack's /bank/resolve
// endpoint before calling this. The resolved name is sent in the request
// body and stored as-is — it is the provider's authoritative answer.
func (h *MerchantHandler) SaveBankAccount(c *gin.Context) {
	var req struct {
		BankCode      string `json:"bankCode"      binding:"required"`
		BankName      string `json:"bankName"      binding:"required"`
		AccountNumber string `json:"accountNumber" binding:"required"`
		AccountName   string `json:"accountName"   binding:"required"` // provider-resolved
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	acct, err := h.merchant.SaveBankAccount(c.Request.Context(), h.userID(c),
		req.BankCode, req.BankName, req.AccountNumber, req.AccountName)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{
		"bankCode":      acct.BankCode,
		"bankName":      acct.BankName,
		"accountNumber": acct.AccountNumber,
		"accountName":   acct.AccountName,
	}))
}

// Withdraw — POST /merchant/withdraw
// Debits the merchant wallet and enqueues a bank transfer via the asynq worker.
func (h *MerchantHandler) Withdraw(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Idempotency-Key header required", "Idempotency-Key"))
		return
	}
	var req struct {
		AmountKobo     int64  `json:"amountKobo"     binding:"required,min=100000"` // min ₦1,000
		PIN            string `json:"pin"            binding:"required,len=4"`
		WithdrawalType string `json:"withdrawalType"` // "instant" | "standard" (default)
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
	if err := h.wallet.MerchantWithdraw(c.Request.Context(), merchant.ID, h.userID(c),
		req.AmountKobo, req.PIN, idempotencyKey, req.WithdrawalType); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"message": "withdrawal initiated"}))
}

// ListPrescriptions — GET /merchant/prescriptions?status=pending
func (h *MerchantHandler) ListPrescriptions(c *gin.Context) {
	merchant, err := h.merchant.ResolveByUserID(c.Request.Context(), h.userID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Merchant profile not found", ""))
		return
	}
	prescriptions, err := h.catalog.ListPrescriptionsForMerchant(c.Request.Context(), merchant.ID, c.Query("status"))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"prescriptions": prescriptions}))
}

// ReviewPrescription — POST /merchant/prescriptions/:id/review {approve: bool, note?: string}
func (h *MerchantHandler) ReviewPrescription(c *gin.Context) {
	prescriptionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid prescription ID", "id"))
		return
	}
	var req struct {
		Approve bool    `json:"approve"`
		Note    *string `json:"note"`
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
	p, err := h.catalog.ReviewPrescription(c.Request.Context(), h.userID(c), merchant.ID, prescriptionID, req.Approve, req.Note)
	if err != nil {
		if err == service.ErrPrescriptionUsed {
			c.JSON(http.StatusConflict, errResp("VALIDATION_ERROR", err.Error(), ""))
			return
		}
		c.JSON(http.StatusForbidden, errResp("FORBIDDEN", "Access denied", ""))
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"id": p.ID, "status": p.Status}))
}
