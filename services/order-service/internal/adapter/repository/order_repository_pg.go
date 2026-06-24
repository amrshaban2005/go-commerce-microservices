package repository

import (
	"context"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/domain"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/port"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type orderRepositoryPG struct {
	db *gorm.DB
}

type OrderDataModel struct {
	ID          uuid.UUID            `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	CustomerID  uuid.UUID            `gorm:"ype:uuid;column:customer_id"`
	Status      string               `gorm:"column:status"`
	TotalAmount float64              `gorm:"column:total_amount"`
	CreatedAt   time.Time            `gorm:"column:created_at"`
	UpdatedAt   time.Time            `gorm:"column:updated_at"`
	Items       []OrderItemDataModel `gorm:"foreignKey:OrderID;references:ID"`
}

func (OrderDataModel) TableName() string {
	return "orders"
}

type OrderItemDataModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	OrderID     uuid.UUID `gorm:"type:uuid;column:order_id"`
	ProductID   uuid.UUID `gorm:"type:uuid;column:product_id"`
	ProductName string    `gorm:"column:product_name"`
	UnitPrice   float64   `gorm:"column:unit_price"`
	Quantity    int       `gorm:"column:quantity"`
	Subtotal    float64   `gorm:"column:subtotal"`
}

func (OrderItemDataModel) TableName() string {
	return "order_items"
}

type OutboxDataModel struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey"`
	AggregateID   uuid.UUID  `gorm:"type:uuid;column:aggregate_id"`
	AggregateType string     `gorm:"column:aggregate_type"`
	EventType     string     `gorm:"column:event_type"`
	Payload       []byte     `gorm:"column:payload;type:jsonb"`
	RetryCount    int        `gorm:"column:retry_count"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	ProcessedAt   *time.Time `gorm:"column:processed_at"`
}

func (OutboxDataModel) TableName() string {
	return "outbox_messages"
}

func NewOrderRepositoryPG(db *gorm.DB) port.OrderRespository {
	return &orderRepositoryPG{db}
}

func (r orderRepositoryPG) CreateWithOutbox(ctx context.Context, order *domain.Order, message *domain.OutboxMessage) error {

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		orderModel := toOrderDataModel(order)
		if err := tx.Create(&orderModel).Error; err != nil {
			return err
		}

		orderItemsModel := toOrderItemDataModels(order.Items)

		if err := tx.Create(&orderItemsModel).Error; err != nil {
			return err
		}

		outboxDataModel := toOutboxDataModel(message)
		if err := tx.Create(&outboxDataModel).Error; err != nil {
			return err
		}
		return nil
	})

}

func (r orderRepositoryPG) ConfirmOrder(ctx context.Context, orderID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&OrderDataModel{}).
		Where("id = ? AND status = ?", orderID, domain.OrderStatusPending).
		Updates(map[string]any{
			"status":     domain.OrderStatusConfirmed,
			"updated_at": time.Now().UTC(),
		}).
		Error
}

func (r orderRepositoryPG) RejectOrder(ctx context.Context, orderID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&OrderDataModel{}).
		Where("id = ? AND status = ?", orderID, domain.OrderStatusPending).
		Updates(map[string]any{
			"status":     domain.OrderStatusFailed,
			"updated_at": time.Now().UTC(),
		}).
		Error
}

func (r orderRepositoryPG) GetOrder(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	var orderModel OrderDataModel

	err := r.db.WithContext(ctx).Preload("Items").
		Where("id = ? ", orderID).First(&orderModel).Error

	if err != nil {
		return nil, err
	}
	return toDomainOrder(orderModel, orderModel.Items), nil
}

func toOrderDataModel(order *domain.Order) OrderDataModel {
	return OrderDataModel{
		ID:          order.ID,
		CustomerID:  order.CustomerID,
		Status:      order.Status,
		TotalAmount: order.TotalAmount,
		CreatedAt:   order.CreatedAt,
		UpdatedAt:   order.UpdatedAt,
	}
}

func toOrderItemDataModels(items []domain.OrderItems) []OrderItemDataModel {
	models := make([]OrderItemDataModel, 0, len(items))

	for _, item := range items {
		models = append(models, OrderItemDataModel{
			ID:          item.ID,
			OrderID:     item.OrderID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			UnitPrice:   item.UnitPrice,
			Quantity:    item.Quantity,
			Subtotal:    item.Subtotal,
		})
	}

	return models
}

func toDomainOrder(orderModel OrderDataModel, itemModels []OrderItemDataModel) *domain.Order {
	return &domain.Order{
		ID:          orderModel.ID,
		CustomerID:  orderModel.CustomerID,
		Status:      orderModel.Status,
		TotalAmount: orderModel.TotalAmount,
		Items:       toDomainOrderItems(itemModels),
		CreatedAt:   orderModel.CreatedAt,
		UpdatedAt:   orderModel.UpdatedAt,
	}
}

func toDomainOrderItems(itemModels []OrderItemDataModel) []domain.OrderItems {
	items := make([]domain.OrderItems, 0, len(itemModels))

	for _, item := range itemModels {
		items = append(items, domain.OrderItems{
			ID:          item.ID,
			OrderID:     item.OrderID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			UnitPrice:   item.UnitPrice,
			Quantity:    item.Quantity,
			Subtotal:    item.Subtotal,
		})
	}

	return items
}
