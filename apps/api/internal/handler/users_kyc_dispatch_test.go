package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/service"
)

// ── UsersHandler stubs ────────────────────────────────────────────────────────

type stubUserRepoFull struct {
	user            *model.User
	addresses       []model.Address
	driverProfile   *model.DriverProfile
	merchantProfile *model.MerchantProfile
	err             error
}

func (s *stubUserRepoFull) PhoneByID(_ context.Context, _ uuid.UUID) (string, error) { return "", nil }
func (s *stubUserRepoFull) Create(_ context.Context, _ *model.User) error              { return s.err }
func (s *stubUserRepoFull) FindByPhone(_ context.Context, _ string) (*model.User, error) {
	return s.user, s.err
}
func (s *stubUserRepoFull) FindByUsername(_ context.Context, _ string) (*model.User, error) {
	return s.user, s.err
}
func (s *stubUserRepoFull) FindByReferralCode(_ context.Context, _ string) (*model.User, error) {
	return s.user, s.err
}
func (s *stubUserRepoFull) CreateRefreshToken(_ context.Context, _ *model.RefreshToken) error {
	return nil
}
func (s *stubUserRepoFull) FindRefreshToken(_ context.Context, _ string) (*model.RefreshToken, error) {
	return nil, nil
}
func (s *stubUserRepoFull) FindRefreshTokenAny(_ context.Context, _ string) (*model.RefreshToken, error) {
	return nil, nil
}
func (s *stubUserRepoFull) RevokeRefreshToken(_ context.Context, _ string, _ time.Time) error {
	return nil
}
func (s *stubUserRepoFull) RevokeRefreshFamily(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (s *stubUserRepoFull) CreateOTP(_ context.Context, _ *model.OTPCode) error { return nil }
func (s *stubUserRepoFull) InvalidatePreviousOTPs(_ context.Context, _, _ string) error {
	return nil
}
func (s *stubUserRepoFull) FindActiveOTP(_ context.Context, _, _ string) (*model.OTPCode, error) {
	return nil, nil
}
func (s *stubUserRepoFull) MarkOTPUsed(_ context.Context, _ uuid.UUID, _ time.Time) error { return nil }
func (s *stubUserRepoFull) UpsertPIN(_ context.Context, _ uuid.UUID, _ string) error     { return nil }
func (s *stubUserRepoFull) FindPIN(_ context.Context, _ uuid.UUID) (*model.PIN, error)   { return nil, nil }
func (s *stubUserRepoFull) IncrementPINFailure(_ context.Context, _ uuid.UUID, _ *time.Time) error {
	return nil
}
func (s *stubUserRepoFull) ResetPINFailures(_ context.Context, _ uuid.UUID) error { return nil }
func (s *stubUserRepoFull) FindDriverBankAccount(_ context.Context, _ uuid.UUID) (*model.DriverBankAccount, error) {
	return nil, nil
}
func (s *stubUserRepoFull) UpsertDriverBankAccount(_ context.Context, _ *model.DriverBankAccount) error {
	return nil
}

func (s *stubUserRepoFull) FindByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return s.user, s.err
}
func (s *stubUserRepoFull) Update(_ context.Context, _ *model.User) error { return s.err }
func (s *stubUserRepoFull) ListAddresses(_ context.Context, _ uuid.UUID) ([]model.Address, error) {
	return s.addresses, s.err
}
func (s *stubUserRepoFull) CreateAddress(_ context.Context, _ *model.Address) error { return s.err }
func (s *stubUserRepoFull) FindDriverProfile(_ context.Context, _ uuid.UUID) (*model.DriverProfile, error) {
	return s.driverProfile, s.err
}
func (s *stubUserRepoFull) FindMerchantProfile(_ context.Context, _ uuid.UUID) (*model.MerchantProfile, error) {
	return s.merchantProfile, s.err
}

