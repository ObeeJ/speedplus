package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/crypto"
	"github.com/speedplus/api/internal/dto"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/observability"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrIllegalTransition = errors.New("illegal order state transition")
	ErrQuoteInvalid      = errors.New("quote invalid or expired")
	ErrMerchantClosed    = errors.New("merchant is currently closed")
	ErrRxRequired        = errors.New("prescription required for this order")
	ErrRxNotApproved     = errors.New("prescription is not yet approved by the pharmacy")
)

type CreateOrderInput struct {
	CustomerID        uuid.UUID
	MerchantID        uuid.UUID
	QuoteID           uuid.UUID
	Vertical          string
	Items             []OrderItemInput
	DeliveryAddrID    uuid.UUID
	RecipientName     *string
	RecipientPhone    *string
	PrescriptionID    *uuid.UUID
	TipKobo           int64
	ScheduledFor      *time.Time
	PaymentMethod     string
	DeclaredValueKobo *int64  // package vertical only
	IdempotencyKey    string
	Stops             []OrderStopInput
}

type OrderStopInput struct {
	Sequence       int
	AddressID      uuid.UUID
	RecipientName  *string
	RecipientPhone *string
	Notes          *string
}

type OrderItemInput struct {
	ProductID        uuid.UUID
	Name             string
	Quantity         int
	UnitPriceKobo    int64
	WeightKg         float64
	SizeCategory     string
	Customizations   *string
	SubstitutionPref *string
}

type OrderService struct {
	db            *gorm.DB
	pricing       *PricingService
	ledger        *LedgerService
	tier          *TierService
	dispatch      *DispatchService
	hub           wsPublisher
	deliveryCodes *DeliveryCodeService
	recipients    *crypto.Cipher // encrypts/decrypts recipient name+phone at rest
	email         orderEmailSender
	users         repo.UserRepo
	// enqueueReview schedules post-review rating/badge recomputation. Injected
	// after construction to avoid a service→worker import cycle.
	enqueueReview func(revieweeID, revieweeType string) error
}

// InjectReviewQueue wires the asynq enqueue function for post-review
// aggregation. Until called, reviews persist but aggregation is skipped.
func (s *OrderService) InjectReviewQueue(fn func(revieweeID, revieweeType string) error) {
	s.enqueueReview = fn
}

// EnqueueBadgeCheck schedules a driver badge re-evaluation — e.g. once a
// delivery completes. Call it *after* the enclosing transaction commits: the
// worker counts delivered orders, so running it against an uncommitted tx
// undercounts and would silently skip a driver's first_delivery badge.
// Enqueue failure is logged, not returned: the delivery itself already
// succeeded and must not be rolled back over a badge.
func (s *OrderService) EnqueueBadgeCheck(ctx context.Context, driverID uuid.UUID) {
	if s.enqueueReview == nil {
		return
	}
	if err := s.enqueueReview(driverID.String(), "driver"); err != nil {
		observability.CaptureError(ctx, err, "EnqueueBadgeCheck: enqueue failed",
			"driver_id", driverID.String())
	}
}

// orderEmailSender is the subset of email.Client used by OrderService.
type orderEmailSender interface {
	SendDeliveryCode(ctx context.Context, toEmail, firstName, code, orderID string)
}

// wsPublisher is the subset of ws.Hub used by OrderService.
type wsPublisher interface {
	Publish(ctx context.Context, channel, event string, data interface{}) error
}

func NewOrderService(db *gorm.DB, pricing *PricingService, ledger *LedgerService, tier *TierService) *OrderService {
	return &OrderService{db: db, pricing: pricing, ledger: ledger, tier: tier}
}

func (s *OrderService) InjectDispatch(d *DispatchService, hub wsPublisher) {
	s.dispatch = d
	s.hub = hub
}

func (s *OrderService) InjectDeliveryCodes(dc *DeliveryCodeService) {
	s.deliveryCodes = dc
}

// InjectEmail wires the email client and user repo so Transition can send
// the delivery code to the customer when the order goes in_transit.
func (s *OrderService) InjectEmail(e orderEmailSender, u repo.UserRepo) {
	s.email = e
	s.users = u
}

// InjectRecipientCipher wires the AES-GCM cipher used to encrypt/decrypt
// recipient name+phone. Required before any order carrying recipient data is
// created — see encryptRecipient.
func (s *OrderService) InjectRecipientCipher(c *crypto.Cipher) {
	s.recipients = c
}

// encryptRecipient encrypts a recipient PII field for storage. nil in, nil
// out. If the field is set but no cipher is configured, this fails loudly
// rather than ever writing plaintext under an "_enc" column.
func (s *OrderService) encryptRecipient(v *string) (*string, error) {
	if v == nil {
		return nil, nil
	}
	if s.recipients == nil {
		return nil, fmt.Errorf("recipient encryption not configured")
	}
	return s.recipients.EncryptPtr(v)
}

