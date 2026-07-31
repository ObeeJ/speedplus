package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
)

type SubscriptionRepo interface {
	CreateSubscription(ctx context.Context, sub *model.Subscription) error
	PauseByCustomer(ctx context.Context, subID, customerID uuid.UUID) error
	CancelByCustomer(ctx context.Context, subID, customerID uuid.UUID) error
	ListDue(ctx context.Context) ([]model.Subscription, error)
	UpdateDunning(ctx context.Context, subID uuid.UUID, dunningCount int, nextCharge time.Time) error
	PauseByID(ctx context.Context, subID uuid.UUID) error
	UpdateNextCharge(ctx context.Context, subID uuid.UUID, nextCharge time.Time) error
	FindCheapestProduct(ctx context.Context, merchantID uuid.UUID) (*model.Product, error)
	FindProduct(ctx context.Context, id uuid.UUID) (*model.Product, error)
	FindProductBySpec(ctx context.Context, merchantID, specID uuid.UUID) (*model.Product, error)
	FindCylinderSpec(ctx context.Context, specID uuid.UUID) (*model.CylinderSpec, error)
	BurnRateStats(ctx context.Context) ([]SubscriptionBurnRow, error)
	UpdateBurnRate(ctx context.Context, customerID string, avgDays float64, predictedRunout time.Time) error
	GetLiveLPGPrice(ctx context.Context, region string) (*model.LPGPriceIndex, error)
	GetPrevLPGPrice(ctx context.Context, region string) (*model.LPGPriceIndex, error)
	CreateLPGPriceRow(ctx context.Context, row *model.LPGPriceIndex) error
}

type SubscriptionBurnRow struct {
	CustomerID          string
	AvgDaysBetweenFills float64
	LastDeliveredAt     time.Time
}

type subscriptionRepo struct{ db *gorm.DB }

func NewSubscriptionRepo(db *gorm.DB) SubscriptionRepo { return &subscriptionRepo{db: db} }

func (r *subscriptionRepo) CreateSubscription(ctx context.Context, sub *model.Subscription) error {
	return r.db.WithContext(ctx).Create(sub).Error
}

func (r *subscriptionRepo) PauseByCustomer(ctx context.Context, subID, customerID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.Subscription{}).
		Where("id = ? AND customer_id = ? AND status = 'active'", subID, customerID).
		Updates(map[string]interface{}{"status": "paused", "paused_reason": "customer"}).Error
}

func (r *subscriptionRepo) CancelByCustomer(ctx context.Context, subID, customerID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.Subscription{}).
		Where("id = ? AND customer_id = ?", subID, customerID).
		Update("status", "cancelled").Error
}

func (r *subscriptionRepo) ListDue(ctx context.Context) ([]model.Subscription, error) {
	var subs []model.Subscription
	err := r.db.WithContext(ctx).Where("status = 'active' AND next_charge_at <= NOW()").Find(&subs).Error
	return subs, err
}

func (r *subscriptionRepo) UpdateDunning(ctx context.Context, subID uuid.UUID, dunningCount int, nextCharge time.Time) error {
	return r.db.WithContext(ctx).Model(&model.Subscription{}).Where("id = ?", subID).
		Updates(map[string]interface{}{"dunning_count": dunningCount, "next_charge_at": nextCharge}).Error
}

func (r *subscriptionRepo) PauseByID(ctx context.Context, subID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.Subscription{}).Where("id = ?", subID).
		Updates(map[string]interface{}{"status": "paused", "paused_reason": "dunning"}).Error
}

func (r *subscriptionRepo) UpdateNextCharge(ctx context.Context, subID uuid.UUID, nextCharge time.Time) error {
	return r.db.WithContext(ctx).Model(&model.Subscription{}).Where("id = ?", subID).
		Updates(map[string]interface{}{"dunning_count": 0, "next_charge_at": nextCharge}).Error
}

func (r *subscriptionRepo) FindCheapestProduct(ctx context.Context, merchantID uuid.UUID) (*model.Product, error) {
	var p model.Product
	err := r.db.WithContext(ctx).
		Where("merchant_id = ? AND is_available = true", merchantID).
		Order("price_kobo ASC").First(&p).Error
	return &p, err
}

func (r *subscriptionRepo) FindProduct(ctx context.Context, id uuid.UUID) (*model.Product, error) {
	var p model.Product
	err := r.db.WithContext(ctx).First(&p, id).Error
	return &p, err
}

func (r *subscriptionRepo) FindProductBySpec(ctx context.Context, merchantID, specID uuid.UUID) (*model.Product, error) {
	var p model.Product
	err := r.db.WithContext(ctx).
		Where("merchant_id = ? AND cylinder_spec_id = ? AND is_available = true", merchantID, specID).
		First(&p).Error
	return &p, err
}

func (r *subscriptionRepo) FindCylinderSpec(ctx context.Context, specID uuid.UUID) (*model.CylinderSpec, error) {
	var spec model.CylinderSpec
	err := r.db.WithContext(ctx).First(&spec, "id = ?", specID).Error
	return &spec, err
}

func (r *subscriptionRepo) BurnRateStats(ctx context.Context) ([]SubscriptionBurnRow, error) {
	var rows []SubscriptionBurnRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			o.customer_id::text AS customer_id,
			AVG(gap_days)       AS avg_days_between_fills,
			MAX(o.delivered_at) AS last_delivered_at
		FROM (
			SELECT customer_id, delivered_at,
				EXTRACT(EPOCH FROM (delivered_at - LAG(delivered_at) OVER (
					PARTITION BY customer_id ORDER BY delivered_at
				))) / 86400.0 AS gap_days
			FROM orders
			WHERE vertical = 'gas' AND status = 'delivered' AND delivered_at IS NOT NULL
		) o
		WHERE gap_days IS NOT NULL
		GROUP BY o.customer_id
	`).Scan(&rows).Error
	return rows, err
}

func (r *subscriptionRepo) UpdateBurnRate(ctx context.Context, customerID string, avgDays float64, predictedRunout time.Time) error {
	return r.db.WithContext(ctx).Model(&model.Subscription{}).
		Where("customer_id = ? AND vertical = 'gas' AND status = 'active'", customerID).
		Updates(map[string]interface{}{
			"avg_days_between_refills": avgDays,
			"predicted_runout_at":      predictedRunout,
		}).Error
}

func (r *subscriptionRepo) GetLiveLPGPrice(ctx context.Context, region string) (*model.LPGPriceIndex, error) {
	var row model.LPGPriceIndex
	err := r.db.WithContext(ctx).
		Where("region = ? AND effective_at <= NOW()", region).
		Order("effective_at DESC").First(&row).Error
	return &row, err
}

func (r *subscriptionRepo) GetPrevLPGPrice(ctx context.Context, region string) (*model.LPGPriceIndex, error) {
	return r.GetLiveLPGPrice(ctx, region)
}

func (r *subscriptionRepo) CreateLPGPriceRow(ctx context.Context, row *model.LPGPriceIndex) error {
	return r.db.WithContext(ctx).Create(row).Error
}
