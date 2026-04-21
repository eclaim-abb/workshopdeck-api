package auth

import "eclaim-workshop-deck-api/internal/models"

type RegisterRequest struct {
	UserProfileNo uint   `json:"user_profile_no"`
	RoleNo        uint   `json:"role_no" binding:"required"`
	Name          string `json:"user_name" binding:"required"`
	UserId        string `json:"user_id" binding:"required"`
	Email         string `json:"email" binding:"required,email"`
	Password      string `json:"password" binding:"required,min=6"`
	CreatedBy     uint   `json:"created_by" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type FindByEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ChangePasswordRequest struct {
	Email           string `json:"email" binding:"required,email"`
	Username        string `json:"user_id" binding:"required"`
	Password        string `json:"password" binding:"required,min=6"`
	NewPassword     string `json:"new_password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

type UpdateAccountRequest struct {
	Email           string `json:"email" binding:"email"`
	Username        string `json:"user_id"`
	Password        string `json:"password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
	UserNo          uint   `json:"user_no"`
}

type AuthResponse struct {
	User            *models.User            `json:"user"`
	WorkshopDetails *models.WorkshopDetails `json:"workshop_details"`
	AccessToken     string                  `json:"access_token"`
	RefreshToken    string                  `json:"refresh_token"`
	TokenType       string                  `json:"token_type"`
	ExpiresIn       int                     `json:"expires_in"` // in seconds
}

type LoginPendingResponse struct {
	RequiresTwoFactor bool   `json:"requires_two_factor"`
	UserNo            uint   `json:"user_no"`
	Message           string `json:"message"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ResetPasswordRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required"`
}

type VerifyTwoFactorRequest struct {
	UserNo uint   `json:"user_no" binding:"required"`
	Token  string `json:"token" binding:"required"`
}
