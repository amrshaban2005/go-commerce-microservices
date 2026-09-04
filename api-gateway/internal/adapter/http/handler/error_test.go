package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/dto"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWriteServiceErrorMapsTimeoutToGatewayTimeout(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "context deadline", err: context.DeadlineExceeded},
		{name: "grpc deadline", err: status.Error(codes.DeadlineExceeded, "deadline exceeded")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			response := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(response)

			writeServiceError(c, test.err, http.StatusInternalServerError, "request failed: ")

			if response.Code != http.StatusGatewayTimeout {
				t.Fatalf("expected status %d, got %d", http.StatusGatewayTimeout, response.Code)
			}
			var body dto.ErrorResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Error != "request timed out" {
				t.Fatalf("unexpected error response %q", body.Error)
			}
		})
	}
}

func TestWriteServiceErrorKeepsDefaultStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)

	writeServiceError(c, errors.New("unavailable"), http.StatusBadGateway, "request failed: ")

	if response.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, response.Code)
	}
}

func TestWriteServiceErrorMapsGRPCCodes(t *testing.T) {
	tests := []struct {
		name       string
		code       codes.Code
		wantStatus int
	}{
		{name: "invalid argument", code: codes.InvalidArgument, wantStatus: http.StatusBadRequest},
		{name: "not found", code: codes.NotFound, wantStatus: http.StatusNotFound},
		{name: "unavailable", code: codes.Unavailable, wantStatus: http.StatusBadGateway},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			response := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(response)

			writeServiceError(c, status.Error(test.code, test.name), http.StatusInternalServerError, "request failed: ")

			if response.Code != test.wantStatus {
				t.Fatalf("expected status %d, got %d", test.wantStatus, response.Code)
			}
		})
	}
}
