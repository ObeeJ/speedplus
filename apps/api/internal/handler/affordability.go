package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/service"
)

type AffordabilityHandler struct {
	svc *service.AffordabilityService
}

func NewAffordabilityHandler(svc *service.AffordabilityService) *AffordabilityHandler {
	return &AffordabilityHandler{svc: svc}
}

// GetAffordability returns what the user's current wallet balance can buy
// based on real-time prices from nearby merchants.
//
// GET /api/v1/wallet/affordability?lat=6.5244&lng=3.3792&radius=5
//
// lat and lng are required. radius is optional (default 5km).
// The response is designed to power the "Your balance covers X" UI widget.
func (h *AffordabilityHandler) GetAffordability(c *gin.Context) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")

	if latStr == "" || lngStr == "" {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "lat and lng query parameters are required", "lat"))
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "lat must be a valid decimal number", "lat"))
		return
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "lng must be a valid decimal number", "lng"))
		return
	}

	radius := 5.0
	if r := c.Query("radius"); r != "" {
		if parsed, err := strconv.ParseFloat(r, 64); err == nil && parsed > 0 && parsed <= 50 {
			radius = parsed
		}
	}

	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	results, err := h.svc.GetAffordability(c.Request.Context(), userID, lat, lng, radius)
	if err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{
		"verticals": results,
		"radiusKm":  radius,
	}))
}
