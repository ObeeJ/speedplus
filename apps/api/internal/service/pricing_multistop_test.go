package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/config"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

// fakeOSRMTrip serves a canned /trip/v1/driving response so multistop
// pricing tests never depend on a live network call.
func fakeOSRMTrip(t *testing.T, distanceM, durationS float64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"trips": []map[string]any{
				{"distance": distanceM, "duration": durationS},
			},
		})
	}))
}

func TestOSRMTrip_ParsesDistanceAndETA(t *testing.T) {
	srv := fakeOSRMTrip(t, 12000, 1200) // 12km, 20min
	defer srv.Close()

	svc := &PricingService{osrmURL: srv.URL, httpClient: srv.Client()}
	distKm, etaMin, err := svc.osrmTrip(context.Background(), []LatLng{
		{Lat: 6.5, Lng: 3.3}, {Lat: 6.6, Lng: 3.4}, {Lat: 6.7, Lng: 3.5},
	})
	if err != nil {
		t.Fatalf("osrmTrip: %v", err)
	}
	if distKm != 12.0 {
		t.Errorf("distKm = %v, want 12.0", distKm)
	}
	if etaMin != 25 { // 20 + 5min pickup buffer
		t.Errorf("etaMin = %v, want 25", etaMin)
	}
}

func TestOSRMTrip_RequiresAtLeastTwoWaypoints(t *testing.T) {
	svc := &PricingService{osrmURL: "http://unused"}
	if _, _, err := svc.osrmTrip(context.Background(), []LatLng{{Lat: 1, Lng: 1}}); err == nil {
		t.Fatal("expected error for a single waypoint")
	}
}

func TestQuoteMultiStop_PricesRouteAndPerStopFee(t *testing.T) {
	srv := fakeOSRMTrip(t, 10000, 900) // 10km route
	defer srv.Close()

	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		cfg := &config.Config{JWTSecret: "test-secret-at-least-32-bytes-long!!"}
		svc := &PricingService{db: tx, cfg: cfg, osrmURL: srv.URL, httpClient: srv.Client()}

		// pricing_quotes.customer_id/merchant_id carry FKs — seed real rows.
		customer := seedWalletOwner(t, tx)
		merchantUser := seedWalletOwner(t, tx)
		merchant := &model.Merchant{
			ID: uuid.New(), UserID: merchantUser.ID,
			BusinessName: "MultiStop " + uuid.NewString()[:8], Vertical: model.VerticalPackage,
		}
		mustCreate(t, tx, merchant)
		customerID, merchantID := customer.ID, merchant.ID
		quote, err := svc.QuoteMultiStop(context.Background(), MultiStopQuoteRequest{
			CustomerID:   customerID,
			MerchantID:   merchantID,
			Vertical:     "package",
			SubtotalKobo: 0,
			Origin:       LatLng{Lat: 6.5, Lng: 3.3},
			Stops: []LatLng{
				{Lat: 6.55, Lng: 3.35},
				{Lat: 6.60, Lng: 3.40},
				{Lat: 6.65, Lng: 3.45},
			},
		})
		if err != nil {
			t.Fatalf("QuoteMultiStop: %v", err)
		}

		fees := DefaultFeeTable["package"]
		wantDelivery := fees.BaseFeeKobo + int64(10.0*float64(fees.PerKmKobo)) + int64(2)*fees.PerStopKobo // 3 stops => 2 extra
		if quote.DeliveryKobo != wantDelivery {
			t.Errorf("DeliveryKobo = %d, want %d (base + 10km*perKm + 2*perStop)", quote.DeliveryKobo, wantDelivery)
		}
		if quote.StopCount != 3 {
			t.Errorf("StopCount = %d, want 3", quote.StopCount)
		}
		if quote.DistanceKm != 10.0 {
			t.Errorf("DistanceKm = %v, want 10.0", quote.DistanceKm)
		}
	})
}

func TestQuoteMultiStop_RejectsZeroStops(t *testing.T) {
	svc := &PricingService{cfg: &config.Config{JWTSecret: "test-secret-at-least-32-bytes-long!!"}}
	_, err := svc.QuoteMultiStop(context.Background(), MultiStopQuoteRequest{
		Vertical: "package",
		Origin:   LatLng{Lat: 6.5, Lng: 3.3},
		Stops:    nil,
	})
	if err == nil {
		t.Fatal("expected error when no stops are provided")
	}
}

func TestQuoteMultiStop_RejectsTooManyStops(t *testing.T) {
	svc := &PricingService{cfg: &config.Config{JWTSecret: "test-secret-at-least-32-bytes-long!!"}}
	stops := make([]LatLng, maxQuoteStops+1)
	_, err := svc.QuoteMultiStop(context.Background(), MultiStopQuoteRequest{
		Vertical: "package",
		Origin:   LatLng{Lat: 6.5, Lng: 3.3},
		Stops:    stops,
	})
	if err == nil {
		t.Fatal("expected error when stops exceed the guardrail")
	}
}

// The signed hash must cover StopCount — otherwise a client could request a
// 3-stop quote, then submit the order claiming only 1 stop's worth of fee by
// tampering the persisted row.
func TestSignQuote_CoversStopCount(t *testing.T) {
	svc := &PricingService{cfg: &config.Config{JWTSecret: "test-secret-at-least-32-bytes-long!!"}}

	id := uuid.New()
	custID, merchID := uuid.New(), uuid.New()
	exp := time.Now().Add(10 * time.Minute)

	q1 := &model.PricingQuote{ID: id, CustomerID: custID, MerchantID: merchID, StopCount: 1, SubtotalKobo: 1000, DeliveryKobo: 500, ServiceKobo: 50, TotalKobo: 1550, ExpiresAt: exp}
	q3 := &model.PricingQuote{ID: id, CustomerID: custID, MerchantID: merchID, StopCount: 3, SubtotalKobo: 1000, DeliveryKobo: 500, ServiceKobo: 50, TotalKobo: 1550, ExpiresAt: exp}

	h1 := svc.signQuote(q1)
	h3 := svc.signQuote(q3)
	if h1 == h3 {
		t.Fatal("signQuote must produce a different hash when StopCount differs — otherwise stop count can be tampered after signing")
	}
}
