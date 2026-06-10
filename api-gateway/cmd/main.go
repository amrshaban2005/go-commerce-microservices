package main

import (
	"fmt"
	"log"
	"os"

	grpcclient "github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/adapter/grpc-client"
	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/adapter/http/handler"
	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/adapter/http/router"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func sanityCheck() {
	envProps := []string{
		"APP_PORT",
		"CATALOG_READ_GRPC_ADDR",
		"CATALOG_WRITE_GRPC_ADDR",
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

	// register routes, run server
	catalogReadClient, closeReadCatalogClient, err := grpcclient.NewReadCatalogClient(os.Getenv("CATALOG_READ_GRPC_ADDR"))
	if err != nil {
		log.Fatalf("failed to connect to catalog read grpc server: %v", err.Error())
	}
	defer closeReadCatalogClient()

	catalogWriteClient, closeWriteCatalogClient, err := grpcclient.NewWriteCatalogClient(os.Getenv("CATALOG_WRITE_GRPC_ADDR"))
	if err != nil {
		log.Fatalf("failed to connect to catalog write grpc server: %v", err.Error())
	}
	defer closeWriteCatalogClient()

	orderClient, closeOrderClient, err := grpcclient.NewOrderClient(os.Getenv("CATALOG_ORDER_GRPC_ADDR"))
	if err != nil {
		log.Fatalf("failed to connect to order grpc server: %v", err.Error())
	}
	defer closeOrderClient()

	prodcutHandler := handler.NewProductHandler(catalogReadClient, catalogWriteClient)
	orderHandler := handler.NewOrderHandler(orderClient)

	r := gin.Default()
	api := r.Group("/api/v1")

	router.RegisterProductRoutes(api, prodcutHandler)
	router.RegisterOrderRoutes(api, orderHandler)

	addr := ":" + os.Getenv("APP_PORT")
	log.Printf("api gateway is running on: %s", addr)

	err = r.Run(addr)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to server: %v", err.Error()))
	}
}
