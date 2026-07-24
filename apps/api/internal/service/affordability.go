package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/repo"
)

// VerticalAffordabilityResult is what the handler returns to the frontend.
type VerticalAffordabilityResult struct {
	Vertical      string `json:"vertical"`
	CanAfford     bool   `json:"canAfford"`
	MedianKobo    int64  `json:"medianKobo"`
	MinKobo       int64  `json:"minKobo"`
	MaxKobo       int64  `json:"maxKobo"`
	MerchantCount int    `json:"merchantCount"`
	// ShortfallKobo is how much more the user needs to afford the median order.
	// Zero when canAfford is true.
	ShortfallKobo int64  `json:"shortfallKobo"`
	// Summary is a human-readable line for the UI.
	// e.g. "Covers 1 meal" or "₦1,200 short for gas"
	Summary string `json:"summary"`
}

// AffordabilityService computes what a user's wallet balance can buy
// based on real-time prices from nearby merchants.
type AffordabilityService struct {
	ledger *LedgerService
	repo   repo.AffordabilityRepo
}

func NewAffordabilityService(ledger *LedgerService, r repo.AffordabilityRepo) *AffordabilityService {
	return &AffordabilityService{ledger: ledger, repo: r}
}

// GetAffordability returns the affordability map for a user at a given location.
// radiusKm defaults to 5.0 if zero.
func (s *AffordabilityService) GetAffordability(ctx context.Context, userID uuid.UUID, lat, lng, radiusKm float64) ([]VerticalAffordabilityResult, error) {
	if radiusKm <= 0 {
		radiusKm = 5.0
	}

	balance, err := s.ledger.GetBalance(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("affordability: get balance: %w", err)
	}

	costs, err := s.repo.NearbyVerticalCosts(ctx, lat, lng, radiusKm)
	if err != nil {
		return nil, fmt.Errorf("affordability: nearby costs: %w", err)
	}

	results := make([]VerticalAffordabilityResult, 0, len(costs))
	for _, c := range costs {
		canAfford := balance >= c.MedianKobo
		shortfall := int64(0)
		if !canAfford {
			shortfall = c.MedianKobo - balance
		}

		results = append(results, VerticalAffordabilityResult{
			Vertical:      c.Vertical,
			CanAfford:     canAfford,
			MedianKobo:    c.MedianKobo,
			MinKobo:       c.MinKobo,
			MaxKobo:       c.MaxKobo,
			MerchantCount: c.MerchantCount,
			ShortfallKobo: shortfall,
			Summary:       buildSummary(c.Vertical, canAfford, c.MedianKobo, shortfall),
		})
	}
	return results, nil
}

func buildSummary(vertical string, canAfford bool, medianKobo, shortfallKobo int64) string {
	label := verticalLabel(vertical)
	if canAfford {
		return fmt.Sprintf("Covers %s (avg %s)", label, naira(medianKobo))
	}
	return fmt.Sprintf("%s short for %s", naira(shortfallKobo), label)
}

func verticalLabel(v string) string {
	switch v {
	case "food":
		return "a meal"
	case "gas":
		return "gas"
	case "grocery":
		return "groceries"
	case "pharmacy":
		return "pharmacy"
	case "package":
		return "a delivery"
	default:
		return v
	}
}

func naira(kobo int64) string {
	nairaVal := kobo / 100
	rem := kobo % 100
	if rem == 0 {
		return fmt.Sprintf("₦%s", formatThousands(nairaVal))
	}
	return fmt.Sprintf("₦%s.%02d", formatThousands(nairaVal), rem)
}

func formatThousands(n int64) string {
	s := fmt.Sprintf("%d", n)
	out := ""
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += ","
		}
		out += string(ch)
	}
	return out
}
