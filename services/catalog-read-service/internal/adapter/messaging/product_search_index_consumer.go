package messaging

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/domain"
	indexingproduct "github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/features/products/indexing_product"
	"github.com/mehdihadeli/go-mediatr"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type ProductSearchIndexConsumer struct {
	channel   *amqp.Channel
	exchange  string
	queueName string
	logger    *zap.Logger
}

func NewProductSearchIndexConsumer(
	channel *amqp.Channel,
	exchange string,
	queueName string,
	logger *zap.Logger,
) *ProductSearchIndexConsumer {
	return &ProductSearchIndexConsumer{
		channel:   channel,
		exchange:  exchange,
		queueName: queueName,
		logger:    logger,
	}
}

func (c *ProductSearchIndexConsumer) Start(ctx context.Context) error {
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

	c.logger.Info("product search index consumer started", zap.String("queue", queue.Name))

	for {
		select {
		case <-ctx.Done():
			return nil

		case delivery, ok := <-deliveries:
			if !ok {
				return errors.New("RabbitMQ search-index deliveries channel closed")
			}
			c.handleMessage(ctx, delivery)
		}
	}
}

func (c *ProductSearchIndexConsumer) handleMessage(ctx context.Context, delivery amqp.Delivery) {
	var event ProductCreatedEvent
	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		c.logger.Error("failed to unmarshal product created search event", zap.Error(err))
		_ = delivery.Nack(false, false)
		return
	}

	_, err := mediatr.Send[*indexingproduct.Command, *struct{}](
		ctx,
		&indexingproduct.Command{
			Product: domain.Product{
				ID:          event.ProductID,
				Name:        event.Name,
				Description: event.Description,
				Price:       event.Price,
				Status:      event.Status,
			},
		},
	)
	if err != nil {
		c.logger.Error(
			"failed to index product created event",
			zap.String("message_id", event.MessageID),
			zap.String("product_id", event.ProductID),
			zap.Error(err),
		)
		_ = delivery.Nack(false, true)
		return
	}

	_ = delivery.Ack(false)
	c.logger.Info(
		"product created event indexed",
		zap.String("message_id", event.MessageID),
		zap.String("product_id", event.ProductID),
	)
}
