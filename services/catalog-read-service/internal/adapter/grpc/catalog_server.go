package grpcadapter

import (
	"context"

	catalogv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/catalog/v1"
	gettingproducts "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/features/products/getting_products"
	searchingproducts "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/features/products/searching_products"
	"github.com/mehdihadeli/go-mediatr"
)

type CatalogServer struct {
	catalogv1.UnimplementedCatalogReadServiceServer
}

func NewCatalogServer() *CatalogServer {
	return &CatalogServer{}
}

func (c *CatalogServer) GetProducts(ctx context.Context, req *catalogv1.GetProductsRequest) (*catalogv1.GetProductsResponse, error) {

	result, err := mediatr.Send[*gettingproducts.Query, *gettingproducts.Result](
		ctx,
		&gettingproducts.Query{},
	)
	if err != nil {
		return nil, err
	}

	products := result.Products
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

func (c *CatalogServer) SearchProducts(
	ctx context.Context,
	req *catalogv1.SearchProductsRequest,
) (*catalogv1.SearchProductsResponse, error) {
	result, err := mediatr.Send[*searchingproducts.Query, *searchingproducts.Result](
		ctx,
		&searchingproducts.Query{Text: req.Query},
	)
	if err != nil {
		return nil, err
	}

	response := make([]*catalogv1.Product, 0, len(result.Products))
	for _, product := range result.Products {
		response = append(response, &catalogv1.Product{
			Id:          product.ID,
			Name:        product.Name,
			Description: product.Description,
			Price:       product.Price,
		})
	}

	return &catalogv1.SearchProductsResponse{Products: response}, nil
}
