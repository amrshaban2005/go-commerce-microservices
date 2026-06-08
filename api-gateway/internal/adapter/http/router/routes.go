package router

import (
	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/adapter/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterProductRoutes(rg *gin.RouterGroup, h *handler.ProductHandler) {
	products := rg.Group("/products")
	{
		products.GET("", h.GetProducts)
		products.POST("", h.CreateProduct)
	}
}
