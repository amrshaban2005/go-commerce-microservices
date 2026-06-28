package searchingproducts

import (
	"context"
	"errors"
	"strings"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/port"
)

type Handler struct {
	searchRepo port.ProductSearchRepository
}

func NewHandler(searchRepo port.ProductSearchRepository) *Handler {
	return &Handler{searchRepo: searchRepo}
}

func (h *Handler) Handle(ctx context.Context, query *Query) (*Result, error) {
	text := strings.TrimSpace(query.Text)
	if text == "" {
		return nil, errors.New("search text is required")
	}

	products, err := h.searchRepo.Search(ctx, text)
	if err != nil {
		return nil, err
	}

	return &Result{Products: products}, nil
}
