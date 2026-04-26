package handlers

import (
	"backend/auth"
	"backend/middleware"
	"backend/models"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB  *gorm.DB
	JWT *auth.JWTService
}

func NewAuthHandler(db *gorm.DB, jwtService *auth.JWTService) *AuthHandler {
	return &AuthHandler{
		DB:  db,
		JWT: jwtService,
	}
}

type registerRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type authUserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid register payload", err.Error())
		return
	}

	email := normalizeEmail(req.Email)

	var existing models.User
	err := h.DB.Select("id").Where("email = ?", email).First(&existing).Error
	if err == nil {
		writeError(c, http.StatusConflict, "EMAIL_EXISTS", "email already registered", nil)
		return
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to check existing user", nil)
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password", nil)
		return
	}

	user := models.User{
		Email:        email,
		PasswordHash: hashedPassword,
	}
	if err := h.DB.Create(&user).Error; err != nil {
		if isDuplicateError(err) {
			writeError(c, http.StatusConflict, "EMAIL_EXISTS", "email already registered", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create user", nil)
		return
	}

	token, err := h.JWT.GenerateToken(user.ID, user.Email)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create access token", nil)
		return
	}

	writeSuccess(c, http.StatusCreated, "register successful", gin.H{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   h.JWT.ExpiresInSeconds(),
		"user": authUserResponse{
			ID:    user.ID.String(),
			Email: user.Email,
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid login payload", err.Error())
		return
	}

	email := normalizeEmail(req.Email)

	var user models.User
	if err := h.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "email or password is incorrect", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load user", nil)
		return
	}

	if !auth.VerifyPassword(req.Password, user.PasswordHash) {
		writeError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "email or password is incorrect", nil)
		return
	}

	token, err := h.JWT.GenerateToken(user.ID, user.Email)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create access token", nil)
		return
	}

	writeSuccess(c, http.StatusOK, "login successful", gin.H{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   h.JWT.ExpiresInSeconds(),
		"user": authUserResponse{
			ID:    user.ID.String(),
			Email: user.Email,
		},
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authenticated user", nil)
		return
	}

	var user models.User
	if err := h.DB.Select("id", "email").Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "user not found", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load user", nil)
		return
	}

	writeSuccess(c, http.StatusOK, "profile fetched", authUserResponse{
		ID:    user.ID.String(),
		Email: user.Email,
	})
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "duplicate key") || strings.Contains(errStr, "unique constraint")
}
