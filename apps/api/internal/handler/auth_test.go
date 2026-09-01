package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/service"
)

func init() { gin.SetMode(gin.TestMode) }

// ── stub ──────────────────────────────────────────────────────────────────────

type stubAuth struct {
	registerUser    *model.User
	registerAccess  string
	registerRefresh string
	registerErr     error

	loginUser    *model.User
	loginAccess  string
	loginRefresh string
	loginErr     error

	refreshUser        *model.User
	refreshAccess      string
	refreshRefresh     string
	refreshErr         error
	refreshCalledWith  string // capture for assertion

	verifyOTPUser    *model.User
	verifyOTPAccess  string
	verifyOTPRefresh string
	verifyOTPErr     error

	logoutErr error
}

func (s *stubAuth) Register(_ context.Context, _ service.RegisterInput) (*model.User, string, string, error) {
	return s.registerUser, s.registerAccess, s.registerRefresh, s.registerErr
}
func (s *stubAuth) Login(_ context.Context, _, _ string) (*model.User, string, string, error) {
	return s.loginUser, s.loginAccess, s.loginRefresh, s.loginErr
}
func (s *stubAuth) Refresh(_ context.Context, raw string) (*model.User, string, string, error) {
	s.refreshCalledWith = raw
	return s.refreshUser, s.refreshAccess, s.refreshRefresh, s.refreshErr
}
func (s *stubAuth) VerifyOTP(_ context.Context, _, _, _ string) (*model.User, string, string, error) {
	return s.verifyOTPUser, s.verifyOTPAccess, s.verifyOTPRefresh, s.verifyOTPErr
}
func (s *stubAuth) Logout(_ context.Context, _ string) error        { return s.logoutErr }
func (s *stubAuth) RequestOTP(_ context.Context, _, _ string) (string, error) { return "", nil }
func (s *stubAuth) SetPIN(_ context.Context, _ uuid.UUID, _ string) error     { return nil }
func (s *stubAuth) VerifyPIN(_ context.Context, _ uuid.UUID, _ string) error  { return nil }
func (s *stubAuth) ResetPassword(_ context.Context, _, _, _ string) error     { return nil }

// ── helpers ───────────────────────────────────────────────────────────────────

func authRouter(svc *stubAuth) *gin.Engine {
	r := gin.New()
	h := &AuthHandler{auth: svc}
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)
	r.POST("/logout", h.Logout)
	r.POST("/otp/verify", h.VerifyOTP)
	return r
}

func postJSON(r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b)))
	return w
}

// cookieVal reads a Set-Cookie header value by cookie name.
func cookieVal(w *httptest.ResponseRecorder, name string) string {
	for k, v := range w.Header() {
		if strings.EqualFold(k, "Set-Cookie") {
			for _, h := range v {
				if strings.Contains(h, name+"=") {
					parts := strings.Split(h, ";")
					for _, p := range parts {
						kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
						if len(kv) == 2 && kv[0] == name {
							return kv[1]
						}
					}
				}
			}
		}
	}
	return ""
}

// cookieMaxAge returns (maxAge, found). found=false means the cookie was not
// present at all in the response — distinguishable from MaxAge=0 (session).
func cookieMaxAge(w *httptest.ResponseRecorder, name string) (int, bool) {
	for k, v := range w.Header() {
		if strings.EqualFold(k, "Set-Cookie") {
			for _, h := range v {
				if strings.Contains(h, name+"=") {
					parts := strings.Split(h, ";")
					for _, p := range parts {
						kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
						if len(kv) == 2 && strings.EqualFold(kv[0], "Max-Age") {
							var age int
							fmt.Sscanf(kv[1], "%d", &age)
							return age, true
						}
					}
					// cookie present but no Max-Age directive = session cookie
					return 0, true
				}
			}
		}
	}
	return 0, false
}

func makeUser(role model.UserRole) *model.User {
	return &model.User{
		ID:           uuid.New(),
		Role:         role,
		FirstName:    "Test",
		LastName:     "User",
		Phone:        "+2348" + uuid.NewString()[:9],
		ReferralCode: "TEST" + uuid.NewString()[:4],
		IsVerified:   true,
		IsActive:     true,
	}
}

// ── Register ──────────────────────────────────────────────────────────────────

