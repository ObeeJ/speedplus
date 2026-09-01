package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type cachedResponse struct {
	Status      int             `json:"status"`
	Body        json.RawMessage `json:"body"`
	BodyFingerprint string      `json:"fp"` // SHA-256 of original request body
}

const idemProcessingSentinel = "__processing__"
const idemProcessingWaitTTL = 30 * time.Second
const idemErrorTTL = 2 * time.Minute

// Idempotency enforces Idempotency-Key on money-moving POSTs.
// The key is bound to a SHA-256 fingerprint of the request body.
// If the same key arrives with a different body, 422 is returned —
// per Stripe/IETF idempotency semantics.
//
// strict=true: Redis unavailable → 503 (fail-closed). Use on all money paths.
// strict=false: Redis unavailable → pass-through (fail-open). Use on non-money POSTs.
func Idempotency(rdb *redis.Client, ttl time.Duration, strict ...bool) gin.HandlerFunc {
	failClosed := len(strict) > 0 && strict[0]
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "VALIDATION_ERROR",
					"message": "Idempotency-Key header is required",
					"field":   "Idempotency-Key",
				},
			})
			return
		}

		// Read and fingerprint the body, then restore it for the handler.
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		sum := sha256.Sum256(body)
		fingerprint := hex.EncodeToString(sum[:])

		// Scope the key to the authenticated user so two users sharing an
		// Idempotency-Key value can't collide (cross-user cached response leak).
		// All idempotency-protected routes sit behind Auth, so CtxUserID is set.
		userID := c.GetString(CtxUserID)
		if userID == "" {
			// Idempotency middleware must sit behind Auth. An empty userID means
			// Auth has not run yet — all callers would share the same key namespace,
			// letting any user replay another user's cached response.
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "An unexpected error occurred",
				},
			})
			return
		}
		redisKey := "idem:" + userID + ":" + key
		ctx := c.Request.Context()

		claimed, err := rdb.SetNX(ctx, redisKey, idemProcessingSentinel, idemProcessingWaitTTL).Result()
		if err != nil {
			if failClosed {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
					"error": gin.H{
						"code":    "SERVICE_UNAVAILABLE",
						"message": "Replay protection unavailable. Please retry.",
					},
				})
				return
			}
			c.Next()
			return
		}

		if !claimed {
			raw, getErr := rdb.Get(ctx, redisKey).Bytes()
			if getErr == redis.Nil {
				// Key expired between SetNX and Get. Re-attempt SetNX once.
				claimed, err = rdb.SetNX(ctx, redisKey, idemProcessingSentinel, idemProcessingWaitTTL).Result()
				if err == nil && !claimed {
					// Re-fetch raw if another request claimed it
					raw, getErr = rdb.Get(ctx, redisKey).Bytes()
				}
			}
			if claimed {
				// Successfully claimed on retry after expiration
			} else if getErr == nil && string(raw) != idemProcessingSentinel {
				var cached cachedResponse
				if json.Unmarshal(raw, &cached) == nil {
					// Body mismatch — same key, different payload: reject per IETF semantics
					if cached.BodyFingerprint != "" && cached.BodyFingerprint != fingerprint {
						c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
							"error": gin.H{
								"code":    "IDEMPOTENCY_CONFLICT",
								"message": "Idempotency-Key reused with a different request body",
							},
						})
						return
					}
					c.Data(cached.Status, "application/json", cached.Body)
					c.Abort()
					return
				}
			} else {
				c.AbortWithStatusJSON(http.StatusConflict, gin.H{
					"error": gin.H{
						"code":    "REQUEST_IN_PROGRESS",
						"message": "A request with this Idempotency-Key is already being processed",
					},
				})
				return
			}
		}

		w := &responseWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = w
		c.Next()

		resp := cachedResponse{Status: w.status, Body: w.body.Bytes(), BodyFingerprint: fingerprint}
		b, marshalErr := json.Marshal(resp)
		if marshalErr != nil {
			rdb.Del(ctx, redisKey)
			return
		}

		if w.status >= 200 && w.status < 300 {
			rdb.Set(ctx, redisKey, b, ttl)
		} else {
			rdb.Set(ctx, redisKey, b, idemErrorTTL)
		}
	}
}

type responseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *responseWriter) Write(b []byte) (int, error) {
	// FIX #6: capture status on first Write if WriteHeader was never called
	// explicitly (Gin calls Write before WriteHeader for c.JSON responses).
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
