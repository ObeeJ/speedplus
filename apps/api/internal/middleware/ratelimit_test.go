package middleware

// Tests for the fixed-window Redis rate limiter added in b43056f. This
// guards auth/OTP endpoints against brute force, so getting the boundary
// wrong (off-by-one on the limit, or fail-open when it should fail-closed
// during a Redis outage) directly weakens the account-takeover defense.
//
// Uses a real Redis instance (REDIS_URL, defaulting to localhost:6379 same
// as local dev / docker-compose) rather than a mock — there is no existing
// miniredis/mock convention in this package to follow, and the Lua EVAL
// script this middleware depends on needs a real server to execute.
// Skips if Redis is unreachable, mirroring the DATABASE_URL skip pattern
// used for the Postgres-backed service tests.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func testRedis(t *testing.T) *redis.Client {
	t.Helper()
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379"
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		t.Fatalf("parse REDIS_URL: %v", err)
	}
	rdb := redis.NewClient(opt)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skip("redis not reachable - skipping rate limit tests")
	}
	return rdb
}

func rateLimitRouter(rdb *redis.Client, routeKey string, limit int, window time.Duration, failClosed ...bool) *gin.Engine {
	r := gin.New()
	r.GET("/probe", RateLimit(rdb, routeKey, limit, window, failClosed...), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func doProbe(r *gin.Engine) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.RemoteAddr = "203.0.113.7:5555" // fixed IP so every call in a test hits the same key
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRateLimit_AllowsWithinLimit(t *testing.T) {
	rdb := testRedis(t)
	routeKey := "test-allow-" + uuid.NewString()[:8]
	r := rateLimitRouter(rdb, routeKey, 3, time.Minute)
	defer rdb.Del(context.Background(), "rl:"+routeKey+":203.0.113.7")

	for i := 1; i <= 3; i++ {
		w := doProbe(r)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i, w.Code)
		}
	}
}

func TestRateLimit_BlocksOverLimit(t *testing.T) {
	rdb := testRedis(t)
	routeKey := "test-block-" + uuid.NewString()[:8]
	r := rateLimitRouter(rdb, routeKey, 2, time.Minute)
	defer rdb.Del(context.Background(), "rl:"+routeKey+":203.0.113.7")

	for i := 1; i <= 2; i++ {
		w := doProbe(r)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i, w.Code)
		}
	}
	w := doProbe(r)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("request 3 (over limit): status = %d, want 429", w.Code)
	}
	// The block must persist for subsequent requests within the window too,
	// not just the exact one that tipped over the limit.
	w2 := doProbe(r)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("request 4 (still over limit): status = %d, want 429", w2.Code)
	}
}

func TestRateLimit_SeparateKeysPerRoute(t *testing.T) {
	rdb := testRedis(t)
	routeA := "test-route-a-" + uuid.NewString()[:8]
	routeB := "test-route-b-" + uuid.NewString()[:8]
	defer rdb.Del(context.Background(), "rl:"+routeA+":203.0.113.7", "rl:"+routeB+":203.0.113.7")

	rA := rateLimitRouter(rdb, routeA, 1, time.Minute)
	rB := rateLimitRouter(rdb, routeB, 1, time.Minute)

	if w := doProbe(rA); w.Code != http.StatusOK {
		t.Fatalf("route A first request: status = %d, want 200", w.Code)
	}
	// Route A is now exhausted, but route B has its own independent budget.
	if w := doProbe(rB); w.Code != http.StatusOK {
		t.Fatalf("route B first request: status = %d, want 200 (independent limit from route A)", w.Code)
	}
	if w := doProbe(rA); w.Code != http.StatusTooManyRequests {
		t.Fatalf("route A second request: status = %d, want 429", w.Code)
	}
}

func TestRateLimit_FailClosedOnRedisError(t *testing.T) {
	// A client pointed at a port nothing listens on simulates a Redis outage
	// without needing to shut down the shared test Redis instance.
	badRdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 200 * time.Millisecond})
	routeKey := "test-failclosed-" + uuid.NewString()[:8]
	r := rateLimitRouter(badRdb, routeKey, 5, time.Minute, true)

	w := doProbe(r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("failClosed with Redis down: status = %d, want 503", w.Code)
	}
}

func TestRateLimit_FailOpenOnRedisErrorByDefault(t *testing.T) {
	// Without failClosed, a Redis outage must not lock legitimate users out
	// of the whole API — only auth/OTP endpoints opt into fail-closed.
	badRdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 200 * time.Millisecond})
	routeKey := "test-failopen-" + uuid.NewString()[:8]
	r := rateLimitRouter(badRdb, routeKey, 5, time.Minute)

	w := doProbe(r)
	if w.Code != http.StatusOK {
		t.Fatalf("fail-open with Redis down: status = %d, want 200", w.Code)
	}
}
