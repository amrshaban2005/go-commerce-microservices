package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/domain"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/port"
	"github.com/google/uuid"
)

type productService struct {
	repo port.ProductRepository
}

func NewProductService(repo port.ProductRepository) port.ProductService {
	return &productService{repo}
}

func (p productService) CreateProduct(ctx context.Context, name, description string, price float64) (*domain.Product, error) {
	product, err := domain.NewProduct(name, description, price)
	if err != nil {
		return nil, err
	}

	eventPayload := map[string]any{
		"message_id":  uuid.New().String(),
		"product_id":  product.ID.String(),
		"name":        product.Name,
		"description": product.Description,
		"price":       product.Price,
		"status":      product.Status,
	}

	payloadBytes, err := json.Marshal(eventPayload)
	if err != nil {
		return nil, err
	}

	outboxMessage := &domain.OutboxMessage{
		ID:            uuid.New(),
		AggregateID:   product.ID,
		AggregateType: "Product",
		EventType:     "ProductCreated",
		Payload:       payloadBytes,
		RetryCount:    0,
		CreatedAt:     time.Now().UTC(),
	}

	err = p.repo.CreateWithOutbox(ctx, product, outboxMessage)
	if err != nil {
		return nil, err
	}

	return product, nil

}
