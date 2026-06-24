package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	OrderStatusPending   = "PENDING"
	OrderStatusConfirmed = "CONFIRMED"
	OrderStatusFailed    = "FAILED"
)

type Order struct {
	ID          uuid.UUID
	CustomerID  uuid.UUID
	Status      string
	TotalAmount float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Items       []OrderItems
}

type OrderItems struct {
	ID          uuid.UUID
	OrderID     uuid.UUID
	ProductID   uuid.UUID
	ProductName string
	UnitPrice   float64
	Quantity    int
	Subtotal    float64
}

func NewOrder(customerID uuid.UUID, items []OrderItems) (*Order, error) {

	orderID := uuid.New()
	now := time.Now().UTC()

	var total float64

	for i := range items {

		items[i].ID = uuid.New()
		items[i].OrderID = orderID
		items[i].Subtotal = items[i].UnitPrice * float64(items[i].Quantity)

		total += items[i].Subtotal
	}

	return &Order{
		ID:          orderID,
		CustomerID:  customerID,
		Status:      OrderStatusPending,
		TotalAmount: total,
		Items:       items,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}
