package messaging

import (
	"fmt"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type RabbitMQOptions struct {
	URL                     string        `mapstructure:"url"`
	Exchange                string        `mapstructure:"exchange"`
	OutboxIntervalSeconds   int           `mapstructure:"outboxIntervalSeconds"`
	ConnectionTimeout       time.Duration `mapstructure:"connectionTimeout"`
	PublishTimeout          time.Duration `mapstructure:"publishTimeout"`
	OutboxProcessingTimeout time.Duration `mapstructure:"outboxProcessingTimeout"`
}

func LoadRabbitMQOptions() (*RabbitMQOptions, error) {
	return configloader.BindKey[RabbitMQOptions](
		"rabbitMQOptions",
		map[string]string{
			"url":                     "RABBITMQ_URL",
			"exchange":                "RABBITMQ_EXCHANGE",
			"outboxIntervalSeconds":   "OUTBOX_INTERVAL_SECONDS",
			"connectionTimeout":       "RABBITMQ_CONNECTION_TIMEOUT",
			"publishTimeout":          "RABBITMQ_PUBLISH_TIMEOUT",
			"outboxProcessingTimeout": "OUTBOX_PROCESSING_TIMEOUT",
		},
	)
}

func (options *RabbitMQOptions) Validate() error {
	if options.URL == "" {
		return fmt.Errorf("rabbitMQOptions.url is required")
	}
	if options.Exchange == "" {
		return fmt.Errorf("rabbitMQOptions.exchange is required")
	}
	if options.OutboxIntervalSeconds <= 0 {
		return fmt.Errorf("rabbitMQOptions.outboxIntervalSeconds must be greater than zero")
	}
	if options.ConnectionTimeout <= 0 {
		return fmt.Errorf("rabbitMQOptions.connectionTimeout must be greater than zero")
	}
	if options.PublishTimeout <= 0 {
		return fmt.Errorf("rabbitMQOptions.publishTimeout must be greater than zero")
	}
	if options.OutboxProcessingTimeout <= 0 {
		return fmt.Errorf("rabbitMQOptions.outboxProcessingTimeout must be greater than zero")
	}

	return nil
}