func (s *stubUserRepoFull) FindAddress(_ context.Context, _ uuid.UUID) (*model.Address, error) {
	return nil, s.err
}
func (s *stubUserRepoFull) CreateDriverProfile(_ context.Context, _ *model.DriverProfile) error {
	return s.err
}
func (s *stubUserRepoFull) UpdateDriverProfile(_ context.Context, _ *model.DriverProfile) error {
	return s.err
}
func (s *stubUserRepoFull) CreateMerchantProfile(_ context.Context, _ *model.MerchantProfile) error {
	return s.err
}
func (s *stubUserRepoFull) UpdateMerchantProfile(_ context.Context, _ *model.MerchantProfile) error {
	return s.err
}

func usersRouter(repo *stubUserRepoFull) *gin.Engine {
	r := gin.New()
	h := NewUsersHandler(repo)
	r.GET("/users/me", seedCtx("customer"), h.Me)
	r.PUT("/users/me", seedCtx("customer"), h.UpdateMe)
	r.GET("/users/me/addresses", seedCtx("customer"), h.ListAddresses)
	r.POST("/users/me/addresses", seedCtx("customer"), h.CreateAddress)
	r.GET("/users/me/driver-profile", seedCtx("driver"), h.GetDriverProfile)
	r.GET("/users/me/merchant-profile", seedCtx("merchant"), h.GetMerchantProfile)
	return r
}

func makeTestUser() *model.User {
	return &model.User{
		ID: uuid.New(), Role: model.RoleCustomer,
		FirstName: "Ada", LastName: "Obi", Phone: "08012345678",
		IsVerified: true, IsActive: true, CreatedAt: time.Now(),
	}
}

