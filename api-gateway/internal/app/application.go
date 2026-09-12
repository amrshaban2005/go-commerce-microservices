package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	appconfig "github.com/amrshaban2005/go-commerce-microservices/api-gateway/config"
	grpcclient "github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/adapter/grpc-client"
	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/adapter/http/handler"
	httpmiddleware "github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/adapter/http/middleware"
	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/adapter/http/router"
	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/health"
	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/observability"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Application struct {
	HTTPServer       *http.Server
	ManagementServer *http.Server
	Health           *health.Handler

	listener           net.Listener
	managementListener net.Listener
	closers            []func() error
}

func New(options *appconfig.AppOptions) (*Application, error) {
	return newApplication(options, net.Listen)
}

func newApplication(options *appconfig.AppOptions, listen func(network, address string) (net.Listener, error)) (*Application, error) {
	if options == nil {
		return nil, errors.New("app options are required")
	}
	if err := options.Validate(); err != nil {
		return nil, fmt.Errorf("validate app options: %w", err)
	}

	readCatalogClient, closeReadCatalogClient, err := grpcclient.NewReadCatalogClient(
		options.CatalogReadGrpcAddr,
		options.GRPCReadTimeout,
	)
	if err != nil {
		return nil, fmt.Errorf("create catalog read client: %w", err)
	}
	closers := []func() error{closeReadCatalogClient}

	writeCatalogClient, closeWriteCatalogClient, err := grpcclient.NewWriteCatalogClient(
		options.CatalogWriteGrpcAddr,
		options.GRPCWriteTimeout,
	)
	if err != nil {
		_ = closeAll(closers)
		return nil, fmt.Errorf("create catalog write client: %w", err)
	}
	closers = append(closers, closeWriteCatalogClient)

	orderClient, closeOrderClient, err := grpcclient.NewOrderClient(
		options.OrderGrpcUrl,
		options.GRPCReadTimeout,
		options.GRPCWriteTimeout,
	)
	if err != nil {
		_ = closeAll(closers)
		return nil, fmt.Errorf("create order client: %w", err)
	}
	closers = append(closers, closeOrderClient)

	listener, err := listen("tcp", ":"+options.AppPort)
	if err != nil {
		_ = closeAll(closers)
		return nil, fmt.Errorf("listen on port %s: %w", options.AppPort, err)
	}

	healthHandler := health.New()
	registry := observability.NewRegistry()
	httpMetrics := observability.NewHTTPMetrics(registry)
	engine := buildRouter(readCatalogClient, writeCatalogClient, orderClient, httpMetrics, options.RequestTimeout)
	managementHandler := buildManagementHandler(healthHandler, observability.Handler(registry))

	managementListener, err := listen("tcp", ":"+options.ManagementPort)
	if err != nil {
		_ = listener.Close()
		_ = closeAll(closers)
		return nil, fmt.Errorf("listen on management port %s: %w", options.ManagementPort, err)
	}

	return &Application{
		HTTPServer: &http.Server{
			Addr:              ":" + options.AppPort,
			Handler:           engine,
			ReadHeaderTimeout: options.ReadHeaderTimeout,
			ReadTimeout:       options.ReadTimeout,
			WriteTimeout:      options.WriteTimeout,
			IdleTimeout:       options.IdleTimeout,
		},
		ManagementServer: &http.Server{
			Addr:              ":" + options.ManagementPort,
			Handler:           managementHandler,
			ReadHeaderTimeout: options.ReadHeaderTimeout,
			ReadTimeout:       options.ReadTimeout,
			WriteTimeout:      options.WriteTimeout,
			IdleTimeout:       options.IdleTimeout,
		},
		Health:             healthHandler,
		listener:           listener,
		managementListener: managementListener,
		closers:            closers,
	}, nil
}

func (a *Application) Run(errCh chan<- error) {
	a.Health.SetReady(true)

	go func() {
		if err := a.HTTPServer.Serve(a.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.Health.SetReady(false)
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()

	go func() {
		if err := a.ManagementServer.Serve(a.managementListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.Health.SetReady(false)
			errCh <- fmt.Errorf("management server: %w", err)
		}
	}()
}

func (a *Application) Shutdown(ctx context.Context) error {
	a.Health.SetReady(false)

	shutdownCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var shutdownErrors []error
	if err := a.HTTPServer.Shutdown(shutdownCtx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("http server shutdown: %w", err))
	}
	if err := a.ManagementServer.Shutdown(shutdownCtx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("management server shutdown: %w", err))
	}
	if err := closeAll(a.closers); err != nil {
		shutdownErrors = append(shutdownErrors, err)
	}

	return errors.Join(shutdownErrors...)
}

func (a *Application) Addr() net.Addr {
	return a.listener.Addr()
}

func (a *Application) ManagementAddr() net.Addr {
	return a.managementListener.Addr()
}

func buildRouter(
	readCatalogClient *grpcclient.ReadCatalogClient,
	writeCatalogClient *grpcclient.WriteCatalogClient,
	orderClient *grpcclient.OrderClient,
	httpMetrics *observability.HTTPMetrics,
	requestTimeout time.Duration,
) *gin.Engine {
	productHandler := handler.NewProductHandler(readCatalogClient, writeCatalogClient)
	orderHandler := handler.NewOrderHandler(orderClient)

	engine := gin.New()
	engine.Use(httpMetrics.Middleware(), gin.Logger(), gin.Recovery())
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := engine.Group("/api/v1")
	api.Use(httpmiddleware.RequestTimeout(requestTimeout))
	router.RegisterProductRoutes(api, productHandler)
	router.RegisterOrderRoutes(api, orderHandler)

	return engine
}

func buildManagementHandler(healthHandler *health.Handler, metricsHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health/liveness", healthHandler.Liveness)
	mux.HandleFunc("/health/readiness", healthHandler.Readiness)
	mux.Handle("/metrics", metricsHandler)
	return mux
}

func closeAll(closers []func() error) error {
	var closeErrors []error
	for i := len(closers) - 1; i >= 0; i-- {
		if err := closers[i](); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}

	return errors.Join(closeErrors...)
}