func TestAuthHandler_Register(t *testing.T) {
	t.Run("customer sets __role=customer and fourdat_refresh", func(t *testing.T) {
		user := makeUser(model.RoleCustomer)
		svc := &stubAuth{registerUser: user, registerAccess: "acc", registerRefresh: "ref"}
		w := postJSON(authRouter(svc), "/register", map[string]any{
			"firstName": "Test", "lastName": "User",
			"phone": "+2348000000001", "password": "password123",
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
		}
		if got := cookieVal(w, "__role"); got != "customer" {
			t.Errorf("__role = %q, want customer", got)
		}
		if got := cookieVal(w, "fourdat_refresh"); got != "ref" {
			t.Errorf("fourdat_refresh = %q, want ref", got)
		}
	})

	t.Run("driver sets __role=driver", func(t *testing.T) {
		user := makeUser(model.RoleDriver)
		svc := &stubAuth{registerUser: user, registerAccess: "acc", registerRefresh: "ref"}
		w := postJSON(authRouter(svc), "/register", map[string]any{
			"firstName": "Test", "lastName": "Driver",
			"phone": "+2348000000002", "password": "password123",
			"role": "driver", "vehicleType": "motorcycle", "vehiclePlate": "LAG-001",
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
		}
		if got := cookieVal(w, "__role"); got != "driver" {
			t.Errorf("__role = %q, want driver", got)
		}
	})

	t.Run("phone conflict returns 409 and no cookies", func(t *testing.T) {
		svc := &stubAuth{registerErr: service.ErrPhoneExists}
		w := postJSON(authRouter(svc), "/register", map[string]any{
			"firstName": "Test", "lastName": "User",
			"phone": "+2348000000001", "password": "password123",
		})
		if w.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", w.Code)
		}
		if got := cookieVal(w, "__role"); got != "" {
			t.Errorf("__role should not be set on error, got %q", got)
		}
	})

	t.Run("missing fields returns 400", func(t *testing.T) {
		svc := &stubAuth{}
		w := postJSON(authRouter(svc), "/register", map[string]any{"firstName": "only"})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

// ── Login ─────────────────────────────────────────────────────────────────────

func TestAuthHandler_Login(t *testing.T) {
	t.Run("success sets __role and fourdat_refresh", func(t *testing.T) {
		user := makeUser(model.RoleCustomer)
		svc := &stubAuth{loginUser: user, loginAccess: "acc", loginRefresh: "ref"}
		w := postJSON(authRouter(svc), "/login", map[string]any{
			"phone": "+2348000000001", "password": "password123",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
		if got := cookieVal(w, "__role"); got != "customer" {
			t.Errorf("__role = %q, want customer", got)
		}
		if got := cookieVal(w, "fourdat_refresh"); got != "ref" {
			t.Errorf("fourdat_refresh = %q, want ref", got)
		}
	})

	t.Run("merchant sets __role=merchant", func(t *testing.T) {
		user := makeUser(model.RoleMerchant)
		svc := &stubAuth{loginUser: user, loginAccess: "acc", loginRefresh: "ref"}
		w := postJSON(authRouter(svc), "/login", map[string]any{
			"phone": "+2348000000003", "password": "password123",
		})
		if got := cookieVal(w, "__role"); got != "merchant" {
			t.Errorf("__role = %q, want merchant", got)
		}
	})

	t.Run("invalid credentials returns 401 and no cookies", func(t *testing.T) {
		svc := &stubAuth{loginErr: service.ErrInvalidCredentials}
		w := postJSON(authRouter(svc), "/login", map[string]any{
			"phone": "+2348000000001", "password": "wrong",
		})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
		if got := cookieVal(w, "__role"); got != "" {
			t.Errorf("__role should not be set on failed login, got %q", got)
		}
		if got := cookieVal(w, "fourdat_refresh"); got != "" {
			t.Errorf("fourdat_refresh should not be set on failed login, got %q", got)
		}
	})

	t.Run("response body contains accessToken and user.role", func(t *testing.T) {
		user := makeUser(model.RoleDriver)
		svc := &stubAuth{loginUser: user, loginAccess: "the-token", loginRefresh: "ref"}
		w := postJSON(authRouter(svc), "/login", map[string]any{
			"phone": "+2348000000002", "password": "password123",
		})
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]any)
		if data["accessToken"] != "the-token" {
			t.Errorf("accessToken = %v, want the-token", data["accessToken"])
		}
		userBlock := data["user"].(map[string]any)
		if userBlock["role"] != "driver" {
			t.Errorf("user.role = %v, want driver", userBlock["role"])
		}
	})
}

// ── Logout ────────────────────────────────────────────────────────────────────

func TestAuthHandler_Logout(t *testing.T) {
	t.Run("clears __role and fourdat_refresh with MaxAge=-1", func(t *testing.T) {
		svc := &stubAuth{}
		req := httptest.NewRequest(http.MethodPost, "/logout", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "fourdat_refresh", Value: "some-token"})
		w := httptest.NewRecorder()
		authRouter(svc).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if age, found := cookieMaxAge(w, "__role"); !found || age != 0 {
			t.Errorf("__role MaxAge = %d found=%v, want 0 (expired)", age, found)
		}
		if age, found := cookieMaxAge(w, "fourdat_refresh"); !found || age != 0 {
			t.Errorf("fourdat_refresh MaxAge = %d found=%v, want 0 (expired)", age, found)
		}
	})

	t.Run("succeeds with no refresh token", func(t *testing.T) {
		svc := &stubAuth{}
		w := postJSON(authRouter(svc), "/logout", map[string]any{})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})
}

// ── Refresh ───────────────────────────────────────────────────────────────────

func TestAuthHandler_Refresh(t *testing.T) {
	t.Run("valid token rotates fourdat_refresh and re-sets __role", func(t *testing.T) {
		user := makeUser(model.RoleCustomer)
		svc := &stubAuth{
			refreshUser: user, refreshAccess: "new-acc", refreshRefresh: "new-ref",
		}
		req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "fourdat_refresh", Value: "old-ref"})
		w := httptest.NewRecorder()
		authRouter(svc).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
		// Verify the handler passed the cookie value to the service
		if svc.refreshCalledWith != "old-ref" {
			t.Errorf("Refresh called with %q, want old-ref", svc.refreshCalledWith)
		}
		if got := cookieVal(w, "fourdat_refresh"); got != "new-ref" {
			t.Errorf("fourdat_refresh = %q, want new-ref", got)
		}
		// __role must be re-set so Cloudflare routing survives token rotation
		if got := cookieVal(w, "__role"); got != "customer" {
			t.Errorf("__role = %q, want customer after refresh", got)
		}
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]any)
		if data["accessToken"] != "new-acc" {
			t.Errorf("accessToken = %v, want new-acc", data["accessToken"])
		}
	})

	t.Run("invalid token clears both cookies and returns 401", func(t *testing.T) {
		svc := &stubAuth{refreshErr: service.ErrTokenInvalid}
		req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "fourdat_refresh", Value: "bad"})
		w := httptest.NewRecorder()
		authRouter(svc).ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
		if age, found := cookieMaxAge(w, "__role"); !found || age != 0 {
			t.Errorf("__role MaxAge = %d found=%v, want 0 (expired) on failed refresh", age, found)
		}
		if age, found := cookieMaxAge(w, "fourdat_refresh"); !found || age != 0 {
			t.Errorf("fourdat_refresh MaxAge = %d found=%v, want 0 (expired) on failed refresh", age, found)
		}
	})

	t.Run("no token returns 400", func(t *testing.T) {
		svc := &stubAuth{}
		w := postJSON(authRouter(svc), "/refresh", map[string]any{})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

// ── VerifyOTP ─────────────────────────────────────────────────────────────────

func TestAuthHandler_VerifyOTP(t *testing.T) {
	t.Run("valid OTP sets __role matching user role", func(t *testing.T) {
		user := makeUser(model.RoleDriver)
		svc := &stubAuth{verifyOTPUser: user, verifyOTPAccess: "acc", verifyOTPRefresh: "ref"}
		w := postJSON(authRouter(svc), "/otp/verify", map[string]any{
			"phone": "+2348000000002", "otp": "123456", "purpose": "phone_verification",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
		if got := cookieVal(w, "__role"); got != "driver" {
			t.Errorf("__role = %q, want driver", got)
		}
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]any)
		if data["verified"] != true {
			t.Errorf("verified = %v, want true", data["verified"])
		}
	})

	t.Run("invalid OTP returns 422 and no cookies", func(t *testing.T) {
		svc := &stubAuth{verifyOTPErr: service.ErrOTPInvalid}
		w := postJSON(authRouter(svc), "/otp/verify", map[string]any{
			"phone": "+2348000000002", "otp": "000000", "purpose": "phone_verification",
		})
		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", w.Code)
		}
		if got := cookieVal(w, "__role"); got != "" {
			t.Errorf("__role should not be set on OTP failure, got %q", got)
		}
	})
}

// ── Full lifecycle: register → login → refresh → logout ──────────────────────

func TestAuthHandler_FullLifecycle(t *testing.T) {
	user := makeUser(model.RoleCustomer)
	svc := &stubAuth{
		registerUser: user, registerAccess: "reg-acc", registerRefresh: "reg-ref",
		loginUser: user, loginAccess: "login-acc", loginRefresh: "login-ref",
		refreshUser: user, refreshAccess: "new-acc", refreshRefresh: "new-ref",
	}
	r := authRouter(svc)

	// Step 1: Register
	w1 := postJSON(r, "/register", map[string]any{
		"firstName": "Full", "lastName": "Lifecycle",
		"phone": "+2348000000099", "password": "password123",
	})
	if w1.Code != http.StatusCreated {
		t.Fatalf("register: status = %d: %s", w1.Code, w1.Body.String())
	}
	if cookieVal(w1, "__role") != "customer" {
		t.Errorf("register: __role = %q, want customer", cookieVal(w1, "__role"))
	}

	// Step 2: Login
	w2 := postJSON(r, "/login", map[string]any{
		"phone": "+2348000000099", "password": "password123",
	})
	if w2.Code != http.StatusOK {
		t.Fatalf("login: status = %d: %s", w2.Code, w2.Body.String())
	}
	if cookieVal(w2, "__role") != "customer" {
		t.Errorf("login: __role = %q, want customer", cookieVal(w2, "__role"))
	}
	refreshToken := cookieVal(w2, "fourdat_refresh")
	if refreshToken == "" {
		t.Fatal("login: fourdat_refresh cookie not set")
	}

	// Step 3: Refresh — browser sends the cookie back
	req3 := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(`{}`))
	req3.Header.Set("Content-Type", "application/json")
	req3.AddCookie(&http.Cookie{Name: "fourdat_refresh", Value: refreshToken})
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("refresh: status = %d: %s", w3.Code, w3.Body.String())
	}
	if svc.refreshCalledWith != refreshToken {
		t.Errorf("refresh: service called with %q, want %q", svc.refreshCalledWith, refreshToken)
	}
	if cookieVal(w3, "fourdat_refresh") != "new-ref" {
		t.Errorf("refresh: fourdat_refresh = %q, want new-ref", cookieVal(w3, "fourdat_refresh"))
	}
	if cookieVal(w3, "__role") != "customer" {
		t.Errorf("refresh: __role = %q, want customer (must survive token rotation)", cookieVal(w3, "__role"))
	}

	// Step 4: Logout — both cookies must be expired
	req4 := httptest.NewRequest(http.MethodPost, "/logout", strings.NewReader(`{}`))
	req4.Header.Set("Content-Type", "application/json")
	req4.AddCookie(&http.Cookie{Name: "fourdat_refresh", Value: "new-ref"})
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Fatalf("logout: status = %d", w4.Code)
	}
	if age, found := cookieMaxAge(w4, "__role"); !found || age != 0 {
		t.Errorf("logout: __role MaxAge = %d found=%v, want 0 (expired)", age, found)
	}
	if age, found := cookieMaxAge(w4, "fourdat_refresh"); !found || age != 0 {
		t.Errorf("logout: fourdat_refresh MaxAge = %d found=%v, want 0 (expired)", age, found)
	}
}

