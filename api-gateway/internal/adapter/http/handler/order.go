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

// CreateOrder godoc
// @Summary Create order
// @Description Create an order and start the stock reservation workflow
// @Tags Orders
// @Accept json
// @Produce json
// @Param request body dto.CreateOrderRequest true "Order data"
// @Success 201 {object} dto.OrderResponse
// @Failure 400 {object} dto.ErrorResponse
// @Router /orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req dto.CreateOrderRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body " + err.Error()})
		return
	}
	order, err := h.orderClient.CreateOrder(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "failed to create order " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, dto.FromOrderResponse(order))
}

// GetOrder godoc
// @Summary Get order
// @Description Return an order by its ID
// @Tags Orders
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} dto.OrderResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /orders/{id} [get]
func (h *OrderHandler) GetOrder(c *gin.Context) {
	order, err := h.orderClient.GetOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "failed to get order " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.FromOrderResponse(order))
}
