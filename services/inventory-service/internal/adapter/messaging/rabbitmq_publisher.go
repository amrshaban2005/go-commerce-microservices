package messaging

import (
	"context"
	"errors"
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQPublisher struct {
	channel  *amqp.Channel
	exchange string
	returns  <-chan amqp.Return
	mu       sync.Mutex
}

func NewRabbitMQPublisher(
	channel *amqp.Channel,
	exchange string,
) (*RabbitMQPublisher, error) {
	if err := channel.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return nil, err
	}

	if err := channel.Confirm(false); err != nil {
		return nil, err
	}

	return &RabbitMQPublisher{
		channel:  channel,
		exchange: exchange,
		returns:  channel.NotifyReturn(make(chan amqp.Return, 1)),
	}, nil
}

func (p *RabbitMQPublisher) Publish(ctx context.Context, eventType string, payload []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	routingKey, err := eventRoutingKey(eventType)
	if err != nil {
		return err
	}

	confirmation, err := p.channel.PublishWithDeferredConfirmWithContext(ctx, p.exchange, routingKey, true, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         payload,
	})

	if err != nil {
		return err
	}

	acked, err := confirmation.WaitContext(ctx)
	if err != nil {
		return err
	}
	if !acked {
		return errors.New("RabbitMQ negatively acknowledged message")
	}

	select {
	case returned, ok := <-p.returns:
		if !ok {
			return errors.New("RabbitMQ return notification channel closed")
		}
		return fmt.Errorf(
			"RabbitMQ could not route message: code=%d reason=%s routing_key=%s",
			returned.ReplyCode,
			returned.ReplyText,
			returned.RoutingKey,
		)
	default:
		return nil
	}
}

func eventRoutingKey(eventType string) (string, error) {
	switch eventType {
	case "StockReserved":
		return "stock.reserved", nil
	case "StockReservationFailed":
		return "stock.notreserved", nil
	default:
		return "", fmt.Errorf("unsupported event type: %s", eventType)
	}
}
