package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/dto"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/service"
)

type PricingHandler struct {
	pricing *service.PricingService
}

func NewPricingHandler(pricing *service.PricingService) *PricingHandler {
	return &PricingHandler{pricing: pricing}
}

func (h *PricingHandler) Quote(c *gin.Context) {
	var req dto.QuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	merchantID, err := uuid.Parse(req.MerchantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid merchantId", "merchantId"))
		return
	}

	customerID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))

	quote, err := h.pricing.Quote(c.Request.Context(), service.QuoteRequest{
		CustomerID:   customerID,
		MerchantID:   merchantID,
		Vertical:     req.Vertical,
		SubtotalKobo: req.SubtotalKobo,
		OriginLat:    req.OriginLat,
		OriginLng:    req.OriginLng,
		DestLat:      req.DestLat,
		DestLng:      req.DestLng,
		WeightKg:     req.WeightKg,
		SizeCategory: service.SizeCategory(req.SizeCategory),
	})
	if err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.OK(dto.QuoteResponse{
		ID:              quote.ID,
		SubtotalKobo:    quote.SubtotalKobo,
		DeliveryKobo:    quote.DeliveryKobo,
		ServiceKobo:     quote.ServiceKobo,
		TotalKobo:       quote.TotalKobo,
		DistanceKm:      quote.DistanceKm,
		ETAMinutes:      quote.ETAMinutes,
		WeightKg:        quote.WeightKg,
		SizeCategory:    quote.SizeCategory,
		WeatherAdvisory: quote.WeatherAdvisory,
		ExpiresAt:       quote.ExpiresAt,
	}))
}
