package dto

import catalogv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/catalog/v1"

type ProductResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type CreateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

func FromCatalogProduct(product *catalogv1.Product) ProductResponse {
	return ProductResponse{
		ID:          product.Id,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
	}
}

func FromCatalogProducts(products []*catalogv1.Product) []ProductResponse {
	responses := make([]ProductResponse, 0, len(products))

	for _, product := range products {
		responses = append(responses, FromCatalogProduct(product))
	}

	return responses
}
