package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimit returns a fixed-window limiter: `limit` requests per `window`.
// Key is scoped to IP + routeKey so /auth/* and /otp/* can have tighter limits.
// failClosed: when true, a Redis error returns 503 instead of allowing the request.
// Set failClosed=true for auth/OTP endpoints to prevent brute-force during Redis outages.
func RateLimit(rdb *redis.Client, routeKey string, limit int, window time.Duration, failClosed ...bool) gin.HandlerFunc {
	closed := len(failClosed) > 0 && failClosed[0]
	return func(c *gin.Context) {
		key := fmt.Sprintf("rl:%s:%s", routeKey, c.ClientIP())
		ctx := c.Request.Context()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			if closed {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
					"error": gin.H{"code": "SERVICE_UNAVAILABLE", "message": "Service temporarily unavailable. Please try again shortly."},
				})
				return
			}
			// fail open for non-critical endpoints
			c.Next()
			return
		}
		if count == 1 {
			rdb.Expire(ctx, key, window)
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		remaining := int64(limit) - count
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		if count > int64(limit) {
			ttl, _ := rdb.TTL(ctx, key).Result()
			if ttl > 0 {
				c.Header("Retry-After", fmt.Sprintf("%.0f", ttl.Seconds()))
			}
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "Too many requests. Please slow down.",
				},
			})
			return
		}
		c.Next()
	}
}
