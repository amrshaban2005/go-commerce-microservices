package gettingproducts

import "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/domain"

type Result struct {
	Products []domain.Product
}
