package main

import (
	"context"
	"fmt"
	"log"
	"net"

	catalogv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/catalog/v1"
	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
	appconfig "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/config"
	grpcadapter "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/adapter/grpc"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/adapter/messaging"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/adapter/repository"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/database"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/service"
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

	mongoOptions, err := database.LoadMongoOptions()
	if err != nil {
		log.Fatalf("failed to load mongo options: %v", err)
	}
	if err := mongoOptions.Validate(); err != nil {
		log.Fatalf("invalid mongo options: %v", err)
	}

	rabbitMQOptions, err := messaging.LoadRabbitMQOptions()
	if err != nil {
		log.Fatalf("failed to load rabbit mq options: %v", err)
	}
	if err := rabbitMQOptions.Validate(); err != nil {
		log.Fatalf("invalid rabbit mq options: %v", err)
	}

	// connect mongo db
	client, err := database.ConnectMongo(mongoOptions)
	if err != nil {
		log.Fatalf("failed to connect mongo: %v", err.Error())
	}
	db := client.Database(mongoOptions.Database)

	// intialize
	productRepo := repository.NewProductRepositoryMongo(db)
	inboxRepo, err := repository.NewInboxMessageMongoRepository(db)
	if err != nil {
		log.Fatalf("failed to create db index: %v", err.Error())
	}
	prodcutService := service.NewProductService(productRepo, inboxRepo)

	// run rabbitmq
	conn, err := amqp.Dial(rabbitMQOptions.URL)
	if err != nil {
		log.Fatalf("failed to connect to rabbit mq: %v", err.Error())
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to connect to rabbit mq channel: %v", err.Error())
	}
	defer channel.Close()

	consumer := messaging.NewProductCreatedConsumer(channel, rabbitMQOptions.Exchange, rabbitMQOptions.ProductCreatedQueue, prodcutService)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumerErrChan := make(chan error, 1)
	go func() {
		consumerErrChan <- consumer.Start(ctx)
	}()

	// run grpc server
	list, err := net.Listen("tcp", ":"+appOptions.GRPCPort)
	if err != nil {
		log.Fatalf("failed to listen to tcp: %v", err.Error())
	}

	server := grpc.NewServer()
	catalogv1.RegisterCatalogReadServiceServer(server, grpcadapter.NewCatalogServer(prodcutService))

	log.Printf("Catalog read service grpc is running on: %s", appOptions.GRPCPort)
	if err = server.Serve(list); err != nil {
		panic(fmt.Sprintf("failed to connect to grpc: %v", err.Error()))
	}

	select {
	case err := <-consumerErrChan:
		if err != nil {
			log.Println("consumer stopped with error:", err)
		}
		log.Println("consumer stopped")
	case <-ctx.Done():
		log.Println("inventory-service stopping")
	}
}
