package service

// Post-delivery review + driver badge tests.
//
// The central guarantee here is authorization: a customer may only review the
// driver/merchant who actually fulfilled their own delivered order. revieweeID
// is derived from the order server-side and is never accepted from the client,
// so a caller cannot inflate or tank a stranger's rating.
//
// Requires DATABASE_URL (shared fixtures testDB/withTx/mustCreate live in
// ledger_money_test.go). Skips locally if unset.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

// reviewFixture is a delivered order plus its participants.
type reviewFixture struct {
	customer   *model.User
	driverUser *model.User
	merchant   *model.Merchant
	order      *model.Order
}

func seedDeliveredOrder(t *testing.T, tx *gorm.DB, status model.OrderStatus) reviewFixture {
	t.Helper()
	unique := uuid.NewString()[:8]

	customer := &model.User{ID: uuid.New(), Role: model.RoleCustomer, FirstName: "Rev", LastName: "Customer", Phone: "+234820" + unique, PasswordHash: "x"}
	mustCreate(t, tx, customer)
	driverUser := &model.User{ID: uuid.New(), Role: model.RoleDriver, FirstName: "Rev", LastName: "Driver", Phone: "+234821" + unique, PasswordHash: "x"}
	mustCreate(t, tx, driverUser)
	merchantUser := &model.User{ID: uuid.New(), Role: model.RoleMerchant, FirstName: "Rev", LastName: "Merchant", Phone: "+234822" + unique, PasswordHash: "x"}
	mustCreate(t, tx, merchantUser)
	merchant := &model.Merchant{ID: uuid.New(), UserID: merchantUser.ID, BusinessName: "Rev Merchant " + unique, Vertical: model.VerticalFood}
	mustCreate(t, tx, merchant)

	addr := &model.Address{ID: uuid.New(), UserID: customer.ID, Street: "1 Review St", City: "Lagos", State: "Lagos", Lat: 6.5, Lng: 3.3}
	mustCreate(t, tx, addr)

	// orders.quote_id is NOT NULL with an FK to pricing_quotes.
	quote := &model.PricingQuote{
		ID: uuid.New(), CustomerID: customer.ID, MerchantID: merchant.ID,
		SubtotalKobo: 100_000, DeliveryKobo: 50_000, ServiceKobo: 5_000, TotalKobo: 155_000,
		QuoteHash: "rev-hash-" + unique, ExpiresAt: time.Now().Add(time.Hour),
	}
	mustCreate(t, tx, quote)

	order := &model.Order{
		ID:         uuid.New(),
		CustomerID: customer.ID,
		MerchantID: merchant.ID,
		DriverID:   &driverUser.ID,
		QuoteID:    quote.ID,
		Status:     status,
		Vertical:   "food",
		SubtotalKobo: 100_000, DeliveryKobo: 50_000, ServiceKobo: 5_000, TotalKobo: 155_000,
		DeliveryAddressID: addr.ID,
		IdempotencyKey:    "rev-order-" + unique,
	}
	mustCreate(t, tx, order)

	return reviewFixture{customer: customer, driverUser: driverUser, merchant: merchant, order: order}
}