// generateTrackingRef produces a short human-readable tracking reference.
// Format: SPX-XXXXX where X is uppercase alphanumeric (no ambiguous chars).
// Collision probability at 1M orders: ~0.5% — DB unique index is the hard guard.
func generateTrackingRef() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 5)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			b[i] = chars[0]
			continue
		}
		b[i] = chars[n.Int64()]
	}
	return "SPX-" + string(b)
}

func (s *OrderService) Create(ctx context.Context, in CreateOrderInput) (*model.Order, error) {
	// Idempotency check
	var existing model.Order
	if err := s.db.WithContext(ctx).Where("idempotency_key = ?", in.IdempotencyKey).First(&existing).Error; err == nil {
		return &existing, nil
	}

	// Validate quote
	var subtotal int64
	for _, item := range in.Items {
		subtotal += item.UnitPriceKobo * int64(item.Quantity)
	}
	quote, err := s.pricing.ValidateQuote(ctx, in.QuoteID, subtotal)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrQuoteInvalid, err)
	}

	// Pharmacy vertical requires a prescription that has actually been
	// approved by the merchant — a present-but-pending/rejected ID must not
	// be enough to place the order (the comment claimed "approved
	// prescription" but the code never checked status until now).
	if in.Vertical == "pharmacy" {
		if in.PrescriptionID == nil {
			return nil, ErrRxRequired
		}
		var rx model.Prescription
		if err := s.db.WithContext(ctx).First(&rx, *in.PrescriptionID).Error; err != nil {
			return nil, ErrRxRequired
		}
		if rx.CustomerID != in.CustomerID {
			return nil, ErrRxRequired
		}
		if rx.Status != "approved" {
			return nil, ErrRxNotApproved
		}
	}

	// Pay-on-arrival gate: trust-ladder checks only.
	// Settlement path (card scan → PIN → wallet debit) is implemented in
	// PaycodeService.ConfirmByCard — POD orders are fully settleable.
	if in.PaymentMethod == "pay_on_arrival" {
		if s.tier == nil {
			return nil, ErrPODTierLocked
		}
		totalKobo := quote.TotalKobo + in.TipKobo
		if err := s.tier.CanUsePayOnArrival(ctx, in.CustomerID, totalKobo); err != nil {
			return nil, err
		}
		// Eligible — fall through to order creation below (no escrow hold for POD).
	}

	recipientNameEnc, err := s.encryptRecipient(in.RecipientName)
	if err != nil {
		return nil, fmt.Errorf("encrypt recipient name: %w", err)
	}
	recipientPhoneEnc, err := s.encryptRecipient(in.RecipientPhone)
	if err != nil {
		return nil, fmt.Errorf("encrypt recipient phone: %w", err)
	}

	var order *model.Order
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check merchant is open + KYC approved
		var merchant model.Merchant
		if err := tx.First(&merchant, in.MerchantID).Error; err != nil {
			return fmt.Errorf("merchant not found")
		}
		if !merchant.IsOpen {
			return ErrMerchantClosed
		}

		order = &model.Order{
			ID:                uuid.New(),
			CustomerID:        in.CustomerID,
			MerchantID:        in.MerchantID,
			QuoteID:           in.QuoteID,
			Vertical:          in.Vertical,
			Status:            model.OrderPending,
			SubtotalKobo:      quote.SubtotalKobo,
			DeliveryKobo:      quote.DeliveryKobo,
			ServiceKobo:       quote.ServiceKobo,
			TipKobo:           in.TipKobo,
			TotalKobo:         quote.TotalKobo + in.TipKobo,
			DeliveryAddressID: in.DeliveryAddrID,
			RecipientNameEnc:  recipientNameEnc,
			RecipientPhoneEnc: recipientPhoneEnc,
			PrescriptionID:    in.PrescriptionID,
			ScheduledFor:      in.ScheduledFor,
			PaymentMethod:     "wallet",
			DeclaredValueKobo: in.DeclaredValueKobo,
			IdempotencyKey:    in.IdempotencyKey,
		}
		if in.PaymentMethod == "pay_on_arrival" {
			order.PaymentMethod = "pay_on_arrival"
		}
		// Generate a short human-readable tracking reference for package orders.
		// Retry once on the rare collision — DB unique index is the final guard.
		if in.Vertical == "package" {
			ref := generateTrackingRef()
			order.TrackingRef = &ref
		}

		for _, item := range in.Items {
			order.Items = append(order.Items, model.OrderItem{
				ID:               uuid.New(),
				OrderID:          order.ID,
				ProductID:        item.ProductID,
				Name:             item.Name,
				Quantity:         item.Quantity,
				UnitPriceKobo:    item.UnitPriceKobo,
				TotalKobo:        item.UnitPriceKobo * int64(item.Quantity),
				WeightKg:         item.WeightKg,
				SizeCategory:     item.SizeCategory,
				Customizations:   item.Customizations,
				SubstitutionPref: item.SubstitutionPref,
			})
		}

		if err := tx.Create(order).Error; err != nil {
			return err
		}

		// Mark quote used
		if err := s.pricing.MarkQuoteUsed(ctx, in.QuoteID); err != nil {
			return err
		}

		// Multi-drop stops (package vertical)
		for _, stop := range in.Stops {
			stopNameEnc, err := s.encryptRecipient(stop.RecipientName)
			if err != nil {
				return fmt.Errorf("encrypt stop %d recipient name: %w", stop.Sequence, err)
			}
			stopPhoneEnc, err := s.encryptRecipient(stop.RecipientPhone)
			if err != nil {
				return fmt.Errorf("encrypt stop %d recipient phone: %w", stop.Sequence, err)
			}
			order.Stops = append(order.Stops, model.OrderStop{
				ID:                uuid.New(),
				OrderID:           order.ID,
				Sequence:          stop.Sequence,
				AddressID:         stop.AddressID,
				RecipientNameEnc:  stopNameEnc,
				RecipientPhoneEnc: stopPhoneEnc,
				Notes:             stop.Notes,
				QRCode:            "", // generated when order transitions to in_transit
				Status:            "pending",
			})
		}
		if len(in.Stops) > 0 {
			if err := tx.Create(&order.Stops).Error; err != nil {
				return fmt.Errorf("create stops: %w", err)
			}
		}

		// Escrow hold — debit customer wallet for wallet-payment orders.
		// Pay-on-arrival orders skip this: the wallet is debited at the door
		// when the rider scans the customer's SpeedPlus card + PIN.
		if order.PaymentMethod != "pay_on_arrival" {
			if err := s.ledger.HoldEscrow(ctx, tx, order.ID, in.CustomerID, order.TotalKobo); err != nil {
				return fmt.Errorf("escrow hold: %w", err)
			}
		}

		// Audit event
		return tx.Create(&model.OrderEvent{
			ID:        uuid.New(),
			OrderID:   order.ID,
			ToStatus:  model.OrderPending,
			ActorID:   in.CustomerID,
			ActorRole: "customer",
		}).Error
	})
	if err != nil {
		return nil, err
	}

	// Trigger dispatch asynchronously after the transaction commits.
	// Merchant lat/lng is used as the pickup point for KNN driver search.
	if s.dispatch != nil {
		go func(o *model.Order) {
			var merchant model.Merchant
			if dbErr := s.db.First(&merchant, o.MerchantID).Error; dbErr != nil {
				return
			}
			var dropoff model.Address
			_ = s.db.First(&dropoff, o.DeliveryAddressID).Error // best-effort label; ok if empty

			candidates, dispErr := s.dispatch.Dispatch(context.Background(), o, merchant.Lat, merchant.Lng)
			if dispErr != nil || len(candidates) == 0 {
				return
			}
			// Notify each targeted driver via WS on their personal channel.
			// Payload matches what the driver app's offer card expects:
			// offerId (so accept/reject hits the right row), addresses, distance.
			if s.hub != nil {
				for _, cand := range candidates {
					_ = s.hub.Publish(context.Background(),
						"driver:"+cand.DriverID.String(),
						"new_offer",
						map[string]interface{}{
							"offerId":         cand.OfferID,
							"orderId":         o.ID,
							"vertical":        o.Vertical,
							"totalKobo":       o.TotalKobo,
							"distanceKm":      cand.DistanceKm,
							"pickupAddress":   merchant.BusinessName,
							"dropoffAddress":  fmt.Sprintf("%s, %s", dropoff.Street, dropoff.City),
						},
					)
				}
			}
			// Notify customer that we're searching.
			if s.hub != nil {
				_ = s.hub.Publish(context.Background(),
					"order:"+o.ID.String(),
					"searching_driver",
					map[string]interface{}{"orderId": o.ID},
				)
			}
		}(order)
	}

	return order, nil
}

