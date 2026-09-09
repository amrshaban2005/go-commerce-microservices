package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	catalogv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/catalog/v1"
	applogger "github.com/amrshaban2005/go-commerce-microservices/pkg/logger"
	appconfig "github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/config"
	grpcadapter "github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/adapter/grpc"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/adapter/messaging"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/adapter/repository"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/database"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/health"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/port"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/service"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/worker"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

func Module() fx.Option {
	return fx.Options(
		fx.Provide(
			provideAppOptions,
			providePostgresOptions,
			provideRabbitMQOptions,
			provideLogger,
			provideDB,
			repository.NewProductRepositryPG,
			repository.NewOutboxRepositoryPG,
			service.NewProductService,
			providePublisher,
			provideOutboxWorker,
			provideHealthHandler,
		),
		fx.Invoke(
			StartOutboxWorker,
			StartGRPCServer,
			StartHealthServer,
			ManageReadiness,
		),
	)
}

func provideAppOptions() (*appconfig.AppOptions, error) {
	appOptions, err := appconfig.LoadAppOptions()
	if err != nil {
		return nil, err
	}
	return appOptions, appOptions.Validate()
}

func providePostgresOptions() (*database.PostgresOptions, error) {
	postgresOptions, err := database.LoadPostgresOptions()
	if err != nil {
		return nil, err
	}
	return postgresOptions, postgresOptions.Validate()
}

func provideRabbitMQOptions() (*messaging.RabbitMQOptions, error) {
	rabbitMQOptions, err := messaging.LoadRabbitMQOptions()
	if err != nil {
		return nil, err
	}
	return rabbitMQOptions, rabbitMQOptions.Validate()
}

func provideHealthHandler() *health.Handler {
	return health.New()
}

func provideLogger(lifeCycle fx.Lifecycle) (*zap.Logger, error) {
	options, err := applogger.LoadOptions()
	if err != nil {
		return nil, err
	}
	if err := options.Validate(); err != nil {
		return nil, err
	}

	logger, err := applogger.New(*options, "catalog-write-service")

	lifeCycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return applogger.Sync(logger)
		},
	})

	return logger, err
}

func provideDB(postgresOptions *database.PostgresOptions, lifeCycle fx.Lifecycle, logger *zap.Logger) (*gorm.DB, error) {
	db, err := database.ConnectPostgres(logger, postgresOptions)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	lifeCycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("closing postgress connection")
			return sqlDB.Close()
		},
	})
	return db, nil

}

func providePublisher(options *messaging.RabbitMQOptions, lifecycle fx.Lifecycle, logger *zap.Logger) (port.EventPublisher, error) {
	conn, err := amqp.DialConfig(options.URL, amqp.Config{
		Dial: amqp.DefaultDial(options.ConnectionTimeout),
	})
	if err != nil {
		return nil, err
	}
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("closing rabbitmq connection")
			channelErr := channel.Close()
			connErr := conn.Close()

			if channelErr != nil {
				return channelErr
			}

			return connErr

		},
	})

	return messaging.NewRabbitMQPublisher(channel, options.Exchange, options.PublishTimeout)
}

func provideOutboxWorker(outboxRepo port.OutboxRepository, publisher port.EventPublisher, options *messaging.RabbitMQOptions, logger *zap.Logger) *worker.OutboxWorker {
	return worker.NewOutboxWorker(
		outboxRepo,
		publisher,
		time.Duration(options.OutboxIntervalSeconds)*time.Second,
		options.OutboxProcessingTimeout,
		20, logger)
}

func StartOutboxWorker(lifeCycle fx.Lifecycle, worker *worker.OutboxWorker, logger *zap.Logger) {
	workerCtx, stopWorker := context.WithCancel(context.Background())
	done := make(chan struct{})
	lifeCycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			logger.Info("starting outbox worker")
			go func() {
				defer close(done)
				worker.Start(workerCtx)
			}()
			return nil
		},
		OnStop: func(shutdownCtx context.Context) error {
			logger.Info("stopping outbox worker")
			stopWorker()
			return waitForDone(shutdownCtx, done, "outbox worker")
		},
	})
}

func StartGRPCServer(
	options *appconfig.AppOptions,
	lifeCycle fx.Lifecycle,
	productService port.ProductService,
	shutdowner fx.Shutdowner,
	logger *zap.Logger,
) {
	server := grpc.NewServer()
	catalogv1.RegisterCatalogWriteServiceServer(server, grpcadapter.NewCatalogServer(productService))

	lifeCycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			list, err := net.Listen("tcp", ":"+options.GRPCPort)
			if err != nil {
				return err
			}
			logger.Info("Catalog write service grpc is running", zap.String("grpc_port", options.GRPCPort))
			go func() {
				if serveErr := server.Serve(list); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
					requestShutdown(shutdowner, logger, "grpc server", serveErr)
				}
			}()
			return nil
		}, OnStop: func(context.Context) error {
			logger.Info("stopping grpc server")
			server.GracefulStop()
			return nil
		},
	})
}

func StartHealthServer(
	lifecycle fx.Lifecycle,
	options *appconfig.AppOptions,
	healthHandler *health.Handler,
	shutdowner fx.Shutdowner,
	logger *zap.Logger,
) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health/liveness", healthHandler.Liveness)
	mux.HandleFunc("/health/readiness", healthHandler.Readiness)

	server := &http.Server{
		Addr:              ":" + options.HealthPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			listener, err := net.Listen("tcp", server.Addr)
			if err != nil {
				return err
			}

			logger.Info("catalog write health server is running", zap.String("health_port", options.HealthPort))
			go func() {
				if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
					requestShutdown(shutdowner, logger, "health server", err)
				}
			}()
			return nil
		},
		OnStop: func(shutdownCtx context.Context) error {
			logger.Info("stopping health server")
			return server.Shutdown(shutdownCtx)
		},
	})
}

func ManageReadiness(lifecycle fx.Lifecycle, healthHandler *health.Handler, logger *zap.Logger) {
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			healthHandler.SetReady(true)
			logger.Info("catalog write service is ready")
			return nil
		},
		OnStop: func(context.Context) error {
			healthHandler.SetReady(false)
			logger.Info("catalog write service is not ready")
			return nil
		},
	})
}

func requestShutdown(shutdowner fx.Shutdowner, logger *zap.Logger, component string, runtimeErr error) {
	logger.Error(component+" stopped with error", zap.Error(runtimeErr))
	if err := shutdowner.Shutdown(fx.ExitCode(1)); err != nil {
		logger.Error("failed to request application shutdown", zap.String("component", component), zap.Error(err))
	}
}

func waitForDone(ctx context.Context, done <-chan struct{}, component string) error {
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("stop %s: %w", component, ctx.Err())
	}
}
