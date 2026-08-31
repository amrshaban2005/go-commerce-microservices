package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	appconfig "github.com/amrshaban2005/go-commerce-microservices/api-gateway/config"
	_ "github.com/amrshaban2005/go-commerce-microservices/api-gateway/docs"
	"github.com/amrshaban2005/go-commerce-microservices/api-gateway/internal/app"
)

// @title Go Commerce API
// @version 1.0
// @description Public HTTP API for Go Commerce Microservices.
// @BasePath /api/v1
// @schemes http
func main() {
	appOptions, err := appconfig.LoadAppOptions()
	if err != nil {
		log.Fatalf("failed to load app options: %v", err)
	}

	application, err := app.New(appOptions)
	if err != nil {
		log.Fatalf("failed to create application: %v", err)
	}

	errCh := make(chan error, 1)
	application.Run(errCh)
	log.Printf("api gateway is running on: %s", application.Addr())

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-signalCtx.Done():
		log.Printf("shutdown signal received")
	case runErr := <-errCh:
		log.Printf("api gateway runtime error: %v", runErr)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := application.Shutdown(shutdownCtx); err != nil {
		log.Printf("api gateway shutdown error: %v", err)
	}
	log.Printf("api gateway shutdown complete")
}
