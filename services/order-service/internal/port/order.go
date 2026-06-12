package port

import (
	"context"

	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/domain"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/dto"
	"github.com/google/uuid"
)

type OrderRespository interface {
	CreateWithOutbox(ctx context.Context, order *domain.Order, message *domain.OutboxMessage) error
	ConfirmOrder(ctx context.Context, orderID uuid.UUID) error
	RejectOrder(ctx context.Context, orderID uuid.UUID) error
}

type OrderService interface {
	CreateOrder(ctx context.Context, customerID uuid.UUID, itemsInput []dto.CreateOrderItemInput) (*domain.Order, error)
	HandleConfirmOrder(ctx context.Context, orderID uuid.UUID, messageID uuid.UUID, payload []byte) error
	HandleRejectOrder(ctx context.Context, orderID uuid.UUID, messageID uuid.UUID, payload []byte) error
}
