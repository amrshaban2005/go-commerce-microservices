package messaging

import (
	"fmt"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type RabbitMQOptions struct {
	URL                   string `mapstructure:"url"`
	ConsumerExchange      string `mapstructure:"consumerExchange"`
	PublisherExchange     string `mapstructure:"publisherExchange"`
	ReserveStockQueue     string `mapstructure:"reserveStockQueue"`
	OutboxIntervalSeconds int    `mapstructure:"outboxIntervalSeconds"`
}

func LoadRabbitMQOptions() (*RabbitMQOptions, error) {
	return configloader.BindKey[RabbitMQOptions](
		"rabbitMQOptions",
		map[string]string{
			"url":                   "RABBITMQ_URL",
			"consumerExchange":      "RABBITMQ_EXCHANGE_CONSUMER",
			"publisherExchange":     "RABBITMQ_EXCHANGE_PUBLISHER",
			"reserveStockQueue":     "RESERVE_STOCK_QUEUE",
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
		{name: "consumerExchange", value: options.ConsumerExchange},
		{name: "publisherExchange", value: options.PublisherExchange},
		{name: "reserveStockQueue", value: options.ReserveStockQueue},
	}

	for _, field := range required {
		if field.value == "" {
			return fmt.Errorf("rabbitMQOptions.%s is required", field.name)
		}
	}
	if options.OutboxIntervalSeconds <= 0 {
		return fmt.Errorf("rabbitMQOptions.outboxIntervalSeconds must be greater than zero")
	}

	return nil
}
