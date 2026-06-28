package handler

import (
	"net/http"
	"strings"

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

// SearchProducts godoc
// @Summary Search products
// @Description Search product names and descriptions using Elasticsearch
// @Tags Products
// @Produce json
// @Param q query string true "Search text"
// @Success 200 {array} dto.ProductResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /products/search [get]
func (h *ProductHandler) SearchProducts(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "search query is required"})
		return
	}

	products, err := h.readCatalogClient.SearchProducts(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to search products " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.FromCatalogProducts(products))
}

// GetProducts godoc
// @Summary List products
// @Description Return catalog products using the Redis/MongoDB read path
// @Tags Products
// @Produce json
// @Success 200 {array} dto.ProductResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /products [get]
func (h *ProductHandler) GetProducts(c *gin.Context) {
	products, err := h.readCatalogClient.GetProducts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "failed to get products " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.FromCatalogProducts(products))
}

// CreateProduct godoc
// @Summary Create product
// @Description Create a new catalog product
// @Tags Products
// @Accept json
// @Produce json
// @Param request body dto.CreateProductRequest true "Product data"
// @Success 201 {object} dto.ProductResponse
// @Failure 400 {object} dto.ErrorResponse
// @Router /products [post]
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req dto.CreateProductRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body " + err.Error()})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "error " + err.Error()})
		return
	}

	product, err := h.writeCatalogClient.CreateProducts(c.Request.Context(), req.Name, req.Description, req.Price)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "failed to create products " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, dto.FromCatalogProduct(product))
}
