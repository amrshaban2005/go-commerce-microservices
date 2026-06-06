package handler

import (
	"net/http"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/dto"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/port"
	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	svc port.ProductService
}

func NewProductHandler(svc port.ProductService) *ProductHandler {
	return &ProductHandler{svc}
}

func (h *ProductHandler) GetAllProducts(c *gin.Context) {
	products, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	c.JSON(http.StatusOK, dto.ToProductResponses(products))
}
