package grpcadapter

import (
	"context"

	catalogv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/catalog/v1"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/port"
)

type CatalogServer struct {
	catalogv1.UnimplementedCatalogReadServiceServer
	svc port.ProductService
}

func NewCatalogServer(svc port.ProductService) *CatalogServer {
	return &CatalogServer{svc:svc}
}

func (c *CatalogServer) GetProducts(ctx context.Context, req *catalogv1.GetProductsRequest) (*catalogv1.GetProductsResponse, error) {

	products, err := c.svc.GetProducts(ctx)
	if err != nil {
		return nil, err
	}
	response := make([]*catalogv1.Product, 0, len(products))

	for _, product := range products {
		response = append(response, &catalogv1.Product{
			Id:          product.ID,
			Name:        product.Name,
			Description: product.Description,
			Price:       product.Price,
		})
	}
	return &catalogv1.GetProductsResponse{Products: response}, nil
}
