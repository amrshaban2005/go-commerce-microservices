package gettingproducts

import (
	"context"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/port"
)

type Handler struct {
	productRepo port.ProductRepository
	cacheRepo   port.ProductCacheRepository
}

func NewHandler(productRepo port.ProductRepository, cacheRepo port.ProductCacheRepository) *Handler {
	return &Handler{productRepo: productRepo, cacheRepo: cacheRepo}
}

func (h *Handler) Handle(ctx context.Context, query *Query) (*Result, error) {
	cachedProducts, err := h.cacheRepo.GetProducts(ctx)
	if err != nil {
		return nil, err
	}

	if cachedProducts != nil {
		return &Result{Products: cachedProducts}, nil
	}

	products, err := h.productRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.cacheRepo.SetProducts(ctx, products); err != nil {
		return nil, err
	}

	return &Result{Products: products}, nil
}
