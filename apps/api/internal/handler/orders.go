package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/service"
)

type OrderHandler struct {
	orders *service.OrderService
}

func NewOrderHandler(orders *service.OrderService) *OrderHandler {
	return &OrderHandler{orders: orders}
}

func (h *OrderHandler) Create(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Idempotency-Key header required", "Idempotency-Key"))
		return
	}

	var req struct {
		MerchantID        string  `json:"merchantId" binding:"required"`
		QuoteID           string  `json:"quoteId" binding:"required"`
		Vertical          string  `json:"vertical" binding:"required"`
		DeliveryAddrID    string  `json:"deliveryAddressId" binding:"required"`
		RecipientName     *string `json:"recipientName"`
		RecipientPhone    *string `json:"recipientPhone"`
		PaymentMethod     string  `json:"paymentMethod"`
		PrescriptionID    *string `json:"prescriptionId"`
		TipKobo           int64   `json:"tipKobo"`
		DeclaredValueKobo *int64  `json:"declaredValueKobo"`
		// Gas-specific
		GasMode    *string `json:"gasMode"`
		CylinderID *string `json:"cylinderId"`
		Items []struct {
			ProductID        string  `json:"productId" binding:"required"`
			Name             string  `json:"name"`
			Quantity         int     `json:"quantity" binding:"required,min=1"`
			UnitPriceKobo    int64   `json:"unitPriceKobo"`
			WeightKg         float64 `json:"weightKg"`
			SizeCategory     string  `json:"sizeCategory"`
			Customizations   *string `json:"customizations"`
			SubstitutionPref *string `json:"substitutionPreference"`
		} `json:"items" binding:"required,min=1"`
		Stops []struct {
			Sequence       int     `json:"sequence" binding:"required,min=1"`
			AddressID      string  `json:"addressId" binding:"required"`
			RecipientName  *string `json:"recipientName"`
			RecipientPhone *string `json:"recipientPhone"`
			Notes          *string `json:"notes"`
		} `json:"stops"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	customerID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	merchantID, _ := uuid.Parse(req.MerchantID)
	quoteID, _ := uuid.Parse(req.QuoteID)
	addrID, _ := uuid.Parse(req.DeliveryAddrID)

	in := service.CreateOrderInput{
		CustomerID:        customerID,
		MerchantID:        merchantID,
		QuoteID:           quoteID,
		Vertical:          req.Vertical,
		DeliveryAddrID:    addrID,
		RecipientName:     req.RecipientName,
		RecipientPhone:    req.RecipientPhone,
		PaymentMethod:     req.PaymentMethod,
		TipKobo:           req.TipKobo,
		DeclaredValueKobo: req.DeclaredValueKobo,
		IdempotencyKey:    idempotencyKey,
		GasMode:           req.GasMode,
	}

	if req.PrescriptionID != nil {
		// Previously the parse error was discarded, so a malformed
		// prescriptionId silently became the nil UUID and surfaced later as
		// a confusing PRESCRIPTION_REQUIRED instead of a clear 400.
		pid, err := uuid.Parse(*req.PrescriptionID)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid prescriptionId", "prescriptionId"))
			return
		}
		in.PrescriptionID = &pid
	}

	if req.CylinderID != nil {
		cid, err := uuid.Parse(*req.CylinderID)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid cylinderId", "cylinderId"))
			return
		}
		in.CylinderID = &cid
	}

	for _, item := range req.Items {
		// For the package vertical, productId is a sentinel string (not a real
		// catalog product). Accept uuid.Nil for package items — the service
		// layer does not join on product_id for package orders.
		pid, parseErr := uuid.Parse(item.ProductID)
		if parseErr != nil && req.Vertical != "package" {
			c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR",
				"item productId must be a valid UUID", "items.productId"))
			return
		}
		in.Items = append(in.Items, service.OrderItemInput{
			ProductID:        pid,
			Name:             item.Name,
			Quantity:         item.Quantity,
			UnitPriceKobo:    item.UnitPriceKobo,
			WeightKg:         item.WeightKg,
			SizeCategory:     item.SizeCategory,
			Customizations:   item.Customizations,
			SubstitutionPref: item.SubstitutionPref,
		})
	}

	for _, stop := range req.Stops {
		addrID, _ := uuid.Parse(stop.AddressID)
		in.Stops = append(in.Stops, service.OrderStopInput{
			Sequence:       stop.Sequence,
			AddressID:      addrID,
			RecipientName:  stop.RecipientName,
			RecipientPhone: stop.RecipientPhone,
			Notes:          stop.Notes,
		})
	}

	order, err := h.orders.Create(c.Request.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrQuoteInvalid):
			c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", err.Error(), "quoteId"))
		case errors.Is(err, service.ErrMerchantClosed):
			c.JSON(http.StatusConflict, errResp("MERCHANT_CLOSED", "Merchant is currently closed", ""))
		case errors.Is(err, service.ErrRxRequired):
			c.JSON(http.StatusUnprocessableEntity, errResp("PRESCRIPTION_REQUIRED", "Prescription required", "prescriptionId"))
		case errors.Is(err, service.ErrRxNotApproved):
			c.JSON(http.StatusUnprocessableEntity, errResp("PRESCRIPTION_NOT_APPROVED", err.Error(), "prescriptionId"))
		case errors.Is(err, service.ErrGasValidation):
			c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", err.Error(), "gas"))
		case errors.Is(err, service.ErrInsufficientBalance):
			c.JSON(http.StatusPaymentRequired, errResp("INSUFFICIENT_BALANCE", "Insufficient wallet balance", "wallet"))
		default:
			internalError(c, err)
		}
		return
	}

	c.JSON(http.StatusCreated, successResp(order))
}

func (h *OrderHandler) GetByID(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid order ID", "id"))
		return
	}

	requesterID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	requesterRole := c.GetString(middleware.CtxUserRole)

	order, err := h.orders.GetByID(c.Request.Context(), orderID, requesterID, requesterRole)
	if err != nil {
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Order not found", ""))
		return
	}

	// Enrich with driver profile when a driver is assigned.
	resp := h.orders.ToResponse(c.Request.Context(), order)
	c.JSON(http.StatusOK, successResp(resp))
}

func (h *OrderHandler) GetStops(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid order ID", "id"))
		return
	}
	requesterID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	requesterRole := c.GetString(middleware.CtxUserRole)

	// Ownership check via GetByID
	if _, err := h.orders.GetByID(c.Request.Context(), orderID, requesterID, requesterRole); err != nil {
		c.JSON(http.StatusForbidden, errResp("FORBIDDEN", "Access denied", ""))
		return
	}

	stops, err := h.orders.GetStops(c.Request.Context(), orderID, requesterID, requesterRole)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"stops": stops}))
}

func (h *OrderHandler) ConfirmStop(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid order ID", "id"))
		return
	}
	var req struct {
		Sequence            int      `json:"sequence" binding:"required,min=1"`
		Code                string   `json:"code" binding:"required"`
		EmptyCollected      bool     `json:"emptyCollected"`
		EmptyCylinderSerial *string  `json:"emptyCylinderSerial"`
		CapturedLat         *float64 `json:"capturedLat"`
		CapturedLng         *float64 `json:"capturedLng"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	driverID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.orders.ConfirmStop(c.Request.Context(), orderID, driverID, req.Sequence, service.ConfirmStopInput{
		Code:                req.Code,
		EmptyCollected:      req.EmptyCollected,
		EmptyCylinderSerial: req.EmptyCylinderSerial,
		CapturedLat:         req.CapturedLat,
		CapturedLng:         req.CapturedLng,
	}); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"message": "stop confirmed"}))
}

