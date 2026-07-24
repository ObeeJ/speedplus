package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/speedplus/api/internal/observability"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Attach a cloned Sentry hub to the request context so all
		// downstream CaptureError calls carry this request's trace.
		ctx := observability.SetRequestContext(c.Request.Context())
		c.Request = c.Request.WithContext(ctx)

		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		slog.InfoContext(ctx, "request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"latency_ms", time.Since(start).Milliseconds(),
			"request_id", c.GetString(RequestIDKey),
			"ip", c.ClientIP(),
		)
	}
}
