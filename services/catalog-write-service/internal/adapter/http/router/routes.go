package router

import (
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/adapter/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterProductRoutes(rg *gin.RouterGroup, h *handler.ProductHandler) {
	products := rg.Group("/products")
	{
		products.GET("", h.GetAllProducts)
	}
}
