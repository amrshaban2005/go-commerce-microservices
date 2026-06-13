package database

import (
	"fmt"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type MongoOptions struct {
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"`
}

func LoadMongoOptions() (*MongoOptions, error) {
	return configloader.BindKey[MongoOptions](
		"mongoOptions",
		map[string]string{
			"uri":      "MONGO_URI",
			"database": "MONGO_DATABASE",
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

	return nil
}
