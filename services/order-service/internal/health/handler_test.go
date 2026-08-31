package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLivenessIsUpWhenApplicationIsNotReady(t *testing.T) {
	handler := New()

	response := httptest.NewRecorder()
	handler.Liveness(response, httptest.NewRequest(http.MethodGet, "/health/liveness", nil))

	assertResponse(t, response, http.StatusOK, "UP")
}

func TestReadinessFollowsLifecycleState(t *testing.T) {
	handler := New()

	response := httptest.NewRecorder()
	handler.Readiness(response, httptest.NewRequest(http.MethodGet, "/health/readiness", nil))
	assertResponse(t, response, http.StatusServiceUnavailable, "DOWN")

	handler.SetReady(true)
	response = httptest.NewRecorder()
	handler.Readiness(response, httptest.NewRequest(http.MethodGet, "/health/readiness", nil))
	assertResponse(t, response, http.StatusOK, "UP")

	handler.SetReady(false)
	response = httptest.NewRecorder()
	handler.Readiness(response, httptest.NewRequest(http.MethodGet, "/health/readiness", nil))
	assertResponse(t, response, http.StatusServiceUnavailable, "DOWN")
}

func assertResponse(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	expectedCode int,
	expectedStatus string,
) {
	t.Helper()

	if recorder.Code != expectedCode {
		t.Fatalf("expected status code %d, got %d", expectedCode, recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected application/json content type, got %q", contentType)
	}

	var body response
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != expectedStatus {
		t.Fatalf("expected status %q, got %q", expectedStatus, body.Status)
	}
}
