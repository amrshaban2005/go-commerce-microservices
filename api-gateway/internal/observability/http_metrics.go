package observability

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

const unmatchedRoute = "unmatched"

// HTTPMetrics contains the API gateway's request metrics.
type HTTPMetrics struct {
	requests *prometheus.CounterVec
	errors   *prometheus.CounterVec
	inFlight prometheus.Gauge
	duration *prometheus.HistogramVec
}

func NewHTTPMetrics(registerer prometheus.Registerer) *HTTPMetrics {
	metrics := &HTTPMetrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_server_requests_total",
			Help: "Total number of HTTP requests handled by the API gateway.",
		}, []string{"method", "route", "status"}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_server_errors_total",
			Help: "Total number of API gateway HTTP responses with a 5xx status.",
		}, []string{"method", "route"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "http_server_requests_in_flight",
			Help: "Current number of HTTP requests being handled by the API gateway.",
		}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_server_request_duration_seconds",
			Help:    "API gateway HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route", "status"}),
	}

	registerer.MustRegister(metrics.requests, metrics.errors, metrics.inFlight, metrics.duration)
	return metrics
}

func (m *HTTPMetrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		m.inFlight.Inc()

		defer func() {
			m.inFlight.Dec()

			route := c.FullPath()
			if route == "" {
				route = unmatchedRoute
			}
			status := c.Writer.Status()
			statusLabel := strconv.Itoa(status)

			m.requests.WithLabelValues(c.Request.Method, route, statusLabel).Inc()
			m.duration.WithLabelValues(c.Request.Method, route, statusLabel).Observe(time.Since(startedAt).Seconds())
			if status >= 500 {
				m.errors.WithLabelValues(c.Request.Method, route).Inc()
			}
		}()

		c.Next()
	}
}
