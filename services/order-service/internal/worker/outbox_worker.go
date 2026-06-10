package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/port"
)

type OutboxWorker struct {
	outboxRepo port.OutboxRepository
	publisher  port.EventPublisher
	interval   time.Duration
	batchSize  int
}

func NewOutboxWorker(
	outboxRepo port.OutboxRepository,
	publisher port.EventPublisher,
	interval time.Duration,
	batchSize int,
) *OutboxWorker {
	return &OutboxWorker{
		outboxRepo: outboxRepo,
		publisher:  publisher,
		interval:   interval,
		batchSize:  batchSize,
	}
}

func (w *OutboxWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	slog.Info("outbox worker started")

	for {
		select {
		case <-ctx.Done():
			slog.Info("outbox worker stopped")
			return

		case <-ticker.C:
			w.process(ctx)
		}
	}
}

func (w *OutboxWorker) process(ctx context.Context) {
	messages, err := w.outboxRepo.FindUnprocessed(ctx, w.batchSize)
	if err != nil {
		slog.Error("failed to fetch outbox messages", "error", err)
		return
	}

	for _, message := range messages {
		err := w.publisher.Publish(ctx, message.EventType, message.Payload)
		if err != nil {
			_ = w.outboxRepo.IncrementRetry(ctx, message.ID)
			slog.Error("failed to publish outbox message", "id", message.ID, "error", err)
			continue
		}

		if err := w.outboxRepo.MarkAsProcessed(ctx, message.ID); err != nil {
			slog.Error("failed to mark outbox as processed", "id", message.ID, "error", err)
			continue
		}

		slog.Info("outbox message published", "id", message.ID, "event", message.EventType)
	}
}
