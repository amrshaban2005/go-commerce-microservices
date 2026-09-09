package messaging

import (
	"fmt"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type RabbitMQOptions struct {
	URL                       string        `mapstructure:"url"`
	Exchange                  string        `mapstructure:"exchange"`
	ProductCreatedQueue       string        `mapstructure:"productCreatedQueue"`
	ProductSearchQueue        string        `mapstructure:"productSearchQueue"`
	ConnectionTimeout         time.Duration `mapstructure:"connectionTimeout"`
	ConsumerProcessingTimeout time.Duration `mapstructure:"consumerProcessingTimeout"`
}

func LoadRabbitMQOptions() (*RabbitMQOptions, error) {
	return configloader.BindKey[RabbitMQOptions](
		"rabbitMQOptions",
		map[string]string{
			"url":                       "RABBITMQ_URL",
			"exchange":                  "RABBITMQ_EXCHANGE",
			"productCreatedQueue":       "PRODUCT_CREATED_QUEUE",
			"productSearchQueue":        "PRODUCT_SEARCH_QUEUE",
			"connectionTimeout":         "RABBITMQ_CONNECTION_TIMEOUT",
			"consumerProcessingTimeout": "RABBITMQ_CONSUMER_PROCESSING_TIMEOUT",
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
	if options.ConnectionTimeout <= 0 {
		return fmt.Errorf("rabbitMQOptions.connectionTimeout must be greater than zero")
	}
	if options.ConsumerProcessingTimeout <= 0 {
		return fmt.Errorf("rabbitMQOptions.consumerProcessingTimeout must be greater than zero")
	}

	return nil
}
