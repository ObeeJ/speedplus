package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
)

// ── Pricing ───────────────────────────────────────────────────────────────────

type QuoteRequest struct {
	MerchantID   string  `json:"merchantId"   binding:"required"`
	Vertical     string  `json:"vertical"     binding:"required"`
	SubtotalKobo int64   `json:"subtotalKobo" binding:"required,min=1"`
	OriginLat    float64 `json:"originLat"    binding:"required"`
	OriginLng    float64 `json:"originLng"    binding:"required"`
	DestLat      float64 `json:"destLat"      binding:"required"`
	DestLng      float64 `json:"destLng"      binding:"required"`
	// Package-only fields
	WeightKg     float64 `json:"weightKg"`
	SizeCategory string  `json:"sizeCategory"` // small|medium|large
}

type QuoteResponse struct {
	ID                   uuid.UUID `json:"id"`
	SubtotalKobo         int64     `json:"subtotalKobo"`
	DeliveryKobo         int64     `json:"deliveryKobo"`
	ServiceKobo          int64     `json:"serviceKobo"`
	WeatherSurchargeKobo int64     `json:"weatherSurchargeKobo"` // 0 when surcharge is off
	TotalKobo            int64     `json:"totalKobo"`
	DistanceKm           float64   `json:"distanceKm"`
	ETAMinutes           int       `json:"etaMinutes"`
	StopCount            int       `json:"stopCount,omitempty"`
	WeightKg             float64   `json:"weightKg,omitempty"`
	SizeCategory         string    `json:"sizeCategory,omitempty"`
	WeatherAdvisory      string    `json:"weatherAdvisory,omitempty"`
	ExpiresAt            time.Time `json:"expiresAt"`
}

// MultiStopQuoteRequest prices a package order with one pickup + N dropoffs.
// Stops must be in the order the sender wants them visited (or the order the
// frontend's OSRM-optimized suggestion produced).
type MultiStopQuoteRequest struct {
	MerchantID   string  `json:"merchantId"   binding:"required"`
	Vertical     string  `json:"vertical"     binding:"required"`
	SubtotalKobo int64   `json:"subtotalKobo" binding:"required,min=1"`
	OriginLat    float64 `json:"originLat"    binding:"required"`
	OriginLng    float64 `json:"originLng"    binding:"required"`
	Stops        []struct {
		Lat float64 `json:"lat" binding:"required"`
		Lng float64 `json:"lng" binding:"required"`
	} `json:"stops" binding:"required,min=1,max=6"`
	WeightKg     float64 `json:"weightKg"`
	SizeCategory string  `json:"sizeCategory"`
}

// ── Orders ────────────────────────────────────────────────────────────────────
// Note: order creation and cancellation request binding is done via inline
// structs in handler/orders.go — these DTO types are not used there.

type MoneyResponse struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type OrderItemResponse struct {
	ID               uuid.UUID     `json:"id"`
	ProductID        uuid.UUID     `json:"productId"`
	Name             string        `json:"name"`
	Quantity         int           `json:"quantity"`
	UnitPrice        MoneyResponse `json:"unitPrice"`
	Total            MoneyResponse `json:"total"`
	WeightKg         float64       `json:"weightKg,omitempty"`
	SizeCategory     string        `json:"sizeCategory,omitempty"`
	Customizations   *string       `json:"customizations,omitempty"`
	SubstitutionPref *string       `json:"substitutionPreference,omitempty"`
}

type OrderResponse struct {
	ID                uuid.UUID           `json:"id"`
	CustomerID        uuid.UUID           `json:"customerId"`
	MerchantID        uuid.UUID           `json:"merchantId"`
	DriverID          *uuid.UUID          `json:"driverId,omitempty"`
	DriverName        *string             `json:"driverName,omitempty"`
	DriverRating      *float64            `json:"driverRating,omitempty"`
	DriverVehicle     *string             `json:"driverVehicle,omitempty"`
	DriverPhone       *string             `json:"driverPhone,omitempty"`
	Vertical          string              `json:"vertical"`
	Status            model.OrderStatus   `json:"status"`
	PaymentMethod     string              `json:"paymentMethod"`
	Items             []OrderItemResponse `json:"items"`
	Subtotal          MoneyResponse       `json:"subtotal"`
	DeliveryFee       MoneyResponse       `json:"deliveryFee"`
	ServiceFee        MoneyResponse       `json:"serviceFee"`
	Tip               MoneyResponse       `json:"tip"`
	Total             MoneyResponse       `json:"total"`
	DeliveryAddressID uuid.UUID           `json:"deliveryAddressId"`
	PrescriptionID    *uuid.UUID          `json:"prescriptionId,omitempty"`
	ScheduledFor      *time.Time          `json:"scheduledFor,omitempty"`
	EstimatedAt       *time.Time          `json:"estimatedDeliveryAt,omitempty"`
	DeliveredAt       *time.Time          `json:"deliveredAt,omitempty"`
	CancelReason      *string             `json:"cancellationReason,omitempty"`
	TrackingRef       *string             `json:"trackingRef,omitempty"`
	DeclaredValueKobo *int64              `json:"declaredValueKobo,omitempty"`
	CreatedAt         time.Time           `json:"createdAt"`
	UpdatedAt         time.Time           `json:"updatedAt"`
}

// FromModel converts a model.Order to OrderResponse.
func OrderFromModel(o *model.Order) OrderResponse {
	items := make([]OrderItemResponse, len(o.Items))
	for i, item := range o.Items {
		items[i] = OrderItemResponse{
			ID:               item.ID,
			ProductID:        item.ProductID,
			Name:             item.Name,
			Quantity:         item.Quantity,
			UnitPrice:        MoneyResponse{Amount: item.UnitPriceKobo, Currency: "NGN"},
			Total:            MoneyResponse{Amount: item.TotalKobo, Currency: "NGN"},
			WeightKg:         item.WeightKg,
			SizeCategory:     item.SizeCategory,
			Customizations:   item.Customizations,
			SubstitutionPref: item.SubstitutionPref,
		}
	}
	return OrderResponse{
		ID:                o.ID,
		CustomerID:        o.CustomerID,
		MerchantID:        o.MerchantID,
		DriverID:          o.DriverID,
		Vertical:          o.Vertical,
		Status:            o.Status,
		PaymentMethod:     o.PaymentMethod,
		Items:             items,
		Subtotal:          MoneyResponse{Amount: o.SubtotalKobo, Currency: "NGN"},
		DeliveryFee:       MoneyResponse{Amount: o.DeliveryKobo, Currency: "NGN"},
		ServiceFee:        MoneyResponse{Amount: o.ServiceKobo, Currency: "NGN"},
		Tip:               MoneyResponse{Amount: o.TipKobo, Currency: "NGN"},
		Total:             MoneyResponse{Amount: o.TotalKobo, Currency: "NGN"},
		DeliveryAddressID: o.DeliveryAddressID,
		PrescriptionID:    o.PrescriptionID,
		ScheduledFor:      o.ScheduledFor,
		EstimatedAt:       o.EstimatedAt,
		DeliveredAt:       o.DeliveredAt,
		CancelReason:      o.CancelReason,
		TrackingRef:       o.TrackingRef,
		DeclaredValueKobo: o.DeclaredValueKobo,
		CreatedAt:         o.CreatedAt,
		UpdatedAt:         o.UpdatedAt,
	}
}
