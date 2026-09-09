package database

import (
	"fmt"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type MongoOptions struct {
	URI               string        `mapstructure:"uri"`
	Database          string        `mapstructure:"database"`
	ConnectionTimeout time.Duration `mapstructure:"connectionTimeout"`
}

func LoadMongoOptions() (*MongoOptions, error) {
	return configloader.BindKey[MongoOptions](
		"mongoOptions",
		map[string]string{
			"uri":               "MONGO_URI",
			"database":          "MONGO_DATABASE",
			"connectionTimeout": "MONGO_CONNECTION_TIMEOUT",
		},
	)
}

func (options *MongoOptions) Validate() error {
	if options.URI == "" {
		return fmt.Errorf("mongoOptions.uri is required")
	}
	if options.Database == "" {
		return fmt.Errorf("mongoOptions.database is required")
	}
	if options.ConnectionTimeout <= 0 {
		return fmt.Errorf("mongoOptions.connectionTimeout must be greater than zero")
	}

	return nil
}
