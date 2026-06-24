package e2e

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	catalogv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/catalog/v1"
	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestCatalogProductFlow(t *testing.T) {
	if err := configloader.LoadDotEnv("../.env"); err != nil {
		log.Println("No local .env file found; using system environment variables")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	readConn, err := grpc.NewClient(os.Getenv("CATALOG_READ_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("connect catalog read service: %v", err)
	}
	defer readConn.Close()

	writeConn, err := grpc.NewClient(os.Getenv("CATALOG_WRITE_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("connect catalog write service: %v", err)
	}
	defer writeConn.Close()

	writeClient := catalogv1.NewCatalogWriteServiceClient(writeConn)
	readClient := catalogv1.NewCatalogReadServiceClient(readConn)

	createRes, err := writeClient.CreateProduct(ctx, &catalogv1.CreateProductRequest{
		Name:        "E2E Keyboard",
		Description: "Created from e2e test",
		Price:       100,
	})
	if err != nil {
		t.Fatalf("create product %v", err)
	}

	productID := createRes.Product.Id
	if productID == "" {
		t.Fatal("expected created product id")
	}

	readResp, err := readClient.GetProducts(ctx, &catalogv1.GetProductsRequest{})
	if err != nil {
		t.Fatalf("get product %v", err)
	}

	for _, product := range readResp.Products {
		if product.Id == productID {
			return
		}
	}

	select {
	case <-ctx.Done():
		t.Fatalf("product %s was not projected to catalog read service", productID)
	case <-time.After(500 * time.Millisecond):
	}
}
