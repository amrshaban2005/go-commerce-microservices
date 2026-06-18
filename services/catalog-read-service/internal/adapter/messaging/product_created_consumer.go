package messaging

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/domain"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
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
	logger         *zap.Logger
}

func NewProductCreatedConsumer(
	channel *amqp.Channel,
	exchange string,
	queueName string,
	productService port.ProductService,
	logger *zap.Logger,
) *ProductCreatedConsumer {
	return &ProductCreatedConsumer{
		channel:        channel,
		exchange:       exchange,
		queueName:      queueName,
		productService: productService,
		logger:         logger,
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

	c.logger.Info("product created consumer started", zap.String("queue", queue.Name))

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
		c.logger.Error("failed to unmarshal product created event", zap.Error(err))
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
		c.logger.Error(
			"failed to handle product created event",
			zap.String("message_id", event.MessageID),
			zap.String("product_id", event.ProductID),
			zap.Error(err),
		)
		_ = delivery.Nack(false, true)
		return
	}

	_ = delivery.Ack(false)

	c.logger.Info(
		"product created event consumed",
		zap.String("message_id", event.MessageID),
		zap.String("product_id", event.ProductID),
	)
}
