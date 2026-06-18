package worker

import (
	"context"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/port"
	"go.uber.org/zap"
)

type OutboxWorker struct {
	outboxRepo port.OutboxRepository
	publisher  port.EventPublisher
	interval   time.Duration
	batchSize  int
	logger     *zap.Logger
}

func NewOutboxWorker(
	outboxRepo port.OutboxRepository,
	publisher port.EventPublisher,
	interval time.Duration,
	batchSize int,
	logger *zap.Logger,
) *OutboxWorker {
	return &OutboxWorker{
		outboxRepo: outboxRepo,
		publisher:  publisher,
		interval:   interval,
		batchSize:  batchSize,
		logger:     logger,
	}
}

func (w *OutboxWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.logger.Info("outbox worker started")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("outbox worker stopped")
			return

		case <-ticker.C:
			w.process(ctx)
		}
	}
}

func (w *OutboxWorker) process(ctx context.Context) {
	messages, err := w.outboxRepo.FindUnprocessed(ctx, w.batchSize)
	if err != nil {
		w.logger.Error("failed to fetch outbox messages", zap.Error(err))
		return
	}

	for _, message := range messages {
		err := w.publisher.Publish(ctx, message.EventType, message.Payload)
		if err != nil {
			_ = w.outboxRepo.IncrementRetry(ctx, message.ID)
			w.logger.Error(
				"failed to publish outbox message",
				zap.Any("message_id", message.ID),
				zap.String("event_type", message.EventType),
				zap.Error(err),
			)
			continue
		}

		if err := w.outboxRepo.MarkAsProcessed(ctx, message.ID); err != nil {
			w.logger.Error(
				"failed to mark outbox as processed",
				zap.Any("message_id", message.ID),
				zap.Error(err),
			)
			continue
		}

		w.logger.Info(
			"outbox message published",
			zap.Any("message_id", message.ID),
			zap.String("event_type", message.EventType),
		)
	}
}