func TestUsersHandler_Me(t *testing.T) {
	t.Run("returns user", func(t *testing.T) {
		r := usersRouter(&stubUserRepoFull{user: makeTestUser()})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/me", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]any)
		if data["firstName"] != "Ada" {
			t.Errorf("firstName = %v, want Ada", data["firstName"])
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		r := usersRouter(&stubUserRepoFull{err: errors.New("not found")})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/me", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})
}

func TestUsersHandler_UpdateMe(t *testing.T) {
	t.Run("updates and returns user", func(t *testing.T) {
		r := usersRouter(&stubUserRepoFull{user: makeTestUser()})
		b, _ := json.Marshal(map[string]any{"firstName": "Chidi"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewReader(b)))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
	})

	t.Run("user not found returns 404", func(t *testing.T) {
		r := usersRouter(&stubUserRepoFull{err: errors.New("not found")})
		b, _ := json.Marshal(map[string]any{"firstName": "X"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewReader(b)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})
}

func TestUsersHandler_ListAddresses(t *testing.T) {
	t.Run("returns addresses", func(t *testing.T) {
		addr := model.Address{ID: uuid.New(), Street: "1 Test St", City: "Lagos"}
		r := usersRouter(&stubUserRepoFull{user: makeTestUser(), addresses: []model.Address{addr}})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/me/addresses", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("service error returns 500", func(t *testing.T) {
		r := usersRouter(&stubUserRepoFull{err: errors.New("db error")})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/me/addresses", nil))
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", w.Code)
		}
	})
}

func TestUsersHandler_CreateAddress(t *testing.T) {
	t.Run("creates and returns 201", func(t *testing.T) {
		r := usersRouter(&stubUserRepoFull{user: makeTestUser()})
		b, _ := json.Marshal(map[string]any{
			"street": "1 Test St", "city": "Lagos", "state": "Lagos",
			"lat": 6.5, "lng": 3.3,
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/users/me/addresses", bytes.NewReader(b)))
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
		}
	})

	t.Run("missing required fields returns 400", func(t *testing.T) {
		r := usersRouter(&stubUserRepoFull{user: makeTestUser()})
		b, _ := json.Marshal(map[string]any{"city": "Lagos"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/users/me/addresses", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestUsersHandler_GetDriverProfile(t *testing.T) {
	t.Run("returns driver profile", func(t *testing.T) {
		dp := &model.DriverProfile{ID: uuid.New(), VehicleType: model.VehicleMotorcycle}
		r := usersRouter(&stubUserRepoFull{driverProfile: dp})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/me/driver-profile", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		r := usersRouter(&stubUserRepoFull{err: errors.New("not found")})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/me/driver-profile", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})
}

func TestUsersHandler_GetMerchantProfile(t *testing.T) {
	t.Run("returns merchant profile", func(t *testing.T) {
		mp := &model.MerchantProfile{BusinessName: "Gas Co"}
		r := usersRouter(&stubUserRepoFull{merchantProfile: mp})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/me/merchant-profile", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		r := usersRouter(&stubUserRepoFull{err: errors.New("not found")})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/me/merchant-profile", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
	})
}

// ── KYCHandler stubs ──────────────────────────────────────────────────────────

type stubKYC struct {
	checks []model.KYCCheck
	err    error
}

func (s *stubKYC) SubmitCheck(_ context.Context, _ uuid.UUID, _ model.KYCDocType, _ map[string]string) error {
	return s.err
}
func (s *stubKYC) QueueForAdmin(_ context.Context, _, _ int) ([]model.KYCCheck, error) {
	return s.checks, s.err
}
func (s *stubKYC) AdminApprove(_ context.Context, _, _ uuid.UUID, _ string) error { return s.err }
func (s *stubKYC) AdminReject(_ context.Context, _, _ uuid.UUID, _ string) error  { return s.err }
func (s *stubKYC) GetUserKYC(_ context.Context, _ uuid.UUID) ([]model.KYCCheck, error) {
	return s.checks, s.err
}

func kycRouter(svc *stubKYC) *gin.Engine {
	r := gin.New()
	h := &KYCHandler{kyc: svc}
	r.POST("/kyc/check", seedCtx("customer"), h.SubmitCheck)
	r.GET("/kyc/status", seedCtx("customer"), h.MyKYCStatus)
	r.GET("/admin/kyc/queue", seedCtx("admin"), h.AdminQueue)
	r.POST("/admin/kyc/:id/approve", seedCtx("admin"), h.Approve)
	r.POST("/admin/kyc/:id/reject", seedCtx("admin"), h.Reject)
	r.GET("/admin/users/:id/kyc", seedCtx("admin"), h.AdminGetUserKYC)
	return r
}

func TestKYCHandler_SubmitCheck(t *testing.T) {
	t.Run("success returns 202", func(t *testing.T) {
		r := kycRouter(&stubKYC{})
		b, _ := json.Marshal(map[string]any{"docType": "bvn", "params": map[string]string{"bvn": "12345678901"}})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/kyc/check", bytes.NewReader(b)))
		if w.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want 202: %s", w.Code, w.Body.String())
		}
	})

	t.Run("service error returns 422", func(t *testing.T) {
		r := kycRouter(&stubKYC{err: errors.New("invalid bvn")})
		b, _ := json.Marshal(map[string]any{"docType": "bvn", "params": map[string]string{"bvn": "bad"}})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/kyc/check", bytes.NewReader(b)))
		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", w.Code)
		}
	})

	t.Run("missing docType returns 400", func(t *testing.T) {
		r := kycRouter(&stubKYC{})
		b, _ := json.Marshal(map[string]any{"params": map[string]string{}})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/kyc/check", bytes.NewReader(b)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestKYCHandler_MyKYCStatus(t *testing.T) {
	t.Run("returns checks", func(t *testing.T) {
		r := kycRouter(&stubKYC{checks: []model.KYCCheck{{ID: uuid.New(), DocType: "bvn"}}})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/kyc/status", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})
}

func TestKYCHandler_AdminQueue(t *testing.T) {
	t.Run("returns queue", func(t *testing.T) {
		r := kycRouter(&stubKYC{checks: []model.KYCCheck{{ID: uuid.New()}}})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/kyc/queue", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("service error returns 500", func(t *testing.T) {
		r := kycRouter(&stubKYC{err: errors.New("db error")})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/kyc/queue", nil))
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", w.Code)
		}
	})
}

func TestKYCHandler_Approve(t *testing.T) {
	t.Run("success returns 200", func(t *testing.T) {
		r := kycRouter(&stubKYC{})
		b, _ := json.Marshal(map[string]any{"note": "looks good"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/kyc/"+uuid.NewString()+"/approve", bytes.NewReader(b)))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		r := kycRouter(&stubKYC{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/kyc/bad-id/approve", bytes.NewReader([]byte(`{}`)))) 
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestKYCHandler_Reject(t *testing.T) {
	t.Run("success returns 200", func(t *testing.T) {
		r := kycRouter(&stubKYC{})
		b, _ := json.Marshal(map[string]any{"note": "blurry image"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/kyc/"+uuid.NewString()+"/reject", bytes.NewReader(b)))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("missing note returns 400", func(t *testing.T) {
		r := kycRouter(&stubKYC{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/kyc/"+uuid.NewString()+"/reject", bytes.NewReader([]byte(`{}`))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestKYCHandler_AdminGetUserKYC(t *testing.T) {
	t.Run("returns user KYC history", func(t *testing.T) {
		r := kycRouter(&stubKYC{checks: []model.KYCCheck{{ID: uuid.New(), DocType: "nin"}}})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/users/"+uuid.NewString()+"/kyc", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("invalid user UUID returns 400", func(t *testing.T) {
		r := kycRouter(&stubKYC{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/users/bad-id/kyc", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

// ── DispatchHandler stubs ─────────────────────────────────────────────────────

type stubDispatch struct {
	err error
}

func (s *stubDispatch) SetOnline(_ context.Context, _ uuid.UUID, _ bool) error    { return s.err }
func (s *stubDispatch) SetOffline(_ context.Context, _ uuid.UUID) error            { return s.err }
func (s *stubDispatch) UpdateLocation(_ context.Context, _ uuid.UUID, _, _, _ float64) error {
	return s.err
}
func (s *stubDispatch) AcceptOffer(_ context.Context, _, _ uuid.UUID) error  { return s.err }
func (s *stubDispatch) DeclineOffer(_ context.Context, _, _ uuid.UUID) error { return s.err }
func (s *stubDispatch) RejectOffer(_ context.Context, _, _ uuid.UUID) error  { return s.err }
func (s *stubDispatch) ManualAssign(_ context.Context, _, _ uuid.UUID) error { return s.err }

func dispatchRouter(svc *stubDispatch) *gin.Engine {
	r := gin.New()
	h := &DispatchHandler{dispatch: svc}
	r.PATCH("/drivers/online", seedCtx("driver"), h.SetOnline)
	r.POST("/drivers/location", seedCtx("driver"), h.UpdateLocation)
	r.POST("/drivers/offers/:id/accept", seedCtx("driver"), h.AcceptOffer)
	r.POST("/drivers/offers/:id/reject", seedCtx("driver"), h.RejectOffer)
	r.POST("/admin/dispatch/:orderId/assign", seedCtx("admin"), h.AdminAssign)
	return r
}

func TestDispatchHandler_SetOnline(t *testing.T) {
	t.Run("success returns 200", func(t *testing.T) {
		r := dispatchRouter(&stubDispatch{})
		b, _ := json.Marshal(map[string]any{"online": true})
		req := httptest.NewRequest(http.MethodPatch, "/drivers/online", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
	})

	t.Run("service error returns 500", func(t *testing.T) {
		r := dispatchRouter(&stubDispatch{err: errors.New("db error")})
		b, _ := json.Marshal(map[string]any{"online": true})
		req := httptest.NewRequest(http.MethodPatch, "/drivers/online", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", w.Code)
		}
	})
}

func TestDispatchHandler_UpdateLocation(t *testing.T) {
	t.Run("success returns 200", func(t *testing.T) {
		r := dispatchRouter(&stubDispatch{})
		b, _ := json.Marshal(map[string]any{"lat": 6.5, "lng": 3.3})
		req := httptest.NewRequest(http.MethodPost, "/drivers/location", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
	})

	t.Run("missing lat/lng returns 400", func(t *testing.T) {
		r := dispatchRouter(&stubDispatch{})
		b, _ := json.Marshal(map[string]any{"lat": 6.5})
		req := httptest.NewRequest(http.MethodPost, "/drivers/location", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestDispatchHandler_AcceptOffer(t *testing.T) {
	t.Run("success returns 200", func(t *testing.T) {
		r := dispatchRouter(&stubDispatch{})
		req := httptest.NewRequest(http.MethodPost, "/drivers/offers/"+uuid.NewString()+"/accept", bytes.NewReader([]byte(`{}`)))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
	})

	t.Run("conflict returns 409", func(t *testing.T) {
		r := dispatchRouter(&stubDispatch{err: errors.New("offer expired")})
		req := httptest.NewRequest(http.MethodPost, "/drivers/offers/"+uuid.NewString()+"/accept", bytes.NewReader([]byte(`{}`)))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", w.Code)
		}
	})

	t.Run("invalid offer UUID returns 400", func(t *testing.T) {
		r := dispatchRouter(&stubDispatch{})
		req := httptest.NewRequest(http.MethodPost, "/drivers/offers/bad-id/accept", bytes.NewReader([]byte(`{}`)))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})
}

func TestDispatchHandler_RejectOffer(t *testing.T) {
	t.Run("success returns 200", func(t *testing.T) {
		r := dispatchRouter(&stubDispatch{})
		req := httptest.NewRequest(http.MethodPost, "/drivers/offers/"+uuid.NewString()+"/reject", bytes.NewReader([]byte(`{}`)))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})
}

func TestDispatchHandler_AdminAssign(t *testing.T) {
	t.Run("success returns 200", func(t *testing.T) {
		r := dispatchRouter(&stubDispatch{})
		b, _ := json.Marshal(map[string]any{"driverId": uuid.NewString()})
		req := httptest.NewRequest(http.MethodPost, "/admin/dispatch/"+uuid.NewString()+"/assign", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid order UUID returns 400", func(t *testing.T) {
		r := dispatchRouter(&stubDispatch{})
		b, _ := json.Marshal(map[string]any{"driverId": uuid.NewString()})
		req := httptest.NewRequest(http.MethodPost, "/admin/dispatch/bad-id/assign", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("invalid driver UUID returns 400", func(t *testing.T) {
		r := dispatchRouter(&stubDispatch{})
		b, _ := json.Marshal(map[string]any{"driverId": "bad-id"})
		req := httptest.NewRequest(http.MethodPost, "/admin/dispatch/"+uuid.NewString()+"/assign", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("conflict returns 409", func(t *testing.T) {
		r := dispatchRouter(&stubDispatch{err: errors.New("driver unavailable")})
		b, _ := json.Marshal(map[string]any{"driverId": uuid.NewString()})
		req := httptest.NewRequest(http.MethodPost, "/admin/dispatch/"+uuid.NewString()+"/assign", bytes.NewReader(b))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", w.Code)
		}
	})
}

// ── KYCService interface check ─────────────────────────────────────────────────

// Ensure stubKYC satisfies the kycService interface used by KYCHandler.
var _ interface {
	SubmitCheck(context.Context, uuid.UUID, model.KYCDocType, map[string]string) error
	QueueForAdmin(context.Context, int, int) ([]model.KYCCheck, error)
	AdminApprove(context.Context, uuid.UUID, uuid.UUID, string) error
	AdminReject(context.Context, uuid.UUID, uuid.UUID, string) error
	GetUserKYC(context.Context, uuid.UUID) ([]model.KYCCheck, error)
} = (*stubKYC)(nil)

// Ensure stubDispatch satisfies the dispatchService interface.
var _ interface {
	SetOnline(context.Context, uuid.UUID, bool) error
	UpdateLocation(context.Context, uuid.UUID, float64, float64, float64) error
	AcceptOffer(context.Context, uuid.UUID, uuid.UUID) error
	RejectOffer(context.Context, uuid.UUID, uuid.UUID) error
	ManualAssign(context.Context, uuid.UUID, uuid.UUID) error
} = (*stubDispatch)(nil)

// Silence unused import warning for service package.
var _ = service.ErrOrderNotFound
