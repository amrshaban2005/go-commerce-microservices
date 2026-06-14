package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	orderv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/order/v1"
	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
	appconfig "github.com/amrshaban2005/go-commerce-microservices/services/order-service/config"
	grpcadapter "github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/adapter/grpc"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/adapter/messaging"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/adapter/repository"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/database"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/service"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/worker"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

func main() {
	if err := configloader.LoadDotEnv(); err != nil {
		log.Println("No local .env file found; using system environment variables")
	}

	appOptions, err := appconfig.LoadAppOptions()
	if err != nil {
		log.Fatalf("failed to load app options: %v", err)
	}
	if err := appOptions.Validate(); err != nil {
		log.Fatalf("invalid app options: %v", err)
	}

	postgresOptions, err := database.LoadPostgresOptions()
	if err != nil {
		log.Fatalf("failed to load postgres options: %v", err)
	}
	if err := postgresOptions.Validate(); err != nil {
		log.Fatalf("invalid postgres options: %v", err)
	}

	rabbitMQOptions, err := messaging.LoadRabbitMQOptions()
	if err != nil {
		log.Fatalf("failed to load rabbit mq options: %v", err)
	}
	if err := rabbitMQOptions.Validate(); err != nil {
		log.Fatalf("invalid rabbit mq options: %v", err)
	}

	//connect postgres
	db, err := database.ConnectPostgres(postgresOptions)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err.Error())
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err.Error())
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
		log.Fatal("failed to connect rabbitmq: ", err)
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
		log.Fatal("failed to create rabbitmq publisher: ", err)
	}

	outboxWorker := worker.NewOutboxWorker(outboxRepo, publisher, time.Duration(rabbitMQOptions.OutboxIntervalSeconds)*time.Second, 20)
	// consumers
	stockReservedConsumer := messaging.NewStockReservedConsumer(consumerChannel1, rabbitMQOptions.ConsumerExchange, rabbitMQOptions.StockReservedQueue, orderService)
	stockNotReservedConsumer := messaging.NewStockNotReservedConsumer(consumerChannel2, rabbitMQOptions.ConsumerExchange, rabbitMQOptions.StockNotReservedQueue, orderService)

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
		log.Fatalf("failed to listen to tcp: %v", err.Error())
	}

	server := grpc.NewServer()
	orderv1.RegisterOrderServiceServer(server, grpcadapter.NewOrderServer(orderService))

	log.Printf("Order service grpc is running on: %s", appOptions.GRPCPort)
	if err = server.Serve(list); err != nil {
		panic(fmt.Sprintf("failed to connect to grpc: %v", err.Error()))
	}

	select {
	case err := <-errChan:
		if err != nil {
			log.Println("consumer stopped with error:", err)
		}
		log.Println("consumer stopped")

	case <-ctx.Done():
		log.Println("consumer stopping")
	}
}
