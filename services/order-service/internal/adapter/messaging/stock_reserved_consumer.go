package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/amrshaban2005/go-commerce-microservices/services/order-service/internal/port"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
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
	channel      *amqp.Channel
	exchange     string
	queueName    string
	orderService port.OrderService
}

func NewStockReservedConsumer(
	channel *amqp.Channel,
	exchange string,
	queueName string,
	orderService port.OrderService,
) *StockReservedConsumer {
	return &StockReservedConsumer{
		channel:      channel,
		exchange:     exchange,
		queueName:    queueName,
		orderService: orderService,
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

	slog.Info("stock reserved consumer started", "queue", queue.Name)

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
	var event StockReservedEvent

	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		slog.Error("failed to unmarshal stock reserved event", "error", err)
		_ = delivery.Nack(false, false)
		return
	}

	messageID, err := uuid.Parse(event.MessageID)
	if err != nil {
		slog.Error("failed to parse message id", "error", err)
		_ = delivery.Nack(false, false)
		return
	}
	orderID, err := uuid.Parse(event.OrderID)
	if err != nil {
		slog.Error("failed to parse order id", "error", err)
		_ = delivery.Nack(false, false)
		return
	}

	err = c.orderService.HandleConfirmOrder(
		ctx,
		orderID,
		messageID,
		delivery.Body,
	)
	if err != nil {
		slog.Error("failed to handle stock reserved event", "error", err)
		_ = delivery.Nack(false, true)
		return
	}

	_ = delivery.Ack(false)

	slog.Info("Reserve stock reserved event consumed", "order_id", event.OrderID)
}
