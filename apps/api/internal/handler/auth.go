package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/observability"
	"github.com/speedplus/api/internal/service"
)

// authService is the interface AuthHandler depends on.
// Accept interfaces, return structs — keeps the handler testable with a stub.
type authService interface {
	Register(ctx context.Context, in service.RegisterInput) (*model.User, string, string, error)
	Login(ctx context.Context, phone, password string) (*model.User, string, string, error)
	Refresh(ctx context.Context, raw string) (*model.User, string, string, error)
	VerifyOTP(ctx context.Context, phone, code, purpose string) (*model.User, string, string, error)
	Logout(ctx context.Context, raw string) error
	RequestOTP(ctx context.Context, phone, purpose string) (string, error)
	SetPIN(ctx context.Context, userID uuid.UUID, pin string) error
	VerifyPIN(ctx context.Context, userID uuid.UUID, pin string) error
	ResetPassword(ctx context.Context, phone, otp, newPassword string) error
}

type AuthHandler struct {
	auth   authService
	secure bool // false in development so cookies work over http://localhost
}

func NewAuthHandler(auth *service.AuthService, secure bool) *AuthHandler {
	return &AuthHandler{auth: auth, secure: secure}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		FirstName    string `json:"firstName" binding:"required"`
		LastName     string `json:"lastName" binding:"required"`
		Phone        string `json:"phone" binding:"required"`
		Password     string `json:"password" binding:"required,min=8"`
		Role         string `json:"role"`
		ReferralCode string `json:"referralCode"` // optional
		VehicleType  string `json:"vehicleType"`  // required when role=driver: bicycle|motorcycle|car|van
		VehiclePlate string `json:"vehiclePlate"` // required when role=driver
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

	user, access, refresh, err := h.auth.Register(c.Request.Context(), service.RegisterInput{
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Phone:        req.Phone,
		Password:     req.Password,
		Role:         role,
		ReferralCode: req.ReferralCode,
		VehicleType:  req.VehicleType,
		VehiclePlate: req.VehiclePlate,
	})
	if err != nil {
		switch err {
		case service.ErrPhoneExists:
			c.JSON(http.StatusConflict, errResp("VALIDATION_ERROR", "Phone already registered", "phone"))
		case service.ErrVehicleRequired, service.ErrVehicleTypeInvalid:
			c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", err.Error(), "vehicleType"))
		default:
			internalError(c, err)
		}
		return
	}

	setRefreshTokenCookie(c, refresh, h.secure)
	setRoleCookie(c, user.Role, h.secure)

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

	setRefreshTokenCookie(c, refresh, h.secure)
	setRoleCookie(c, user.Role, h.secure)

	c.JSON(http.StatusOK, successResp(gin.H{
		"accessToken":  access,
		"refreshToken": refresh,
		"user":         userView(user),
	}))
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	c.ShouldBindJSON(&req)
	token := req.RefreshToken
	if token == "" {
		token, _ = c.Cookie("fourdat_refresh")
	}
	if token == "" {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Refresh token required", ""))
		return
	}

	user, access, refresh, err := h.auth.Refresh(c.Request.Context(), token)
	if err != nil {
		setRefreshTokenCookie(c, "", h.secure, -1)
		clearRoleCookie(c, h.secure)
		c.JSON(http.StatusUnauthorized, errResp("UNAUTHORIZED", "Session expired. Please log in again.", ""))
		return
	}

	setRefreshTokenCookie(c, refresh, h.secure)
	setRoleCookie(c, user.Role, h.secure)

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
	token := req.RefreshToken
	if token == "" {
		token, _ = c.Cookie("fourdat_refresh")
	}
	if token != "" {
		_ = h.auth.Logout(c.Request.Context(), token)
	}
	setRefreshTokenCookie(c, "", h.secure, -1)
	clearRoleCookie(c, h.secure)
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

	user, access, refresh, err := h.auth.VerifyOTP(c.Request.Context(), req.Phone, req.Code, req.Purpose)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", "Invalid or expired OTP", "otp"))
		return
	}

	setRefreshTokenCookie(c, refresh, h.secure)
	setRoleCookie(c, user.Role, h.secure)

	c.JSON(http.StatusOK, successResp(gin.H{
		"verified":     true,
		"accessToken":  access,
		"refreshToken": refresh,
	}))
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

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Phone       string `json:"phone" binding:"required"`
		OTP         string `json:"otp" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}
	if err := h.auth.ResetPassword(c.Request.Context(), req.Phone, req.OTP, req.NewPassword); err != nil {
		switch err {
		case service.ErrOTPInvalid:
			c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", "Invalid or expired OTP", "otp"))
		case service.ErrUserNotFound:
			c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_ERROR", "Invalid or expired OTP", "otp"))
		default:
			internalError(c, err)
		}
		return
	}
	c.JSON(http.StatusOK, successResp(gin.H{"message": "Password reset successful"}))
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
	// Do not forward raw Gin/validator error strings — they expose internal
	// struct field names and Go type tags. Return a generic message instead.
	c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "Invalid request body", ""))
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
		"id":           u.ID,
		"role":         u.Role,
		"firstName":    u.FirstName,
		"lastName":     u.LastName,
		"phone":        u.Phone,
		"referralCode": u.ReferralCode,
		"isVerified":   u.IsVerified,
		"createdAt":    u.CreatedAt,
	}
}

func setRefreshTokenCookie(c *gin.Context, token string, secure bool, maxAge ...int) {
	age := 7 * 24 * 3600
	if len(maxAge) > 0 {
		age = maxAge[0]
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("fourdat_refresh", token, age, "/api/v1/auth", ".fourdat.com", secure, true)
}

// setRoleCookie writes a non-HttpOnly cookie Cloudflare edge rules read to
// route the user to the correct subdomain. It carries no secret, only the
// role string. HttpOnly=false is intentional — Cloudflare needs to read it.
// SameSite=Lax is intentional — the cookie must be sent on top-level
// cross-site navigations (e.g. link from WhatsApp) so the edge rule fires.
func setRoleCookie(c *gin.Context, role model.UserRole, secure bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("__role", string(role), 7*24*3600, "/", ".fourdat.com", secure, false)
}

// clearRoleCookie expires the __role cookie. MaxAge=-1 instructs the browser
// to delete it immediately. Never call setRoleCookie("") — use this instead.
func clearRoleCookie(c *gin.Context, secure bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("__role", "", -1, "/", ".fourdat.com", secure, false)
}
