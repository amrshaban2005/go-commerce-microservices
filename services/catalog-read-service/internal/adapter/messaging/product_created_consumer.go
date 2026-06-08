package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/domain"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ProductCreatedEvent struct {
	MessageID   string  `json:"message_id"`
	ProductID   string  `json:"product_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Status      string  `json:"status"`
}

type ProductCreatedConsumer struct {
	channel        *amqp.Channel
	exchange       string
	queueName      string
	productService port.ProductService
}

func NewProductCreatedConsumer(
	channel *amqp.Channel,
	exchange string,
	queueName string,
	productService port.ProductService,
) *ProductCreatedConsumer {
	return &ProductCreatedConsumer{
		channel:        channel,
		exchange:       exchange,
		queueName:      queueName,
		productService: productService,
	}
}

func (c *ProductCreatedConsumer) Start(ctx context.Context) error {
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
		"product.created",
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

	slog.Info("product created consumer started", "queue", queue.Name)

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

func (c *ProductCreatedConsumer) handleMessage(ctx context.Context, delivery amqp.Delivery) {
	var event ProductCreatedEvent

	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		slog.Error("failed to unmarshal ProductCreated event", "error", err)
		_ = delivery.Nack(false, false)
		return
	}

	product := domain.Product{
		ID:          event.ProductID,
		Name:        event.Name,
		Description: event.Description,
		Price:       event.Price,
		Status:      event.Status,
	}

	err := c.productService.HandleProductCreated(
		ctx,
		event.MessageID,
		product,
		delivery.Body,
	)
	if err != nil {
		slog.Error("failed to handle ProductCreated event", "error", err)
		_ = delivery.Nack(false, true)
		return
	}

	_ = delivery.Ack(false)

	slog.Info("ProductCreated event consumed", "product_id", event.ProductID)
}
