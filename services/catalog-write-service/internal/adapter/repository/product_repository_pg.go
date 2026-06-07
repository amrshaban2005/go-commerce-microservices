package repository

import (
	"context"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/domain"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/port"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductDataModel struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name        string
	Description string
	Price       float64
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (p *ProductDataModel) TableName() string {
	return "products"
}

type productRepositryPG struct {
	db *gorm.DB
}

func NewProductRepositryPG(db *gorm.DB) port.ProductRepository {
	return &productRepositryPG{db}
}

func (r productRepositryPG) FindAll(ctx context.Context) ([]domain.Product, error) {
	var products []ProductDataModel

	if err := r.db.WithContext(ctx).Find(&products).Error; err != nil {
		return nil, err
	}

	return toDomainProducts(products), nil
}

func toDomainProducts(models []ProductDataModel) []domain.Product {
	products := make([]domain.Product, 0, len(models))
	for _, model := range models {
		products = append(products, toDomainProduct(model))
	}
	return products
}

func toDomainProduct(model ProductDataModel) domain.Product {
	return domain.Product{
		ID:          model.ID,
		Name:        model.Name,
		Description: model.Description,
		Price:       model.Price,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}
