package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/port"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type StockReservedEvent struct {
	MessageID string `json:"message_id"`
	OrderID   string `json:"order_id"`
	Items     []Item
}

type Item struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type StockReservedConsumer struct {
	channel           *amqp.Channel
	exchange          string
	queueName         string
	orderService      port.OrderService
	logger            *zap.Logger
	processingTimeout time.Duration
	retry             *retryPolicy
}

func NewStockReservedConsumer(
	channel *amqp.Channel,
	exchange string,
	queueName string,
	orderService port.OrderService,
	logger *zap.Logger,
	processingTimeout time.Duration,
	retryDelay time.Duration,
	maxAttempts int,
	publishTimeout time.Duration,
) *StockReservedConsumer {
	return &StockReservedConsumer{
		channel:           channel,
		exchange:          exchange,
		queueName:         queueName,
		orderService:      orderService,
		logger:            logger,
		processingTimeout: processingTimeout,
		retry: newRetryPolicy(
			channel,
			exchange,
			"stock.reserved",
			queueName,
			maxAttempts,
			retryDelay,
			publishTimeout,
		),
	}
}

func (c *StockReservedConsumer) Start(ctx context.Context) error {
	if err := c.channel.ExchangeDeclare(
		c.exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	queue, err := c.channel.QueueDeclare(
		c.queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}
	if err := c.retry.declare(); err != nil {
		return err
	}

	if err := c.channel.QueueBind(
		queue.Name,
		"stock.reserved",
		c.exchange,
		false,
		nil,
	); err != nil {
		return err
	}

	deliveries, err := c.channel.Consume(
		queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	c.logger.Info("stock reserved consumer started", zap.String("queue", queue.Name))

	for {
		select {
		case <-ctx.Done():
			return nil

		case delivery, ok := <-deliveries:
			if !ok {
				return errors.New("RabbitMQ deliveries channel closed")
			}
			c.handleMessage(ctx, delivery)
		}
	}
}

func (c *StockReservedConsumer) handleMessage(ctx context.Context, delivery amqp.Delivery) {
	processingCtx, cancel := context.WithTimeout(ctx, c.processingTimeout)
	defer cancel()

	var event StockReservedEvent

	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		c.logger.Error("failed to unmarshal stock reserved event", zap.Error(err))
		if deadLetterErr := c.retry.deadLetter(ctx, delivery, "invalid JSON"); deadLetterErr != nil {
			c.logger.Error("failed to dead-letter invalid stock reserved event", zap.Error(deadLetterErr))
		}
		return
	}

	messageID, err := uuid.Parse(event.MessageID)
	if err != nil {
		c.logger.Error("failed to parse message id", zap.String("message_id", event.MessageID), zap.Error(err))
		if deadLetterErr := c.retry.deadLetter(ctx, delivery, "invalid message_id"); deadLetterErr != nil {
			c.logger.Error("failed to dead-letter stock reserved event", zap.Error(deadLetterErr))
		}
		return
	}
	orderID, err := uuid.Parse(event.OrderID)
	if err != nil {
		c.logger.Error("failed to parse order id", zap.String("order_id", event.OrderID), zap.Error(err))
		if deadLetterErr := c.retry.deadLetter(ctx, delivery, "invalid order_id"); deadLetterErr != nil {
			c.logger.Error("failed to dead-letter stock reserved event", zap.Error(deadLetterErr))
		}
		return
	}

	err = c.orderService.HandleConfirmOrder(
		processingCtx,
		orderID,
		messageID,
		delivery.Body,
	)
	if err != nil {
		c.logger.Error(
			"failed to handle stock reserved event",
			zap.String("message_id", event.MessageID),
			zap.String("order_id", event.OrderID),
			zap.Error(err),
		)
		disposition, retryErr := c.retry.retry(ctx, delivery, "processing failed")
		if retryErr != nil {
			c.logger.Error("failed to schedule stock reserved event failure", zap.Error(retryErr))
		} else {
			c.logger.Warn(string(disposition), zap.String("message_id", event.MessageID))
		}
		return
	}

	if err := delivery.Ack(false); err != nil {
		c.logger.Error("failed to acknowledge stock reserved event", zap.Error(err))
		return
	}

	c.logger.Info(
		"stock reserved event consumed",
		zap.String("message_id", event.MessageID),
		zap.String("order_id", event.OrderID),
	)
}
