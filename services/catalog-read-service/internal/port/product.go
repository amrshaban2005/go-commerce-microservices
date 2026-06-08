package port

import (
	"context"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/domain"
)

type ProductRepository interface {
	Upsert(ctx context.Context, product domain.Product) error
	FindAll(ctx context.Context) ([]domain.Product, error)
}

type ProductService interface {
	HandleProductCreated(ctx context.Context, messageID string, product domain.Product, payload []byte) error
	GetProducts(ctx context.Context) ([]domain.Product, error)
}
