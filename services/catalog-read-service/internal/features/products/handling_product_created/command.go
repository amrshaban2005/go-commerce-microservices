package handlingproductcreated

import "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/domain"

type Command struct {
	MessageID string
	Product   domain.Product
	Payload   []byte
}
