package port

import "context"

type InboxRepository interface {
	IsProcessed(ctx context.Context, messageID string) (bool, error)
	SaveProcessed(ctx context.Context, messageID string, eventType string, payload []byte) error
}
