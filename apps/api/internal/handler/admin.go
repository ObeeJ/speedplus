package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/dto"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/service"
)

type AdminHandler struct {
	admin      *service.AdminService
	ledger     *service.LedgerService
	feeConfigs *service.FeeConfigService
}

func NewAdminHandler(admin *service.AdminService, ledger *service.LedgerService, feeConfigs *service.FeeConfigService) *AdminHandler {
	return &AdminHandler{admin: admin, ledger: ledger, feeConfigs: feeConfigs}
}

// ── Merchants ─────────────────────────────────────────────────────────────────

func (h *AdminHandler) ListMerchants(c *gin.Context) {
	status := c.Query("status") // optional filter: pending|active|suspended
	page := queryInt(c, "page", 0)
	merchants, err := h.admin.ListMerchants(c.Request.Context(), status, page, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"merchants": merchants}))
}

func (h *AdminHandler) SetMerchantStatus(c *gin.Context) {
	merchantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid merchant ID", "id"))
		return
	}
	var req struct {
		Status string `json:"status" binding:"required,oneof=active suspended"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", err.Error(), ""))
		return
	}
	adminID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.admin.SetMerchantStatus(c.Request.Context(), merchantID, adminID, req.Status, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.MessageResponse{Message: "merchant status updated"}))
}

// ── Drivers ───────────────────────────────────────────────────────────────────

func (h *AdminHandler) ListDrivers(c *gin.Context) {
	status := c.Query("status") // optional filter: pending|under_review|approved|suspended
	page := queryInt(c, "page", 0)
	drivers, err := h.admin.ListDrivers(c.Request.Context(), status, page, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"drivers": drivers}))
}

func (h *AdminHandler) SetDriverStatus(c *gin.Context) {
	driverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid driver ID", "id"))
		return
	}
	var req struct {
		Status string `json:"status" binding:"required,oneof=approved suspended under_review"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", err.Error(), ""))
		return
	}
	adminID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.admin.SetDriverStatus(c.Request.Context(), driverID, adminID, req.Status, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.MessageResponse{Message: "driver status updated"}))
}

// ── Orders ────────────────────────────────────────────────────────────────────

func (h *AdminHandler) SearchOrders(c *gin.Context) {
	q := c.Query("q")       // order ID prefix, customer phone, or status
	status := c.Query("status")
	page := queryInt(c, "page", 0)
	orders, err := h.admin.SearchOrders(c.Request.Context(), q, status, page, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"orders": orders}))
}

func (h *AdminHandler) GetOrderDetail(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid order ID", "id"))
		return
	}
	order, err := h.admin.GetOrderDetail(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Fail("NOT_FOUND", "Order not found", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(order))
}

// ── Disputes (escrow freeze / release) ───────────────────────────────────────

// FreezeEscrow locks an escrow hold pending dispute investigation.
// Does NOT move money — only transitions EscrowHeld → EscrowFrozen.
func (h *AdminHandler) FreezeEscrow(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("orderId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid order ID", "orderId"))
		return
	}
	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", err.Error(), "reason"))
		return
	}
	adminID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.admin.FreezeEscrow(c.Request.Context(), orderID, adminID, req.Reason); err != nil {
		c.JSON(http.StatusConflict, dto.Fail("VALIDATION_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.MessageResponse{Message: "escrow frozen"}))
}

// ReleaseEscrow resolves a dispute by routing funds through the ledger journal.
// Requires a reason and explicit recipient (customer=refund, merchant=settle).
// Dual-admin approval: first call records ApprovalOne; second call (different admin) executes.
func (h *AdminHandler) ReleaseEscrow(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("orderId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid order ID", "orderId"))
		return
	}
	var req struct {
		Recipient string `json:"recipient" binding:"required,oneof=customer merchant"`
		Reason    string `json:"reason"    binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", err.Error(), ""))
		return
	}
	adminID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	result, err := h.admin.ReleaseEscrow(c.Request.Context(), orderID, adminID, req.Recipient, req.Reason)
	if err != nil {
		c.JSON(http.StatusConflict, dto.Fail("VALIDATION_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"message": result}))
}

// ── Cancellation rules ────────────────────────────────────────────────────────

func (h *AdminHandler) ListCancellationRules(c *gin.Context) {
	rules, err := h.admin.ListCancellationRules(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"rules": rules}))
}

func (h *AdminHandler) UpsertCancellationRule(c *gin.Context) {
	var req struct {
		Vertical            string  `json:"vertical"            binding:"required"`
		OrderStatusAtCancel string  `json:"orderStatusAtCancel" binding:"required"`
		MerchantCompKobo    int64   `json:"merchantCompKobo"`
		MerchantCompPct     float64 `json:"merchantCompPct"`
		RiderCompPctOfDelivery float64 `json:"riderCompPctOfDelivery"`
		FullRefund          bool    `json:"fullRefund"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", err.Error(), ""))
		return
	}
	rule, err := h.admin.UpsertCancellationRule(c.Request.Context(), service.CancellationRuleInput{
		Vertical:               req.Vertical,
		OrderStatusAtCancel:    req.OrderStatusAtCancel,
		MerchantCompKobo:       req.MerchantCompKobo,
		MerchantCompPct:        req.MerchantCompPct,
		RiderCompPctOfDelivery: req.RiderCompPctOfDelivery,
		FullRefund:             req.FullRefund,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(rule))
}

func (h *AdminHandler) DeleteCancellationRule(c *gin.Context) {
	ruleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid rule ID", "id"))
		return
	}
	if err := h.admin.DeleteCancellationRule(c.Request.Context(), ruleID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.MessageResponse{Message: "rule deleted"}))
}