// TestSubmitReview_DerivesRevieweeFromOrder is the regression test for the
// IDOR: even though the caller supplies no reviewee, the review must land on
// the order's actual driver — and on nobody else.
func TestSubmitReview_DerivesRevieweeFromOrder(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		f := seedDeliveredOrder(t, tx, model.OrderDelivered)

		// An unrelated driver who must NOT accumulate this review.
		bystander := &model.User{ID: uuid.New(), Role: model.RoleDriver, FirstName: "By", LastName: "Stander", Phone: "+2348290" + uuid.NewString()[:6], PasswordHash: "x"}
		mustCreate(t, tx, bystander)

		svc := &OrderService{orders: repo.NewOrderRepo(tx)}
		comment := "great rider"
		if err := svc.SubmitReview(context.Background(), f.order.ID, f.customer.ID, "driver", 5, &comment); err != nil {
			t.Fatalf("SubmitReview: %v", err)
		}

		var got model.OrderReview
		if err := tx.Where("order_id = ? AND reviewee_type = 'driver'", f.order.ID).First(&got).Error; err != nil {
			t.Fatalf("review not persisted: %v", err)
		}
		if got.RevieweeID != f.driverUser.ID {
			t.Errorf("revieweeID = %s, want the order's driver %s", got.RevieweeID, f.driverUser.ID)
		}
		if got.RevieweeID == bystander.ID {
			t.Error("review landed on an unrelated driver — IDOR regression")
		}
		if got.ReviewerID != f.customer.ID {
			t.Errorf("reviewerID = %s, want %s", got.ReviewerID, f.customer.ID)
		}

		// Merchant reviews must resolve to the order's merchant, not the driver.
		if err := svc.SubmitReview(context.Background(), f.order.ID, f.customer.ID, "merchant", 4, nil); err != nil {
			t.Fatalf("SubmitReview(merchant): %v", err)
		}
		var mrev model.OrderReview
		if err := tx.Where("order_id = ? AND reviewee_type = 'merchant'", f.order.ID).First(&mrev).Error; err != nil {
			t.Fatalf("merchant review not persisted: %v", err)
		}
		if mrev.RevieweeID != f.merchant.ID {
			t.Errorf("merchant revieweeID = %s, want %s", mrev.RevieweeID, f.merchant.ID)
		}
	})
}

// TestSubmitReview_RejectsNonOwner: only the order's own customer may review.
func TestSubmitReview_RejectsNonOwner(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		f := seedDeliveredOrder(t, tx, model.OrderDelivered)
		stranger := uuid.New()

		svc := &OrderService{orders: repo.NewOrderRepo(tx)}
		if err := svc.SubmitReview(context.Background(), f.order.ID, stranger, "driver", 5, nil); err == nil {
			t.Fatal("a non-owner must not be able to review someone else's order")
		}

		var count int64
		tx.Model(&model.OrderReview{}).Where("order_id = ?", f.order.ID).Count(&count)
		if count != 0 {
			t.Errorf("rejected review still persisted %d row(s)", count)
		}
	})
}

// TestSubmitReview_RequiresDelivered: reviews only after the job is done.
func TestSubmitReview_RequiresDelivered(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		f := seedDeliveredOrder(t, tx, model.OrderInTransit)

		svc := &OrderService{orders: repo.NewOrderRepo(tx)}
		if err := svc.SubmitReview(context.Background(), f.order.ID, f.customer.ID, "driver", 5, nil); err == nil {
			t.Fatal("an in-transit order must not be reviewable")
		}
	})
}

// TestSubmitReview_RejectsBadInput covers the rating bounds and reviewee type
// allow-list at the service boundary (not only the HTTP binding).
func TestSubmitReview_RejectsBadInput(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		f := seedDeliveredOrder(t, tx, model.OrderDelivered)
		svc := &OrderService{orders: repo.NewOrderRepo(tx)}

		for _, tc := range []struct {
			name         string
			rating       int
			revieweeType string
		}{
			{"rating zero", 0, "driver"},
			{"rating six", 6, "driver"},
			{"rating negative", -1, "driver"},
			{"unknown reviewee type", 5, "admin"},
			{"empty reviewee type", 5, ""},
		} {
			t.Run(tc.name, func(t *testing.T) {
				if err := svc.SubmitReview(context.Background(), f.order.ID, f.customer.ID, tc.revieweeType, tc.rating, nil); err == nil {
					t.Errorf("SubmitReview(%q, rating=%d) = nil, want error", tc.revieweeType, tc.rating)
				}
			})
		}
	})
}

// TestSubmitReview_OnePerRolePerOrder: the unique(order_id, reviewee_type)
// constraint must make a double submission fail rather than double-count.
func TestSubmitReview_OnePerRolePerOrder(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		f := seedDeliveredOrder(t, tx, model.OrderDelivered)
		svc := &OrderService{orders: repo.NewOrderRepo(tx)}

		if err := svc.SubmitReview(context.Background(), f.order.ID, f.customer.ID, "driver", 5, nil); err != nil {
			t.Fatalf("first review: %v", err)
		}

		// The duplicate insert raises a constraint error, which aborts the
		// enclosing transaction in Postgres. Fence it with a savepoint so the
		// outer test transaction stays usable for the assertions below.
		// (In production each request has its own transaction, so this is a
		// test-harness concern only.)
		tx.SavePoint("before_dup")
		if err := svc.SubmitReview(context.Background(), f.order.ID, f.customer.ID, "driver", 1, nil); err == nil {
			t.Fatal("second review for the same role must be rejected")
		}
		tx.RollbackTo("before_dup")

		var count int64
		tx.Model(&model.OrderReview{}).Where("order_id = ? AND reviewee_type = 'driver'", f.order.ID).Count(&count)
		if count != 1 {
			t.Errorf("got %d driver reviews, want exactly 1", count)
		}
	})
}

