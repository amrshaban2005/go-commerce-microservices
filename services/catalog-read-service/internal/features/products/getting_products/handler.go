package gettingproducts

import (
	"context"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/port"
)

type Handler struct {
	productRepo port.ProductRepository
}

func NewHandler(productRepo port.ProductRepository) *Handler {
	return &Handler{productRepo: productRepo}
}

func (h *Handler) Handle(ctx context.Context, query *Query) (*Result, error) {
	products, err := h.productRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	return &Result{Products: products}, nil
}
