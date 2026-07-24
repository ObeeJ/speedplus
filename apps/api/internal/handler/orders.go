package handler

import (
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
		MerchantID     string  `json:"merchantId" binding:"required"`
		QuoteID        string  `json:"quoteId" binding:"required"`
		Vertical       string  `json:"vertical" binding:"required"`
		DeliveryAddrID string  `json:"deliveryAddressId" binding:"required"`
		RecipientName  *string `json:"recipientName"`
		RecipientPhone *string `json:"recipientPhone"`
		PaymentMethod  string  `json:"paymentMethod"`
		PrescriptionID *string `json:"prescriptionId"`
		TipKobo        int64   `json:"tipKobo"`
		Items          []struct {
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
		CustomerID:     customerID,
		MerchantID:     merchantID,
		QuoteID:        quoteID,
		Vertical:       req.Vertical,
		DeliveryAddrID: addrID,
		RecipientName:  req.RecipientName,
		RecipientPhone: req.RecipientPhone,
		PaymentMethod:  req.PaymentMethod,
		TipKobo:        req.TipKobo,
		IdempotencyKey: idempotencyKey,
	}

	if req.PrescriptionID != nil {
		pid, _ := uuid.Parse(*req.PrescriptionID)
		in.PrescriptionID = &pid
	}

	for _, item := range req.Items {
		pid, _ := uuid.Parse(item.ProductID)
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
		switch err {
		case service.ErrQuoteInvalid:
			c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", err.Error(), "quoteId"))
		case service.ErrMerchantClosed:
			c.JSON(http.StatusConflict, errResp("MERCHANT_CLOSED", "Merchant is currently closed", ""))
		case service.ErrRxRequired:
			c.JSON(http.StatusUnprocessableEntity, errResp("PRESCRIPTION_REQUIRED", "Prescription required", "prescriptionId"))
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

	c.JSON(http.StatusOK, successResp(order))
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
		Sequence int    `json:"sequence" binding:"required,min=1"`
		Code     string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	driverID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.orders.ConfirmStop(c.Request.Context(), orderID, driverID, req.Sequence, req.Code); err != nil {
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
		if err == service.ErrIllegalTransition {
			c.JSON(http.StatusConflict, errResp("VALIDATION_ERROR", err.Error(), ""))
			return
		}
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{"message": "order cancelled"}))
}
