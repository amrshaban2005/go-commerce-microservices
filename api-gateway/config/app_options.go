package config

import (
	"fmt"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type AppOptions struct {
	AppPort              string        `mapstructure:"appPort"`
	CatalogReadGrpcAddr  string        `mapstructure:"catalogReadGrpcAddr"`
	CatalogWriteGrpcAddr string        `mapstructure:"catalogWriteGrpcAddr"`
	OrderGrpcUrl         string        `mapstructure:"orderGrpcUrl"`
	RequestTimeout       time.Duration `mapstructure:"requestTimeout"`
	GRPCReadTimeout      time.Duration `mapstructure:"grpcReadTimeout"`
	GRPCWriteTimeout     time.Duration `mapstructure:"grpcWriteTimeout"`
	ReadHeaderTimeout    time.Duration `mapstructure:"readHeaderTimeout"`
	ReadTimeout          time.Duration `mapstructure:"readTimeout"`
	WriteTimeout         time.Duration `mapstructure:"writeTimeout"`
	IdleTimeout          time.Duration `mapstructure:"idleTimeout"`
}

func LoadAppOptions() (*AppOptions, error) {
	return configloader.BindKey[AppOptions](
		"appOptions",
		map[string]string{
			"appPort":              "APP_PORT",
			"catalogReadGrpcAddr":  "CATALOG_READ_GRPC_ADDR",
			"catalogWriteGrpcAddr": "CATALOG_WRITE_GRPC_ADDR",
			"orderGrpcUrl":         "ORDER_GRPC_ADDR",
			"requestTimeout":       "REQUEST_TIMEOUT",
			"grpcReadTimeout":      "GRPC_READ_TIMEOUT",
			"grpcWriteTimeout":     "GRPC_WRITE_TIMEOUT",
			"readHeaderTimeout":    "HTTP_READ_HEADER_TIMEOUT",
			"readTimeout":          "HTTP_READ_TIMEOUT",
			"writeTimeout":         "HTTP_WRITE_TIMEOUT",
			"idleTimeout":          "HTTP_IDLE_TIMEOUT",
		},
	)
}

func (options *AppOptions) Validate() error {
	required := []struct {
		name  string
		value string
	}{
		{name: "appPort", value: options.AppPort},
		{name: "catalogReadGrpcAddr", value: options.CatalogReadGrpcAddr},
		{name: "catalogWriteGrpcAddr", value: options.CatalogWriteGrpcAddr},
		{name: "orderGrpcUrl", value: options.OrderGrpcUrl},
	}

	for _, field := range required {
		if field.value == "" {
			return fmt.Errorf("AppOptions.%s is required", field.name)
		}
	}

	durations := []struct {
		name  string
		value time.Duration
	}{
		{name: "requestTimeout", value: options.RequestTimeout},
		{name: "grpcReadTimeout", value: options.GRPCReadTimeout},
		{name: "grpcWriteTimeout", value: options.GRPCWriteTimeout},
		{name: "readHeaderTimeout", value: options.ReadHeaderTimeout},
		{name: "readTimeout", value: options.ReadTimeout},
		{name: "writeTimeout", value: options.WriteTimeout},
		{name: "idleTimeout", value: options.IdleTimeout},
	}
	for _, field := range durations {
		if field.value <= 0 {
			return fmt.Errorf("AppOptions.%s must be greater than zero", field.name)
		}
	}
	if options.WriteTimeout <= options.RequestTimeout {
		return fmt.Errorf("AppOptions.writeTimeout must be greater than requestTimeout")
	}
	if options.GRPCReadTimeout >= options.RequestTimeout {
		return fmt.Errorf("AppOptions.grpcReadTimeout must be shorter than requestTimeout")
	}
	if options.GRPCWriteTimeout >= options.RequestTimeout {
		return fmt.Errorf("AppOptions.grpcWriteTimeout must be shorter than requestTimeout")
	}

	return nil
}
