package grpcadapter

import (
	"context"

	catalogv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/catalog/v1"
)

type CatalogServer struct {
	catalogv1.UnimplementedCatalogReadServiceServer
}

func NewCatalogServer() *CatalogServer {
	return &CatalogServer{}
}

func (c *CatalogServer) GetProducts(ctx context.Context, req *catalogv1.GetProductsRequest) (*catalogv1.GetProductsResponse, error) {
	return &catalogv1.GetProductsResponse{
		Products: []*catalogv1.Product{
			{
				Id:          "11111111-1111-1111-1111-111111111111",
				Name:        "Logitech MX Master 3S",
				Description: "Wireless mouse",
				Price:       459.00,
			},
			{
				Id:          "22222222-2222-2222-2222-222222222222",
				Name:        "Keychron K2",
				Description: "Mechanical keyboard",
				Price:       399.00,
			},
		},
	}, nil
}