// TestSubmitReview_EnqueuesAggregate proves the async hook is actually wired.
// This is the shape of test that catches an orphaned enqueue function: the
// review persists, and the aggregation task is handed to the queue exactly
// once with the derived reviewee.
func TestSubmitReview_EnqueuesAggregate(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		f := seedDeliveredOrder(t, tx, model.OrderDelivered)

		type enqueued struct {
			revieweeID   string
			revieweeType string
		}
		var calls []enqueued

		svc := &OrderService{orders: repo.NewOrderRepo(tx)}
		svc.InjectReviewQueue(func(revieweeID, revieweeType string) error {
			calls = append(calls, enqueued{revieweeID, revieweeType})
			return nil
		})

		if err := svc.SubmitReview(context.Background(), f.order.ID, f.customer.ID, "driver", 5, nil); err != nil {
			t.Fatalf("SubmitReview: %v", err)
		}

		if len(calls) != 1 {
			t.Fatalf("enqueue called %d times, want exactly 1", len(calls))
		}
		if calls[0].revieweeID != f.driverUser.ID.String() {
			t.Errorf("enqueued revieweeID = %s, want %s", calls[0].revieweeID, f.driverUser.ID)
		}
		if calls[0].revieweeType != "driver" {
			t.Errorf("enqueued revieweeType = %q, want \"driver\"", calls[0].revieweeType)
		}
	})
}

// TestSubmitReview_EnqueueFailureDoesNotLoseReview: the review is already
// durable when the enqueue runs, so a queue outage must not surface as a
// failed review to the customer.
func TestSubmitReview_EnqueueFailureDoesNotLoseReview(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		f := seedDeliveredOrder(t, tx, model.OrderDelivered)

		svc := &OrderService{orders: repo.NewOrderRepo(tx)}
		svc.InjectReviewQueue(func(string, string) error {
			return context.DeadlineExceeded // simulate Redis down
		})

		if err := svc.SubmitReview(context.Background(), f.order.ID, f.customer.ID, "driver", 5, nil); err != nil {
			t.Fatalf("enqueue failure must not fail the review, got: %v", err)
		}
		var count int64
		tx.Model(&model.OrderReview{}).Where("order_id = ?", f.order.ID).Count(&count)
		if count != 1 {
			t.Errorf("review rows = %d, want 1 (review must survive enqueue failure)", count)
		}
	})
}

// TestUpdateAggregateRating_Driver verifies the average lands on the driver
// profile keyed by user_id — dispatch selects dp.user_id AS driver_id, so
// order.DriverID is a User.ID, not a DriverProfile.ID.
func TestUpdateAggregateRating_Driver(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		f := seedDeliveredOrder(t, tx, model.OrderDelivered)
		profile := &model.DriverProfile{
			ID: uuid.New(), UserID: f.driverUser.ID,
			VehicleType: model.VehicleMotorcycle, VehiclePlate: "TEST-123", Rating: 5.0,
		}
		mustCreate(t, tx, profile)

		// Two reviews: 5 and 3 -> average 4.0
		for _, r := range []int{5, 3} {
			extra := seedDeliveredOrder(t, tx, model.OrderDelivered)
			mustCreate(t, tx, &model.OrderReview{
				ID: uuid.New(), OrderID: extra.order.ID, ReviewerID: f.customer.ID,
				RevieweeID: f.driverUser.ID, RevieweeType: "driver", Rating: r,
			})
		}

		svc := &OrderService{orders: repo.NewOrderRepo(tx)}
		if err := svc.UpdateAggregateRating(context.Background(), f.driverUser.ID, "driver"); err != nil {
			t.Fatalf("UpdateAggregateRating: %v", err)
		}

		var got model.DriverProfile
		if err := tx.Where("user_id = ?", f.driverUser.ID).First(&got).Error; err != nil {
			t.Fatalf("reload profile: %v", err)
		}
		if got.Rating < 3.99 || got.Rating > 4.01 {
			t.Errorf("driver rating = %v, want 4.0 — a mismatch here means the "+
				"aggregate is keyed on the wrong ID and ratings silently never persist", got.Rating)
		}
	})
}

