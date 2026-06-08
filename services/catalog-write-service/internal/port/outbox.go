package port

import (
	"context"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/domain"
	"github.com/google/uuid"
)

type OutboxRepository interface {
	FindUnprocessed(ctx context.Context, limit int) ([]domain.OutboxMessage, error)
	MarkAsProcessed(ctx context.Context, id uuid.UUID) error
	IncrementRetry(ctx context.Context, id uuid.UUID) error
}

type EventPublisher interface {
	Publish(ctx context.Context, eventType string, payload []byte) error
}
