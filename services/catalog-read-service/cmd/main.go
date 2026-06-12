package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	catalogv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/catalog/v1"
	grpcadapter "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/adapter/grpc"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/adapter/messaging"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/adapter/repository"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/database"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/service"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

func sanityCheck() {
	envProps := []string{
		"GRPC_PORT",
		"MONGO_URI",
		"MONGO_DATABASE",
		"RABBITMQ_URL",
		"RABBITMQ_EXCHANGE",
		"PRODUCT_CREATED_QUEUE",
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

	// connect mongo db
	client, err := database.ConnectMongo(os.Getenv("MONGO_URI"))
	if err != nil {
		log.Fatalf("failed to connect mongo: %v", err.Error())
	}
	db := client.Database(os.Getenv("MONGO_DATABASE"))

	// intialize
	productRepo := repository.NewProductRepositoryMongo(db)
	inboxRepo, err := repository.NewInboxMessageMongoRepository(db)
	if err != nil {
		log.Fatalf("failed to create db index: %v", err.Error())
	}
	prodcutService := service.NewProductService(productRepo, inboxRepo)

	// run rabbitmq
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Fatalf("failed to connect to rabbit mq: %v", err.Error())
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to connect to rabbit mq channel: %v", err.Error())
	}
	defer channel.Close()

	consumer := messaging.NewProductCreatedConsumer(channel, os.Getenv("RABBITMQ_EXCHANGE"), os.Getenv("PRODUCT_CREATED_QUEUE"), prodcutService)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumerErrChan := make(chan error, 1)
	go func() {
		consumerErrChan <- consumer.Start(ctx)
	}()

	// run grpc server
	list, err := net.Listen("tcp", ":"+os.Getenv("GRPC_PORT"))
	if err != nil {
		log.Fatalf("failed to listen to tcp: %v", err.Error())
	}

	server := grpc.NewServer()
	catalogv1.RegisterCatalogReadServiceServer(server, grpcadapter.NewCatalogServer(prodcutService))

	log.Printf("Catalog read service grpc is running on: %s", os.Getenv("GRPC_PORT"))
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
