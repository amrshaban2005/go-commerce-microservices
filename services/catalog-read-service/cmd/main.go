package main

import (
	"fmt"
	"log"
	"net"
	"os"

	catalogv1 "github.com/amrshaban2005/go-commerce-microservices/api/gen/go/catalog/v1"
	grpcadapter "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/adapter/grpc"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

func sanityCheck() {
	envProps := []string{
		"GRPC_PORT",
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

	list, err := net.Listen("tcp", ":"+os.Getenv("GRPC_PORT"))
	if err != nil {
		log.Fatalf("failed to listen to tcp: %v", err.Error())
	}

	server := grpc.NewServer()
	catalogv1.RegisterCatalogReadServiceServer(server, grpcadapter.NewCatalogServer())

	log.Printf("Catalog read service grpc is running on: %s", os.Getenv("GRPC_PORT"))
	if err = server.Serve(list); err != nil {
		panic(fmt.Sprintf("failed to connect to grpc: %v", err.Error()))
	}

}
