package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	catalogv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/catalog/v1"
	applogger "github.com/amrshaban2005/go-commerce-microservices/pkg/logger"
	appconfig "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/config"
	grpcadapter "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/adapter/grpc"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/adapter/messaging"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/adapter/repository"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/database"
	gettingproducts "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/features/products/getting_products"
	handlingproductcreated "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/features/products/handling_product_created"
	indexingproduct "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/features/products/indexing_product"
	searchingproducts "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/features/products/searching_products"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/health"
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
			provideElasticsearchOptions,
			provideRabbitMQOptions,
			provideLogger,
			provideMongoClient,
			provideMongoDatabase,
			database.NewRedisClient,
			database.NewElasticsearchClient,
			repository.NewProductRepositoryMongo,
			repository.NewInboxMessageMongoRepository,
			repository.NewProductCacheRepositoryRedis,
			repository.NewProductSearchRepositoryElasticsearch,
			gettingproducts.NewHandler,
			handlingproductcreated.NewHandler,
			indexingproduct.NewHandler,
			searchingproducts.NewHandler,
			provideRabbitMQConnection,
			provideRabbitMQChannel,
			provideProductSearchChannel,
			provideProductCreatedConsumer,
			provideProductSearchIndexConsumer,
			provideHealthHandler,
		),
		fx.Invoke(
			StartConsumer,
			StartGRPCServer,
			RegisterMediatorHandlers,
			StartHealthServer,
			ManageReadiness,
		),
	)
}

func provideElasticsearchOptions() (*database.ElasticsearchOptions, error) {
	options, err := database.LoadElasticsearchOptions()
	if err != nil {
		return nil, err
	}
	return options, options.Validate()
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

type productSearchChannel struct {
	channel *amqp.Channel
}

func provideProductSearchChannel(
	conn *amqp.Connection,
	lifecycle fx.Lifecycle,
) (*productSearchChannel, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	lifecycle.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return channel.Close()
		},
	})

	return &productSearchChannel{channel: channel}, nil
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

func provideProductSearchIndexConsumer(
	channel *productSearchChannel,
	options *messaging.RabbitMQOptions,
	logger *zap.Logger,
) *messaging.ProductSearchIndexConsumer {
	return messaging.NewProductSearchIndexConsumer(
		channel.channel,
		options.Exchange,
		options.ProductSearchQueue,
		logger.With(zap.String("component", "product_search_index_consumer")),
	)
}

func StartConsumer(
	lifecycle fx.Lifecycle,
	consumer *messaging.ProductCreatedConsumer,
	searchIndexConsumer *messaging.ProductSearchIndexConsumer,
	shutdowner fx.Shutdowner,
	logger *zap.Logger,
) {
	workerCtx, stopWorkers := context.WithCancel(context.Background())
	done := make(chan struct{})

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			logger.Info("starting catalog projection consumers")
			var workers sync.WaitGroup
			workers.Add(2)

			go func() {
				defer workers.Done()
				err := consumer.Start(workerCtx)
				if err != nil {
					requestShutdown(shutdowner, logger, "product created consumer", err)
					return
				}

				logger.Info("consumer stopped")
			}()

			go func() {
				defer workers.Done()
				err := searchIndexConsumer.Start(workerCtx)
				if err != nil {
					requestShutdown(shutdowner, logger, "product search index consumer", err)
					return
				}

				logger.Info("product search index consumer stopped")
			}()

			go func() {
				workers.Wait()
				close(done)
			}()

			return nil
		},
		OnStop: func(shutdownCtx context.Context) error {
			logger.Info("consumer stopping")
			stopWorkers()
			return waitForDone(shutdownCtx, done, "catalog projection consumers")
		},
	})
}

func StartGRPCServer(
	lifecycle fx.Lifecycle,
	appOptions *appconfig.AppOptions,
	shutdowner fx.Shutdowner,
	logger *zap.Logger,
) {
	server := grpc.NewServer()
	catalogv1.RegisterCatalogReadServiceServer(server, grpcadapter.NewCatalogServer())

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			listener, err := net.Listen("tcp", ":"+appOptions.GRPCPort)
			if err != nil {
				return err
			}

			logger.Info("catalog read service grpc is running", zap.String("grpc_port", appOptions.GRPCPort))

			go func() {
				if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
					requestShutdown(shutdowner, logger, "grpc server", err)
				}
			}()

			return nil
		},
		OnStop: func(context.Context) error {
			logger.Info("stopping grpc server")
			server.GracefulStop()
			return nil
		},
	})
}

func StartHealthServer(
	lifecycle fx.Lifecycle,
	appOptions *appconfig.AppOptions,
	healthHandler *health.Handler,
	shutdowner fx.Shutdowner,
	logger *zap.Logger,
) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health/liveness", healthHandler.Liveness)
	mux.HandleFunc("/health/readiness", healthHandler.Readiness)

	server := &http.Server{
		Addr:              ":" + appOptions.HealthPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			listener, err := net.Listen("tcp", server.Addr)
			if err != nil {
				return err
			}

			logger.Info("catalog read health server is running", zap.String("health_port", appOptions.HealthPort))
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
			logger.Info("catalog read service is ready")
			return nil
		},
		OnStop: func(context.Context) error {
			healthHandler.SetReady(false)
			logger.Info("catalog read service is not ready")
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

func RegisterMediatorHandlers(
	getProductsHandler *gettingproducts.Handler,
	productCreatedHandler *handlingproductcreated.Handler,
	indexProductHandler *indexingproduct.Handler,
	searchProductsHandler *searchingproducts.Handler,
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

	if err := mediatr.RegisterRequestHandler[*indexingproduct.Command, *struct{}](
		indexProductHandler,
	); err != nil {
		return err
	}

	if err := mediatr.RegisterRequestHandler[*searchingproducts.Query, *searchingproducts.Result](
		searchProductsHandler,
	); err != nil {
		return err
	}

	return nil
}
