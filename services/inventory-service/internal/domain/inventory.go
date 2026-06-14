package domain

import "github.com/google/uuid"

const (
	ReservationStatusReserved = "RESERVED"
	ReservationStatusFailed   = "FAILED"
)

type ReserveStockItem struct {
	ProductID uuid.UUID
	Quantity  int
}
