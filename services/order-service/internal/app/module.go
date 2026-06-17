package app

import (
	"context"
	"net"
	"time"

	orderv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/order/v1"
	applogger "github.com/amrshaban2005/go-commerce-microservices/pkg/logger"
	appconfig "github.com/amrshaban2005/go-commerce-microservices/services/order-service/config"
	grpcadapter "github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/adapter/grpc"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/adapter/messaging"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/adapter/repository"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/database"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/port"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/service"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/worker"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

type PublisherChannelOut struct {
	fx.Out

	Channel *amqp.Channel `name:"publisher_channel"`
}

type StockReservedChannelOut struct {
	fx.Out

	Channel *amqp.Channel `name:"stock_reserved_channel"`
}

type StockNotReservedChannelOut struct {
	fx.Out

	Channel *amqp.Channel `name:"stock_not_reserved_channel"`
}

type PublisherParams struct {
	fx.In

	Channel *amqp.Channel `name:"publisher_channel"`
	Options *messaging.RabbitMQOptions
}

type StockReservedConsumerParams struct {
	fx.In

	Channel      *amqp.Channel `name:"stock_reserved_channel"`
	Options      *messaging.RabbitMQOptions
	OrderService port.OrderService
	Logger       *zap.Logger
}

type StockNotReservedConsumerParams struct {
	fx.In

	Channel      *amqp.Channel `name:"stock_not_reserved_channel"`
	Options      *messaging.RabbitMQOptions
	OrderService port.OrderService
	Logger       *zap.Logger
}

func Module() fx.Option {
	return fx.Options(
		fx.Provide(
			provideAppOptions,
			providePostgresOptions,
			provideRabbitMQOptions,
			provideLogger,
			provideDB,
			repository.NewOrderRepositoryPG,
			repository.NewOutboxRepositoryPG,
			repository.NewInboxMessageRepository,
			service.NewOrderService,
			provideRabbitMQConnection,
			providePublisherChannel,
			provideConsumerChannel1,
			provideConsumerChannel2,
			providePublisher,
			provideOutboxWorker,
			provideStockReservedConsumer,
			provideStockNotReservedConsumer,
		), fx.Invoke(
			StartOutboxWorker,
			StartConsumers,
			StartGRPCServer,
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

func provideLogger(lifecycle fx.Lifecycle) (*zap.Logger, error) {
	options, err := applogger.LoadOptions()
	if err != nil {
		return nil, err
	}
	if err := options.Validate(); err != nil {
		return nil, err
	}

	logger, err := applogger.New(*options, "order-service")

	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return logger.Sync()
		},
	})

	return logger, err
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
	return PublisherChannelOut{Channel: ch}, err
}

func provideConsumerChannel1(conn *amqp.Connection, lifecycle fx.Lifecycle) (StockReservedChannelOut, error) {
	ch, err := conn.Channel()
	if err != nil {
		return StockReservedChannelOut{}, err
	}
	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return ch.Close()
		},
	})
	return StockReservedChannelOut{Channel: ch}, err
}

func provideConsumerChannel2(conn *amqp.Connection, lifecycle fx.Lifecycle) (StockNotReservedChannelOut, error) {
	ch, err := conn.Channel()
	if err != nil {
		return StockNotReservedChannelOut{}, err
	}
	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return ch.Close()
		},
	})
	return StockNotReservedChannelOut{Channel: ch}, err
}

func providePublisher(params PublisherParams,
) (port.EventPublisher, error) {
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

func provideStockReservedConsumer(
	params StockReservedConsumerParams,
) *messaging.StockReservedConsumer {
	return messaging.NewStockReservedConsumer(
		params.Channel,
		params.Options.ConsumerExchange,
		params.Options.StockReservedQueue,
		params.OrderService,
		params.Logger.With(zap.String("component", "stock_reserved_consumer")),
	)
}

func provideStockNotReservedConsumer(
	params StockNotReservedConsumerParams,
) *messaging.StockNotReservedConsumer {
	return messaging.NewStockNotReservedConsumer(
		params.Channel,
		params.Options.ConsumerExchange,
		params.Options.StockNotReservedQueue,
		params.OrderService,
		params.Logger.With(zap.String("component", "stock_not_reserved_consumer")),
	)
}

func StartConsumers(
	lifecycle fx.Lifecycle,
	stockReservedConsumer *messaging.StockReservedConsumer,
	stockNotReservedConsumer *messaging.StockNotReservedConsumer,
	logger *zap.Logger,
) {
	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error, 1)

	lifecycle.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			logger.Info("starting order consumers")

			go func() {
				errChan <- messaging.Start(startCtx, messaging.Consumers{
					StockReserved:    stockReservedConsumer,
					StockNotReserved: stockNotReservedConsumer,
				})
			}()

			go func() {
				err := <-errChan
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

func StartOutboxWorker(
	lifecycle fx.Lifecycle,
	outboxWorker *worker.OutboxWorker,
	logger *zap.Logger,
) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("starting outbox worker")

			go outboxWorker.Start(ctx)

			return nil
		},
	})
}

func StartGRPCServer(
	lifecycle fx.Lifecycle,
	appOptions *appconfig.AppOptions,
	orderService port.OrderService,
	logger *zap.Logger,
) {
	server := grpc.NewServer()
	orderv1.RegisterOrderServiceServer(server, grpcadapter.NewOrderServer(orderService))

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", ":"+appOptions.GRPCPort)
			if err != nil {
				return err
			}

			logger.Info("order service grpc is running", zap.String("grpc_port", appOptions.GRPCPort))

			go func() {
				if err := server.Serve(listener); err != nil {
					logger.Error("grpc server stopped with error", zap.Error(err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("stopping grpc server")
			server.GracefulStop()
			return nil
		},
	})
}
