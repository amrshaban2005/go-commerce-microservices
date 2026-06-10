package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const OrderStatusPending = "PENDING"

var (
	ErrOrderItemsRequired = errors.New("order must have at least one item")
	ErrInvalidQuantity    = errors.New("order item quantity must be greater than zero")
	ErrInvalidUnitPrice   = errors.New("order item unit price must be greater than or equal to zero")
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
	if len(items) == 0 {
		return nil, ErrOrderItemsRequired
	}

	orderID := uuid.New()
	now := time.Now().UTC()

	var total float64

	for i := range items {
		if items[i].Quantity <= 0 {
			return nil, ErrInvalidQuantity
		}

		if items[i].UnitPrice < 0 {
			return nil, ErrInvalidUnitPrice
		}

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
