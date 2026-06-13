package main

import (
	"context"
	"log"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/adapter/messaging"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/adapter/repository"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/database"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/service"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/worker"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	if err := configloader.LoadDotEnv(); err != nil {
		log.Println("No local .env file found; using system environment variables")
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
	inventoryRepo := repository.NewInventoryRepositoryPG(db)
	inventoryService := service.NewInventoryService(inventoryRepo)
	outboxRepo := repository.NewOutboxRepositoryPG(db)
	// run rabbitmq
	rabbitConn, err := amqp.Dial(rabbitMQOptions.URL)
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

	consumer := messaging.NewReserveStockRequestedConsumer(consumerChannel, rabbitMQOptions.ConsumerExchange, rabbitMQOptions.ReserveStockQueue, inventoryService)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publisher, err := messaging.NewRabbitMQPublisher(publisherChannel, rabbitMQOptions.PublisherExchange)
	if err != nil {
		log.Fatal("failed to create rabbitmq publisher: ", err)
	}

	outboxWorker := worker.NewOutboxWorker(outboxRepo, publisher, time.Duration(rabbitMQOptions.OutboxIntervalSeconds)*time.Second, 20)

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
