package dto

import "github.com/google/uuid"

type CreateOrderItemInput struct {
	ProductID   uuid.UUID
	ProductName string
	UnitPrice   float64
	Quantity    int
}
	