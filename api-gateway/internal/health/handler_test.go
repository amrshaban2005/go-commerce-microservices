package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLivenessIsAlwaysUp(t *testing.T) {
	handler := New()

	response := httptest.NewRecorder()
	handler.Liveness(response, httptest.NewRequest(http.MethodGet, "/health/liveness", nil))

	assertHealthResponse(t, response, http.StatusOK, "UP")
}

func TestReadinessFollowsLifecycleState(t *testing.T) {
	handler := New()

	response := httptest.NewRecorder()
	handler.Readiness(response, httptest.NewRequest(http.MethodGet, "/health/readiness", nil))
	assertHealthResponse(t, response, http.StatusServiceUnavailable, "DOWN")

	handler.SetReady(true)
	response = httptest.NewRecorder()
	handler.Readiness(response, httptest.NewRequest(http.MethodGet, "/health/readiness", nil))
	assertHealthResponse(t, response, http.StatusOK, "UP")

	handler.SetReady(false)
	response = httptest.NewRecorder()
	handler.Readiness(response, httptest.NewRequest(http.MethodGet, "/health/readiness", nil))
	assertHealthResponse(t, response, http.StatusServiceUnavailable, "DOWN")
}

func assertHealthResponse(t *testing.T, response *httptest.ResponseRecorder, expectedCode int, expectedStatus string) {
	t.Helper()

	if response.Code != expectedCode {
		t.Fatalf("expected status code %d, got %d", expectedCode, response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected application/json content type, got %q", contentType)
	}

	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != expectedStatus {
		t.Fatalf("expected health status %q, got %q", expectedStatus, body["status"])
	}
}
