package service

import (
	"context"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/domain"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/port"
)

type productService struct {
	repo port.ProductRepository
}

func NewProductService(repo port.ProductRepository) port.ProductService {
	return &productService{repo}
}

func (p productService) List(ctx context.Context) ([]domain.Product, error) {
	products, err := p.repo.FindAll(ctx)

	if err != nil {
		return nil, err
	}

	return products, nil
}
