package handler

import (
	"net/http"

	grpcclient "github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/adapter/grpc-client"
	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/dto"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	orderClient *grpcclient.OrderClient
}

func NewOrderHandler(orderClient *grpcclient.OrderClient) *OrderHandler {
	return &OrderHandler{orderClient}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req dto.CreateOrderRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"invalid request body": err.Error()})
		return
	}
	order, err := h.orderClient.CreateOrder(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"failed to create products": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, dto.FromOrderResponse(order))
}
