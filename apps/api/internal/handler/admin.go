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
	admin            *service.AdminService
	ledger           *service.LedgerService
	feeConfigs       *service.FeeConfigService
	weatherSurcharge *service.WeatherSurchargeService
	velocity         *service.VelocityService
}

func NewAdminHandler(admin *service.AdminService, ledger *service.LedgerService, feeConfigs *service.FeeConfigService) *AdminHandler {
	return &AdminHandler{admin: admin, ledger: ledger, feeConfigs: feeConfigs}
}

func (h *AdminHandler) InjectWeatherSurcharge(ws *service.WeatherSurchargeService) {
	h.weatherSurcharge = ws
}

func (h *AdminHandler) InjectVelocity(v *service.VelocityService) {
	h.velocity = v
}

// ── Merchants ─────────────────────────────────────────────────────────────────

func (h *AdminHandler) ListMerchants(c *gin.Context) {
	status := c.Query("status")
	cursor := parseCursor(c)
	merchants, err := h.admin.ListMerchants(c.Request.Context(), status, cursor, 20)
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
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
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
	status := c.Query("status")
	cursor := parseCursor(c)
	drivers, err := h.admin.ListDrivers(c.Request.Context(), status, cursor, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"drivers": drivers}))
}

// ── Admin console listings ────────────────────────────────────────────────────
//
// These four back admin pages that shipped in the client before the routes
// existed and 404'd on load. All are read-only and sit behind
// RequireRole("admin") plus the admin rate limiter.

func (h *AdminHandler) ListUsers(c *gin.Context) {
	users, err := h.admin.ListUsers(c.Request.Context(), c.Query("role"), c.Query("q"), parseCursor(c), 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"users": users}))
}

func (h *AdminHandler) ListRuns(c *gin.Context) {
	runs, err := h.admin.ListRuns(c.Request.Context(), c.Query("status"), parseCursor(c), 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"runs": runs}))
}

func (h *AdminHandler) ListSubscriptions(c *gin.Context) {
	subs, err := h.admin.ListSubscriptions(c.Request.Context(), c.Query("status"), parseCursor(c), 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"subscriptions": subs}))
}

func (h *AdminHandler) ListPrescriptions(c *gin.Context) {
	rx, err := h.admin.ListPrescriptions(c.Request.Context(), c.Query("status"), parseCursor(c), 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"prescriptions": rx}))
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
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
		return
	}
	adminID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.admin.SetDriverStatus(c.Request.Context(), driverID, adminID, req.Status, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.MessageResponse{Message: "driver status updated"}))
}

// ── Users ─────────────────────────────────────────────────────────────────────

func (h *AdminHandler) SetUserActive(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid user ID", "id"))
		return
	}
	var req struct {
		Active bool   `json:"active"`
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
		return
	}
	adminID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.admin.SetUserActive(c.Request.Context(), userID, adminID, req.Active, req.Reason); err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, dto.Fail("NOT_FOUND", "User not found", "id"))
			return
		}
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.MessageResponse{Message: "user status updated"}))
}

// ── Orders ────────────────────────────────────────────────────────────────────

func (h *AdminHandler) SearchOrders(c *gin.Context) {
	q := c.Query("q")
	status := c.Query("status")
	cursor := parseCursor(c)
	orders, err := h.admin.SearchOrders(c.Request.Context(), q, status, cursor, 20)
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
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid request body", "reason"))
		return
	}
	adminID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.admin.FreezeEscrow(c.Request.Context(), orderID, adminID, req.Reason); err != nil {
		c.JSON(http.StatusConflict, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
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
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
		return
	}
	adminID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	result, err := h.admin.ReleaseEscrow(c.Request.Context(), orderID, adminID, req.Recipient, req.Reason)
	if err != nil {
		c.JSON(http.StatusConflict, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
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
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
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
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
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
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
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

// GetMetrics — GET /admin/metrics
// Returns today's operational health numbers.
func (h *AdminHandler) GetMetrics(c *gin.Context) {
	m, err := h.admin.GetMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(m))
}

// queryInt reads an integer query param with a fallback default.
func queryInt(c *gin.Context, key string, def int) int {
	var v int
	if _, err := fmt.Sscanf(c.Query(key), "%d", &v); err != nil {
		return def
	}
	return v
}

func parseCursor(c *gin.Context) *uuid.UUID {
	if raw := c.Query("cursor"); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			return &id
		}
	}
	return nil
}

// ── Gas: fill_status ──────────────────────────────────────────────────────────

func (h *AdminHandler) ListGasMerchants(c *gin.Context) {
	cursor := parseCursor(c)
	merchants, err := h.admin.ListGasMerchants(c.Request.Context(), c.Query("fillStatus"), cursor, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"merchants": merchants}))
}

