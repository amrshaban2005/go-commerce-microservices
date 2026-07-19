package main

import (
	"log"

	"net/http"
	_ "net/http/pprof"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/app"
	"go.uber.org/fx"
)

func startPprof() {
	go func() {
		log.Println("pprof running on http://localhost:6060/debug/pprof/")

		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Println(err)
		}
	}()
}

func main() {
	startPprof()

	if err := configloader.LoadDotEnv(); err != nil {
		log.Println("No local .env file found; using system environment variables")
	}

	fxApp := fx.New(app.Module(), fx.NopLogger)
	if err := fxApp.Err(); err != nil {
		log.Fatalf("failed to build app: %v", err)
	}
	fxApp.Run()
}
