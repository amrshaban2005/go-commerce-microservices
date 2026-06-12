package config

import (
	"errors"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type AppOptions struct {
	GRPCPort string `mapstructure:"grpcPort"`
}

func LoadAppOptions() (*AppOptions, error) {
	return configloader.BindKey[AppOptions](
		"appOptions",
		map[string]string{
			"grpcPort": "GRPC_PORT",
		},
	)
}

func (options *AppOptions) Validate() error {
	if options.GRPCPort == "" {
		return errors.New("appOptions.grpcPort is required")
	}

	return nil
}
