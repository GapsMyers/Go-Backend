package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS applies a minimal origin whitelist for browser-based clients.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowAll, allowedSet, wildcardPrefixes := buildOriginSet(allowedOrigins)

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		allowOrigin := ""

		if origin != "" && originAllowed(origin, allowAll, allowedSet, wildcardPrefixes) {
			allowOrigin = origin
		}

		if allowOrigin != "" {
			c.Header("Vary", "Origin,Access-Control-Request-Method,Access-Control-Request-Headers")
			c.Header("Access-Control-Allow-Origin", allowOrigin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")

			allowHeaders := strings.TrimSpace(c.GetHeader("Access-Control-Request-Headers"))
			if allowHeaders == "" {
				allowHeaders = "Authorization,Content-Type,Accept,Origin,X-Requested-With"
			}
			c.Header("Access-Control-Allow-Headers", allowHeaders)
		}

		if c.Request.Method == http.MethodOptions {
			if origin != "" && allowOrigin == "" {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func buildOriginSet(origins []string) (bool, map[string]bool, []string) {
	set := make(map[string]bool, len(origins))
	wildcardPrefixes := make([]string, 0, len(origins))
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
		if strings.HasSuffix(o, "*") {
			wildcardPrefixes = append(wildcardPrefixes, strings.TrimSuffix(o, "*"))
			continue
		}
		set[o] = true
	}

	return allowAll, set, wildcardPrefixes
}

func originAllowed(origin string, allowAll bool, allowedSet map[string]bool, wildcardPrefixes []string) bool {
	if allowAll || allowedSet[origin] {
		return true
	}

	for _, prefix := range wildcardPrefixes {
		if strings.HasPrefix(origin, prefix) {
			return true
		}
	}

	return false
}
