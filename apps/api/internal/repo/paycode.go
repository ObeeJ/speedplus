package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/speedplus/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PaycodeRepo interface {
	Create(ctx context.Context, pc *model.Paycode) error
	FindByNonce(ctx context.Context, orderID uuid.UUID, nonce string) (*model.Paycode, error)
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	FindByIDTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.Paycode, error)
	SaveTx(ctx context.Context, tx *gorm.DB, pc *model.Paycode) error
	LockOrderTx(ctx context.Context, tx *gorm.DB, orderID uuid.UUID) (*model.Order, error)
	SaveOrderTx(ctx context.Context, tx *gorm.DB, o *model.Order) error
	CreateOrderEventTx(ctx context.Context, tx *gorm.DB, e *model.OrderEvent) error
	FindActiveInTransitOrder(ctx context.Context, tx *gorm.DB, customerID uuid.UUID) (*model.Order, error)
	CountWeightProof(ctx context.Context, tx *gorm.DB, orderID uuid.UUID) (int64, error)
	FindAddress(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.Address, error)
}

type paycodeRepo struct{ db *gorm.DB }

func NewPaycodeRepo(db *gorm.DB) PaycodeRepo { return &paycodeRepo{db: db} }

func (r *paycodeRepo) Create(ctx context.Context, pc *model.Paycode) error {
	return r.db.WithContext(ctx).Create(pc).Error
}

func (r *paycodeRepo) FindByNonce(ctx context.Context, orderID uuid.UUID, nonce string) (*model.Paycode, error) {
	var pc model.Paycode
	err := r.db.WithContext(ctx).
		Where("order_id = ? AND nonce = ? AND confirmed_at IS NULL", orderID, nonce).
		First(&pc).Error
	return &pc, err
}

func (r *paycodeRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *paycodeRepo) FindByIDTx(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.Paycode, error) {
	var pc model.Paycode
	err := tx.WithContext(ctx).First(&pc, id).Error
	return &pc, err
}

func (r *paycodeRepo) SaveTx(ctx context.Context, tx *gorm.DB, pc *model.Paycode) error {
	return tx.WithContext(ctx).Save(pc).Error
}

func (r *paycodeRepo) LockOrderTx(ctx context.Context, tx *gorm.DB, orderID uuid.UUID) (*model.Order, error) {
	var o model.Order
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&o, orderID).Error
	return &o, err
}

func (r *paycodeRepo) SaveOrderTx(ctx context.Context, tx *gorm.DB, o *model.Order) error {
	return tx.WithContext(ctx).Save(o).Error
}

func (r *paycodeRepo) CreateOrderEventTx(ctx context.Context, tx *gorm.DB, e *model.OrderEvent) error {
	return tx.WithContext(ctx).Create(e).Error
}

func (r *paycodeRepo) FindActiveInTransitOrder(ctx context.Context, tx *gorm.DB, customerID uuid.UUID) (*model.Order, error) {
	var o model.Order
	err := tx.WithContext(ctx).
		Where("customer_id = ? AND status = ?", customerID, model.OrderInTransit).
		Order("created_at DESC").First(&o).Error
	return &o, err
}

func (r *paycodeRepo) CountWeightProof(ctx context.Context, tx *gorm.DB, orderID uuid.UUID) (int64, error) {
	var count int64
	err := tx.WithContext(ctx).Model(&model.ProofMedia{}).
		Where("order_id = ? AND kind = 'weight_photo'", orderID).Count(&count).Error
	return count, err
}

func (r *paycodeRepo) FindAddress(ctx context.Context, tx *gorm.DB, id uuid.UUID) (*model.Address, error) {
	var a model.Address
	err := tx.WithContext(ctx).First(&a, id).Error
	return &a, err
}
