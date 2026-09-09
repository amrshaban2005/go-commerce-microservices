package database

import (
	"fmt"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type ElasticsearchOptions struct {
	URL               string        `mapstructure:"url"`
	ProductsIndex     string        `mapstructure:"productsIndex"`
	ConnectionTimeout time.Duration `mapstructure:"connectionTimeout"`
}

func LoadElasticsearchOptions() (*ElasticsearchOptions, error) {
	return configloader.BindKey[ElasticsearchOptions](
		"elasticsearchOptions",
		map[string]string{
			"url":               "ELASTICSEARCH_URL",
			"productsIndex":     "ELASTICSEARCH_PRODUCTS_INDEX",
			"connectionTimeout": "ELASTICSEARCH_CONNECTION_TIMEOUT",
		},
	)
}

func (options *ElasticsearchOptions) Validate() error {
	if options.URL == "" {
		return fmt.Errorf("elasticsearchOptions.url is required")
	}
	if options.ProductsIndex == "" {
		return fmt.Errorf("elasticsearchOptions.productsIndex is required")
	}
	if options.ConnectionTimeout <= 0 {
		return fmt.Errorf("elasticsearchOptions.connectionTimeout must be greater than zero")
	}

	return nil
}