// TestAwardBadgeIfEligible_MilestonesAndIdempotency: badges are awarded at the
// right thresholds and re-running the award is a no-op (worker retries replay).
func TestAwardBadgeIfEligible_MilestonesAndIdempotency(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		f := seedDeliveredOrder(t, tx, model.OrderDelivered)
		svc := &OrderService{orders: repo.NewOrderRepo(tx)}

		// Exactly one delivered order exists -> first_delivery only.
		if err := svc.AwardBadgeIfEligible(context.Background(), f.driverUser.ID); err != nil {
			t.Fatalf("AwardBadgeIfEligible: %v", err)
		}

		badges, err := svc.GetDriverBadges(context.Background(), f.driverUser.ID)
		if err != nil {
			t.Fatalf("GetDriverBadges: %v", err)
		}
		if len(badges) != 1 || badges[0].BadgeType != "first_delivery" {
			t.Fatalf("got %d badge(s) %+v, want exactly [first_delivery]", len(badges), badges)
		}

		// Replay must not duplicate — unique(driver_id, badge_type).
		if err := svc.AwardBadgeIfEligible(context.Background(), f.driverUser.ID); err != nil {
			t.Fatalf("second AwardBadgeIfEligible: %v", err)
		}
		badges, _ = svc.GetDriverBadges(context.Background(), f.driverUser.ID)
		if len(badges) != 1 {
			t.Errorf("after replay got %d badges, want 1 — award is not idempotent", len(badges))
		}
	})
}

// TestAwardBadgeIfEligible_TopRatedThreshold: top_rated needs both a high
// average and enough reviews, so a single 5-star cannot mint it.
func TestAwardBadgeIfEligible_TopRatedThreshold(t *testing.T) {
	gdb := testDB(t)
	withTx(t, gdb, func(tx *gorm.DB) {
		f := seedDeliveredOrder(t, tx, model.OrderDelivered)
		svc := &OrderService{orders: repo.NewOrderRepo(tx)}

		first := seedDeliveredOrder(t, tx, model.OrderDelivered)
		mustCreate(t, tx, &model.OrderReview{
			ID: uuid.New(), OrderID: first.order.ID, ReviewerID: f.customer.ID,
			RevieweeID: f.driverUser.ID, RevieweeType: "driver", Rating: 5,
		})
		if err := svc.AwardBadgeIfEligible(context.Background(), f.driverUser.ID); err != nil {
			t.Fatalf("AwardBadgeIfEligible: %v", err)
		}
		badges, _ := svc.GetDriverBadges(context.Background(), f.driverUser.ID)
		for _, b := range badges {
			if b.BadgeType == "top_rated" {
				t.Fatalf("top_rated awarded with 1 review; needs >= %d", topRatedMinReviews)
			}
		}

		// Now clear the review bar with a 5.0 average.
		for i := 0; i < topRatedMinReviews; i++ {
			o := seedDeliveredOrder(t, tx, model.OrderDelivered)
			mustCreate(t, tx, &model.OrderReview{
				ID: uuid.New(), OrderID: o.order.ID, ReviewerID: f.customer.ID,
				RevieweeID: f.driverUser.ID, RevieweeType: "driver", Rating: 5,
			})
		}
		if err := svc.AwardBadgeIfEligible(context.Background(), f.driverUser.ID); err != nil {
			t.Fatalf("AwardBadgeIfEligible (eligible): %v", err)
		}
		badges, _ = svc.GetDriverBadges(context.Background(), f.driverUser.ID)
		var found bool
		for _, b := range badges {
			if b.BadgeType == "top_rated" {
				found = true
			}
		}
		if !found {
			t.Errorf("top_rated not awarded at %d reviews / 5.0 avg; got %+v", topRatedMinReviews+1, badges)
		}
	})
}
