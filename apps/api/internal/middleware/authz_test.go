package middleware

// Authorization-gate tests.
//
// These three middlewares stand in front of every protected route in the API —
// RequireRole guards the admin, driver, merchant and customer groups,
// RequireVerified guards order creation and every wallet money movement, and
// RequireActiveUser is the only thing stopping a suspended account from acting
// on its still-valid access token (JWTs are stateless, so revoking refresh
// tokens alone locks nobody out until the access token expires).
//
// Before this file the package sat at 5.9% coverage with none of them tested.
// A regression here would fail no other test in the repo; it would simply open
// every route it protects.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
)

func init() { gin.SetMode(gin.TestMode) }

// stubUserFinder satisfies the unexported userFinder interface that
// RequireActiveUser depends on, without touching a database.
type stubUserFinder struct {
	user *model.User
	err  error
}

func (s *stubUserFinder) FindByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

// runChain builds a single-route router, seeds the context values Auth would
// normally set from the JWT, and reports the resulting status.
func runChain(seed func(c *gin.Context), mw gin.HandlerFunc) *httptest.ResponseRecorder {
	r := gin.New()
	r.GET("/probe", func(c *gin.Context) { seed(c); c.Next() }, mw, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/probe", nil))
	return w
}

// ── RequireRole ───────────────────────────────────────────────────────────────

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		allowed  []string
		wantCode int
	}{
		{name: "matching single role passes", actual: "admin", allowed: []string{"admin"}, wantCode: http.StatusOK},
		{name: "one of several allowed roles passes", actual: "driver", allowed: []string{"admin", "driver"}, wantCode: http.StatusOK},
		{name: "wrong role is rejected", actual: "customer", allowed: []string{"admin"}, wantCode: http.StatusForbidden},
		{name: "driver cannot reach an admin route", actual: "driver", allowed: []string{"admin"}, wantCode: http.StatusForbidden},
		{name: "customer cannot reach a merchant route", actual: "customer", allowed: []string{"merchant"}, wantCode: http.StatusForbidden},
		{name: "empty role is rejected", actual: "", allowed: []string{"admin"}, wantCode: http.StatusForbidden},
		// Role comparison must be exact — a prefix or case variant must not pass.
		{name: "role is matched exactly, not by prefix", actual: "admins", allowed: []string{"admin"}, wantCode: http.StatusForbidden},
		{name: "role match is case sensitive", actual: "Admin", allowed: []string{"admin"}, wantCode: http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := runChain(
				func(c *gin.Context) { c.Set(CtxUserRole, tc.actual) },
				RequireRole(tc.allowed...),
			)
			if w.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d (role %q against %v)", w.Code, tc.wantCode, tc.actual, tc.allowed)
			}
		})
	}
}

// ── RequireVerified ───────────────────────────────────────────────────────────

func TestRequireVerified(t *testing.T) {
	t.Run("verified user passes", func(t *testing.T) {
		w := runChain(func(c *gin.Context) { c.Set(CtxIsVerified, true) }, RequireVerified())
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("unverified user is blocked", func(t *testing.T) {
		w := runChain(func(c *gin.Context) { c.Set(CtxIsVerified, false) }, RequireVerified())
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	// A missing flag must NOT read as verified. GetBool returns false for an
	// absent key, so this is fail-closed — pinned here because the opposite
	// would silently open wallet funding and transfers.
	t.Run("absent verification flag fails closed", func(t *testing.T) {
		w := runChain(func(c *gin.Context) {}, RequireVerified())
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403 — a missing flag must never be treated as verified", w.Code)
		}
	})
}

// ── RequireActiveUser ─────────────────────────────────────────────────────────

func TestRequireActiveUser(t *testing.T) {
	activeID := uuid.New()

	tests := []struct {
		name     string
		userID   string
		finder   *stubUserFinder
		wantCode int
	}{
		{
			name:     "active user passes",
			userID:   activeID.String(),
			finder:   &stubUserFinder{user: &model.User{ID: activeID, IsActive: true}},
			wantCode: http.StatusOK,
		},
		{
			// The whole point of this middleware: a suspended account still
			// holding a valid access token must be stopped on the next request.
			name:     "deactivated user is blocked despite a valid token",
			userID:   activeID.String(),
			finder:   &stubUserFinder{user: &model.User{ID: activeID, IsActive: false}},
			wantCode: http.StatusForbidden,
		},
		{
			name:     "lookup failure fails closed",
			userID:   activeID.String(),
			finder:   &stubUserFinder{err: errors.New("connection refused")},
			wantCode: http.StatusForbidden,
		},
		{
			name:     "unparseable user id is unauthorized",
			userID:   "not-a-uuid",
			finder:   &stubUserFinder{user: &model.User{ID: activeID, IsActive: true}},
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "missing user id is unauthorized",
			userID:   "",
			finder:   &stubUserFinder{user: &model.User{ID: activeID, IsActive: true}},
			wantCode: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := runChain(
				func(c *gin.Context) { c.Set(CtxUserID, tc.userID) },
				RequireActiveUser(tc.finder),
			)
			if w.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", w.Code, tc.wantCode)
			}
		})
	}
}

// A rejected request must never reach the handler. Writing a status but letting
// the chain continue would run the money-moving handler anyway.
func TestAuthzMiddleware_AbortsBeforeHandler(t *testing.T) {
	cases := []struct {
		name string
		seed func(*gin.Context)
		mw   gin.HandlerFunc
	}{
		{
			name: "RequireRole",
			seed: func(c *gin.Context) { c.Set(CtxUserRole, "customer") },
			mw:   RequireRole("admin"),
		},
		{
			name: "RequireVerified",
			seed: func(c *gin.Context) { c.Set(CtxIsVerified, false) },
			mw:   RequireVerified(),
		},
		{
			name: "RequireActiveUser",
			seed: func(c *gin.Context) { c.Set(CtxUserID, uuid.NewString()) },
			mw:   RequireActiveUser(&stubUserFinder{user: &model.User{IsActive: false}}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reached := false
			r := gin.New()
			r.GET("/probe", func(c *gin.Context) { tc.seed(c); c.Next() }, tc.mw, func(c *gin.Context) {
				reached = true
				c.Status(http.StatusOK)
			})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/probe", nil))

			if reached {
				t.Fatal("handler executed despite the authorization gate rejecting the request")
			}
			if w.Code == http.StatusOK {
				t.Fatal("status = 200 on a rejected request")
			}
		})
	}
}
