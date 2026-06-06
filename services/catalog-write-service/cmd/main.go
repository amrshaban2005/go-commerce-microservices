package main

import (
	"log"
	"os"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/adapter/http/handler"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/adapter/http/router"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/adapter/repository"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/database"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-write-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	// load env
	err := godotenv.Load()
	if err != nil {
		panic("No env. file found.")
	}

	// connect postgress database
	db, err := database.ConnectPostgres()
	if err != nil {
		panic(err.Error())
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(err.Error())
	}
	defer sqlDB.Close()
	_ = db

	// initialization
	productRepo := repository.NewProductRepositryPG(db)
	productService := service.NewProductService(productRepo)
	productHandler := handler.NewProductHandler(productService)

	// register routes, run server
	r := gin.Default()
	api := r.Group("/api/v1")

	router.RegisterProductRoutes(api, productHandler)

	addr := ":" + os.Getenv("APP_PORT")
	err = r.Run(addr)
	if err != nil {
		panic(err.Error())
	}

	log.Printf("Catalog write service is running on: %s", addr)
}
