package grpcclient

import (
	"context"
	"time"

	catalogv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/catalog/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type WriteCatalogClient struct {
	client  catalogv1.CatalogWriteServiceClient
	timeout time.Duration
}

func NewWriteCatalogClient(addr string, timeout time.Duration) (*WriteCatalogClient, func() error, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	client := catalogv1.NewCatalogWriteServiceClient(conn)

	return &WriteCatalogClient{client: client, timeout: timeout}, conn.Close, nil
}

func (c WriteCatalogClient) CreateProducts(ctx context.Context, name string,
	description string,
	price float64) (*catalogv1.Product, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.client.CreateProduct(ctx, &catalogv1.CreateProductRequest{
		Name:        name,
		Description: description,
		Price:       price,
	})
	if err != nil {
		return nil, err
	}
	return response.Product, nil
}
