package port

import (
	"context"

	"github.com/google/uuid"
)

type InboxRepository interface {
	IsProcessed(ctx context.Context, messageID uuid.UUID) (bool, error)
	SaveProcessed(ctx context.Context, messageID uuid.UUID, eventType string, payload []byte) error
}
