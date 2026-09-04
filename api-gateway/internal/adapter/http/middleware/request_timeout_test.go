package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRequestTimeoutAddsDeadlineAndCancelsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(RequestTimeout(time.Millisecond))
	engine.GET("/", func(c *gin.Context) {
		if _, ok := c.Request.Context().Deadline(); !ok {
			t.Fatal("expected request deadline")
		}
		<-c.Request.Context().Done()
		c.Status(http.StatusGatewayTimeout)
	})

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected status %d, got %d", http.StatusGatewayTimeout, response.Code)
	}
}
