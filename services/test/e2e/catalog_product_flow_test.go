package e2e

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestCatalogProductFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := newAPIClient()
	created := productResponse{}
	client.doJSON(t, ctx, http.MethodPost, "/api/v1/products", map[string]any{
		"name":        "E2E Keyboard",
		"description": "Created from e2e test",
		"price":       100,
	}, http.StatusCreated, &created)

	if created.ID == "" {
		t.Fatal("expected created product id")
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		products := []productResponse{}
		client.doJSON(t, ctx, http.MethodGet, "/api/v1/products", nil, http.StatusOK, &products)

		for _, product := range products {
			if product.ID == created.ID {
				return
			}
		}

		select {
		case <-ctx.Done():
			t.Fatalf("product %s was not projected before deadline: %v", created.ID, ctx.Err())
		case <-ticker.C:
		}
	}
}
