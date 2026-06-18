package app

import (
	"context"
	"net"
	"time"

	catalogv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/catalog/v1"
	applogger "github.com/amrshaban2005/go-commerce-microservices/pkg/logger"
	appconfig "github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/config"
	grpcadapter "github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/adapter/grpc"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/adapter/messaging"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/adapter/repository"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/database"
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
		),
		fx.Invoke(
			StartOutboxWorker,
			StartGRPCServer,
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
			return logger.Sync()
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
	conn, err := amqp.Dial(options.URL)
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

	return messaging.NewRabbitMQPublisher(channel, options.Exchange)
}

func provideOutboxWorker(outboxRepo port.OutboxRepository, publisher port.EventPublisher, options *messaging.RabbitMQOptions, logger *zap.Logger) *worker.OutboxWorker {
	return worker.NewOutboxWorker(
		outboxRepo,
		publisher,
		time.Duration(options.OutboxIntervalSeconds)*time.Second,
		20, logger)
}

func StartOutboxWorker(lifeCycle fx.Lifecycle, worker *worker.OutboxWorker, logger *zap.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	lifeCycle.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			logger.Info("starting outbox worker")
			go worker.Start(ctx)
			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			logger.Info("stopping outbox worker")
			cancel()
			return nil
		},
	})
}

func StartGRPCServer(options *appconfig.AppOptions, lifeCycle fx.Lifecycle, productService port.ProductService, logger *zap.Logger) {
	server := grpc.NewServer()
	catalogv1.RegisterCatalogWriteServiceServer(server, grpcadapter.NewCatalogServer(productService))

	lifeCycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			list, err := net.Listen("tcp", ":"+options.GRPCPort)
			if err != nil {
				return err
			}
			logger.Info("Catalog write service grpc is running", zap.String("grpc_port", options.GRPCPort))
			go func() {
				if err = server.Serve(list); err != nil {
					logger.Error("grpc server stopped with error", zap.Error(err))
				}
			}()
			return nil
		}, OnStop: func(ctx context.Context) error {
			logger.Info("stopping grpc server")
			server.GracefulStop()
			return nil
		},
	})
}
