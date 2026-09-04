package grpcclient

import (
	"context"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/dto"
	orderv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/order/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type OrderClient struct {
	client       orderv1.OrderServiceClient
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func NewOrderClient(addr string, readTimeout, writeTimeout time.Duration) (*OrderClient, func() error, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	client := orderv1.NewOrderServiceClient(conn)

	return &OrderClient{
		client:       client,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}, conn.Close, nil
}

func (c OrderClient) CreateOrder(ctx context.Context, req *dto.CreateOrderRequest) (*orderv1.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, c.writeTimeout)
	defer cancel()

	response, err := c.client.CreateOrder(ctx, dto.FromOrderRequest(req))
	if err != nil {
		return nil, err
	}
	return response.Order, nil

}

func (c OrderClient) GetOrder(ctx context.Context, orderID string) (*orderv1.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, c.readTimeout)
	defer cancel()

	response, err := c.client.GetOrder(ctx, &orderv1.GetOrderRequest{OrderId: orderID})
	if err != nil {
		return nil, err
	}
	return response.Order, nil
}

func (c OrderClient) GetOrders(ctx context.Context) ([]*orderv1.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, c.readTimeout)
	defer cancel()

	response, err := c.client.GetOrders(ctx, &orderv1.GetOrdersRequest{})
	if err != nil {
		return nil, err
	}
	return response.Orders, nil
}
