package messaging

import (
	"testing"
	"time"
)

func TestRabbitMQOptionsValidateRequiresOperationTimeouts(t *testing.T) {
	valid := RabbitMQOptions{
		URL: "amqp://localhost", PublisherExchange: "orders", ConsumerExchange: "inventory",
		StockReservedQueue: "reserved", StockNotReservedQueue: "not-reserved",
		OutboxIntervalSeconds: 5, ConnectionTimeout: 5 * time.Second, PublishTimeout: 5 * time.Second,
		OutboxProcessingTimeout: 30 * time.Second, ConsumerProcessingTimeout: 10 * time.Second,
		ConsumerRetryDelay: 10 * time.Second, ConsumerMaxAttempts: 3,
	}

	tests := []struct {
		name    string
		options RabbitMQOptions
		wantErr bool
	}{
		{name: "valid", options: valid},
		{name: "missing publish timeout", options: func() RabbitMQOptions {
			options := valid
			options.PublishTimeout = 0
			return options
		}(), wantErr: true},
		{name: "missing consumer retry delay", options: func() RabbitMQOptions {
			options := valid
			options.ConsumerRetryDelay = 0
			return options
		}(), wantErr: true},
		{name: "missing consumer max attempts", options: func() RabbitMQOptions {
			options := valid
			options.ConsumerMaxAttempts = 0
			return options
		}(), wantErr: true},
		{name: "missing outbox processing timeout", options: func() RabbitMQOptions {
			options := valid
			options.OutboxProcessingTimeout = 0
			return options
		}(), wantErr: true},
		{name: "missing connection timeout", options: func() RabbitMQOptions {
			options := valid
			options.ConnectionTimeout = 0
			return options
		}(), wantErr: true},
		{name: "missing consumer timeout", options: func() RabbitMQOptions {
			options := valid
			options.ConsumerProcessingTimeout = 0
			return options
		}(), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.options.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
