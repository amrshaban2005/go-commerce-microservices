package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID          uuid.UUID
	Name        string
	Description string
	Price       float64
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

var (
	errProductNameRequired  = errors.New("Product name is required")
	errProductPriceRequired = errors.New("Product price is required")
)

const PRODUCT_ACTIVE_STATUS = "ACTIVE"

func NewProduct(name, description string, price float64) (*Product, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errProductNameRequired
	}
	if price < 0 {
		return nil, errProductPriceRequired
	}

	now := time.Now().UTC()
	return &Product{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Price:       price,
		Status:      PRODUCT_ACTIVE_STATUS,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}