// Transition moves an order through the state machine.
// Returns 409-equivalent error on illegal transition.
// Transition moves an order through the state machine. actorID must be the
// *business-entity* ID for the actor's role, not their login user ID — i.e.
// for a merchant, the resolved model.Merchant.ID (see LedgerService.ResolveWalletOwner
// for the same wrinkle on the wallet side); for a driver, order.DriverID
// already IS the login user ID. Callers must resolve that before invoking.
func (s *OrderService) Transition(ctx context.Context, orderID, actorID uuid.UUID, actorRole string, to model.OrderStatus, note *string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, orderID).Error; err != nil {
			return err
		}

		// Row-level ownership — fail closed. Without this, any authenticated
		// caller could transition any order's status by ID (BOLA).
		switch actorRole {
		case "merchant":
			if order.MerchantID != actorID {
				return errors.New("forbidden")
			}
		case "driver":
			if order.DriverID == nil || *order.DriverID != actorID {
				return errors.New("forbidden")
			}
		case "customer":
			if order.CustomerID != actorID {
				return errors.New("forbidden")
			}
		case "admin":
			// admin may transition any order
		default:
			return errors.New("forbidden")
		}

		allowed := model.ValidTransitions[order.Status]
		valid := false
		for _, st := range allowed {
			if st == to {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("%w: %s → %s", ErrIllegalTransition, order.Status, to)
		}

		from := order.Status
		order.Status = to
		if to == model.OrderDelivered {
			now := time.Now()
			order.DeliveredAt = &now
		}

		if err := tx.Save(&order).Error; err != nil {
			return err
		}

		if err := tx.Create(&model.OrderEvent{
			ID:         uuid.New(),
			OrderID:    orderID,
			FromStatus: from,
			ToStatus:   to,
			ActorID:    actorID,
			ActorRole:  actorRole,
			Note:       note,
		}).Error; err != nil {
			return err
		}

		// When order goes in_transit: generate delivery code and send to customer.
		// Done inside the transaction so a failed code generation rolls back the
		// transition — the driver cannot proceed without a code the customer has.
		if to == model.OrderInTransit && s.deliveryCodes != nil {
			code, err := s.deliveryCodes.Generate(ctx, orderID)
			if err != nil {
				return fmt.Errorf("delivery code generate: %w", err)
			}
			// Send code to customer — best-effort, non-blocking after tx commits.
			if s.email != nil && s.users != nil {
				go func(customerID uuid.UUID, c, oid string) {
					u, err := s.users.FindByID(context.Background(), customerID)
					if err != nil || u.Email == nil {
						return
					}
					s.email.SendDeliveryCode(context.Background(), *u.Email, u.FirstName, c, oid)
				}(order.CustomerID, code, orderID.String())
			}
			// Also push the code to the customer via WS so the app shows it immediately.
			if s.hub != nil {
				_ = s.hub.Publish(ctx,
					"order:"+orderID.String(),
					"delivery_code",
					map[string]interface{}{"code": code, "orderId": orderID},
				)
			}
		}

		// Notify customer when a driver is assigned — the finding page listens
		// for this event to navigate to the tracking screen.
		if to == model.OrderDriverAssigned && s.hub != nil {
			_ = s.hub.Publish(ctx,
				"order:"+orderID.String(),
				"driver_assigned",
				map[string]interface{}{"orderId": orderID, "driverId": order.DriverID},
			)
		}

		// Broadcast every status change so the customer tracking page updates
		// in real-time without relying solely on polling.
		if s.hub != nil {
			_ = s.hub.Publish(ctx,
				"order:"+orderID.String(),
				"order_status_changed",
				map[string]interface{}{"orderId": orderID, "status": to},
			)
		}

		// Notify on delivery so the tracking page transitions immediately
		// without waiting for the next 5-second poll cycle.
		if to == model.OrderDelivered && s.hub != nil {
			_ = s.hub.Publish(ctx,
				"order:"+orderID.String(),
				"order_delivered",
				map[string]interface{}{"orderId": orderID},
			)
		}

		return nil
	})
}

