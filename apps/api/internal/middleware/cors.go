package middleware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if slices.Contains(allowedOrigins, origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, Idempotency-Key")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}

		// HSTS — 1 year, includeSubDomains
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		// CSP — tight policy for an API server (no HTML served, so default-src 'none' is safe)
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// AllowedOrigins builds the list from a comma-separated env string.
func AllowedOrigins(raw string) []string {
	var out []string
	for _, o := range strings.Split(raw, ",") {
		if s := strings.TrimSpace(o); s != "" {
			out = append(out, s)
		}
	}
	return out
}
