package handlingproductcreated

import (
	"context"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/port"
)

type Handler struct {
	productRepo port.ProductRepository
	inboxRepo   port.InboxRepository
}

func NewHandler(
	productRepo port.ProductRepository,
	inboxRepo port.InboxRepository,
) *Handler {
	return &Handler{
		productRepo: productRepo,
		inboxRepo:   inboxRepo,
	}
}

func (h *Handler) Handle(ctx context.Context, command *Command) (*struct{}, error) {
	isProcessed, err := h.inboxRepo.IsProcessed(ctx, command.MessageID)
	if err != nil {
		return nil, err
	}
	if isProcessed {
		return &struct{}{}, nil
	}

	if err := h.productRepo.Upsert(ctx, command.Product); err != nil {
		return nil, err
	}

	return &struct{}{}, h.inboxRepo.SaveProcessed(
		ctx,
		command.MessageID,
		"ProductCreated",
		command.Payload,
	)
}
