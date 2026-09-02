package handler

import (
	"errors"
	"io"
	"net/http"

	"ai-ats-platform/backend/internal/middleware"
	"ai-ats-platform/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type signupRequest struct {
	CompanyName string `json:"company_name" binding:"required,min=1,max=255"`
	CompanySlug string `json:"company_slug" binding:"omitempty,max=255"`
	Email       string `json:"email" binding:"required,email,max=320"`
	Password    string `json:"password" binding:"required,min=8,max=72"`
	FirstName   string `json:"first_name" binding:"required,min=1,max=100"`
	LastName    string `json:"last_name" binding:"required,min=1,max=100"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req signupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", validationMessage(err))
		return
	}

	result, err := h.authService.Signup(c.Request.Context(), service.SignupInput{
		CompanyName: req.CompanyName,
		CompanySlug: req.CompanySlug,
		Email:       req.Email,
		Password:    req.Password,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
	})
	if err != nil {
		handleAuthError(c, err)
		return
	}

	respondSuccess(c, http.StatusCreated, result)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", validationMessage(err))
		return
	}

	result, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, result)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "invalid_token", "Invalid token context")
		return
	}

	user, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		handleAuthError(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"user": user})
}

func validationMessage(err error) string {
	if errors.Is(err, io.EOF) {
		return "Request body is required. Send JSON with Content-Type: application/json"
	}
	return err.Error()
}

func handleAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEmailAlreadyExists):
		respondError(c, http.StatusConflict, "email_exists", "Email is already registered")
	case errors.Is(err, service.ErrSlugAlreadyExists):
		respondError(c, http.StatusConflict, "slug_exists", "Company slug is already taken")
	case errors.Is(err, service.ErrInvalidCredentials):
		respondError(c, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
	case errors.Is(err, service.ErrAccountInactive):
		respondError(c, http.StatusForbidden, "account_inactive", "Account is inactive")
	default:
		respondError(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
	}
}
