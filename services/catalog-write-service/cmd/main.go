package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	catalogv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/catalog/v1"
	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
	appconfig "github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/config"
	grpcadapter "github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/adapter/grpc"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/adapter/messaging"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/adapter/repository"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/database"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/service"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/worker"
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

	// connect postgress database
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
	productRepo := repository.NewProductRepositryPG(db)
	outboxRepo := repository.NewOutboxRepositoryPG(db)
	productService := service.NewProductService(productRepo)

	// start rabbitmq/messaging
	rabbitConn, err := amqp.Dial(rabbitMQOptions.URL)
	if err != nil {
		log.Fatal("failed to connect rabbitmq: ", err)
	}
	defer rabbitConn.Close()

	rabbitChannel, err := rabbitConn.Channel()
	if err != nil {
		log.Fatal("failed to open rabbitmq channel: ", err)
	}
	defer rabbitChannel.Close()

	publisher, err := messaging.NewRabbitMQPublisher(rabbitChannel, rabbitMQOptions.Exchange)
	if err != nil {
		log.Fatal("failed to create rabbitmq publisher: ", err)
	}

	outboxWorker := worker.NewOutboxWorker(
		outboxRepo,
		publisher,
		time.Duration(rabbitMQOptions.OutboxIntervalSeconds)*time.Second,
		20,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go outboxWorker.Start(ctx)

	// run grpc server
	list, err := net.Listen("tcp", ":"+appOptions.GRPCPort)
	if err != nil {
		log.Fatalf("failed to listen to tcp: %v", err.Error())
	}

	server := grpc.NewServer()
	catalogv1.RegisterCatalogWriteServiceServer(server, grpcadapter.NewCatalogServer(productService))

	log.Printf("Catalog write service grpc is running on: %s", appOptions.GRPCPort)
	if err = server.Serve(list); err != nil {
		panic(fmt.Sprintf("failed to connect to grpc: %v", err.Error()))
	}

}
