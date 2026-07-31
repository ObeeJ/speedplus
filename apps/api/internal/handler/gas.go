package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/service"
)

// authedCustomerID resolves the authenticated caller's ID. Discarding a parse
// error here (as the previous version did) yields uuid.Nil and lets the
// request proceed as an unauthenticated identity — return 401 instead.
func authedCustomerID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err != nil {
		c.JSON(http.StatusUnauthorized, errResp("UNAUTHORIZED", "Invalid or missing session", ""))
		return uuid.Nil, false
	}
	return id, true
}

type GasHandler struct {
	gas *service.GasService
}

func NewGasHandler(gas *service.GasService) *GasHandler {
	return &GasHandler{gas: gas}
}

// ListSpecs — GET /api/v1/gas/specs (public)
func (h *GasHandler) ListSpecs(c *gin.Context) {
	specs, err := h.gas.ListSpecs(c.Request.Context())
	if err != nil {
		internalError(c, err)
		return
	}
	// Specs are seeded data — safe to cache publicly for 5 minutes.
	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, successResp(gin.H{"specs": specs}))
}

// ListCylinders — GET /api/v1/cylinders (customer)
func (h *GasHandler) ListCylinders(c *gin.Context) {
	customerID, ok := authedCustomerID(c)
	if !ok {
		return
	}
	cylinders, err := h.gas.ListCylinders(c.Request.Context(), customerID)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"cylinders": cylinders}))
}

// RegisterCylinder — POST /api/v1/cylinders (customer)
func (h *GasHandler) RegisterCylinder(c *gin.Context) {
	var req struct {
		SpecID          *string  `json:"specId"`
		Serial          string   `json:"serial"          binding:"required"`
		ManufactureYear *int16   `json:"manufactureYear"`
		ValveType       *string  `json:"valveType"`
		TareKg          *float64 `json:"tareKg"`
		LastRecertAt    *string  `json:"lastRecertAt"` // ISO date YYYY-MM-DD
		Notes           *string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	customerID, ok := authedCustomerID(c)
	if !ok {
		return
	}

	in := service.RegisterCylinderInput{
		Serial:          req.Serial,
		ManufactureYear: req.ManufactureYear,
		ValveType:       req.ValveType,
		TareKg:          req.TareKg,
		Notes:           req.Notes,
	}
	if req.SpecID != nil {
		id, err := uuid.Parse(*req.SpecID)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid specId", "specId"))
			return
		}
		in.SpecID = &id
	}
	if req.LastRecertAt != nil {
		t, err := time.Parse("2006-01-02", *req.LastRecertAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "lastRecertAt must be YYYY-MM-DD", "lastRecertAt"))
			return
		}
		in.LastRecertAt = &t
	}

	cyl, err := h.gas.RegisterCylinder(c.Request.Context(), customerID, in)
	if err != nil {
		// RegisterCylinder's errors are client errors (duplicate serial,
		// invalid spec reference) — 422, not internalError's 500. Do not
		// forward err.Error() though; it can carry internal detail.
		c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", "Could not register cylinder", ""))
		return
	}
	c.JSON(http.StatusCreated, successResp(gin.H{"cylinder": cyl}))
}

// RetireCylinder — POST /api/v1/cylinders/:id/retire (customer)
func (h *GasHandler) RetireCylinder(c *gin.Context) {
	cylinderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid cylinder ID", "id"))
		return
	}
	customerID, ok := authedCustomerID(c)
	if !ok {
		return
	}
	if err := h.gas.RetireCylinder(c.Request.Context(), customerID, cylinderID); err != nil {
		switch {
		case errors.Is(err, service.ErrCylinderNotFound):
			c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Cylinder not found", ""))
		case errors.Is(err, service.ErrCylinderForbidden):
			c.JSON(http.StatusForbidden, errResp("FORBIDDEN", "Access denied", ""))
		default:
			c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", "Could not retire cylinder", ""))
		}
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"message": "cylinder retired"}))
}
