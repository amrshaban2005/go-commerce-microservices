package main

import (
	"fmt"
	"log"

	appconfig "github.com/amrshaban2005/go-commerce-microservices/api-gateway/config"
	grpcclient "github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/adapter/grpc-client"
	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/adapter/http/handler"
	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/adapter/http/router"
	"github.com/gin-gonic/gin"
)

func main() {
	appOptions, err := appconfig.LoadAppOptions()
	if err != nil {
		log.Fatalf("failed to load app options: %v", err)
	}
	if err := appOptions.Validate(); err != nil {
		log.Fatalf("invalid app options: %v", err)
	}

	// register routes, run server
	catalogReadClient, closeReadCatalogClient, err := grpcclient.NewReadCatalogClient(appOptions.CatalogReadGrpcAddr)
	if err != nil {
		log.Fatalf("failed to connect to catalog read grpc server: %v", err.Error())
	}
	defer closeReadCatalogClient()

	catalogWriteClient, closeWriteCatalogClient, err := grpcclient.NewWriteCatalogClient(appOptions.CatalogWriteGrpcAddr)
	if err != nil {
		log.Fatalf("failed to connect to catalog write grpc server: %v", err.Error())
	}
	defer closeWriteCatalogClient()

	orderClient, closeOrderClient, err := grpcclient.NewOrderClient(appOptions.OrderGrpcUrl)
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

	addr := ":" + appOptions.AppPort
	log.Printf("api gateway is running on: %s", addr)

	err = r.Run(addr)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to server: %v", err.Error()))
	}
}
