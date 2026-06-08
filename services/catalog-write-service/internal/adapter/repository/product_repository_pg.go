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
	Status      string
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

func (r productRepositryPG) CreateWithOutbox(ctx context.Context, product *domain.Product, message *domain.OutboxMessage) error {

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		productDataModel := toProductDataModel(product)
		if err := tx.Create(&productDataModel).Error; err != nil {
			return err
		}

		outboxMessageDataModel := toOutboxDataModel(message)
		if err := tx.Create(&outboxMessageDataModel).Error; err != nil {
			return err
		}
		return nil
	})

}

func toProductDataModel(product *domain.Product) ProductDataModel {
	return ProductDataModel{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Status:      product.Status,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}
}
