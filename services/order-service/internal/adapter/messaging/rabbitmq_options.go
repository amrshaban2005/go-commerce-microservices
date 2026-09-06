package messaging

import (
	"fmt"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type RabbitMQOptions struct {
	URL                       string        `mapstructure:"url"`
	PublisherExchange         string        `mapstructure:"publisherExchange"`
	ConsumerExchange          string        `mapstructure:"consumerExchange"`
	StockReservedQueue        string        `mapstructure:"stockReservedQueue"`
	StockNotReservedQueue     string        `mapstructure:"stockNotReservedQueue"`
	OutboxIntervalSeconds     int           `mapstructure:"outboxIntervalSeconds"`
	ConnectionTimeout         time.Duration `mapstructure:"connectionTimeout"`
	PublishTimeout            time.Duration `mapstructure:"publishTimeout"`
	OutboxProcessingTimeout   time.Duration `mapstructure:"outboxProcessingTimeout"`
	ConsumerProcessingTimeout time.Duration `mapstructure:"consumerProcessingTimeout"`
}

func LoadRabbitMQOptions() (*RabbitMQOptions, error) {
	return configloader.BindKey[RabbitMQOptions](
		"rabbitMQOptions",
		map[string]string{
			"url":                       "RABBITMQ_URL",
			"publisherExchange":         "RABBITMQ_EXCHANGE_PUBLISHER",
			"consumerExchange":          "RABBITMQ_EXCHANGE_CONSUMER",
			"stockReservedQueue":        "STOCK_RESERVED_QUEUE",
			"stockNotReservedQueue":     "STOCK_NOT_RESERVED_QUEUE",
			"outboxIntervalSeconds":     "OUTBOX_INTERVAL_SECONDS",
			"connectionTimeout":         "RABBITMQ_CONNECTION_TIMEOUT",
			"publishTimeout":            "RABBITMQ_PUBLISH_TIMEOUT",
			"outboxProcessingTimeout":   "OUTBOX_PROCESSING_TIMEOUT",
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
		{name: "publisherExchange", value: options.PublisherExchange},
		{name: "consumerExchange", value: options.ConsumerExchange},
		{name: "stockReservedQueue", value: options.StockReservedQueue},
		{name: "stockNotReservedQueue", value: options.StockNotReservedQueue},
	}

	for _, field := range required {
		if field.value == "" {
			return fmt.Errorf("rabbitMQOptions.%s is required", field.name)
		}
	}
	if options.OutboxIntervalSeconds <= 0 {
		return fmt.Errorf("rabbitMQOptions.outboxIntervalSeconds must be greater than zero")
	}
	if options.PublishTimeout <= 0 {
		return fmt.Errorf("rabbitMQOptions.publishTimeout must be greater than zero")
	}
	if options.ConnectionTimeout <= 0 {
		return fmt.Errorf("rabbitMQOptions.connectionTimeout must be greater than zero")
	}
	if options.OutboxProcessingTimeout <= 0 {
		return fmt.Errorf("rabbitMQOptions.outboxProcessingTimeout must be greater than zero")
	}
	if options.ConsumerProcessingTimeout <= 0 {
		return fmt.Errorf("rabbitMQOptions.consumerProcessingTimeout must be greater than zero")
	}

	return nil
}
