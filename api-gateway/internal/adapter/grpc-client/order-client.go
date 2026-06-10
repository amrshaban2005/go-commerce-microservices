package grpcclient

import (
	"context"

	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/dto"
	orderv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/order/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type OrderClient struct {
	client orderv1.OrderServiceClient
}

func NewOrderClient(addr string) (*OrderClient, func() error, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	client := orderv1.NewOrderServiceClient(conn)

	return &OrderClient{client}, conn.Close, nil
}

func (c OrderClient) CreateOrder(ctx context.Context, req *dto.CreateOrderRequest) (*orderv1.Order, error) {

	response, err := c.client.CreateOrder(ctx, dto.FromOrderRequest(req))
	if err != nil {
		return nil, err
	}
	return response.Order, nil

}
