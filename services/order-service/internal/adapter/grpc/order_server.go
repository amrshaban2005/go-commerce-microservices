package grpcadapter

import (
	"context"

	orderv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/order/v1"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/domain"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/dto"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/port"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OrderServer struct {
	orderv1.UnimplementedOrderServiceServer
	svc port.OrderService
}

func NewOrderServer(svc port.OrderService) *OrderServer {
	return &OrderServer{svc: svc}
}

func (s *OrderServer) CreateOrder(ctx context.Context, in *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	customerID, err := uuid.Parse(in.CustomerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid customer_id")
	}

	itemsInput := make([]dto.CreateOrderItemInput, 0, len(in.Items))

	for _, item := range in.Items {
		productID, err := uuid.Parse(item.ProductId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid product_id")
		}
		itemsInput = append(itemsInput, dto.CreateOrderItemInput{
			ProductID:   productID,
			ProductName: item.ProductName,
			UnitPrice:   item.UnitPrice,
			Quantity:    int(item.Quantity),
		})
	}
	order, err := s.svc.CreateOrder(ctx, dto.CreateOrderInput{
		CustomerID: customerID,
		Items:      itemsInput,
	})

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &orderv1.CreateOrderResponse{
		Order: toProtoOrder(order),
	}, nil

}

func (s *OrderServer) GetOrder(ctx context.Context, in *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	orderID, err := uuid.Parse(in.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}
	order, err := s.svc.GetOrder(ctx, orderID)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &orderv1.GetOrderResponse{Order: toProtoOrder(order)}, nil
}

func toProtoOrder(order *domain.Order) *orderv1.Order {
	items := make([]*orderv1.OrderItem, 0, len(order.Items))

	for _, item := range order.Items {
		items = append(items, &orderv1.OrderItem{
			Id:          item.ID.String(),
			OrderId:     item.OrderID.String(),
			ProductId:   item.ProductID.String(),
			ProductName: item.ProductName,
			UnitPrice:   item.UnitPrice,
			Quantity:    int32(item.Quantity),
			Subtotal:    item.Subtotal,
		})
	}

	return &orderv1.Order{
		Id:          order.ID.String(),
		CustomerId:  order.CustomerID.String(),
		Status:      order.Status,
		TotalAmount: order.TotalAmount,
		Items:       items,
	}
}
