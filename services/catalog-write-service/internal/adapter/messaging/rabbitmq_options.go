package messaging

import (
	"fmt"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type RabbitMQOptions struct {
	URL                   string `mapstructure:"url"`
	Exchange              string `mapstructure:"exchange"`
	OutboxIntervalSeconds int    `mapstructure:"outboxIntervalSeconds"`
}

func LoadRabbitMQOptions() (*RabbitMQOptions, error) {
	return configloader.BindKey[RabbitMQOptions](
		"rabbitMQOptions",
		map[string]string{
			"url":                   "RABBITMQ_URL",
			"exchange":              "RABBITMQ_EXCHANGE",
			"outboxIntervalSeconds": "OUTBOX_INTERVAL_SECONDS",
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

	return nil
}
