package messaging

import (
	"fmt"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type RabbitMQOptions struct {
	URL                 string `mapstructure:"url"`
	Exchange            string `mapstructure:"exchange"`
	ProductCreatedQueue string `mapstructure:"productCreatedQueue"`
	ProductSearchQueue  string `mapstructure:"productSearchQueue"`
}

func LoadRabbitMQOptions() (*RabbitMQOptions, error) {
	return configloader.BindKey[RabbitMQOptions](
		"rabbitMQOptions",
		map[string]string{
			"url":                 "RABBITMQ_URL",
			"exchange":            "RABBITMQ_EXCHANGE",
			"productCreatedQueue": "PRODUCT_CREATED_QUEUE",
			"productSearchQueue":  "PRODUCT_SEARCH_QUEUE",
		},
	)
}

func (options *RabbitMQOptions) Validate() error {
	required := []struct {
		name  string
		value string
	}{
		{name: "url", value: options.URL},
		{name: "exchange", value: options.Exchange},
		{name: "productCreatedQueue", value: options.ProductCreatedQueue},
		{name: "productSearchQueue", value: options.ProductSearchQueue},
	}

	for _, field := range required {
		if field.value == "" {
			return fmt.Errorf("rabbitMQOptions.%s is required", field.name)
		}
	}

	return nil
}
