package domain

import (
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
