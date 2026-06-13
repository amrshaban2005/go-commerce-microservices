package database

import (
	"fmt"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type PostgresOptions struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"sslMode"`
}

func LoadPostgresOptions() (*PostgresOptions, error) {
	return configloader.BindKey[PostgresOptions](
		"postgresOptions",
		map[string]string{
			"host":     "DB_HOST",
			"port":     "DB_PORT",
			"user":     "DB_USER",
			"password": "DB_PASSWORD",
			"database": "DB_NAME",
			"sslMode":  "DB_SSLMODE",
		},
	)
}

func (options *PostgresOptions) Validate() error {
	required := []struct {
		name  string
		value string
	}{
		{name: "host", value: options.Host},
		{name: "port", value: options.Port},
		{name: "user", value: options.User},
		{name: "password", value: options.Password},
		{name: "database", value: options.Database},
		{name: "sslMode", value: options.SSLMode},
	}

	for _, field := range required {
		if field.value == "" {
			return fmt.Errorf("postgresOptions.%s is required", field.name)
		}
	}

	return nil
}