func (s *OrderService) GetByID(ctx context.Context, orderID, requesterID uuid.UUID, requesterRole string) (*model.Order, error) {
	var order model.Order
	if err := s.db.WithContext(ctx).Preload("Items").Preload("Events").First(&order, orderID).Error; err != nil {
		return nil, err
	}
	// Row-level ownership. Fail closed: only the four known roles are allowed,
	// and each (except admin) must own the row. An empty or unrecognized role
	// — e.g. a middleware slip — must NOT fall through to full access (BOLA).
	switch requesterRole {
	case "customer":
		if order.CustomerID != requesterID {
			return nil, errors.New("forbidden")
		}
	case "driver":
		if order.DriverID == nil || *order.DriverID != requesterID {
			return nil, errors.New("forbidden")
		}
	case "merchant":
		if order.MerchantID != requesterID {
			return nil, errors.New("forbidden")
		}
	case "admin":
		// admin sees all
	default:
		return nil, errors.New("forbidden")
	}
	return &order, nil
}

// ToResponse converts an Order to the API response shape, enriching with
// driver profile data when a driver is assigned.
func (s *OrderService) ToResponse(ctx context.Context, o *model.Order) dto.OrderResponse {
	resp := dto.OrderFromModel(o)
	if o.DriverID == nil {
		return resp
	}
	// Best-effort driver enrichment — never blocks the response.
	var dp model.DriverProfile
	if err := s.db.WithContext(ctx).Where("user_id = ?", o.DriverID).First(&dp).Error; err != nil {
		return resp
	}
	var u model.User
	if err := s.db.WithContext(ctx).First(&u, o.DriverID).Error; err != nil {
		return resp
	}
	name := u.FirstName + " " + u.LastName
	vehicle := string(dp.VehicleType)
	resp.DriverName = &name
	resp.DriverRating = &dp.Rating
	resp.DriverVehicle = &vehicle
	if u.Phone != "" {
		resp.DriverPhone = &u.Phone
	}
	return resp
}

