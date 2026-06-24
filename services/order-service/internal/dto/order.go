package dto

import (
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
)

type CreateOrderInput struct {
	CustomerID uuid.UUID
	Items      []CreateOrderItemInput
}

type CreateOrderItemInput struct {
	ProductID   uuid.UUID
	ProductName string
	UnitPrice   float64
	Quantity    int
}

func (i CreateOrderInput) Validate() error {
	if err := validation.ValidateStruct(&i,
		validation.Field(&i.CustomerID, validation.Required),
		validation.Field(&i.Items, validation.Required, validation.Length(1, 0)),
	); err != nil {
		return err
	}
	for index, item := range i.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("items[%d]: %w", index, err)
		}
	}
	return nil
}

func (i CreateOrderItemInput) Validate() error {
	return validation.ValidateStruct(
		&i,
		validation.Field(&i.ProductID, validation.Required),
		validation.Field(&i.ProductName, validation.Required),
		validation.Field(&i.UnitPrice, validation.Min(0.0)),
		validation.Field(&i.Quantity, validation.Required, validation.Min(1)),
	)
}
