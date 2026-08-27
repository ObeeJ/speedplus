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
	// Lua script: atomically increment and set TTL only on first call.
	// KEYS[1] = rate-limit key, ARGV[1] = window seconds.
	// Returns the new counter value.
	// Using a script eliminates the race between INCR and EXPIRE: if the
	// process dies after INCR but before EXPIRE the key would have no TTL
	// and permanently lock out the IP.
	const luaIncr = `
local v = redis.call('INCR', KEYS[1])
if v == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return v
`
	return func(c *gin.Context) {
		key := fmt.Sprintf("rl:%s:%s", routeKey, c.ClientIP())
		ctx := c.Request.Context()

		res, err := rdb.Eval(ctx, luaIncr, []string{key}, int(window.Seconds())).Int64()
		if err != nil {
			if closed {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
					"error": gin.H{"code": "SERVICE_UNAVAILABLE", "message": "Service temporarily unavailable. Please try again shortly."},
				})
				return
			}
			c.Next()
			return
		}
		count := res

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
