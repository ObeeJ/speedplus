package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/service"
)

type PaycodeHandler struct {
	paycodes *service.PaycodeService
}

func NewPaycodeHandler(paycodes *service.PaycodeService) *PaycodeHandler {
	return &PaycodeHandler{paycodes: paycodes}
}

// Generate — called by customer app to produce the QR payload for their order.
// POST /paycodes/generate
func (h *PaycodeHandler) Generate(c *gin.Context) {
	var req struct {
		OrderID string `json:"orderId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid order ID", "orderId"))
		return
	}

	callerID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	pc, err := h.paycodes.Generate(c.Request.Context(), orderID, callerID)
	if err != nil {
		switch err {
		case service.ErrNotOwner:
			c.JSON(http.StatusForbidden, errResp("FORBIDDEN", "You can only generate a code for your own order", ""))
		case service.ErrPaycodeInvalid:
			c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Order not found", "orderId"))
		default:
			internalError(c, err)
		}
		return
	}

	c.JSON(http.StatusCreated, successResp(gin.H{
		"id":        pc.ID,
		"payload":   pc.Payload,
		"expiresAt": pc.ExpiresAt,
	}))
}

// Resolve — driver scans QR; returns order summary before confirming.
// POST /paycodes/resolve
func (h *PaycodeHandler) Resolve(c *gin.Context) {
	var req struct {
		Payload string `json:"payload" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	driverID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	order, pc, err := h.paycodes.Resolve(c.Request.Context(), req.Payload, driverID)
	if err != nil {
		switch err {
		case service.ErrPaycodeExpired:
			c.JSON(http.StatusGone, errResp("VALIDATION_ERROR", "Paycode expired", "payload"))
		case service.ErrNotAssigned:
			c.JSON(http.StatusForbidden, errResp("FORBIDDEN", "You are not the assigned driver", ""))
		default:
			c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", "Invalid paycode", "payload"))
		}
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{
		"paycodeId":  pc.ID,
		"orderId":    order.ID,
		"totalKobo":  order.TotalKobo,
		"status":     order.Status,
		"merchantId": order.MerchantID,
		"customerId": order.CustomerID,
	}))
}

// ConfirmByCode — PRIMARY delivery confirmation path.
// Rider enters the 6-digit code the customer shared with them.
// On success: escrow releases, order marked delivered.
// Fallback when customer has no signal: POST /paycodes/scan-card (card.go).
// POST /paycodes/confirm-code
func (h *PaycodeHandler) ConfirmByCode(c *gin.Context) {
	var req struct {
		OrderID string   `json:"orderId" binding:"required"`
		Code    string   `json:"code"    binding:"required,len=6"`
		Lat     *float64 `json:"lat"` // optional rider GPS — evidence only, never blocks
		Lng     *float64 `json:"lng"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid order ID", "orderId"))
		return
	}

	driverID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.paycodes.ConfirmByCode(c.Request.Context(), orderID, driverID, req.Code, req.Lat, req.Lng); err != nil {
		switch err {
		case service.ErrNotAssigned:
			c.JSON(http.StatusForbidden, errResp("FORBIDDEN", "You are not the assigned driver", ""))
		case service.ErrDeliveryCodeLocked:
			c.JSON(http.StatusTooManyRequests, errResp("LOCKED", "Too many failed attempts. Contact support.", ""))
		case service.ErrDeliveryCodeInvalid:
			c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", "Code expired or not found", "code"))
		default:
			// Wrong code — err.Error() contains remaining attempts count
			c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", err.Error(), "code"))
		}
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{"message": "delivery confirmed", "status": "delivered"}))
}

// Confirm — QR paycode confirmation path (legacy / secondary).
// Used when the customer scans the driver's QR or vice versa.
// POST /paycodes/:id/confirm
func (h *PaycodeHandler) Confirm(c *gin.Context) {
	paycodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid paycode ID", "id"))
		return
	}

	var req struct {
		PIN *string `json:"pin"`
	}
	c.ShouldBindJSON(&req)

	driverID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.paycodes.Confirm(c.Request.Context(), paycodeID, driverID, req.PIN); err != nil {
		switch err {
		case service.ErrAlreadyConfirmed:
			c.JSON(http.StatusConflict, errResp("VALIDATION_ERROR", "Paycode already confirmed", ""))
		case service.ErrPaycodeExpired:
			c.JSON(http.StatusGone, errResp("VALIDATION_ERROR", "Paycode expired", ""))
		case service.ErrNotAssigned:
			c.JSON(http.StatusForbidden, errResp("FORBIDDEN", "You are not the assigned driver", ""))
		default:
			internalError(c, err)
		}
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{"message": "delivery confirmed", "status": "delivered"}))
}
