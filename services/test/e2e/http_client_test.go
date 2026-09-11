package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

const defaultAPIGatewayURL = "http://127.0.0.1:8080"

type apiClient struct {
	baseURL string
	client  *http.Client
}

type productResponse struct {
	ID string `json:"id"`
}

type orderResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func newAPIClient() *apiClient {
	baseURL := os.Getenv("API_GATEWAY_URL")
	if baseURL == "" {
		baseURL = defaultAPIGatewayURL
	}

	return &apiClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{},
	}
}

func (client *apiClient) doJSON(
	t *testing.T,
	ctx context.Context,
	method string,
	path string,
	requestBody any,
	expectedStatus int,
	responseBody any,
) {
	t.Helper()

	var body io.Reader
	if requestBody != nil {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("encode %s %s request: %v", method, path, err)
		}
		body = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, body)
	if err != nil {
		t.Fatalf("create %s %s request: %v", method, path, err)
	}
	if requestBody != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := client.client.Do(request)
	if err != nil {
		t.Fatalf("send %s %s request: %v", method, path, err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != expectedStatus {
		payload, _ := io.ReadAll(io.LimitReader(response.Body, 8*1024))
		t.Fatalf(
			"%s %s returned %d, expected %d: %s",
			method,
			path,
			response.StatusCode,
			expectedStatus,
			string(payload),
		)
	}

	if responseBody == nil {
		return
	}
	if err := json.NewDecoder(response.Body).Decode(responseBody); err != nil {
		t.Fatalf("decode %s %s response: %v", method, path, err)
	}
}
