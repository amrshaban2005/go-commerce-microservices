package messaging

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/port"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type StockNotReservedEvent struct {
	MessageID string `json:"message_id"`
	OrderID   string `json:"order_id"`
	Reason    string `json:"reason"`
	Items     []Item
}

type StockNotReservedConsumer struct {
	channel      *amqp.Channel
	exchange     string
	queueName    string
	orderService port.OrderService
	logger       *zap.Logger
}

func NewStockNotReservedConsumer(
	channel *amqp.Channel,
	exchange string,
	queueName string,
	orderService port.OrderService,
	logger *zap.Logger,
) *StockNotReservedConsumer {
	return &StockNotReservedConsumer{
		channel:      channel,
		exchange:     exchange,
		queueName:    queueName,
		orderService: orderService,
		logger:       logger,
	}
}

func (c *StockNotReservedConsumer) Start(ctx context.Context) error {
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

	if err := c.channel.QueueBind(
		queue.Name,
		"stock.notreserved",
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

	c.logger.Info("stock not reserved consumer started", zap.String("queue", queue.Name))

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

func (c *StockNotReservedConsumer) handleMessage(ctx context.Context, delivery amqp.Delivery) {
	var event StockNotReservedEvent

	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		c.logger.Error("failed to unmarshal stock not reserved event", zap.Error(err))
		_ = delivery.Nack(false, false)
		return
	}

	messageID, err := uuid.Parse(event.MessageID)
	if err != nil {
		c.logger.Error("failed to parse message id", zap.String("message_id", event.MessageID), zap.Error(err))
		_ = delivery.Nack(false, false)
		return
	}
	orderID, err := uuid.Parse(event.OrderID)
	if err != nil {
		c.logger.Error("failed to parse order id", zap.String("order_id", event.OrderID), zap.Error(err))
		_ = delivery.Nack(false, false)
		return
	}

	err = c.orderService.HandleRejectOrder(
		ctx,
		orderID,
		messageID,
		delivery.Body,
	)
	if err != nil {
		c.logger.Error(
			"failed to handle stock not reserved event",
			zap.String("message_id", event.MessageID),
			zap.String("order_id", event.OrderID),
			zap.Error(err),
		)
		_ = delivery.Nack(false, true)
		return
	}

	_ = delivery.Ack(false)

	c.logger.Info(
		"stock not reserved event consumed",
		zap.String("message_id", event.MessageID),
		zap.String("order_id", event.OrderID),
	)
}
