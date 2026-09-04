package grpcclient

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/dto"
	catalogv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/catalog/v1"
	orderv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/order/v1"
	"google.golang.org/grpc"
)

type blockingReadCatalogClient struct {
	catalogv1.CatalogReadServiceClient
}

func (blockingReadCatalogClient) GetProducts(
	ctx context.Context,
	_ *catalogv1.GetProductsRequest,
	_ ...grpc.CallOption,
) (*catalogv1.GetProductsResponse, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

type recordingOrderClient struct {
	orderv1.OrderServiceClient
	deadlines chan time.Duration
}

func (c recordingOrderClient) CreateOrder(
	ctx context.Context,
	_ *orderv1.CreateOrderRequest,
	_ ...grpc.CallOption,
) (*orderv1.CreateOrderResponse, error) {
	c.recordDeadline(ctx)
	return &orderv1.CreateOrderResponse{}, nil
}

func (c recordingOrderClient) GetOrder(
	ctx context.Context,
	_ *orderv1.GetOrderRequest,
	_ ...grpc.CallOption,
) (*orderv1.GetOrderResponse, error) {
	c.recordDeadline(ctx)
	return &orderv1.GetOrderResponse{}, nil
}

func (c recordingOrderClient) recordDeadline(ctx context.Context) {
	deadline, ok := ctx.Deadline()
	if !ok {
		c.deadlines <- 0
		return
	}
	c.deadlines <- time.Until(deadline)
}

func TestReadCatalogClientStopsAtConfiguredDeadline(t *testing.T) {
	client := ReadCatalogClient{
		client:  blockingReadCatalogClient{},
		timeout: 10 * time.Millisecond,
	}

	_, err := client.GetProducts(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func TestOrderClientUsesReadAndWriteDeadlines(t *testing.T) {
	deadlines := make(chan time.Duration, 2)
	client := OrderClient{
		client:       recordingOrderClient{deadlines: deadlines},
		readTimeout:  time.Second,
		writeTimeout: 2 * time.Second,
	}

	if _, err := client.GetOrder(context.Background(), "order-1"); err != nil {
		t.Fatalf("get order: %v", err)
	}
	readDeadline := <-deadlines

	if _, err := client.CreateOrder(context.Background(), &dto.CreateOrderRequest{}); err != nil {
		t.Fatalf("create order: %v", err)
	}
	writeDeadline := <-deadlines

	if readDeadline <= 0 || readDeadline > client.readTimeout {
		t.Fatalf("read deadline %s is outside expected range", readDeadline)
	}
	if writeDeadline <= client.readTimeout || writeDeadline > client.writeTimeout {
		t.Fatalf("write deadline %s is outside expected range", writeDeadline)
	}
}

func TestOrderClientPreservesShorterParentDeadline(t *testing.T) {
	deadlines := make(chan time.Duration, 1)
	client := OrderClient{
		client:      recordingOrderClient{deadlines: deadlines},
		readTimeout: time.Second,
	}
	parentTimeout := 50 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), parentTimeout)
	defer cancel()

	if _, err := client.GetOrder(ctx, "order-1"); err != nil {
		t.Fatalf("get order: %v", err)
	}
	remaining := <-deadlines

	if remaining <= 0 || remaining > parentTimeout {
		t.Fatalf("deadline %s did not preserve shorter parent deadline", remaining)
	}
}
