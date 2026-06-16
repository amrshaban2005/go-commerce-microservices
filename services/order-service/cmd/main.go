package main

import (
	"context"
	"log"
	"net"
	"time"

	orderv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/order/v1"
	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
	applogger "github.com/amrshaban2005/go-commerce-microservices/pkg/logger"
	appconfig "github.com/amrshaban2005/go-commerce-microservices/services/order-service/config"
	grpcadapter "github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/adapter/grpc"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/adapter/messaging"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/adapter/repository"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/database"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/service"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/worker"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	if err := configloader.LoadDotEnv(); err != nil {
		log.Println("No local .env file found; using system environment variables")
	}

	logOptions, err := applogger.LoadOptions()
	if err != nil {
		log.Fatalf("failed to load log options: %v", err)
	}
	if err := logOptions.Validate(); err != nil {
		log.Fatalf("invalid log options: %v", err)
	}

	logger, err := applogger.New(*logOptions, "order-service")
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	appOptions, err := appconfig.LoadAppOptions()
	if err != nil {
		logger.Fatal("failed to load app options", zap.Error(err))
	}
	if err := appOptions.Validate(); err != nil {
		logger.Fatal("invalid app options", zap.Error(err))
	}

	postgresOptions, err := database.LoadPostgresOptions()
	if err != nil {
		logger.Fatal("failed to load postgres options", zap.Error(err))
	}
	if err := postgresOptions.Validate(); err != nil {
		logger.Fatal("invalid postgres options", zap.Error(err))
	}

	rabbitMQOptions, err := messaging.LoadRabbitMQOptions()
	if err != nil {
		logger.Fatal("failed to load rabbit mq options", zap.Error(err))
	}
	if err := rabbitMQOptions.Validate(); err != nil {
		logger.Fatal("invalid rabbit mq options", zap.Error(err))
	}

	//connect postgres
	db, err := database.ConnectPostgres(logger.With(zap.String("connection", "postgres")), postgresOptions)
	if err != nil {
		logger.Fatal("failed to connect to postgres", zap.Error(err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer sqlDB.Close()

	// initialization
	orderRepo := repository.NewOrderRepositoryPG(db)
	outboxRepo := repository.NewOutboxRepositoryPG(db)
	inboxRepo := repository.NewInboxMessageRepository(db)
	orderService := service.NewOrderService(orderRepo, inboxRepo)

	// run rabbitmq
	rabbitConn, err := amqp.Dial(rabbitMQOptions.URL)
	if err != nil {
		logger.Fatal("failed to connect rabbitmq", zap.Error(err))
	}
	defer rabbitConn.Close()

	publishChannel, _ := rabbitConn.Channel()
	consumerChannel1, _ := rabbitConn.Channel()
	consumerChannel2, _ := rabbitConn.Channel()

	defer publishChannel.Close()
	defer consumerChannel1.Close()
	defer consumerChannel2.Close()

	// publisher
	publisher, err := messaging.NewRabbitMQPublisher(publishChannel, rabbitMQOptions.PublisherExchange)
	if err != nil {
		logger.Fatal("failed to create rabbitmq publisher", zap.Error(err))
	}

	outboxWorker := worker.NewOutboxWorker(
		outboxRepo,
		publisher,
		time.Duration(rabbitMQOptions.OutboxIntervalSeconds)*time.Second,
		20,
		logger.With(zap.String("component", "outbox_worker")),
	)
	// consumers
	stockReservedConsumer := messaging.NewStockReservedConsumer(
		consumerChannel1,
		rabbitMQOptions.ConsumerExchange,
		rabbitMQOptions.StockReservedQueue,
		orderService,
		logger.With(zap.String("component", "stock_reserved_consumer")),
	)
	stockNotReservedConsumer := messaging.NewStockNotReservedConsumer(
		consumerChannel2,
		rabbitMQOptions.ConsumerExchange,
		rabbitMQOptions.StockNotReservedQueue,
		orderService,
		logger.With(zap.String("component", "stock_not_reserved_consumer")),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go outboxWorker.Start(ctx)

	errChan := make(chan error, 1)
	go func() {
		errChan <- messaging.Start(ctx, messaging.Consumers{
			StockReserved:    stockReservedConsumer,
			StockNotReserved: stockNotReservedConsumer,
		})
	}()
	// run grpc server
	list, err := net.Listen("tcp", ":"+appOptions.GRPCPort)
	if err != nil {
		logger.Fatal("failed to listen to tcp", zap.Error(err))
	}

	server := grpc.NewServer()
	orderv1.RegisterOrderServiceServer(server, grpcadapter.NewOrderServer(orderService))

	logger.Info("order service grpc is running", zap.String("grpc_port", appOptions.GRPCPort))
	if err = server.Serve(list); err != nil {
		logger.Info("failed to connect to grpc", zap.Error(err))
	}

	select {
	case err := <-errChan:
		if err != nil {
			logger.Info("consumer stopped with error:", zap.Error(err))
		}
		logger.Info("consumer stopped")

	case <-ctx.Done():
		logger.Info("consumer stopping")
	}
}
