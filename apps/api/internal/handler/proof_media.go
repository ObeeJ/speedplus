package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/dto"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/service"
)

type ProofMediaHandler struct {
	media *service.ProofMediaService
}

func NewProofMediaHandler(media *service.ProofMediaService) *ProofMediaHandler {
	return &ProofMediaHandler{media: media}
}

// PresignUpload — POST /orders/:id/proof/presign (driver only, assigned driver enforced in service)
func (h *ProofMediaHandler) PresignUpload(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid order ID", "id"))
		return
	}
	var req struct {
		StopID      *string `json:"stopId"`
		Kind        string  `json:"kind" binding:"required"`
		ContentType string  `json:"contentType" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", err.Error(), ""))
		return
	}
	var stopID *uuid.UUID
	if req.StopID != nil {
		id, err := uuid.Parse(*req.StopID)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid stopId", "stopId"))
			return
		}
		stopID = &id
	}

	driverID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	uploadURL, key, err := h.media.PresignUpload(c.Request.Context(), orderID, stopID, req.Kind, driverID, req.ContentType)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"uploadUrl": uploadURL, "key": key}))
}

// ConfirmUpload — POST /orders/:id/proof/confirm (driver only)
func (h *ProofMediaHandler) ConfirmUpload(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid order ID", "id"))
		return
	}
	var req struct {
		StopID      *string  `json:"stopId"`
		Kind        string   `json:"kind" binding:"required"`
		Key         string   `json:"key" binding:"required"`
		SHA256      string   `json:"sha256" binding:"required"`
		SealSerial  *string  `json:"sealSerial"`
		MeasuredKg  *float64 `json:"measuredKg"`
		CapturedLat *float64 `json:"capturedLat"`
		CapturedLng *float64 `json:"capturedLng"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", err.Error(), ""))
		return
	}
	var stopID *uuid.UUID
	if req.StopID != nil {
		id, err := uuid.Parse(*req.StopID)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid stopId", "stopId"))
			return
		}
		stopID = &id
	}

	driverID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	media, err := h.media.ConfirmUpload(c.Request.Context(), driverID, service.ConfirmUploadInput{
		OrderID: orderID, StopID: stopID, Kind: req.Kind, Key: req.Key, SHA256: req.SHA256,
		SealSerial: req.SealSerial, MeasuredKg: req.MeasuredKg,
		CapturedLat: req.CapturedLat, CapturedLng: req.CapturedLng,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", err.Error(), ""))
		return
	}
	c.JSON(http.StatusCreated, dto.OK(gin.H{"id": media.ID}))
}

// GetMedia — GET /orders/:id/proof (customer/admin)
func (h *ProofMediaHandler) GetMedia(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid order ID", "id"))
		return
	}
	requesterID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	requesterRole := c.GetString(middleware.CtxUserRole)

	media, err := h.media.GetMediaForOrder(c.Request.Context(), orderID, requesterID, requesterRole)
	if err != nil {
		c.JSON(http.StatusForbidden, dto.Fail("FORBIDDEN", "Access denied", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"media": media}))
}
