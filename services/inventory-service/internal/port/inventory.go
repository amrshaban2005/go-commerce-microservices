package port

import (
	"context"

	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/domain"
	"github.com/google/uuid"
)

type InventoryRepository interface {
	ReserveStockWithInboxAndOutbox(
		ctx context.Context,
		messageID uuid.UUID,
		orderID uuid.UUID,
		items []domain.ReserveStockItem,
		incomingPayload []byte,
	) error
}

type InventoryService interface {
	HandleReserveStockRequested(
		ctx context.Context,
		messageID uuid.UUID,
		orderID uuid.UUID,
		items []domain.ReserveStockItem,
		incomingPayload []byte,
	) error
}
