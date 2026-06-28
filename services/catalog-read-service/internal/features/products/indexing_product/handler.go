package indexingproduct

import (
	"context"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/port"
)

type Handler struct {
	searchRepo port.ProductSearchRepository
}

func NewHandler(searchRepo port.ProductSearchRepository) *Handler {
	return &Handler{searchRepo: searchRepo}
}

func (h *Handler) Handle(ctx context.Context, command *Command) (*struct{}, error) {
	if err := h.searchRepo.Index(ctx, command.Product); err != nil {
		return nil, err
	}

	return &struct{}{}, nil
}
