package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/dto"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/service"
)

type DispatchHandler struct {
	dispatch *service.DispatchService
}

func NewDispatchHandler(dispatch *service.DispatchService) *DispatchHandler {
	return &DispatchHandler{dispatch: dispatch}
}

func (h *DispatchHandler) UpdateLocation(c *gin.Context) {
	var req struct {
		Lat     float64  `json:"lat"     binding:"required"`
		Lng     float64  `json:"lng"     binding:"required"`
		Heading *float64 `json:"heading"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
		return
	}

	driverID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	heading := 0.0
	if req.Heading != nil {
		heading = *req.Heading
	}

	if err := h.dispatch.UpdateLocation(c.Request.Context(), driverID, req.Lat, req.Lng, heading); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.MessageResponse{Message: "location updated"}))
}

func (h *DispatchHandler) AcceptOffer(c *gin.Context) {
	offerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid offer ID", "id"))
		return
	}

	driverID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.dispatch.AcceptOffer(c.Request.Context(), offerID, driverID); err != nil {
		c.JSON(http.StatusConflict, dto.Fail("VALIDATION_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.MessageResponse{Message: "offer accepted"}))
}

func (h *DispatchHandler) RejectOffer(c *gin.Context) {
	offerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid offer ID", "id"))
		return
	}
	driverID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.dispatch.RejectOffer(c.Request.Context(), offerID, driverID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.MessageResponse{Message: "offer rejected"}))
}

func (h *DispatchHandler) AdminAssign(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("orderId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid order ID", "orderId"))
		return
	}
	var req struct {
		DriverID string `json:"driverId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
		return
	}
	driverID, err := uuid.Parse(req.DriverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid driver ID", "driverId"))
		return
	}

	if err := h.dispatch.ManualAssign(c.Request.Context(), orderID, driverID); err != nil {
		c.JSON(http.StatusConflict, dto.Fail("VALIDATION_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.MessageResponse{Message: "driver assigned"}))
}
