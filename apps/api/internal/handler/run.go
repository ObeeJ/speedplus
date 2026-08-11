package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/service"
)

type RunHandler struct{ runs *service.RunService }

func NewRunHandler(runs *service.RunService) *RunHandler { return &RunHandler{runs: runs} }

func (h *RunHandler) GetRun(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid run ID", "id"))
		return
	}
	requesterID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	requesterRole := c.GetString(middleware.CtxUserRole)
	run, err := h.runs.GetRun(c.Request.Context(), id, requesterID, requesterRole)
	if err != nil {
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "Run not found", ""))
		return
	}
	c.JSON(http.StatusOK, successResp(run))
}
