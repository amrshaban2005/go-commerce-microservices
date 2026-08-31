package config

import (
	"errors"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type AppOptions struct {
	GRPCPort   string `mapstructure:"grpcPort"`
	HealthPort string `mapstructure:"healthPort"`
}

func LoadAppOptions() (*AppOptions, error) {
	return configloader.BindKey[AppOptions](
		"appOptions",
		map[string]string{
			"grpcPort":   "GRPC_PORT",
			"healthPort": "HEALTH_PORT",
		},
	)
}

func (options *AppOptions) Validate() error {
	required := []struct {
		name  string
		value string
	}{
		{name: "grpcPort", value: options.GRPCPort},
		{name: "healthPort", value: options.HealthPort},
	}

	for _, field := range required {
		if field.value == "" {
			return errors.New("appOptions." + field.name + " is required")
		}
	}

	return nil
}
