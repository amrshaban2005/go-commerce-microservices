package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/adapter/messaging"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/adapter/repository"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/database"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/service"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/worker"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
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
		"RABBITMQ_EXCHANGE_CONSUMER",
		"RESERVE_STOCK_QUEUE",
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
	inventoryRepo := repository.NewInventoryRepositoryPG(db)
	inventoryService := service.NewInventoryService(inventoryRepo)
	outboxRepo := repository.NewOutboxRepositoryPG(db)
	// run rabbitmq
	rabbitConn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Fatal("failed to connect rabbitmq: ", err)
	}
	defer rabbitConn.Close()

	consumerChannel, err := rabbitConn.Channel()
	if err != nil {
		log.Fatal("failed to open consumer channel: ", err)
	}
	defer consumerChannel.Close()

	publisherChannel, err := rabbitConn.Channel()
	if err != nil {
		log.Fatal("failed to open publisher channel: ", err)
	}
	defer publisherChannel.Close()

	consumer := messaging.NewReserveStockRequestedConsumer(consumerChannel, os.Getenv("RABBITMQ_EXCHANGE_CONSUMER"), os.Getenv("RESERVE_STOCK_QUEUE"), inventoryService)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publisher, err := messaging.NewRabbitMQPublisher(publisherChannel, os.Getenv("RABBITMQ_EXCHANGE_PUBLISHER"))
	if err != nil {
		log.Fatal("failed to create rabbitmq publisher: ", err)
	}

	outboxWorker := worker.NewOutboxWorker(outboxRepo, publisher, 5*time.Second, 20)

	go outboxWorker.Start(ctx)

	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.Start(ctx)
	}()

	log.Println("inventory-service started")
	select {
	case err := <-errCh:
		if err != nil {
			log.Println("consumer stopped with error:", err)
		}
		log.Println("consumer stopped")
	case <-ctx.Done():
		log.Println("inventory-service stopping")
	}

}
