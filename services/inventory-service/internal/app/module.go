package app

import (
	"context"
	"time"

	applogger "github.com/amrshaban2005/go-commerce-microservices/pkg/logger"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/adapter/messaging"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/adapter/repository"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/database"
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
		),
		fx.Invoke(
			StartOutboxWorker,
			StartConsumer,
		),
	)
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
			return logger.Sync()
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
	conn, err := amqp.Dial(options.URL)
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
	)
}

func StartOutboxWorker(
	lifecycle fx.Lifecycle,
	outboxWorker *worker.OutboxWorker,
	logger *zap.Logger,
) {
	ctx, cancel := context.WithCancel(context.Background())

	lifecycle.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			logger.Info("starting outbox worker")
			go outboxWorker.Start(ctx)
			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			logger.Info("stopping outbox worker")
			cancel()
			return nil
		},
	})
}

func StartConsumer(
	lifecycle fx.Lifecycle,
	consumer *messaging.ReserveStockRequestedConsumer,
	logger *zap.Logger,
) {
	ctx, cancel := context.WithCancel(context.Background())

	lifecycle.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			logger.Info("starting reserve stock requested consumer")

			go func() {
				err := consumer.Start(ctx)
				if err != nil {
					logger.Error("consumer stopped with error", zap.Error(err))
					return
				}

				logger.Info("consumer stopped")
			}()

			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			logger.Info("consumer stopping")
			cancel()
			return nil
		},
	})
}
