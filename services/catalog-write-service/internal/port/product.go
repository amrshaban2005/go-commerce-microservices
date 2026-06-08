package port

import (
	"context"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/domain"
)

type ProductRepository interface {
	CreateWithOutbox(ctx context.Context, product *domain.Product, message *domain.OutboxMessage) error
}

type ProductService interface {
	CreateProduct(ctx context.Context, name, description string, price float64) (*domain.Product, error)
}
