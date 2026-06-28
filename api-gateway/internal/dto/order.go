package dto

import orderv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/order/v1"

type CreateOrderRequest struct {
	CustomerID string            `json:"customer_id"`
	OrderItems []CreateOrderItem `json:"order_items"`
}

type CreateOrderItem struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	UnitPrice   float64 `json:"unit_price"`
	Quantity    int     `json:"quantity"`
}

type OrderResponse struct {
	ID          string      `json:"id"`
	CustomerID  string      `json:"customer_id"`
	Status      string      `json:"status"`
	TotalAmount float64     `json:"total_amount"`
	Items       []OrderItem `json:"items"`
}

type OrderItem struct {
	ID          string  `json:"id"`
	OrderID     string  `json:"order_id"`
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	UnitPrice   float64 `json:"unit_price"`
	Quantity    int     `json:"quantity"`
	Subtotal    float64 `json:"subtotal"`
}

func FromOrderResponse(order *orderv1.Order) OrderResponse {
	items := make([]OrderItem, 0, len(order.Items))

	for _, item := range order.Items {
		items = append(items, OrderItem{
			ID:          item.Id,
			OrderID:     item.OrderId,
			ProductID:   item.ProductId,
			ProductName: item.ProductName,
			UnitPrice:   item.UnitPrice,
			Quantity:    int(item.Quantity),
			Subtotal:    item.Subtotal,
		})
	}

	return OrderResponse{
		ID:          order.Id,
		CustomerID:  order.CustomerId,
		Status:      order.Status,
		TotalAmount: order.TotalAmount,
		Items:       items,
	}

}

func FromOrderRequest(order *CreateOrderRequest) *orderv1.CreateOrderRequest {
	items := make([]*orderv1.CreateOrderItem, 0, len(order.OrderItems))

	for _, item := range order.OrderItems {
		items = append(items, &orderv1.CreateOrderItem{
			ProductId:   item.ProductID,
			ProductName: item.ProductName,
			UnitPrice:   item.UnitPrice,
			Quantity:    int32(item.Quantity),
		})
	}

	return &orderv1.CreateOrderRequest{
		CustomerId: order.CustomerID,
		Items:      items,
	}

}
