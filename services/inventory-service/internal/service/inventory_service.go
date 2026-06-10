package service

import (
	"context"

	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/domain"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/port"
	"github.com/google/uuid"
)

type inventoryService struct {
	inventoryRepo port.InventoryRepository
}

func NewInventoryService(inventoryRepo port.InventoryRepository) port.InventoryService {
	return &inventoryService{
		inventoryRepo: inventoryRepo,
	}
}

func (s *inventoryService) HandleReserveStockRequested(
	ctx context.Context,
	messageID uuid.UUID,
	orderID uuid.UUID,
	items []domain.ReserveStockItem,
	incomingPayload []byte,
) error {
	return s.inventoryRepo.ReserveStockWithInboxAndOutbox(
		ctx,
		messageID,
		orderID,
		items,
		incomingPayload,
	)
}
