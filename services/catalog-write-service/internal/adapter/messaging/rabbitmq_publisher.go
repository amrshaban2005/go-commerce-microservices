package messaging

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQPublisher struct {
	channel  *amqp.Channel
	exchange string
}

func NewRabbitMQPublisher(channel *amqp.Channel, exchange string) (*RabbitMQPublisher, error) {
	err := channel.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &RabbitMQPublisher{
		channel:  channel,
		exchange: exchange,
	}, nil
}

func (p *RabbitMQPublisher) Publish(ctx context.Context, eventType string, payload []byte) error {
	routingKey := eventRoutingKey(eventType)

	return p.channel.PublishWithContext(
		ctx,
		p.exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         payload,
		},
	)
}

func eventRoutingKey(eventType string) string {
	switch eventType {
	case "ProductCreated":
		return "product.created"
	default:
		return "event.unknown"
	}
}
