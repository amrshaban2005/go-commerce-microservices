package messaging

import (
	"context"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const deliveryAttemptHeader = "x-delivery-attempt"

type failureDisposition string

const (
	dispositionRetry      failureDisposition = "retry scheduled"
	dispositionDeadLetter failureDisposition = "sent to dead-letter queue"
)

type retryPolicy struct {
	channel            *amqp.Channel
	sourceExchange     string
	sourceRoutingKey   string
	retryQueue         string
	deadLetterExchange string
	deadLetterQueue    string
	maxAttempts        int
	retryDelay         time.Duration
	publishTimeout     time.Duration
}

func newRetryPolicy(
	channel *amqp.Channel,
	sourceExchange string,
	sourceRoutingKey string,
	queueName string,
	maxAttempts int,
	retryDelay time.Duration,
	publishTimeout time.Duration,
) *retryPolicy {
	return &retryPolicy{
		channel:            channel,
		sourceExchange:     sourceExchange,
		sourceRoutingKey:   sourceRoutingKey,
		retryQueue:         queueName + ".retry",
		deadLetterExchange: sourceExchange + ".dead-letter",
		deadLetterQueue:    queueName + ".dlq",
		maxAttempts:        maxAttempts,
		retryDelay:         retryDelay,
		publishTimeout:     publishTimeout,
	}
}

func (p *retryPolicy) declare() error {
	if err := p.channel.Confirm(false); err != nil {
		return fmt.Errorf("enable publisher confirms for retries: %w", err)
	}
	if err := p.channel.ExchangeDeclare(
		p.deadLetterExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare dead-letter exchange: %w", err)
	}
	if _, err := p.channel.QueueDeclare(
		p.deadLetterQueue,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare dead-letter queue: %w", err)
	}
	if err := p.channel.QueueBind(
		p.deadLetterQueue,
		p.deadLetterQueue,
		p.deadLetterExchange,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("bind dead-letter queue: %w", err)
	}
	if _, err := p.channel.QueueDeclare(
		p.retryQueue,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-message-ttl":             p.retryDelay.Milliseconds(),
			"x-dead-letter-exchange":    p.sourceExchange,
			"x-dead-letter-routing-key": p.sourceRoutingKey,
		},
	); err != nil {
		return fmt.Errorf("declare retry queue: %w", err)
	}

	return nil
}

func (p *retryPolicy) retry(ctx context.Context, delivery amqp.Delivery, reason string) (failureDisposition, error) {
	attempt := deliveryAttempt(delivery.Headers)
	if attempt >= p.maxAttempts {
		return dispositionDeadLetter, p.deadLetter(ctx, delivery, reason)
	}

	headers := copyHeaders(delivery.Headers)
	headers[deliveryAttemptHeader] = int32(attempt + 1)
	headers["x-last-failure-reason"] = reason

	return dispositionRetry, p.publishThenAck(ctx, delivery, "", p.retryQueue, headers)
}

func (p *retryPolicy) deadLetter(ctx context.Context, delivery amqp.Delivery, reason string) error {
	headers := copyHeaders(delivery.Headers)
	headers[deliveryAttemptHeader] = int32(deliveryAttempt(delivery.Headers))
	headers["x-dead-letter-reason"] = reason
	headers["x-dead-lettered-at"] = time.Now().UTC().Format(time.RFC3339)

	return p.publishThenAck(
		ctx,
		delivery,
		p.deadLetterExchange,
		p.deadLetterQueue,
		headers,
	)
}

func (p *retryPolicy) publishThenAck(
	ctx context.Context,
	delivery amqp.Delivery,
	exchange string,
	routingKey string,
	headers amqp.Table,
) error {
	publishCtx, cancel := context.WithTimeout(ctx, p.publishTimeout)
	defer cancel()

	confirmation, err := p.channel.PublishWithDeferredConfirmWithContext(
		publishCtx,
		exchange,
		routingKey,
		true,
		false,
		amqp.Publishing{
			Headers:         headers,
			ContentType:     delivery.ContentType,
			ContentEncoding: delivery.ContentEncoding,
			DeliveryMode:    amqp.Persistent,
			Priority:        delivery.Priority,
			CorrelationId:   delivery.CorrelationId,
			ReplyTo:         delivery.ReplyTo,
			MessageId:       delivery.MessageId,
			Timestamp:       delivery.Timestamp,
			Type:            delivery.Type,
			AppId:           delivery.AppId,
			Body:            delivery.Body,
		},
	)
	if err == nil {
		var acknowledged bool
		acknowledged, err = confirmation.WaitContext(publishCtx)
		if err == nil && !acknowledged {
			err = errors.New("RabbitMQ negatively acknowledged retry or dead-letter message")
		}
	}
	if err != nil {
		if nackErr := delivery.Nack(false, true); nackErr != nil {
			return errors.Join(err, fmt.Errorf("requeue original message: %w", nackErr))
		}
		return err
	}

	if err := delivery.Ack(false); err != nil {
		return fmt.Errorf("acknowledge original message: %w", err)
	}
	return nil
}

func deliveryAttempt(headers amqp.Table) int {
	if headers == nil {
		return 1
	}
	switch value := headers[deliveryAttemptHeader].(type) {
	case int:
		return max(value, 1)
	case int8:
		return max(int(value), 1)
	case int16:
		return max(int(value), 1)
	case int32:
		return max(int(value), 1)
	case int64:
		return max(int(value), 1)
	default:
		return 1
	}
}

func copyHeaders(headers amqp.Table) amqp.Table {
	result := make(amqp.Table, len(headers)+3)
	for key, value := range headers {
		result[key] = value
	}
	return result
}
