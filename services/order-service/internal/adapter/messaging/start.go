package messaging

import (
	"context"
	"fmt"
)

type Consumer interface {
	Start(ctx context.Context) error
}

type Consumers struct {
	StockReserved    Consumer
	StockNotReserved Consumer
}

func Start(ctx context.Context, group Consumers) error {
	errCh := make(chan error, 1)
	if group.StockReserved != nil {
		go func() {
			if err := group.StockReserved.Start(ctx); err != nil {
				errCh <- fmt.Errorf("stock reserved consumer: %w", err)
			}
		}()
	}

	if group.StockNotReserved != nil {
		go func() {
			if err := group.StockNotReserved.Start(ctx); err != nil {
				errCh <- fmt.Errorf("stock not reserved consumer: %w", err)
			}
		}()
	}

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return nil
	}
}
