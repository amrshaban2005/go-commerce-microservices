package app

import (
	"context"
	"net"

	catalogv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/catalog/v1"
	applogger "github.com/amrshaban2005/go-commerce-microservices/pkg/logger"
	appconfig "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/config"
	grpcadapter "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/adapter/grpc"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/adapter/messaging"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/adapter/repository"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/database"
	gettingproducts "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/features/products/getting_products"
	handlingproductcreated "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/features/products/handling_product_created"
	"github.com/mehdihadeli/go-mediatr"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func Module() fx.Option {
	return fx.Options(
		fx.Provide(
			provideAppOptions,
			provideMongoOptions,
			database.LoadRedisOptions,
			provideRabbitMQOptions,
			provideLogger,
			provideMongoClient,
			provideMongoDatabase,
			database.NewRedisClient,
			repository.NewProductRepositoryMongo,
			repository.NewInboxMessageMongoRepository,
			repository.NewProductCacheRepositoryRedis,
			gettingproducts.NewHandler,
			handlingproductcreated.NewHandler,
			provideRabbitMQConnection,
			provideRabbitMQChannel,
			provideProductCreatedConsumer,
		),
		fx.Invoke(
			StartConsumer,
			StartGRPCServer,
			RegisterMediatorHandlers,
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

func provideMongoOptions() (*database.MongoOptions, error) {
	options, err := database.LoadMongoOptions()
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

	logger, err := applogger.New(*options, "catalog-read-service")
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

func provideMongoClient(
	options *database.MongoOptions,
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
) (*mongo.Client, error) {
	client, err := database.ConnectMongo(logger.With(zap.String("connection", "mongo")), options)
	if err != nil {
		return nil, err
	}

	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("closing mongo connection")
			return client.Disconnect(ctx)
		},
	})

	return client, nil
}

func provideMongoDatabase(client *mongo.Client, options *database.MongoOptions) *mongo.Database {
	return client.Database(options.Database)
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

func provideRabbitMQChannel(conn *amqp.Connection, lifecycle fx.Lifecycle) (*amqp.Channel, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return channel.Close()
		},
	})

	return channel, nil
}

func provideProductCreatedConsumer(
	channel *amqp.Channel,
	options *messaging.RabbitMQOptions,
	logger *zap.Logger,
) *messaging.ProductCreatedConsumer {
	return messaging.NewProductCreatedConsumer(
		channel,
		options.Exchange,
		options.ProductCreatedQueue,
		logger.With(zap.String("component", "product_created_consumer")),
	)
}

func StartConsumer(
	lifecycle fx.Lifecycle,
	consumer *messaging.ProductCreatedConsumer,
	logger *zap.Logger,
) {
	ctx, cancel := context.WithCancel(context.Background())

	lifecycle.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			logger.Info("starting product created consumer")

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

func StartGRPCServer(
	lifecycle fx.Lifecycle,
	appOptions *appconfig.AppOptions,
	logger *zap.Logger,
) {
	server := grpc.NewServer()
	catalogv1.RegisterCatalogReadServiceServer(server, grpcadapter.NewCatalogServer())

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", ":"+appOptions.GRPCPort)
			if err != nil {
				return err
			}

			logger.Info("catalog read service grpc is running", zap.String("grpc_port", appOptions.GRPCPort))

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

func RegisterMediatorHandlers(
	getProductsHandler *gettingproducts.Handler,
	productCreatedHandler *handlingproductcreated.Handler,
) error {
	if err := mediatr.RegisterRequestHandler[*gettingproducts.Query, *gettingproducts.Result](
		getProductsHandler,
	); err != nil {
		return err
	}

	if err := mediatr.RegisterRequestHandler[*handlingproductcreated.Command, *struct{}](
		productCreatedHandler,
	); err != nil {
		return err
	}

	return nil
}
