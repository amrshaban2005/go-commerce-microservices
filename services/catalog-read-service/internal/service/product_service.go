package service

import (
	"context"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/domain"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/port"
)

type productService struct {
	productRepo port.ProductRepository
	inboxRepo   port.InboxRepository
}

func NewProductService(productRepo port.ProductRepository, inboxRepo port.InboxRepository) port.ProductService {
	return &productService{productRepo, inboxRepo}
}

func (s productService) HandleProductCreated(ctx context.Context, messageID string, product domain.Product, payload []byte) error {

	isProcessed, err := s.inboxRepo.IsProcessed(ctx, messageID)
	if err != nil {
		return err
	}
	if isProcessed {
		return nil
	}

	if err := s.productRepo.Upsert(ctx, product); err != nil {
		return err
	}

	return s.inboxRepo.SaveProcessed(ctx, messageID, "ProductCreated", payload)

}

func (s productService) GetProducts(ctx context.Context) ([]domain.Product, error) {
	return s.productRepo.FindAll(ctx)
}
