package repository

import (
	"context"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/domain"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/port"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type outboxRepositoryPG struct {
	db *gorm.DB
}

type OutboxMessageDataModle struct {
	ID            uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	AggregateID   uuid.UUID  `gorm:"column:aggregate_id"`
	AggregateType string     `gorm:"column:aggregate_type"`
	EventType     string     `gorm:"column:event_type"`
	Payload       []byte     `gorm:"column:payload"`
	RetryCount    int        `gorm:"column:retry_count"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	ProcessedAt   *time.Time `gorm:"column:processed_at"`
}

func (OutboxMessageDataModle) TableName() string {
	return "outbox_messages"
}

func NewOutboxRepositoryPG(db *gorm.DB) port.OutboxRepository {
	return &outboxRepositoryPG{db}
}

func (r outboxRepositoryPG) FindUnprocessed(ctx context.Context, limit int) ([]domain.OutboxMessage, error) {
	var models []OutboxMessageDataModle

	err := r.db.WithContext(ctx).
		Where("processed_at IS NULL").
		Order("created_at ASC").
		Limit(limit).
		Find(&models).Error

	if err != nil {
		return nil, err
	}

	messages := make([]domain.OutboxMessage, 0, len(models))
	for _, model := range models {
		messages = append(messages, fromOutboxDataModel(&model))
	}

	return messages, nil

}

func (r outboxRepositoryPG) MarkAsProcessed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&OutboxMessageDataModle{}).Where("id = ?", id).Update("processed_at", time.Now().UTC()).Error
}

func (r outboxRepositoryPG) IncrementRetry(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&OutboxMessageDataModle{}).Where("id = ?", id).Update("retry_count", gorm.Expr("retry_count + 1")).Error
}

func toOutboxDataModel(message *domain.OutboxMessage) OutboxMessageDataModle {
	return OutboxMessageDataModle{
		ID:            message.ID,
		AggregateID:   message.AggregateID,
		AggregateType: message.AggregateType,
		EventType:     message.EventType,
		Payload:       message.Payload,
		RetryCount:    message.RetryCount,
		CreatedAt:     message.CreatedAt,
		ProcessedAt:   message.ProcessedAt,
	}
}

func fromOutboxDataModel(message *OutboxMessageDataModle) domain.OutboxMessage {
	return domain.OutboxMessage{
		ID:            message.ID,
		AggregateID:   message.AggregateID,
		AggregateType: message.AggregateType,
		EventType:     message.EventType,
		Payload:       message.Payload,
		RetryCount:    message.RetryCount,
		CreatedAt:     message.CreatedAt,
		ProcessedAt:   message.ProcessedAt,
	}
}
