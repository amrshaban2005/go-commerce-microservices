package main

import (
	"log"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/app"
	"go.uber.org/fx"
)

func main() {
	if err := configloader.LoadDotEnv(); err != nil {
		log.Println("No local .env file found; using system environment variables")
	}

	fxApp := fx.New(app.Module(), fx.NopLogger)
	if err := fxApp.Err(); err != nil {
		log.Fatalf("failed to build app: %v", err)
	}
	fxApp.Run()
}
