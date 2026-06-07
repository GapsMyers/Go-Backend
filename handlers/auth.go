package handlers

import (
	"backend/auth"
	"backend/middleware"
	"backend/models"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type authUserResponse struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	ProfilePhoto string `json:"profile_photo"`
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

	accessToken, refreshToken, err := h.issueTokens(user.ID, user.Email)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create tokens", nil)
		return
	}

	writeSuccess(c, http.StatusCreated, "register successful", gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    h.JWT.ExpiresInSeconds(),
		"user": authUserResponse{
			ID:           user.ID.String(),
			Email:        user.Email,
			ProfilePhoto: user.ProfilePhoto,
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

	accessToken, refreshToken, err := h.issueTokens(user.ID, user.Email)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create tokens", nil)
		return
	}

	writeSuccess(c, http.StatusOK, "login successful", gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    h.JWT.ExpiresInSeconds(),
		"user": authUserResponse{
			ID:           user.ID.String(),
			Email:        user.Email,
			ProfilePhoto: user.ProfilePhoto,
		},
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid refresh payload", err.Error())
		return
	}

	var storedToken models.RefreshToken
	if err := h.DB.Preload("User").Where("token = ?", req.RefreshToken).First(&storedToken).Error; err != nil {
		writeError(c, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired refresh token", nil)
		return
	}

	if storedToken.ExpiresAt.Before(time.Now()) {
		h.DB.Delete(&storedToken)
		writeError(c, http.StatusUnauthorized, "TOKEN_EXPIRED", "refresh token has expired", nil)
		return
	}

	// Issue new tokens (rotation)
	accessToken, newRefreshToken, err := h.issueTokens(storedToken.UserID, storedToken.User.Email)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to issue new tokens", nil)
		return
	}

	// Delete old refresh token
	h.DB.Delete(&storedToken)

	writeSuccess(c, http.StatusOK, "token refreshed", gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"token_type":    "Bearer",
		"expires_in":    h.JWT.ExpiresInSeconds(),
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid logout payload", err.Error())
		return
	}

	if err := h.DB.Where("token = ?", req.RefreshToken).Delete(&models.RefreshToken{}).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to logout", nil)
		return
	}

	writeSuccess(c, http.StatusOK, "logout successful", nil)
}

func (h *AuthHandler) issueTokens(userID uuid.UUID, email string) (string, string, error) {
	accessToken, err := h.JWT.GenerateToken(userID, email)
	if err != nil {
		return "", "", err
	}

	refreshTokenStr, err := h.JWT.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}

	refreshToken := models.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     refreshTokenStr,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days sliding session
	}

	if err := h.DB.Create(&refreshToken).Error; err != nil {
		return "", "", err
	}

	return accessToken, refreshTokenStr, nil
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authenticated user", nil)
		return
	}

	var user models.User
	if err := h.DB.Select("id", "email", "profile_photo").Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "user not found", nil)
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load user", nil)
		return
	}

	writeSuccess(c, http.StatusOK, "profile fetched", authUserResponse{
		ID:           user.ID.String(),
		Email:        user.Email,
		ProfilePhoto: user.ProfilePhoto,
	})
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", err.Error())
		return
	}

	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authenticated user", nil)
		return
	}

	var user models.User
	if err := h.DB.Select("id", "password_hash").Where("id = ?", userID).First(&user).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load user", nil)
		return
	}

	if !auth.VerifyPassword(req.OldPassword, user.PasswordHash) {
		writeError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "old password is incorrect", nil)
		return
	}

	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash new password", nil)
		return
	}

	if err := h.DB.Model(&user).Update("password_hash", hashedPassword).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update password", nil)
		return
	}

	writeSuccess(c, http.StatusOK, "password updated successfully", nil)
}

func (h *AuthHandler) UpdateProfilePhoto(c *gin.Context) {
	file, err := c.FormFile("profile_photo")
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "failed to get file", err.Error())
		return
	}

	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authenticated user", nil)
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		writeError(c, http.StatusBadRequest, "INVALID_FILE_TYPE", "only png, jpg, jpeg allowed", nil)
		return
	}

	uploadDir := "uploads/profiles"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create upload directory", err.Error())
		return
	}

	filename := uuid.New().String() + ext
	filePath := filepath.Join(uploadDir, filename)

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to save file", err.Error())
		return
	}

	photoURL := "/uploads/profiles/" + filename

	var user models.User
	user.ID = userID
	if err := h.DB.Model(&user).Update("profile_photo", photoURL).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update user profile photo", err.Error())
		return
	}

	writeSuccess(c, http.StatusOK, "profile photo updated successfully", gin.H{
		"profile_photo": photoURL,
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