func (h *AdminHandler) SetMerchantFillStatus(c *gin.Context) {
	merchantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid merchant ID", "id"))
		return
	}
	var req struct {
		Status string `json:"status" binding:"required,oneof=good warned probation delisted"`
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
		return
	}
	adminID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.admin.SetMerchantFillStatus(c.Request.Context(), merchantID, adminID, req.Status, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.MessageResponse{Message: "fill status updated"}))
}

// ── Gas: zones / launch_status ────────────────────────────────────────────────

func (h *AdminHandler) ListZones(c *gin.Context) {
	cursor := parseCursor(c)
	zones, err := h.admin.ListZones(c.Request.Context(), c.Query("launchStatus"), cursor, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"zones": zones}))
}

func (h *AdminHandler) SetZoneLaunchStatus(c *gin.Context) {
	zoneID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid zone ID", "id"))
		return
	}
	var req struct {
		Status string `json:"status" binding:"required,oneof=piloting live paused"`
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
		return
	}
	adminID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.admin.SetZoneLaunchStatus(c.Request.Context(), zoneID, adminID, req.Status, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.MessageResponse{Message: "launch status updated"}))
}

// ── Weather surcharge settings ────────────────────────────────────────────────

// GetWeatherSurcharge — GET /admin/settings/weather
func (h *AdminHandler) GetWeatherSurcharge(c *gin.Context) {
	if h.weatherSurcharge == nil {
		c.JSON(http.StatusOK, dto.OK(gin.H{"enabled": false, "amountKobo": 20000}))
		return
	}
	enabled, amountKobo := h.weatherSurcharge.Get(c.Request.Context())
	c.JSON(http.StatusOK, dto.OK(gin.H{"enabled": enabled, "amountKobo": amountKobo}))
}

// SetWeatherSurcharge — PUT /admin/settings/weather
func (h *AdminHandler) SetWeatherSurcharge(c *gin.Context) {
	if h.weatherSurcharge == nil {
		c.JSON(http.StatusServiceUnavailable, dto.Fail("UNAVAILABLE", "Weather surcharge service not configured", ""))
		return
	}
	var req struct {
		Enabled    bool   `json:"enabled"`
		AmountKobo int64  `json:"amountKobo" binding:"min=0"`
		Reason     string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
		return
	}
	adminID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.weatherSurcharge.Set(c.Request.Context(), adminID, req.Enabled, req.AmountKobo, req.Reason); err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"enabled": req.Enabled, "amountKobo": req.AmountKobo}))
}

// ── Velocity limits (AML) ─────────────────────────────────────────────────────

// ListVelocityLimits — GET /admin/velocity/limits
func (h *AdminHandler) ListVelocityLimits(c *gin.Context) {
	if h.velocity == nil {
		c.JSON(http.StatusServiceUnavailable, dto.Fail("UNAVAILABLE", "Velocity service not configured", ""))
		return
	}
	limits, err := h.velocity.ListLimits(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"limits": limits}))
}

// UpsertVelocityLimit — PUT /admin/velocity/limits
func (h *AdminHandler) UpsertVelocityLimit(c *gin.Context) {
	if h.velocity == nil {
		c.JSON(http.StatusServiceUnavailable, dto.Fail("UNAVAILABLE", "Velocity service not configured", ""))
		return
	}
	var req struct {
		Tier        int    `json:"tier"        binding:"min=0,max=1"`
		Operation   string `json:"operation"   binding:"required,oneof=transfer cashout withdraw fund gift_card"`
		PerTxnKobo  int64  `json:"perTxnKobo"  binding:"required,min=1"`
		DailyKobo   int64  `json:"dailyKobo"   binding:"required,min=1"`
		MonthlyKobo int64  `json:"monthlyKobo" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
		return
	}
	limit, err := h.velocity.UpsertLimit(c.Request.Context(), req.Tier, req.Operation, req.PerTxnKobo, req.DailyKobo, req.MonthlyKobo)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(limit))
}

// ListSuspiciousActivity — GET /admin/velocity/suspicious?cursor=...
func (h *AdminHandler) ListSuspiciousActivity(c *gin.Context) {
	if h.velocity == nil {
		c.JSON(http.StatusServiceUnavailable, dto.Fail("UNAVAILABLE", "Velocity service not configured", ""))
		return
	}
	var cursor *uuid.UUID
	if raw := c.Query("cursor"); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			cursor = &id
		}
	}
	rows, err := h.velocity.ListSuspiciousActivity(c.Request.Context(), cursor, 50)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"flags": rows}))
}