func (h *OrderHandler) Cancel(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid order ID", "id"))
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	actorID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	actorRole := c.GetString(middleware.CtxUserRole)

	if err := h.orders.Cancel(c.Request.Context(), orderID, actorID, actorRole, req.Reason); err != nil {
		switch err {
		case service.ErrIllegalTransition:
			c.JSON(http.StatusConflict, errResp("VALIDATION_ERROR", err.Error(), ""))
		case service.ErrOrderNotFound:
			c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Order not found", ""))
		default:
			internalError(c, err)
		}
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{"message": "order cancelled"}))
}

// List — GET /orders?vertical=package&status=delivered&cursor=...
func (h *OrderHandler) List(c *gin.Context) {
	customerID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	var cursor *uuid.UUID
	if raw := c.Query("cursor"); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			cursor = &id
		}
	}
	orders, err := h.orders.ListForCustomer(
		c.Request.Context(), customerID,
		c.Query("vertical"), c.Query("status"),
		cursor, 20,
	)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"orders": orders}))
}

// Receipt — GET /orders/:id/receipt
func (h *OrderHandler) Receipt(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid order ID", "id"))
		return
	}
	customerID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	receipt, err := h.orders.GetReceipt(c.Request.Context(), orderID, customerID)
	if err != nil {
		if errors.Is(err, service.ErrOrderForbidden) {
			c.JSON(http.StatusForbidden, errResp("FORBIDDEN", "Access denied", ""))
			return
		}
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Order not found", ""))
		return
	}
	c.JSON(http.StatusOK, successResp(receipt))
}

// Review — POST /orders/:id/review
func (h *OrderHandler) Review(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Idempotency-Key header required", "Idempotency-Key"))
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid order ID", "id"))
		return
	}
	var req struct {
		RevieweeType string  `json:"revieweeType" binding:"required"`
		Rating       int     `json:"rating"       binding:"required,min=1,max=5"`
		Comment      *string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	customerID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.orders.SubmitReview(c.Request.Context(), orderID, customerID, req.RevieweeType, req.Rating, req.Comment); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusCreated, successResp(gin.H{"message": "review submitted"}))
}

// Badges — GET /drivers/:id/badges
func (h *OrderHandler) DriverBadges(c *gin.Context) {
	driverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid driver ID", "id"))
		return
	}
	badges, err := h.orders.GetDriverBadges(c.Request.Context(), driverID)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"badges": badges}))
}
