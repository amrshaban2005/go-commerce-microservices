package grpcadapter

import (
	"context"

	catalogv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/catalog/v1"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/port"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CatalogServer struct {
	catalogv1.UnimplementedCatalogWriteServiceServer
	svc port.ProductService
}

func NewCatalogServer(svc port.ProductService) *CatalogServer {
	return &CatalogServer{svc: svc}
}

func (c *CatalogServer) CreateProduct(ctx context.Context, req *catalogv1.CreateProductRequest) (*catalogv1.CreateProductResponse, error) {
	product, err := c.svc.CreateProduct(ctx, req.Name, req.Description, req.Price)

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &catalogv1.CreateProductResponse{
		Product: &catalogv1.Product{
			Id:          product.ID.String(),
			Name:        product.Name,
			Description: product.Description,
			Price:       product.Price,
		},
	}, nil

}
