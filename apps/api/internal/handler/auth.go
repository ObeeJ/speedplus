package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/observability"
	"github.com/speedplus/api/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		FirstName string `json:"firstName" binding:"required"`
		LastName  string `json:"lastName" binding:"required"`
		Phone     string `json:"phone" binding:"required"`
		Password  string `json:"password" binding:"required,min=8"`
		Role      string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	role := model.RoleCustomer
	if req.Role == "driver" {
		role = model.RoleDriver
	} else if req.Role == "merchant" {
		role = model.RoleMerchant
	}

	user, access, refresh, err := h.auth.Register(c.Request.Context(), req.FirstName, req.LastName, req.Phone, req.Password, role)
	if err != nil {
		if err == service.ErrPhoneExists {
			c.JSON(http.StatusConflict, errResp("VALIDATION_ERROR", "Phone already registered", "phone"))
			return
		}
		internalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, successResp(gin.H{
		"accessToken":  access,
		"refreshToken": refresh,
		"user":         userView(user),
	}))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Phone    string `json:"phone" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	user, access, refresh, err := h.auth.Login(c.Request.Context(), req.Phone, req.Password)
	if err != nil {
		// Same message for wrong phone or wrong password — no enumeration
		c.JSON(http.StatusUnauthorized, errResp("UNAUTHORIZED", "Invalid credentials", ""))
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{
		"accessToken":  access,
		"refreshToken": refresh,
		"user":         userView(user),
	}))
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	access, refresh, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errResp("UNAUTHORIZED", "Session expired. Please log in again.", ""))
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{
		"accessToken":  access,
		"refreshToken": refresh,
	}))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	c.ShouldBindJSON(&req)
	if req.RefreshToken != "" {
		h.auth.Logout(c.Request.Context(), req.RefreshToken)
	}
	c.JSON(http.StatusOK, successResp(gin.H{"message": "logged out"}))
}

func (h *AuthHandler) RequestOTP(c *gin.Context) {
	var req struct {
		Phone   string `json:"phone" binding:"required"`
		Purpose string `json:"purpose" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	// The code is delivered via email/SMS inside RequestOTP — it must never
	// be echoed back in the HTTP response, in any environment. A prior
	// version gated this behind a "development" context key that was never
	// actually set (dead code), which was one accidental middleware change
	// away from leaking OTPs in production responses.
	if _, err := h.auth.RequestOTP(c.Request.Context(), req.Phone, req.Purpose); err != nil {
		internalError(c, err)
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{"message": "OTP sent"}))
}

func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req struct {
		Phone   string `json:"phone" binding:"required"`
		Code    string `json:"otp" binding:"required"`
		Purpose string `json:"purpose" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	if err := h.auth.VerifyOTP(c.Request.Context(), req.Phone, req.Code, req.Purpose); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", "Invalid or expired OTP", "otp"))
		return
	}

	c.JSON(http.StatusOK, successResp(gin.H{"verified": true}))
}

func (h *AuthHandler) SetPIN(c *gin.Context) {
	var req struct {
		PIN string `json:"pin" binding:"required,len=4"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.auth.SetPIN(c.Request.Context(), userID, req.PIN); err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"message": "PIN set"}))
}

func (h *AuthHandler) VerifyPIN(c *gin.Context) {
	var req struct {
		PIN string `json:"pin" binding:"required,len=4"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.auth.VerifyPIN(c.Request.Context(), userID, req.PIN); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", "Invalid PIN", "pin"))
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"verified": true}))
}

// ── Response helpers ──────────────────────────────────────────────────────────

func successResp(data interface{}) gin.H {
	return gin.H{"success": true, "data": data}
}

func errResp(code, message, field string) gin.H {
	e := gin.H{"code": code, "message": message}
	if field != "" {
		e["field"] = field
	}
	return gin.H{"success": false, "error": e}
}

func validationError(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", err.Error(), ""))
}

func internalError(c *gin.Context, err error) {
	observability.CaptureError(c.Request.Context(), err, "internal error",
		"request_id", c.GetString(middleware.RequestIDKey),
		"path", c.Request.URL.Path,
	)
	c.JSON(http.StatusInternalServerError, errResp("INTERNAL_ERROR", "An unexpected error occurred", ""))
}

func userView(u *model.User) gin.H {
	return gin.H{
		"id":         u.ID,
		"role":       u.Role,
		"firstName":  u.FirstName,
		"lastName":   u.LastName,
		"phone":      u.Phone,
		"isVerified": u.IsVerified,
		"createdAt":  u.CreatedAt,
	}
}