// ── Cookie helper unit tests ──────────────────────────────────────────────────

func TestAuthHandler_CookieHelpers(t *testing.T) {
	t.Run("setRefreshTokenCookie sets fourdat_refresh with httpOnly", func(t *testing.T) {
		r := gin.New()
		r.POST("/test-cookie", func(c *gin.Context) {
			setRefreshTokenCookie(c, "test-refresh-token", false)
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
		w := postJSON(r, "/test-cookie", map[string]any{})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if got := cookieVal(w, "fourdat_refresh"); got != "test-refresh-token" {
			t.Fatalf("fourdat_refresh = %q, want test-refresh-token (headers: %v)", got, w.Header())
		}
	})

	t.Run("setRefreshTokenCookie with -1 maxAge expires cookie", func(t *testing.T) {
		r := gin.New()
		r.POST("/clear-cookie", func(c *gin.Context) {
			setRefreshTokenCookie(c, "", false, -1)
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
		w := postJSON(r, "/clear-cookie", map[string]any{})
		if age, found := cookieMaxAge(w, "fourdat_refresh"); !found || age != 0 {
			t.Fatalf("fourdat_refresh MaxAge = %d found=%v, want 0 (expired) (headers: %v)", age, found, w.Header())
		}
	})

	t.Run("setRoleCookie sets __role value", func(t *testing.T) {
		r := gin.New()
		r.POST("/set-role", func(c *gin.Context) {
			setRoleCookie(c, model.RoleDriver, false)
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
		w := postJSON(r, "/set-role", map[string]any{})
		if got := cookieVal(w, "__role"); got != "driver" {
			t.Errorf("__role = %q, want driver", got)
		}
	})

	t.Run("clearRoleCookie expires __role with MaxAge=-1", func(t *testing.T) {
		r := gin.New()
		r.POST("/clear-role", func(c *gin.Context) {
			clearRoleCookie(c, false)
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
		w := postJSON(r, "/clear-role", map[string]any{})
		if age, found := cookieMaxAge(w, "__role"); !found || age != 0 {
			t.Errorf("__role MaxAge = %d found=%v, want 0 (expired)", age, found)
		}
	})
}
