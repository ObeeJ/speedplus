package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/dto"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/model"
)

type kycService interface {
	SubmitCheck(ctx context.Context, userID uuid.UUID, docType model.KYCDocType, params map[string]string) error
	QueueForAdmin(ctx context.Context, page, pageSize int) ([]model.KYCCheck, error)
	AdminApprove(ctx context.Context, checkID, adminID uuid.UUID, note string) error
	AdminReject(ctx context.Context, checkID, adminID uuid.UUID, note string) error
	GetUserKYC(ctx context.Context, userID uuid.UUID) ([]model.KYCCheck, error)
}

type KYCHandler struct {
	kyc kycService
}

func NewKYCHandler(kyc kycService) *KYCHandler {
	return &KYCHandler{kyc: kyc}
}

func (h *KYCHandler) SubmitCheck(c *gin.Context) {
	var req struct {
		DocType string            `json:"docType" binding:"required"`
		Params  map[string]string `json:"params"  binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
		return
	}

	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.kyc.SubmitCheck(c.Request.Context(), userID, model.KYCDocType(req.DocType), req.Params); err != nil {
		c.JSON(http.StatusUnprocessableEntity, dto.Fail("VALIDATION_ERROR", err.Error(), "docType"))
		return
	}
	c.JSON(http.StatusAccepted, dto.OK(dto.MessageResponse{Message: "KYC check submitted"}))
}

func (h *KYCHandler) AdminQueue(c *gin.Context) {
	page := 0
	checks, err := h.kyc.QueueForAdmin(c.Request.Context(), page, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"checks": checks}))
}

func (h *KYCHandler) Approve(c *gin.Context) {
	checkID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid check ID", "id"))
		return
	}
	var req struct {
		Note string `json:"note"`
	}
	c.ShouldBindJSON(&req)

	adminID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.kyc.AdminApprove(c.Request.Context(), checkID, adminID, req.Note); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.MessageResponse{Message: "approved"}))
}

func (h *KYCHandler) Reject(c *gin.Context) {
	checkID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid check ID", "id"))
		return
	}
	var req struct {
		Note string `json:"note" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", err.Error(), "note"))
		return
	}

	adminID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.kyc.AdminReject(c.Request.Context(), checkID, adminID, req.Note); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.MessageResponse{Message: "rejected"}))
}

// AdminGetUserKYC — GET /admin/users/:id/kyc
// Returns full KYC history for a user — for AML investigation and compliance tracing.
func (h *KYCHandler) AdminGetUserKYC(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid user ID", "id"))
		return
	}
	checks, err := h.kyc.GetUserKYC(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"checks": checks}))
}

// MyKYCStatus — GET /kyc/status
// Lets a user see their own verification checks and statuses.
func (h *KYCHandler) MyKYCStatus(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	checks, err := h.kyc.GetUserKYC(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"checks": checks}))
}

