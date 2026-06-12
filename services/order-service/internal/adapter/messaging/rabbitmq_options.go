package messaging

import (
	"fmt"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type RabbitMQOptions struct {
	URL                   string `mapstructure:"url"`
	PublisherExchange     string `mapstructure:"publisherExchange"`
	ConsumerExchange      string `mapstructure:"consumerExchange"`
	StockReservedQueue    string `mapstructure:"stockReservedQueue"`
	StockNotReservedQueue string `mapstructure:"stockNotReservedQueue"`
	OutboxIntervalSeconds string    `mapstructure:"outboxIntervalSeconds"`
}

func LoadRabbitMQOptions() (*RabbitMQOptions, error) {
	return configloader.BindKey[RabbitMQOptions](
		"rabbitMQOptions",
		map[string]string{
			"url":                   "RABBITMQ_URL",
			"publisherExchange":     "RABBITMQ_EXCHANGE_PUBLISHER",
			"consumerExchange":      "RABBITMQ_EXCHANGE_CONSUMER",
			"stockReservedQueue":    "STOCK_RESERVED_QUEUE",
			"stockNotReservedQueue": "STOCK_NOT_RESERVED_QUEUE",
			"outboxIntervalSeconds": "OUTBOX_INTERVAL_SECONDS",
		},
	)
}

func (options *RabbitMQOptions) Validate() error {
	required := []struct {
		name  string
		value string
	}{
		{name: "url", value: options.URL},
		{name: "publisherExchange", value: options.PublisherExchange},
		{name: "consumerExchange", value: options.ConsumerExchange},
		{name: "stockReservedQueue", value: options.StockReservedQueue},
		{name: "stockNotReservedQueue", value: options.StockNotReservedQueue},
		{name: "outboxIntervalSeconds", value: options.OutboxIntervalSeconds},
	}

	for _, field := range required {
		if field.value == "" {
			return fmt.Errorf("rabbitMQOptions.%s is required", field.name)
		}
	}

	return nil
}
