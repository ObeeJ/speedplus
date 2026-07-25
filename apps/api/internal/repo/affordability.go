package repo

import (
	"context"

	"gorm.io/gorm"
)

// VerticalAffordability holds the aggregated order cost data for one vertical.
type VerticalAffordability struct {
	Vertical    string  `json:"vertical"`
	MedianKobo  int64   `json:"medianKobo"`  // median order total in kobo
	MinKobo     int64   `json:"minKobo"`     // cheapest recent order
	MaxKobo     int64   `json:"maxKobo"`     // most expensive recent order
	MerchantCount int   `json:"merchantCount"` // open merchants within radius
}

// AffordabilityRepo computes what a given wallet balance can afford
// based on real-time prices from nearby merchants.
type AffordabilityRepo interface {
	// NearbyVerticalCosts returns aggregated order cost data for all verticals
	// with at least one open merchant within radiusKm of (lat, lng).
	// Uses delivered orders from the last 30 days as the price signal.
	NearbyVerticalCosts(ctx context.Context, lat, lng float64, radiusKm float64) ([]VerticalAffordability, error)
}

type affordabilityRepo struct{ db *gorm.DB }

func NewAffordabilityRepo(db *gorm.DB) AffordabilityRepo { return &affordabilityRepo{db: db} }

func (r *affordabilityRepo) NearbyVerticalCosts(ctx context.Context, lat, lng, radiusKm float64) ([]VerticalAffordability, error) {
	// Strategy:
	// 1. Find merchant IDs within radiusKm using PostGIS ST_DWithin (uses GIST index)
	// 2. Aggregate delivered orders for those merchants in the last 30 days
	// 3. Return median, min, max per vertical + count of open merchants
	//
	// Median approximation: percentile_cont(0.5) WITHIN GROUP (ORDER BY total_kobo)
	// This is a single pass over the partial index — O(k log n) where k = nearby merchants.
	//
	// All values are parameterised — no string concatenation.
	const query = `
		WITH nearby_merchants AS (
			SELECT m.id, m.vertical
			FROM   merchants m
			WHERE  m.is_open = true
			  AND  ST_DWithin(
			           ST_SetSRID(ST_MakePoint(m.lng, m.lat), 4326)::geography,
			           ST_SetSRID(ST_MakePoint(?, ?)::geography, 4326),
			           ? * 1000  -- km → metres
			       )
		),
		recent_orders AS (
			SELECT o.vertical,
			       o.total_kobo
			FROM   orders o
			JOIN   nearby_merchants nm ON nm.id = o.merchant_id
			WHERE  o.status     = 'delivered'
			  AND  o.created_at > NOW() - INTERVAL '30 days'
		)
		SELECT
			ro.vertical,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY ro.total_kobo), 0)::BIGINT AS median_kobo,
			COALESCE(MIN(ro.total_kobo), 0)                                                  AS min_kobo,
			COALESCE(MAX(ro.total_kobo), 0)                                                  AS max_kobo,
			COUNT(DISTINCT nm.id)                                                             AS merchant_count
		FROM   recent_orders ro
		JOIN   nearby_merchants nm ON nm.vertical = ro.vertical
		GROUP  BY ro.vertical
		ORDER  BY ro.vertical
	`

	var rows []struct {
		Vertical      string `gorm:"column:vertical"`
		MedianKobo    int64  `gorm:"column:median_kobo"`
		MinKobo       int64  `gorm:"column:min_kobo"`
		MaxKobo       int64  `gorm:"column:max_kobo"`
		MerchantCount int    `gorm:"column:merchant_count"`
	}

	if err := r.db.WithContext(ctx).Raw(query, lng, lat, radiusKm).Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]VerticalAffordability, 0, len(rows))
	for _, row := range rows {
		result = append(result, VerticalAffordability{
			Vertical:      row.Vertical,
			MedianKobo:    row.MedianKobo,
			MinKobo:       row.MinKobo,
			MaxKobo:       row.MaxKobo,
			MerchantCount: row.MerchantCount,
		})
	}
	return result, nil
}
