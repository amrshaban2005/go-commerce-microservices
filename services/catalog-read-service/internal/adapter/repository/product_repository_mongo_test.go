package repository

import (
	"context"
	"testing"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/domain"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func Test_Upsert(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	container, err := mongodb.Run(ctx, "mongo:7")
	if err != nil {
		t.Errorf("start mongo container but got %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Errorf("terminate mongo container: %v", err)
		}
	}()
	uri, err := container.ConnectionString(ctx)
	if err != nil {
		t.Errorf("get mongo connection string: %v", err)
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Errorf("expected connect to mongo db but got %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("catalog_read_db_test")

	productRepo := NewProductRepositoryMongo(db)

	product := domain.Product{
		ID:          "product-1",
		Name:        "Keyboard",
		Description: "Mechanical keyboard",
		Price:       100,
		Status:      "active",
	}

	err = productRepo.Upsert(ctx, product)
	if err != nil {
		t.Errorf("expected update product but got %v", err)
	}
	products, err := productRepo.FindAll(ctx)
	if err != nil {
		t.Errorf("expected tp find all products but got %v", err)
	}

	if len(products) != 1 {
		t.Errorf("expected product to find 1 product but got %d", len(products))
	}

	count, err := db.Collection("products").CountDocuments(ctx, bson.M{"_id": "product-1"})
	if err != nil {
		t.Fatalf("count documents: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one mongo document, got %d", count)
	}

}
