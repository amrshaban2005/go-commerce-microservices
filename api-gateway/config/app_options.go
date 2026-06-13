package config

import (
	"fmt"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type AppOptions struct {
	AppPort              string `mapstructure:"appPort"`
	CatalogReadGrpcAddr  string `mapstructure:"catalogReadGrpcAddr"`
	CatalogWriteGrpcAddr string `mapstructure:"catalogWriteGrpcAddr"`
	OrderGrpcUrl         string `mapstructure:"orderGrpcUrl"`
}

func LoadAppOptions() (*AppOptions, error) {
	return configloader.BindKey[AppOptions](
		"appOptions",
		map[string]string{
			"appPort":              "APP_PORT",
			"catalogReadGrpcAddr":  "CATALOG_READ_GRPC_ADDR",
			"catalogWriteGrpcAddr": "CATALOG_WRITE_GRPC_ADDR",
			"orderGrpcUrl":         "ORDER_GRPC_ADDR",
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

	return nil
}