// Cancel handles cancellation with the refund engine.
func (s *OrderService) Cancel(ctx context.Context, orderID, actorID uuid.UUID, actorRole, reason string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.First(&order, orderID).Error; err != nil {
			return err
		}

		// Validate transition
		allowed := model.ValidTransitions[order.Status]
		canCancel := false
		for _, s := range allowed {
			if s == model.OrderCancelled {
				canCancel = true
				break
			}
		}
		if !canCancel {
			return fmt.Errorf("%w: cannot cancel from %s", ErrIllegalTransition, order.Status)
		}

		from := order.Status
		order.Status = model.OrderCancelled
		order.CancelReason = &reason
		if err := tx.Save(&order).Error; err != nil {
			return err
		}

		// Audit
		actorUUID := actorID
		if err := tx.Create(&model.OrderEvent{
			ID:         uuid.New(),
			OrderID:    orderID,
			FromStatus: from,
			ToStatus:   model.OrderCancelled,
			ActorID:    actorUUID,
			ActorRole:  actorRole,
			Note:       &reason,
		}).Error; err != nil {
			return err
		}

		// Refund engine
		if err := s.ledger.ProcessCancellationRefund(ctx, tx, &order); err != nil {
			return err
		}

		// Notify all parties via WS — customer tracking page and driver app
		// both subscribe to order:{id} and must know immediately.
		if s.hub != nil {
			_ = s.hub.Publish(ctx,
				"order:"+orderID.String(),
				"order_cancelled",
				map[string]interface{}{
					"orderId": orderID,
					"reason":  reason,
				},
			)
		}
		return nil
	})
}

