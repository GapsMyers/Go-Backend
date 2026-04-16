package middleware

import (
	"backend/auth"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const userIDContextKey = "user_id"

// JWTAuth verifies JWT access tokens and injects user ID into request context.
func JWTAuth(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			writeAuthError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeAuthError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization format")
			return
		}

		token := strings.TrimSpace(parts[1])
		if token == "" {
			writeAuthError(c, http.StatusUnauthorized, "UNAUTHORIZED", "empty bearer token")
			return
		}

		claims, err := jwtService.ValidateToken(token)
		if err != nil {
			writeAuthError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
			return
		}

		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			writeAuthError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token subject")
			return
		}

		c.Set(userIDContextKey, userID)
		c.Next()
	}
}

// UserIDFromContext returns the authenticated user ID from Gin context.
func UserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	value, exists := c.Get(userIDContextKey)
	if !exists {
		return uuid.Nil, errors.New("authenticated user not found in context")
	}

	userID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("invalid authenticated user type")
	}

	if userID == uuid.Nil {
		return uuid.Nil, errors.New("authenticated user is empty")
	}

	return userID, nil
}

func writeAuthError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
