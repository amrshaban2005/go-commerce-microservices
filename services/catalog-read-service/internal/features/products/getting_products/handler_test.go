package gettingproducts

import (
	"context"
	"errors"
	"testing"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/domain"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/mocks"
	"github.com/stretchr/testify/mock"
)

type fakeProductRepository struct {
	products []domain.Product
	err      error
}

func (f fakeProductRepository) FindAll(ctx context.Context) ([]domain.Product, error) {
	return f.products, f.err
}

func (f fakeProductRepository) Upsert(ctx context.Context, product domain.Product) error {
	return nil
}

func TestHandler_Handler_ReturnsProduct(t *testing.T) {
	products := []domain.Product{
		{
			ID:          "product-1",
			Name:        "Keyboard",
			Description: "Mechanical keyboard",
			Price:       100,
			Status:      "active",
		},
	}

	productRepo := mocks.NewProductRepository(t)
	productRepo.On("FindAll", mock.Anything).Return(products, nil).Once()

	handler := NewHandler(productRepo)
	result, err := handler.Handle(context.Background(), &Query{})
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
	if len(result.Products) != 1 {
		t.Fatalf("expected one product but got %d", len(result.Products))
	}
	if result.Products[0].ID != "product-1" {
		t.Fatalf("expected product id product-1 but got %s", result.Products[0].ID)
	}
}

func Test_Handler_ReturnRepositoryError(t *testing.T) {
	expectedErr := errors.New("mongo db error")
	productRepo := mocks.NewProductRepository(t)
	productRepo.On("FindAll", mock.Anything).Return(nil, expectedErr).Once()

	handler := NewHandler(productRepo)
	result, err := handler.Handle(context.Background(), &Query{})
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	if result != nil {
		t.Fatalf("expected no result but got result %d", len(result.Products))
	}
}
