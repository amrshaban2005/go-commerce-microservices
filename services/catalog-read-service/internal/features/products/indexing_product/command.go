package indexingproduct

import "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/domain"

type Command struct {
	Product domain.Product
}
