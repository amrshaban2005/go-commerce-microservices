package handler

import (
	"net/http"

	grpcclient "github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/adapter/grpc-client"
	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/dto"

	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	readCatalogClient  *grpcclient.ReadCatalogClient
	writeCatalogClient *grpcclient.WriteCatalogClient
}

func NewProductHandler(readCatalogClient *grpcclient.ReadCatalogClient, writeCatalogClient *grpcclient.WriteCatalogClient) *ProductHandler {
	return &ProductHandler{readCatalogClient, writeCatalogClient}
}

func (h *ProductHandler) GetProducts(c *gin.Context) {
	products, err := h.readCatalogClient.GetProducts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"failed to get products": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.FromCatalogProducts(products))
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req dto.CreateProductRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"invalid request body": err.Error()})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product, err := h.writeCatalogClient.CreateProducts(c.Request.Context(), req.Name, req.Description, req.Price)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"failed to create products": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, dto.FromCatalogProduct(product))
}
