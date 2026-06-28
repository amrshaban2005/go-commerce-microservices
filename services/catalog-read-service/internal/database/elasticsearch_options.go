package database

import (
	"fmt"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type ElasticsearchOptions struct {
	URL           string `mapstructure:"url"`
	ProductsIndex string `mapstructure:"productsIndex"`
}

func LoadElasticsearchOptions() (*ElasticsearchOptions, error) {
	return configloader.BindKey[ElasticsearchOptions](
		"elasticsearchOptions",
		map[string]string{
			"url":           "ELASTICSEARCH_URL",
			"productsIndex": "ELASTICSEARCH_PRODUCTS_INDEX",
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

	return nil
}