// OrderStopOut is the API-facing view of a stop. Recipient fields are only
// ever populated by GetStops when the caller is authorized to see them — the
// encrypted columns never leave the service layer.
type OrderStopOut struct {
	ID             uuid.UUID  `json:"id"`
	OrderID        uuid.UUID  `json:"orderId"`
	Sequence       int        `json:"sequence"`
	AddressID      uuid.UUID  `json:"addressId"`
	RecipientName  *string    `json:"recipientName,omitempty"`
	RecipientPhone *string    `json:"recipientPhone,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	Status         string     `json:"status"`
	ConfirmedAt    *time.Time `json:"confirmedAt,omitempty"`
}

// ListForCustomer returns the customer's order history with optional filtering.
// Cursor is stable: uses (created_at, id) composite to prevent duplicates
// when two orders share the same timestamp.
func (s *OrderService) ListForCustomer(ctx context.Context, customerID uuid.UUID, vertical, status string, cursor *uuid.UUID, limit int) ([]model.Order, error) {
	q := s.db.WithContext(ctx).
		Preload("Items").
		Where("customer_id = ?", customerID).
		Order("created_at DESC, id DESC").
		Limit(limit)
	if vertical != "" {
		q = q.Where("vertical = ?", vertical)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if cursor != nil {
		var pivot model.Order
		if err := s.db.WithContext(ctx).First(&pivot, cursor).Error; err == nil {
			q = q.Where("(created_at, id) < (?, ?)", pivot.CreatedAt, pivot.ID)
		}
	}
	var orders []model.Order
	return orders, q.Find(&orders).Error
}

// ReceiptResponse is the full invoice view of a completed order.
type ReceiptResponse struct {
	OrderID       string             `json:"orderId"`
	Vertical      string             `json:"vertical"`
	Status        string             `json:"status"`
	PaymentMethod string             `json:"paymentMethod"`
	Items         []model.OrderItem  `json:"items"`
	SubtotalKobo  int64              `json:"subtotalKobo"`
	DeliveryKobo  int64              `json:"deliveryKobo"`
	ServiceKobo   int64              `json:"serviceKobo"`
	TipKobo       int64              `json:"tipKobo"`
	TotalKobo     int64              `json:"totalKobo"`
	MerchantName  string             `json:"merchantName"`
	DriverName    string             `json:"driverName,omitempty"`
	CreatedAt     string             `json:"createdAt"`
	DeliveredAt   *string            `json:"deliveredAt,omitempty"`
	Review        *model.OrderReview `json:"review,omitempty"`
}

// GetReceipt returns the full invoice for an order the caller owns.
func (s *OrderService) GetReceipt(ctx context.Context, orderID, customerID uuid.UUID) (*ReceiptResponse, error) {
	var order model.Order
	if err := s.db.WithContext(ctx).Preload("Items").First(&order, orderID).Error; err != nil {
		return nil, err
	}
	if order.CustomerID != customerID {
		return nil, fmt.Errorf("forbidden")
	}

	var merchant model.Merchant
	s.db.WithContext(ctx).First(&merchant, order.MerchantID)

	driverName := ""
	if order.DriverID != nil {
		var u model.User
		if err := s.db.WithContext(ctx).First(&u, order.DriverID).Error; err == nil {
			driverName = u.FirstName + " " + u.LastName
		}
	}

	var review model.OrderReview
	var reviewPtr *model.OrderReview
	if err := s.db.WithContext(ctx).Where("order_id = ? AND reviewer_id = ?", orderID, customerID).First(&review).Error; err == nil {
		reviewPtr = &review
	}

	var deliveredAt *string
	if order.DeliveredAt != nil {
		s := order.DeliveredAt.Format("2006-01-02T15:04:05Z07:00")
		deliveredAt = &s
	}

	return &ReceiptResponse{
		OrderID:       order.ID.String(),
		Vertical:      order.Vertical,
		Status:        string(order.Status),
		PaymentMethod: order.PaymentMethod,
		Items:         order.Items,
		SubtotalKobo:  order.SubtotalKobo,
		DeliveryKobo:  order.DeliveryKobo,
		ServiceKobo:   order.ServiceKobo,
		TipKobo:       order.TipKobo,
		TotalKobo:     order.TotalKobo,
		MerchantName:  merchant.BusinessName,
		DriverName:    driverName,
		CreatedAt:     order.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		DeliveredAt:   deliveredAt,
		Review:        reviewPtr,
	}, nil
}

// SubmitReview saves a post-delivery rating + comment.
// Only the order's customer may review, only after delivery, and only about
// the driver/merchant who actually fulfilled that order — revieweeID is
// derived from the order, never trusted from the request.
func (s *OrderService) SubmitReview(ctx context.Context, orderID, customerID uuid.UUID, revieweeType string, rating int, comment *string) error {
	if rating < 1 || rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}
	if revieweeType != "driver" && revieweeType != "merchant" {
		return fmt.Errorf("revieweeType must be driver or merchant")
	}

	var order model.Order
	if err := s.db.WithContext(ctx).First(&order, orderID).Error; err != nil {
		return fmt.Errorf("order not found")
	}
	if order.CustomerID != customerID {
		return fmt.Errorf("forbidden")
	}
	if order.Status != model.OrderDelivered {
		return fmt.Errorf("can only review a delivered order")
	}

	var revieweeID uuid.UUID
	if revieweeType == "driver" {
		if order.DriverID == nil {
			return fmt.Errorf("order has no assigned driver")
		}
		revieweeID = *order.DriverID
	} else {
		if order.MerchantID == uuid.Nil {
			return fmt.Errorf("order has no merchant")
		}
		revieweeID = order.MerchantID
	}

	review := model.OrderReview{
		ID:           uuid.New(),
		OrderID:      orderID,
		ReviewerID:   customerID,
		RevieweeID:   revieweeID,
		RevieweeType: revieweeType,
		Rating:       rating,
		Comment:      comment,
	}
	if err := s.db.WithContext(ctx).Create(&review).Error; err != nil {
		return fmt.Errorf("review already submitted or DB error: %w", err)
	}

	// Rating recomputation + badge award run in the worker: retried on failure,
	// observable, and a panic there cannot take down the API process.
	// The review itself is already durable, so an enqueue failure is logged,
	// not returned — the customer's review must not appear to have failed.
	if s.enqueueReview != nil {
		if err := s.enqueueReview(revieweeID.String(), revieweeType); err != nil {
			observability.CaptureError(ctx, err, "SubmitReview: enqueue aggregate failed",
				"reviewee_id", revieweeID.String(), "reviewee_type", revieweeType)
		}
	}
	return nil
}

// UpdateAggregateRating recalculates and persists the reviewee's average
// rating. Called from the worker so failures retry and surface.
func (s *OrderService) UpdateAggregateRating(ctx context.Context, revieweeID uuid.UUID, revieweeType string) error {
	var avg float64
	if err := s.db.WithContext(ctx).
		Model(&model.OrderReview{}).
		Where("reviewee_id = ? AND reviewee_type = ?", revieweeID, revieweeType).
		Select("COALESCE(AVG(rating), 5.0)").
		Scan(&avg).Error; err != nil {
		return fmt.Errorf("aggregate rating: %w", err)
	}
	// Drivers key on user_id (dispatch selects dp.user_id AS driver_id);
	// merchants key on their own business-entity ID.
	if revieweeType == "driver" {
		return s.db.WithContext(ctx).Model(&model.DriverProfile{}).
			Where("user_id = ?", revieweeID).
			Update("rating", avg).Error
	}
	return s.db.WithContext(ctx).Model(&model.Merchant{}).
		Where("id = ?", revieweeID).
		Update("rating", avg).Error
}

// Thresholds for the top_rated badge.
const (
	topRatedMinReviews = 20
	topRatedMinAvg     = 4.8
)

// AwardBadgeIfEligible checks delivery milestones and awards badges.
var badgeMilestones = []struct {
	Count int
	Type  string
}{
	{1, "first_delivery"},
	{10, "10_deliveries"},
	{50, "50_deliveries"},
	{100, "100_deliveries"},
}

func (s *OrderService) AwardBadgeIfEligible(ctx context.Context, driverID uuid.UUID) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.Order{}).
		Where("driver_id = ? AND status = ?", driverID, model.OrderDelivered).
		Count(&count).Error; err != nil {
		return fmt.Errorf("badge: count deliveries: %w", err)
	}

	for _, m := range badgeMilestones {
		if int(count) >= m.Count {
			badge := model.DriverBadge{
				ID:                uuid.New(),
				DriverID:          driverID,
				BadgeType:         m.Type,
				OrderCountAtAward: int(count),
				AwardedAt:         time.Now(),
			}
			// Insert-if-absent against the unique (driver_id, badge_type)
			// index. FirstOrCreate is wrong here: badge carries a fresh
			// primary key, which GORM folds into the lookup conditions, so
			// it never matches an existing row and the insert then dies on
			// the constraint — breaking worker retries.
			if err := s.db.WithContext(ctx).
				Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "driver_id"}, {Name: "badge_type"}},
					DoNothing: true,
				}).
				Create(&badge).Error; err != nil {
				return fmt.Errorf("badge %s: %w", m.Type, err)
			}
		}
	}

	// top_rated: avg rating >= 4.8 with >= 20 reviews
	var reviewCount int64
	var avgRating float64
	if err := s.db.WithContext(ctx).Model(&model.OrderReview{}).
		Where("reviewee_id = ? AND reviewee_type = 'driver'", driverID).
		Count(&reviewCount).Error; err != nil {
		return fmt.Errorf("badge: count reviews: %w", err)
	}
	if err := s.db.WithContext(ctx).Model(&model.OrderReview{}).
		Where("reviewee_id = ? AND reviewee_type = 'driver'", driverID).
		Select("COALESCE(AVG(rating), 0)").Scan(&avgRating).Error; err != nil {
		return fmt.Errorf("badge: avg rating: %w", err)
	}
	if reviewCount >= topRatedMinReviews && avgRating >= topRatedMinAvg {
		badge := model.DriverBadge{ID: uuid.New(), DriverID: driverID, BadgeType: "top_rated", OrderCountAtAward: int(count), AwardedAt: time.Now()}
		if err := s.db.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "driver_id"}, {Name: "badge_type"}},
				DoNothing: true,
			}).
			Create(&badge).Error; err != nil {
			return fmt.Errorf("badge top_rated: %w", err)
		}
	}
	return nil
}

// GetDriverBadges returns all badges for a driver.
func (s *OrderService) GetDriverBadges(ctx context.Context, driverID uuid.UUID) ([]model.DriverBadge, error) {
	var badges []model.DriverBadge
	err := s.db.WithContext(ctx).Where("driver_id = ?", driverID).Order("awarded_at ASC").Find(&badges).Error
	return badges, err
}

// ListForMerchant returns the merchant's order queue, newest first, optionally
// filtered by status (e.g. "pending" for the incoming-orders tab). merchantID
// is the resolved model.Merchant.ID (business entity), not the login user ID.
func (s *OrderService) ListForMerchant(ctx context.Context, merchantID uuid.UUID, status string, cursor *uuid.UUID, limit int) ([]model.Order, error) {
	q := s.db.WithContext(ctx).
		Preload("Items").
		Where("merchant_id = ?", merchantID).
		Order("created_at DESC").
		Limit(limit)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if cursor != nil {
		q = q.Where("id < ?", *cursor)
	}
	var orders []model.Order
	err := q.Find(&orders).Error
	return orders, err
}

// GetStops returns all stops for a multi-drop package order, ordered by
// sequence, with recipient PII decrypted only for an authorized caller:
//   - the order's own customer, or an admin: every stop's recipient is visible
//     (it's the sender's own data / needed for dispute resolution)
//   - the assigned driver, while the order is in_transit: only the next
//     unconfirmed stop's recipient is visible — never past or future stops,
//     never after delivery. This is the "only for the active stop" rule.
//   - anyone else: recipient fields are omitted entirely.
func (s *OrderService) GetStops(ctx context.Context, orderID, requesterID uuid.UUID, requesterRole string) ([]OrderStopOut, error) {
	var order model.Order
	if err := s.db.WithContext(ctx).First(&order, orderID).Error; err != nil {
		return nil, err
	}

	var stops []model.OrderStop
	if err := s.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("sequence ASC").
		Find(&stops).Error; err != nil {
		return nil, err
	}

	// The active stop for a driver: lowest-sequence stop not yet confirmed/skipped.
	var activeStopID uuid.UUID
	for _, st := range stops {
		if st.Status != "confirmed" && st.Status != "skipped" {
			activeStopID = st.ID
			break
		}
	}

	isOwnerOrAdmin := requesterRole == "admin" || (requesterRole == "customer" && order.CustomerID == requesterID)
	isAssignedDriver := requesterRole == "driver" && order.DriverID != nil && *order.DriverID == requesterID

	out := make([]OrderStopOut, 0, len(stops))
	for _, st := range stops {
		view := OrderStopOut{
			ID:          st.ID,
			OrderID:     st.OrderID,
			Sequence:    st.Sequence,
			AddressID:   st.AddressID,
			Notes:       st.Notes,
			Status:      st.Status,
			ConfirmedAt: st.ConfirmedAt,
		}

		canSeeRecipient := isOwnerOrAdmin ||
			(isAssignedDriver && order.Status == model.OrderInTransit && st.ID == activeStopID)

		if canSeeRecipient && s.recipients != nil {
			if name, err := s.recipients.DecryptPtr(st.RecipientNameEnc); err == nil {
				view.RecipientName = name
			}
			if phone, err := s.recipients.DecryptPtr(st.RecipientPhoneEnc); err == nil {
				view.RecipientPhone = phone
			}
		}

		out = append(out, view)
	}
	return out, nil
}

// ConfirmStop marks a single stop as confirmed using the per-stop delivery code.
// When all stops are confirmed the order transitions to delivered and escrow settles.
func (s *OrderService) ConfirmStop(ctx context.Context, orderID, driverID uuid.UUID, sequence int, code string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Preload("Stops").First(&order, orderID).Error; err != nil {
			return fmt.Errorf("order not found")
		}
		if order.DriverID == nil || *order.DriverID != driverID {
			return fmt.Errorf("not the assigned driver")
		}
		if order.Status != model.OrderInTransit {
			return fmt.Errorf("order not in transit")
		}

		var stop *model.OrderStop
		for i := range order.Stops {
			if order.Stops[i].Sequence == sequence {
				stop = &order.Stops[i]
				break
			}
		}
		if stop == nil {
			return fmt.Errorf("stop %d not found", sequence)
		}
		if stop.Status == "confirmed" {
			return fmt.Errorf("stop already confirmed")
		}

		// Verify delivery code — uses the shared DeliveryCodeService
		if s.deliveryCodes != nil {
			if err := s.deliveryCodes.Verify(ctx, tx, orderID, code); err != nil {
				return err
			}
		}

		now := time.Now()
		stop.Status = "confirmed"
		stop.ConfirmedAt = &now
		if err := tx.Save(stop).Error; err != nil {
			return err
		}

		// Check if all stops are confirmed → settle and deliver
		allDone := true
		for _, st := range order.Stops {
			if st.ID == stop.ID {
				continue
			}
			if st.Status != "confirmed" {
				allDone = false
				break
			}
		}

		if allDone {
			if err := s.ledger.Settle(ctx, tx, &order, uuid.New()); err != nil {
				return fmt.Errorf("settlement: %w", err)
			}
			order.Status = model.OrderDelivered
			order.DeliveredAt = &now
			if err := tx.Save(&order).Error; err != nil {
				return err
			}
			return tx.Create(&model.OrderEvent{
				ID:         uuid.New(),
				OrderID:    orderID,
				FromStatus: model.OrderInTransit,
				ToStatus:   model.OrderDelivered,
				ActorID:    driverID,
				ActorRole:  "driver",
			}).Error
		}
		return nil
	})
}

// recipientPIIRetentionDays: recipient name/phone are third-party PII the
// recipient never consented to us storing — the sender attests consent
// per-order, not the recipient. NDPR data-minimization: purge once the
// order is settled and no dispute is open.
const recipientPIIRetentionDays = 30

// PurgeStaleRecipientPII nulls recipient_*_enc on delivered orders older than
// the retention window, skipping any order whose escrow hold is frozen (an
// open dispute may still need the recipient's identity). Also purges the
// same fields on that order's stops. Returns the number of orders purged.
func (s *OrderService) PurgeStaleRecipientPII(ctx context.Context) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -recipientPIIRetentionDays)

	var orderIDs []uuid.UUID
	err := s.db.WithContext(ctx).
		Model(&model.Order{}).
		Where("delivered_at IS NOT NULL AND delivered_at < ?", cutoff).
		Where("recipient_name_enc IS NOT NULL OR recipient_phone_enc IS NOT NULL").
		Where(`NOT EXISTS (
			SELECT 1 FROM escrow_holds eh
			WHERE eh.order_id = orders.id AND eh.status = ?
		)`, model.EscrowFrozen).
		Pluck("id", &orderIDs).Error
	if err != nil {
		return 0, fmt.Errorf("purge: find stale orders: %w", err)
	}
	if len(orderIDs) == 0 {
		return 0, nil
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Order{}).
			Where("id IN ?", orderIDs).
			Updates(map[string]interface{}{"recipient_name_enc": nil, "recipient_phone_enc": nil}).Error; err != nil {
			return fmt.Errorf("purge orders: %w", err)
		}
		if err := tx.Model(&model.OrderStop{}).
			Where("order_id IN ?", orderIDs).
			Updates(map[string]interface{}{"recipient_name_enc": nil, "recipient_phone_enc": nil}).Error; err != nil {
			return fmt.Errorf("purge stops: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return int64(len(orderIDs)), nil
}
