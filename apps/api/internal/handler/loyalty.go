package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/service"
)

type LoyaltyHandler struct {
	loyalty *service.LoyaltyService
}

func NewLoyaltyHandler(loyalty *service.LoyaltyService) *LoyaltyHandler {
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
