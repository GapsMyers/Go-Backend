package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS applies a minimal origin whitelist for browser-based clients.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowAll, allowedSet := buildOriginSet(allowedOrigins)

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		allowOrigin := ""

		if origin != "" && (allowAll || allowedSet[origin]) {
			allowOrigin = origin
		}

		if allowOrigin != "" {
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Origin", allowOrigin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func buildOriginSet(origins []string) (bool, map[string]bool) {
	set := make(map[string]bool, len(origins))
	allowAll := false

	for _, origin := range origins {
		o := strings.TrimSpace(origin)
		if o == "" {
			continue
		}
		if o == "*" {
			allowAll = true
			continue
		}
		set[o] = true
	}

	return allowAll, set
}
