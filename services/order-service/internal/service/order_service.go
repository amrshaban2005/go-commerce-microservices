package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/domain"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/dto"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/port"
	"github.com/google/uuid"
)

type orderService struct {
	orderRepo port.OrderRespository
	inboxRepo port.InboxRepository
}

func NewOrderService(orderRepo port.OrderRespository, inboxRepo port.InboxRepository) port.OrderService {
	return &orderService{orderRepo, inboxRepo}
}

func (s *orderService) CreateOrder(ctx context.Context, customerID uuid.UUID, itemsInput []dto.CreateOrderItemInput) (*domain.Order, error) {

	orderItems := make([]domain.OrderItems, 0, len(itemsInput))

	for _, item := range itemsInput {
		orderItems = append(orderItems, domain.OrderItems{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			UnitPrice:   item.UnitPrice,
			Quantity:    item.Quantity,
		})
	}
	order, err := domain.NewOrder(customerID, orderItems)
	if err != nil {
		return nil, err
	}

	eventPayload := map[string]any{
		"message_id": uuid.New().String(),
		"order_id":   order.ID.String(),
		"items":      reserveStockItems(order.Items),
	}

	payloadBytes, err := json.Marshal(eventPayload)
	if err != nil {
		return nil, err
	}

	outboxMessage := &domain.OutboxMessage{
		ID:            uuid.New(),
		AggregateID:   order.ID,
		AggregateType: "Order",
		EventType:     "ReserveStockRequested",
		Payload:       payloadBytes,
		RetryCount:    0,
		CreatedAt:     time.Now().UTC(),
	}

	err = s.orderRepo.CreateWithOutbox(ctx, order, outboxMessage)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (s *orderService) HandleConfirmOrder(ctx context.Context, orderID uuid.UUID, messageID uuid.UUID, payload []byte) error {
	isProcessed, err := s.inboxRepo.IsProcessed(ctx, messageID)
	if err != nil {
		return err
	}
	if isProcessed {
		return nil
	}
	// handle confirm order
	if err := s.orderRepo.ConfirmOrder(ctx, orderID); err != nil {
		return err
	}

	return s.inboxRepo.SaveProcessed(ctx, messageID, "StockReserved", payload)
}

func (s *orderService) HandleRejectOrder(ctx context.Context, orderID uuid.UUID, messageID uuid.UUID, payload []byte) error {
	isProcessed, err := s.inboxRepo.IsProcessed(ctx, messageID)
	if err != nil {
		return err
	}
	if isProcessed {
		return nil
	}
	// handle reject order
	if err := s.orderRepo.RejectOrder(ctx, orderID); err != nil {
		return err
	}

	return s.inboxRepo.SaveProcessed(ctx, messageID, "StockReservationFailed", payload)
}

func (s *orderService) GetOrder(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	return s.orderRepo.GetOrder(ctx, orderID)
}

func reserveStockItems(items []domain.OrderItems) []map[string]any {
	result := make([]map[string]any, 0, len(items))

	for _, item := range items {
		result = append(result, map[string]any{
			"product_id": item.ProductID.String(),
			"quantity":   item.Quantity,
		})
	}
	return result
}
