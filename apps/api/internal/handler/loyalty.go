package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/model"
)

// loyaltyService is the subset of service.LoyaltyService used by LoyaltyHandler.
type loyaltyService interface {
	GetBalance(ctx context.Context, userID uuid.UUID) (int, error)
	History(ctx context.Context, userID uuid.UUID, limit int) ([]model.LoyaltyEvent, error)
}

type LoyaltyHandler struct {
	loyalty loyaltyService
}

func NewLoyaltyHandler(loyalty loyaltyService) *LoyaltyHandler {
	return &LoyaltyHandler{loyalty: loyalty}
}

func (h *LoyaltyHandler) GetBalance(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	points, err := h.loyalty.GetBalance(c.Request.Context(), userID)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"points": points}))
}

func (h *LoyaltyHandler) GetHistory(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	events, err := h.loyalty.History(c.Request.Context(), userID, 50)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"events": events}))
}
