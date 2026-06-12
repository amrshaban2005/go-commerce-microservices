package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	orderv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/order/v1"
	grpcadapter "github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/adapter/grpc"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/adapter/messaging"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/adapter/repository"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/database"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/service"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/worker"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

func sanityCheck() {
	envProps := []string{
		"GRPC_PORT",
		"DB_HOST",
		"DB_PORT",
		"DB_USER",
		"DB_PASSWORD",
		"DB_NAME",
		"DB_SSLMODE",
		"RABBITMQ_URL",
		"RABBITMQ_EXCHANGE_PUBLISHER",
		"STOCK_RESERVED_QUEUE",
		"STOCK_NOT_RESERVED_QUEUE",
		"OUTBOX_INTERVAL_SECONDS",
	}

	for _, k := range envProps {
		if os.Getenv(k) == "" {
			log.Fatalf("Enviroment variable %s not provided ", k)
		}
	}
}

func main() {
	// load env
	err := godotenv.Load()
	if err != nil {
		log.Println("No env. file found.")
	}
	sanityCheck()

	//connect postgres
	db, err := database.ConnectPostgres()
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
	rabbitConn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
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
	publisher, err := messaging.NewRabbitMQPublisher(publishChannel, os.Getenv("RABBITMQ_EXCHANGE_PUBLISHER"))
	if err != nil {
		log.Fatal("failed to create rabbitmq publisher: ", err)
	}

	outboxWorker := worker.NewOutboxWorker(outboxRepo, publisher, 5*time.Second, 20)
	// consumers
	stockReservedConsumer := messaging.NewStockReservedConsumer(consumerChannel1, os.Getenv("RABBITMQ_EXCHANGE_CONSUMER"), os.Getenv("STOCK_RESERVED_QUEUE"), orderService)
	stockNotReservedConsumer := messaging.NewStockNotReservedConsumer(consumerChannel2, os.Getenv("RABBITMQ_EXCHANGE_CONSUMER"), os.Getenv("STOCK_NOT_RESERVED_QUEUE"), orderService)

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
	list, err := net.Listen("tcp", ":"+os.Getenv("GRPC_PORT"))
	if err != nil {
		log.Fatalf("failed to listen to tcp: %v", err.Error())
	}

	server := grpc.NewServer()
	orderv1.RegisterOrderServiceServer(server, grpcadapter.NewOrderServer(orderService))

	log.Printf("Order service grpc is running on: %s", os.Getenv("GRPC_PORT"))
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
