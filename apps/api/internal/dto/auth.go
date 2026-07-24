package dto

import "time"

// ── Requests ──────────────────────────────────────────────────────────────────

type RegisterRequest struct {
	FirstName string `json:"firstName" binding:"required"`
	LastName  string `json:"lastName"  binding:"required"`
	Phone     string `json:"phone"     binding:"required"`
	Password  string `json:"password"  binding:"required,min=8"`
	Role      string `json:"role"`
}

type LoginRequest struct {
	Phone    string `json:"phone"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type OTPRequestReq struct {
	Phone   string `json:"phone"   binding:"required"`
	Purpose string `json:"purpose" binding:"required"`
}

type OTPVerifyReq struct {
	Phone   string `json:"phone"   binding:"required"`
	OTP     string `json:"otp"     binding:"required"`
	Purpose string `json:"purpose" binding:"required"`
}

type SetPINRequest struct {
	PIN string `json:"pin" binding:"required,len=6"`
}

type VerifyPINRequest struct {
	PIN string `json:"pin" binding:"required,len=6"`
}

// ── Responses ─────────────────────────────────────────────────────────────────

type UserResponse struct {
	ID         string    `json:"id"`
	Role       string    `json:"role"`
	FirstName  string    `json:"firstName"`
	LastName   string    `json:"lastName"`
	Phone      string    `json:"phone"`
	Email      *string   `json:"email,omitempty"`
	AvatarURL  *string   `json:"avatarUrl,omitempty"`
	IsVerified bool      `json:"isVerified"`
	CreatedAt  time.Time `json:"createdAt"`
}

type AuthResponse struct {
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	User         UserResponse `json:"user"`
}

type TokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type OTPResponse struct {
	Message string  `json:"message"`
	DevCode *string `json:"_dev_code,omitempty"` // only populated in development
}

type VerifiedResponse struct {
	Verified bool `json:"verified"`
}
