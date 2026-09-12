package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestHTTPMetricsRecordsRouteTemplateAndServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry := prometheus.NewRegistry()
	metrics := NewHTTPMetrics(registry)
	engine := gin.New()
	engine.Use(metrics.Middleware())
	engine.GET("/orders/:id", func(c *gin.Context) {
		c.Status(http.StatusInternalServerError)
	})

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/orders/order-123", nil))

	metricsResponse := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		metricsResponse,
		httptest.NewRequest(http.MethodGet, "/metrics", nil),
	)
	body := metricsResponse.Body.String()

	expected := []string{
		"http_server_requests_total{method=\"GET\",route=\"/orders/:id\",status=\"500\"} 1",
		"http_server_errors_total{method=\"GET\",route=\"/orders/:id\"} 1",
		"http_server_request_duration_seconds_count{method=\"GET\",route=\"/orders/:id\",status=\"500\"} 1",
		"http_server_requests_in_flight 0",
	}
	for _, metric := range expected {
		if !strings.Contains(body, metric) {
			t.Fatalf("expected metrics output to contain %q\n%s", metric, body)
		}
	}
	if strings.Contains(body, "order-123") {
		t.Fatal("metrics must use the route template, not the resource ID")
	}
}
