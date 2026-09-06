package worker

import (
	"context"
	"testing"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/domain"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type deadlineRecordingOutboxRepository struct {
	deadlines chan time.Duration
}

func (r deadlineRecordingOutboxRepository) FindUnprocessed(ctx context.Context, _ int) ([]domain.OutboxMessage, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		select {
		case r.deadlines <- 0:
		default:
		}
		return nil, nil
	}
	select {
	case r.deadlines <- time.Until(deadline):
	default:
	}
	return nil, nil
}

func (deadlineRecordingOutboxRepository) MarkAsProcessed(context.Context, uuid.UUID) error {
	return nil
}

func (deadlineRecordingOutboxRepository) IncrementRetry(context.Context, uuid.UUID) error {
	return nil
}

type noOpPublisher struct{}

func (noOpPublisher) Publish(context.Context, string, []byte) error { return nil }

func TestOutboxWorkerAddsProcessingDeadline(t *testing.T) {
	deadlines := make(chan time.Duration, 1)
	processingTimeout := 100 * time.Millisecond
	worker := NewOutboxWorker(
		deadlineRecordingOutboxRepository{deadlines: deadlines},
		noOpPublisher{},
		time.Millisecond,
		processingTimeout,
		1,
		zap.NewNop(),
	)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.Start(ctx)
	}()

	remaining := <-deadlines
	if remaining <= 0 || remaining > processingTimeout {
		t.Fatalf("processing deadline %s is outside the expected range", remaining)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after cancellation")
	}
}
