package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InventoryDataModel struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey"`
	ProductID         uuid.UUID `gorm:"type:uuid;column:product_id"`
	AvailableQuantity int       `gorm:"column:available_quantity"`
	ReservedQuantity  int       `gorm:"column:reserved_quantity"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (InventoryDataModel) TableName() string {
	return "inventory"
}

type StockReservationDataModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	OrderID   uuid.UUID `gorm:"type:uuid;column:order_id"`
	ProductID uuid.UUID `gorm:"type:uuid;column:product_id"`
	Quantity  int       `gorm:"column:quantity"`
	Status    string    `gorm:"column:status"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (StockReservationDataModel) TableName() string {
	return "stock_reservations"
}

type InboxMessageDataModel struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey"`
	MessageID   uuid.UUID      `gorm:"type:uuid;column:message_id"`
	EventType   string         `gorm:"column:event_type"`
	Payload     datatypes.JSON `gorm:"column:payload;type:jsonb"`
	ProcessedAt time.Time      `gorm:"column:processed_at"`
}

func (InboxMessageDataModel) TableName() string {
	return "inbox_messages"
}

type InventoryRepositoryPG struct {
	db *gorm.DB
}

func NewInventoryRepositoryPG(db *gorm.DB) *InventoryRepositoryPG {
	return &InventoryRepositoryPG{db: db}
}

func (r *InventoryRepositoryPG) ReserveStockWithInboxAndOutbox(
	ctx context.Context,
	messageID uuid.UUID,
	orderID uuid.UUID,
	items []domain.ReserveStockItem,
	incomingPayload []byte,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		alreadyProcessed, err := r.isMessageProcessed(tx, messageID)
		if err != nil {
			return err
		}

		if alreadyProcessed {
			return nil
		}

		success, reason, inventories, err := r.checkAndLockStock(tx, items)
		if err != nil {
			return err
		}

		if success {
			if err := r.reserveStock(tx, orderID, items, inventories); err != nil {
				return err
			}

			if err := r.insertOutbox(tx, orderID, "StockReserved", "", items); err != nil {
				return err
			}
		} else {
			if err := r.insertOutbox(tx, orderID, "StockReservationFailed", reason, items); err != nil {
				return err
			}
		}

		return r.insertInbox(tx, messageID, "ReserveStockRequested", incomingPayload)
	})
}

func (r *InventoryRepositoryPG) isMessageProcessed(tx *gorm.DB, messageID uuid.UUID) (bool, error) {
	var count int64

	err := tx.Model(&InboxMessageDataModel{}).
		Where("message_id = ?", messageID).
		Count(&count).Error

	return count > 0, err
}

func (r *InventoryRepositoryPG) checkAndLockStock(
	tx *gorm.DB,
	items []domain.ReserveStockItem,
) (bool, string, map[uuid.UUID]InventoryDataModel, error) {
	inventories := make(map[uuid.UUID]InventoryDataModel)

	for _, item := range items {
		var inventory InventoryDataModel

		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("product_id = ?", item.ProductID).
			First(&inventory).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, "inventory not found for product " + item.ProductID.String(), inventories, nil
		}

		if err != nil {
			return false, "", nil, err
		}

		if inventory.AvailableQuantity < item.Quantity {
			return false, "insufficient stock for product " + item.ProductID.String(), inventories, nil
		}

		inventories[item.ProductID] = inventory
	}

	return true, "", inventories, nil
}

func (r *InventoryRepositoryPG) reserveStock(
	tx *gorm.DB,
	orderID uuid.UUID,
	items []domain.ReserveStockItem,
	inventories map[uuid.UUID]InventoryDataModel,
) error {
	now := time.Now().UTC()

	for _, item := range items {
		inventory := inventories[item.ProductID]

		err := tx.Model(&InventoryDataModel{}).
			Where("id = ?", inventory.ID).
			Updates(map[string]any{
				"available_quantity": inventory.AvailableQuantity - item.Quantity,
				"reserved_quantity":  inventory.ReservedQuantity + item.Quantity,
				"updated_at":         now,
			}).Error

		if err != nil {
			return err
		}

		reservation := StockReservationDataModel{
			ID:        uuid.New(),
			OrderID:   orderID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Status:    domain.ReservationStatusReserved,
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := tx.Create(&reservation).Error; err != nil {
			return err
		}
	}

	return nil
}

func (r *InventoryRepositoryPG) insertOutbox(
	tx *gorm.DB,
	orderID uuid.UUID,
	eventType string,
	reason string,
	items []domain.ReserveStockItem,
) error {
	payloadBytes, err := buildStockEventPayload(orderID, reason, items)
	if err != nil {
		return err
	}

	outbox := OutboxDataModel{
		ID:            uuid.New(),
		AggregateID:   orderID,
		AggregateType: "Order",
		EventType:     eventType,
		Payload:       datatypes.JSON(payloadBytes),
		RetryCount:    0,
		CreatedAt:     time.Now().UTC(),
		ProcessedAt:   nil,
	}

	return tx.Create(&outbox).Error
}

func (r *InventoryRepositoryPG) insertInbox(
	tx *gorm.DB,
	messageID uuid.UUID,
	eventType string,
	payload []byte,
) error {
	inbox := InboxMessageDataModel{
		ID:          uuid.New(),
		MessageID:   messageID,
		EventType:   eventType,
		Payload:     datatypes.JSON(payload),
		ProcessedAt: time.Now().UTC(),
	}

	return tx.Create(&inbox).Error
}

func buildStockEventPayload(
	orderID uuid.UUID,
	reason string,
	items []domain.ReserveStockItem,
) ([]byte, error) {
	eventItems := make([]map[string]any, 0, len(items))

	for _, item := range items {
		eventItems = append(eventItems, map[string]any{
			"product_id": item.ProductID.String(),
			"quantity":   item.Quantity,
		})
	}

	payload := map[string]any{
		"message_id": uuid.New().String(),
		"order_id":   orderID.String(),
		"items":      eventItems,
	}

	if reason != "" {
		payload["reason"] = reason
	}

	return json.Marshal(payload)
}
