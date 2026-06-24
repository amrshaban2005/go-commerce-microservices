package e2e

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	orderv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/order/v1"
	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Test_SuccessOrderFlow(t *testing.T) {
	if err := configloader.LoadDotEnv("../.env"); err != nil {
		log.Println("No local .env file found; using system environment variables")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(os.Getenv("ORDER_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("connect order service: %v", err)
	}
	defer conn.Close()

	orderClient := orderv1.NewOrderServiceClient(conn)
	response, err := orderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		CustomerId: "98d3a6c2-57fc-490b-9baf-6acc0eed3d72",
		Items: []*orderv1.CreateOrderItem{{
			ProductId:   "1c47247b-5f3e-41ae-bd3e-c3191ee63b99",
			ProductName: "keyboard",
			UnitPrice:   60,
			Quantity:    1,
		}, {
			ProductId:   "a6500dba-cb86-42a8-86d2-033091952b15",
			ProductName: "mouse",
			UnitPrice:   100,
			Quantity:    1,
		}},
	})
	if err != nil {
		t.Fatalf("error creating order %v", err)
	}

	orderID := response.Order.Id
	for {
		response, err := orderClient.GetOrder(ctx, &orderv1.GetOrderRequest{OrderId: orderID})
		if err != nil {
			t.Fatalf("error gettig order id %v error %v", orderID, err)
		}
		if response.Order.Status == "CONFIRMED" {
			return
		}

		select {
		case <-ctx.Done():
			t.Fatalf("order not confimred %s ", orderID)
		case <-time.After(500 * time.Millisecond):
		}
	}

}

func Test_FailOrderFlow(t *testing.T) {
	if err := configloader.LoadDotEnv("../.env"); err != nil {
		log.Println("No local .env file found; using system environment variables")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(os.Getenv("ORDER_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("connect order service: %v", err)
	}
	defer conn.Close()

	orderClient := orderv1.NewOrderServiceClient(conn)
	response, err := orderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		CustomerId: "98d3a6c2-57fc-490b-9baf-6acc0eed3d72",
		Items: []*orderv1.CreateOrderItem{{
			ProductId:   "1c47247b-5f3e-41ae-bd3e-c3191ee63b99",
			ProductName: "keyboard",
			UnitPrice:   60,
			Quantity:    100,
		}, {
			ProductId:   "a6500dba-cb86-42a8-86d2-033091952b15",
			ProductName: "mouse",
			UnitPrice:   100,
			Quantity:    1,
		}},
	})
	if err != nil {
		t.Fatalf("error creating order %v", err)
	}

	orderID := response.Order.Id
	for {
		response, err := orderClient.GetOrder(ctx, &orderv1.GetOrderRequest{OrderId: orderID})
		if err != nil {
			t.Fatalf("error gettig order id %v error %v", orderID, err)
		}
		if response.Order.Status == "FAILED" {
			return
		}

		select {
		case <-ctx.Done():
			t.Fatalf("order not failed %s ", orderID)
		case <-time.After(500 * time.Millisecond):
		}
	}

}

