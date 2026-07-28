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
//
// The window's expiry is set only on the first request of the window (when
// the counter goes 0→1). Setting it unconditionally on every increment would
// keep extending the TTL on each request, so a client sending requests faster
// than the window never gets the key to expire — turning a fixed window into
// an effectively permanent lockout under sustained traffic.
func RateLimit(rdb *redis.Client, routeKey string, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("rl:%s:%s", routeKey, c.ClientIP())
		ctx := c.Request.Context()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			// fail open — don't block on Redis errors
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
