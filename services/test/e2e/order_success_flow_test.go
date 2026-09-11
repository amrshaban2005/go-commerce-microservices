package e2e

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestSuccessOrderFlow(t *testing.T) {
	testOrderFlow(t, 1, "CONFIRMED")
}

func TestFailedOrderFlow(t *testing.T) {
	testOrderFlow(t, 100, "FAILED")
}

func testOrderFlow(t *testing.T, keyboardQuantity int, expectedStatus string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := newAPIClient()
	created := orderResponse{}
	client.doJSON(t, ctx, http.MethodPost, "/api/v1/orders", map[string]any{
		"customer_id": "98d3a6c2-57fc-490b-9baf-6acc0eed3d72",
		"order_items": []map[string]any{
			{
				"product_id":   "1c47247b-5f3e-41ae-bd3e-c3191ee63b99",
				"product_name": "keyboard",
				"unit_price":   60,
				"quantity":     keyboardQuantity,
			},
			{
				"product_id":   "a6500dba-cb86-42a8-86d2-033091952b15",
				"product_name": "mouse",
				"unit_price":   100,
				"quantity":     1,
			},
		},
	}, http.StatusCreated, &created)

	if created.ID == "" {
		t.Fatal("expected created order id")
	}

	waitForOrderStatus(t, ctx, client, created.ID, expectedStatus)
}

func waitForOrderStatus(
	t *testing.T,
	ctx context.Context,
	client *apiClient,
	orderID string,
	expectedStatus string,
) {
	t.Helper()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		order := orderResponse{}
		client.doJSON(
			t,
			ctx,
			http.MethodGet,
			fmt.Sprintf("/api/v1/orders/%s", orderID),
			nil,
			http.StatusOK,
			&order,
		)
		if order.Status == expectedStatus {
			return
		}

		select {
		case <-ctx.Done():
			t.Fatalf("order %s did not reach %s before deadline: %v", orderID, expectedStatus, ctx.Err())
		case <-ticker.C:
		}
	}
}
