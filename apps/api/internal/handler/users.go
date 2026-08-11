package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/dto"
	"github.com/speedplus/api/internal/middleware"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
)

type UsersHandler struct {
	users repo.UserRepo
}

func NewUsersHandler(users repo.UserRepo) *UsersHandler {
	return &UsersHandler{users: users}
}

func (h *UsersHandler) Me(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	user, err := h.users.FindByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Fail("NOT_FOUND", "User not found", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(toUserResponse(user)))
}

func (h *UsersHandler) UpdateMe(c *gin.Context) {
	var req struct {
		FirstName *string `json:"firstName"`
		LastName  *string `json:"lastName"`
		Email     *string `json:"email"`
		Username  *string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
		return
	}

	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	user, err := h.users.FindByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Fail("NOT_FOUND", "User not found", ""))
		return
	}

	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.Email != nil {
		user.Email = req.Email
	}
	if req.Username != nil {
		user.Username = req.Username
	}

	if err := h.users.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(toUserResponse(user)))
}

func (h *UsersHandler) ListAddresses(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	addrs, err := h.users.ListAddresses(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"addresses": addrs}))
}

func (h *UsersHandler) CreateAddress(c *gin.Context) {
	var req struct {
		Label                *string `json:"label"`
		Street               string  `json:"street"  binding:"required"`
		City                 string  `json:"city"    binding:"required"`
		State                string  `json:"state"   binding:"required"`
		Lat                  float64 `json:"lat"     binding:"required"`
		Lng                  float64 `json:"lng"     binding:"required"`
		DeliveryInstructions *string `json:"deliveryInstructions"`
		IsDefault            bool    `json:"isDefault"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail("VALIDATION_ERROR", "Invalid request body", ""))
		return
	}

	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	addr := &model.Address{
		ID:                   uuid.New(),
		UserID:               userID,
		Label:                req.Label,
		Street:               req.Street,
		City:                 req.City,
		State:                req.State,
		Country:              "Nigeria",
		Lat:                  req.Lat,
		Lng:                  req.Lng,
		DeliveryInstructions: req.DeliveryInstructions,
		IsDefault:            req.IsDefault,
		CreatedAt:            time.Now(),
	}
	if err := h.users.CreateAddress(c.Request.Context(), addr); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Fail("INTERNAL_ERROR", "An unexpected error occurred", ""))
		return
	}
	c.JSON(http.StatusCreated, dto.OK(addr))
}

func (h *UsersHandler) GetDriverProfile(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	dp, err := h.users.FindDriverProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Fail("NOT_FOUND", "Driver profile not found", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(dp))
}

func (h *UsersHandler) GetMerchantProfile(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	mp, err := h.users.FindMerchantProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Fail("NOT_FOUND", "Merchant profile not found", ""))
		return
	}
	c.JSON(http.StatusOK, dto.OK(mp))
}

func toUserResponse(u *model.User) dto.UserResponse {
	return dto.UserResponse{
		ID:         u.ID.String(),
		Role:       string(u.Role),
		FirstName:  u.FirstName,
		LastName:   u.LastName,
		Phone:      u.Phone,
		Email:      u.Email,
		AvatarURL:  u.AvatarURL,
		IsVerified: u.IsVerified,
		CreatedAt:  u.CreatedAt,
	}
}
