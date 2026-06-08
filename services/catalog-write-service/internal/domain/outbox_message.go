package domain

import (
	"time"

	"github.com/google/uuid"
)

type OutboxMessage struct {
	ID            uuid.UUID
	AggregateID   uuid.UUID
	AggregateType string
	EventType     string
	Payload       []byte
	RetryCount    int
	CreatedAt     time.Time
	ProcessedAt   *time.Time
}

