package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	applogger "github.com/amrshaban2005/go-commerce-microservices/pkg/logger"
	appconfig "github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/config"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/adapter/messaging"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/adapter/repository"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/database"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/health"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/port"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/service"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/worker"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PublisherChannelOut struct {
	fx.Out

	Channel *amqp.Channel `name:"publisher_channel"`
}

type ConsumerChannelOut struct {
	fx.Out

	Channel *amqp.Channel `name:"consumer_channel"`
}

type PublisherParams struct {
	fx.In

	Channel *amqp.Channel `name:"publisher_channel"`
	Options *messaging.RabbitMQOptions
}

type ConsumerParams struct {
	fx.In

	Channel          *amqp.Channel `name:"consumer_channel"`
	Options          *messaging.RabbitMQOptions
	InventoryService port.InventoryService
	Logger           *zap.Logger
}

func Module() fx.Option {
	return fx.Options(
		fx.Provide(
			provideAppOptions,
			providePostgresOptions,
			provideRabbitMQOptions,
			provideLogger,
			provideDB,
			repository.NewInventoryRepositoryPG,
			repository.NewOutboxRepositoryPG,
			service.NewInventoryService,
			provideRabbitMQConnection,
			providePublisherChannel,
			provideConsumerChannel,
			providePublisher,
			provideOutboxWorker,
			provideReserveStockRequestedConsumer,
			provideHealthHandler,
		),
		fx.Invoke(
			StartOutboxWorker,
			StartConsumer,
			StartHealthServer,
			ManageReadiness,
		),
	)
}

func provideAppOptions() (*appconfig.AppOptions, error) {
	options, err := appconfig.LoadAppOptions()
	if err != nil {
		return nil, err
	}
	return options, options.Validate()
}

func providePostgresOptions() (*database.PostgresOptions, error) {
	options, err := database.LoadPostgresOptions()
	if err != nil {
		return nil, err
	}
	return options, options.Validate()
}

func provideRabbitMQOptions() (*messaging.RabbitMQOptions, error) {
	options, err := messaging.LoadRabbitMQOptions()
	if err != nil {
		return nil, err
	}
	return options, options.Validate()
}

func provideHealthHandler() *health.Handler {
	return health.New()
}

func provideLogger(lifecycle fx.Lifecycle) (*zap.Logger, error) {
	options, err := applogger.LoadOptions()
	if err != nil {
		return nil, err
	}
	if err := options.Validate(); err != nil {
		return nil, err
	}

	logger, err := applogger.New(*options, "inventory-service")
	if err != nil {
		return nil, err
	}

	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return applogger.Sync(logger)
		},
	})

	return logger, nil
}

func provideDB(options *database.PostgresOptions, lifecycle fx.Lifecycle, logger *zap.Logger) (*gorm.DB, error) {
	db, err := database.ConnectPostgres(logger.With(zap.String("connection", "postgres")), options)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("closing postgres connection")
			return sqlDB.Close()
		},
	})

	return db, nil
}

func provideRabbitMQConnection(
	options *messaging.RabbitMQOptions,
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
) (*amqp.Connection, error) {
	conn, err := amqp.DialConfig(options.URL, amqp.Config{
		Dial: amqp.DefaultDial(options.ConnectionTimeout),
	})
	if err != nil {
		return nil, err
	}

	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("closing rabbitmq connection")
			return conn.Close()
		},
	})

	return conn, nil
}

func providePublisherChannel(conn *amqp.Connection, lifecycle fx.Lifecycle) (PublisherChannelOut, error) {
	ch, err := conn.Channel()
	if err != nil {
		return PublisherChannelOut{}, err
	}

	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return ch.Close()
		},
	})

	return PublisherChannelOut{Channel: ch}, nil
}

func provideConsumerChannel(conn *amqp.Connection, lifecycle fx.Lifecycle) (ConsumerChannelOut, error) {
	ch, err := conn.Channel()
	if err != nil {
		return ConsumerChannelOut{}, err
	}

	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return ch.Close()
		},
	})

	return ConsumerChannelOut{Channel: ch}, nil
}

func providePublisher(params PublisherParams) (port.EventPublisher, error) {
	return messaging.NewRabbitMQPublisher(
		params.Channel,
		params.Options.PublisherExchange,
		params.Options.PublishTimeout,
	)
}

func provideOutboxWorker(
	outboxRepo port.OutboxRepository,
	publisher port.EventPublisher,
	options *messaging.RabbitMQOptions,
	logger *zap.Logger,
) *worker.OutboxWorker {
	return worker.NewOutboxWorker(
		outboxRepo,
		publisher,
		time.Duration(options.OutboxIntervalSeconds)*time.Second,
		options.OutboxProcessingTimeout,
		20,
		logger.With(zap.String("component", "outbox_worker")),
	)
}

func provideReserveStockRequestedConsumer(params ConsumerParams) *messaging.ReserveStockRequestedConsumer {
	return messaging.NewReserveStockRequestedConsumer(
		params.Channel,
		params.Options.ConsumerExchange,
		params.Options.ReserveStockQueue,
		params.InventoryService,
		params.Logger.With(zap.String("component", "reserve_stock_requested_consumer")),
		params.Options.ConsumerProcessingTimeout,
	)
}

func StartOutboxWorker(
	lifecycle fx.Lifecycle,
	outboxWorker *worker.OutboxWorker,
	logger *zap.Logger,
) {
	workerCtx, stopWorker := context.WithCancel(context.Background())
	done := make(chan struct{})

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			logger.Info("starting outbox worker")
			go func() {
				defer close(done)
				outboxWorker.Start(workerCtx)
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

func StartConsumer(
	lifecycle fx.Lifecycle,
	consumer *messaging.ReserveStockRequestedConsumer,
	shutdowner fx.Shutdowner,
	logger *zap.Logger,
) {
	workerCtx, stopWorker := context.WithCancel(context.Background())
	done := make(chan struct{})

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			logger.Info("starting reserve stock requested consumer")

			go func() {
				defer close(done)
				err := consumer.Start(workerCtx)
				if err != nil {
					requestShutdown(shutdowner, logger, "reserve stock requested consumer", err)
					return
				}

				logger.Info("consumer stopped")
			}()

			return nil
		},
		OnStop: func(shutdownCtx context.Context) error {
			logger.Info("consumer stopping")
			stopWorker()
			return waitForDone(shutdownCtx, done, "reserve stock requested consumer")
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

			logger.Info("inventory health server is running", zap.String("health_port", options.HealthPort))
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
			logger.Info("inventory service is ready")
			return nil
		},
		OnStop: func(context.Context) error {
			healthHandler.SetReady(false)
			logger.Info("inventory service is not ready")
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
