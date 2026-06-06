package dto

import "github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/domain"

type ProductResponse struct {
	ID          string
	Name        string
	Description string
	Price       float64
}

func ToProductResponse(product domain.Product) ProductResponse {
	return ProductResponse{
		ID:          product.ID.String(),
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
	}
}

func ToProductResponses(products []domain.Product) []ProductResponse {
	responses := make([]ProductResponse, 0, len(products))

	for _, product := range products {
		responses = append(responses, ToProductResponse(product))
	}

	return responses
}
