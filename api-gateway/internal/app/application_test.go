package app

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	appconfig "github.com/amrshaban2005/go-commerce-microservices/api-gateway/config"
	"github.com/gin-gonic/gin"
)

func TestApplicationLifecycleControlsReadiness(t *testing.T) {
	gin.SetMode(gin.TestMode)

	listener := newBlockingListener()
	managementListener := newBlockingListener()
	listenCalls := 0
	application, err := newApplication(testOptions(), func(_, _ string) (net.Listener, error) {
		listenCalls++
		if listenCalls == 1 {
			return listener, nil
		}
		return managementListener, nil
	})
	if err != nil {
		t.Fatalf("create application: %v", err)
	}
	if application.Health.IsReady() {
		t.Fatal("application must not be ready before Run")
	}
	if application.HTTPServer.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("expected 5s read header timeout, got %v", application.HTTPServer.ReadHeaderTimeout)
	}
	if application.HTTPServer.ReadTimeout != 15*time.Second {
		t.Fatalf("expected 15s read timeout, got %v", application.HTTPServer.ReadTimeout)
	}
	if application.HTTPServer.WriteTimeout != 15*time.Second {
		t.Fatalf("expected 15s write timeout, got %v", application.HTTPServer.WriteTimeout)
	}
	if application.HTTPServer.IdleTimeout != 60*time.Second {
		t.Fatalf("expected 60s idle timeout, got %v", application.HTTPServer.IdleTimeout)
	}

	errCh := make(chan error, 1)
	application.Run(errCh)
	if !application.Health.IsReady() {
		t.Fatal("application must be ready after Run")
	}

	request := httptest.NewRequest(http.MethodGet, "/health/readiness", nil)
	response := httptest.NewRecorder()
	application.ManagementServer.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected readiness status %d, got %d", http.StatusOK, response.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode readiness response: %v", err)
	}
	if body["status"] != "UP" {
		t.Fatalf("expected readiness UP, got %q", body["status"])
	}

	metricsResponse := httptest.NewRecorder()
	application.ManagementServer.Handler.ServeHTTP(
		metricsResponse,
		httptest.NewRequest(http.MethodGet, "/metrics", nil),
	)
	if metricsResponse.Code != http.StatusOK {
		t.Fatalf("expected metrics status %d, got %d", http.StatusOK, metricsResponse.Code)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := application.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown application: %v", err)
	}
	if application.Health.IsReady() {
		t.Fatal("application must not be ready after Shutdown")
	}

	select {
	case runErr := <-errCh:
		t.Fatalf("unexpected runtime error: %v", runErr)
	default:
	}
}

func TestNewRejectsMissingOptions(t *testing.T) {
	application, err := New(nil)
	if err == nil {
		if application != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_ = application.Shutdown(shutdownCtx)
		}
		t.Fatal("expected missing options error")
	}
}

func testOptions() *appconfig.AppOptions {
	return &appconfig.AppOptions{
		AppPort:              "0",
		ManagementPort:       "0",
		CatalogReadGrpcAddr:  "127.0.0.1:6001",
		CatalogWriteGrpcAddr: "127.0.0.1:6002",
		OrderGrpcUrl:         "127.0.0.1:6005",
		RequestTimeout:       10 * time.Second,
		GRPCReadTimeout:      5 * time.Second,
		GRPCWriteTimeout:     8 * time.Second,
		ReadHeaderTimeout:    5 * time.Second,
		ReadTimeout:          15 * time.Second,
		WriteTimeout:         15 * time.Second,
		IdleTimeout:          60 * time.Second,
	}
}

type blockingListener struct {
	closed    chan struct{}
	closeOnce sync.Once
}

func newBlockingListener() *blockingListener {
	return &blockingListener{closed: make(chan struct{})}
}

func (l *blockingListener) Accept() (net.Conn, error) {
	<-l.closed
	return nil, net.ErrClosed
}

func (l *blockingListener) Close() error {
	l.closeOnce.Do(func() {
		close(l.closed)
	})
	return nil
}

func (l *blockingListener) Addr() net.Addr {
	return testAddr("127.0.0.1:0")
}

type testAddr string

func (a testAddr) Network() string {
	return "tcp"
}

func (a testAddr) String() string {
	return string(a)
}
