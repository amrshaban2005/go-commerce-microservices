package grpcclient

import (
	"context"

	catalogv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/catalog/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ReadCatalogClient struct {
	client catalogv1.CatalogReadServiceClient
}

func NewReadCatalogClient(addr string) (*ReadCatalogClient, func() error, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	client := catalogv1.NewCatalogReadServiceClient(conn)

	return &ReadCatalogClient{client}, conn.Close, nil
}

func (c ReadCatalogClient) GetProducts(ctx context.Context) ([]*catalogv1.Product, error) {
	response, err := c.client.GetProducts(ctx, &catalogv1.GetProductsRequest{})
	if err != nil {
		return nil, err
	}
	return response.Products, nil

}

func (c ReadCatalogClient) SearchProducts(ctx context.Context, query string) ([]*catalogv1.Product, error) {
	response, err := c.client.SearchProducts(ctx, &catalogv1.SearchProductsRequest{Query: query})
	if err != nil {
		return nil, err
	}

	return response.Products, nil
}
