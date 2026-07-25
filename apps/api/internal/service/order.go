package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/crypto"
	"github.com/speedplus/api/internal/dto"
	"github.com/speedplus/api/internal/model"
	"github.com/speedplus/api/internal/repo"
	"gorm.io/gorm"
)

var (
	ErrIllegalTransition = errors.New("illegal order state transition")
	ErrQuoteInvalid      = errors.New("quote invalid or expired")
	ErrMerchantClosed    = errors.New("merchant is currently closed")
	ErrRxRequired        = errors.New("prescription required for this order")
	// POD checkout is gated by the trust ladder, but server-side POD
	// settlement (wallet debit at the door) is not yet implemented — orders
	// must be wallet-funded so escrow can settle. Remove once POD settlement
	// lands in LedgerService.Settle.
	ErrPODNotYetEnabled = errors.New("pay on arrival is not yet available — pay from wallet")
)

type CreateOrderInput struct {
	CustomerID     uuid.UUID
	MerchantID     uuid.UUID
	QuoteID        uuid.UUID
	Vertical       string
	Items          []OrderItemInput
	DeliveryAddrID uuid.UUID
	RecipientName  *string
	RecipientPhone *string
	PrescriptionID *uuid.UUID
	TipKobo        int64
	ScheduledFor   *time.Time
	PaymentMethod  string // ""|"wallet"|"pay_on_arrival"
	IdempotencyKey string
	// Multi-drop stops (package vertical only). When present, DeliveryAddrID
	// is the first stop's address and Stops[0] is the first drop-off.
	Stops []OrderStopInput
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

	// Pharmacy vertical requires approved prescription
	if in.Vertical == "pharmacy" && in.PrescriptionID == nil {
		return nil, ErrRxRequired
	}

	// Pay-on-arrival gate: trust-ladder checks run first so the customer gets
	// the precise reason (tier locked / cap / active POD). Even when eligible,
	// POD checkout is declined until POD settlement is implemented — creating
	// an escrow-less order today would leave it unsettleable at the door.
	if in.PaymentMethod == "pay_on_arrival" {
		if s.tier == nil {
			return nil, ErrPODNotYetEnabled
		}
		totalKobo := quote.TotalKobo + in.TipKobo
		if err := s.tier.CanUsePayOnArrival(ctx, in.CustomerID, totalKobo); err != nil {
			return nil, err
		}
		return nil, ErrPODNotYetEnabled
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
			IdempotencyKey:    in.IdempotencyKey,
		}
		if in.PaymentMethod == "pay_on_arrival" {
			order.PaymentMethod = "pay_on_arrival"
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

		// Escrow hold — debit customer wallet
		if err := s.ledger.HoldEscrow(ctx, tx, order.ID, in.CustomerID, order.TotalKobo); err != nil {
			return fmt.Errorf("escrow hold: %w", err)
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
func (s *OrderService) Transition(ctx context.Context, orderID, actorID uuid.UUID, actorRole string, to model.OrderStatus, note *string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Clauses(/* FOR UPDATE */ ).First(&order, orderID).Error; err != nil {
			return err
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

		return nil
	})
}

func (s *OrderService) GetByID(ctx context.Context, orderID, requesterID uuid.UUID, requesterRole string) (*model.Order, error) {
	var order model.Order
	if err := s.db.WithContext(ctx).Preload("Items").Preload("Events").First(&order, orderID).Error; err != nil {
		return nil, err
	}
	// Row-level ownership: customer sees own orders, driver sees assigned, merchant sees theirs, admin sees all
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
			FromStatus: order.Status,
			ToStatus:   model.OrderCancelled,
			ActorID:    actorUUID,
			ActorRole:  actorRole,
			Note:       &reason,
		}).Error; err != nil {
			return err
		}

		// Refund engine
		return s.ledger.ProcessCancellationRefund(ctx, tx, &order)
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
