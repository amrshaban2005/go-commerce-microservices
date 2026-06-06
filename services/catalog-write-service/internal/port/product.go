package port

import (
	"context"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/domain"
)

type ProductRepository interface {
	FindAll(ctx context.Context) ([]domain.Product, error)
}

type ProductService interface {
	List(ctx context.Context) ([]domain.Product, error)
}
