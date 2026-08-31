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
	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/adapter/http/router"
	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/health"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Application struct {
	HTTPServer *http.Server
	Health     *health.Handler

	listener net.Listener
	closers  []func() error
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

	readCatalogClient, closeReadCatalogClient, err := grpcclient.NewReadCatalogClient(options.CatalogReadGrpcAddr)
	if err != nil {
		return nil, fmt.Errorf("create catalog read client: %w", err)
	}
	closers := []func() error{closeReadCatalogClient}

	writeCatalogClient, closeWriteCatalogClient, err := grpcclient.NewWriteCatalogClient(options.CatalogWriteGrpcAddr)
	if err != nil {
		_ = closeAll(closers)
		return nil, fmt.Errorf("create catalog write client: %w", err)
	}
	closers = append(closers, closeWriteCatalogClient)

	orderClient, closeOrderClient, err := grpcclient.NewOrderClient(options.OrderGrpcUrl)
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
	engine := buildRouter(healthHandler, readCatalogClient, writeCatalogClient, orderClient)

	return &Application{
		HTTPServer: &http.Server{
			Addr:              ":" + options.AppPort,
			Handler:           engine,
			ReadHeaderTimeout: 5 * time.Second,
		},
		Health:   healthHandler,
		listener: listener,
		closers:  closers,
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
}

func (a *Application) Shutdown(ctx context.Context) error {
	a.Health.SetReady(false)

	var shutdownErrors []error
	if err := a.HTTPServer.Shutdown(ctx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("http server shutdown: %w", err))
	}
	if err := closeAll(a.closers); err != nil {
		shutdownErrors = append(shutdownErrors, err)
	}

	return errors.Join(shutdownErrors...)
}

func (a *Application) Addr() net.Addr {
	return a.listener.Addr()
}

func buildRouter(
	healthHandler *health.Handler,
	readCatalogClient *grpcclient.ReadCatalogClient,
	writeCatalogClient *grpcclient.WriteCatalogClient,
	orderClient *grpcclient.OrderClient,
) *gin.Engine {
	productHandler := handler.NewProductHandler(readCatalogClient, writeCatalogClient)
	orderHandler := handler.NewOrderHandler(orderClient)

	engine := gin.Default()
	engine.GET("/health/liveness", gin.WrapF(healthHandler.Liveness))
	engine.GET("/health/readiness", gin.WrapF(healthHandler.Readiness))
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := engine.Group("/api/v1")
	router.RegisterProductRoutes(api, productHandler)
	router.RegisterOrderRoutes(api, orderHandler)

	return engine
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
