package port

import (
	"context"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/domain"
)

type ProductRepository interface {
	Upsert(ctx context.Context, product domain.Product) error
	FindAll(ctx context.Context) ([]domain.Product, error)
}

type ProductCacheRepository interface {
	GetProducts(ctx context.Context) ([]domain.Product, error)
	SetProducts(ctx context.Context, products []domain.Product) error
	DeleteProducts(ctx context.Context) error
}

type ProductSearchRepository interface {
	Index(ctx context.Context, product domain.Product) error
	Search(ctx context.Context, text string) ([]domain.Product, error)
}