// ── Fee configs (pricing engine) ──────────────────────────────────────────────

func (h *AdminHandler) ListFeeConfigs(c *gin.Context) {
	configs, err := h.feeConfigs.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"configs": configs}))
}

func (h *AdminHandler) UpsertFeeConfig(c *gin.Context) {
	var req struct {
		Vertical         string  `json:"vertical"         binding:"required,oneof=food grocery pharmacy gas package"`
		BaseFeeKobo      int64   `json:"baseFeeKobo"      binding:"min=0"`
		PerKmKobo        int64   `json:"perKmKobo"        binding:"min=0"`
		PerKgKobo        int64   `json:"perKgKobo"        binding:"min=0"`
		PerStopKobo      int64   `json:"perStopKobo"      binding:"min=0"`
		ServicePct       float64 `json:"servicePct"       binding:"min=0,max=0.5"`
		MerchantTakeRate float64 `json:"merchantTakeRate" binding:"required,gt=0.5,lte=1"`
		DriverTakeRate   float64 `json:"driverTakeRate"   binding:"required,gte=0.5,lte=1"`
		PlatformTakeRate float64 `json:"platformTakeRate" binding:"min=0"`
		FuelPriceRefKobo int64   `json:"fuelPriceRefKobo" binding:"min=0"`
		Reason           string  `json:"reason"           binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", err.Error(), ""))
		return
	}
	adminID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	row, suggestion, err := h.feeConfigs.Upsert(c.Request.Context(), adminID, service.FeeConfigInput{
		Vertical:         req.Vertical,
		BaseFeeKobo:      req.BaseFeeKobo,
		PerKmKobo:        req.PerKmKobo,
		PerKgKobo:        req.PerKgKobo,
		PerStopKobo:      req.PerStopKobo,
		ServicePct:       req.ServicePct,
		MerchantTakeRate: req.MerchantTakeRate,
		DriverTakeRate:   req.DriverTakeRate,
		PlatformTakeRate: req.PlatformTakeRate,
		FuelPriceRefKobo: req.FuelPriceRefKobo,
		Reason:           req.Reason,
	})
	if err != nil {
		// Service-level validation (take-rate sum, unknown vertical) is a client error.
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"config": row, "fuelSuggestion": suggestion}))
}

// ── Ledger viewer ─────────────────────────────────────────────────────────────

func (h *AdminHandler) GetLedger(c *gin.Context) {
	userID, err := uuid.Parse(c.Query("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "userId query param required", "userId"))
		return
	}
	var cursor *uuid.UUID
	if raw := c.Query("cursor"); raw != "" {
		id, err := uuid.Parse(raw)
		if err == nil {
			cursor = &id
		}
	}
	entries, err := h.ledger.GetTransactions(c.Request.Context(), userID, cursor, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"entries": entries}))
}

// queryInt reads an integer query param with a fallback default.
func queryInt(c *gin.Context, key string, def int) int {
	var v int
	if _, err := fmt.Sscanf(c.Query(key), "%d", &v); err != nil {
		return def
	}
	return v
}
