package messaging

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/domain"
	"github.com/amrshaban2005/go-commerce-microservices/services/inventory-service/internal/port"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type ReserveStockRequestedEvent struct {
	MessageID string `json:"message_id"`
	OrderID   string `json:"order_id"`
	Items     []Item
}

type Item struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type ReserveStockRequestedConsumer struct {
	channel          *amqp.Channel
	exchange         string
	queueName        string
	inventoryService port.InventoryService
	logger           *zap.Logger
}

func NewReserveStockRequestedConsumer(
	channel *amqp.Channel,
	exchange string,
	queueName string,
	inventoryService port.InventoryService,
	logger *zap.Logger,
) *ReserveStockRequestedConsumer {
	return &ReserveStockRequestedConsumer{
		channel:          channel,
		exchange:         exchange,
		queueName:        queueName,
		inventoryService: inventoryService,
		logger:           logger,
	}
}

func (c *ReserveStockRequestedConsumer) Start(ctx context.Context) error {
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
		"order.created",
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

	c.logger.Info("reserve stock requested consumer started", zap.String("queue", queue.Name))

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

func (c *ReserveStockRequestedConsumer) handleMessage(ctx context.Context, delivery amqp.Delivery) {
	var event ReserveStockRequestedEvent

	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		c.logger.Error("failed to unmarshal reserve stock requested event", zap.Error(err))
		_ = delivery.Nack(false, false)
		return
	}

	eventItems := make([]domain.ReserveStockItem, 0, len(event.Items))

	for _, eventItem := range event.Items {
		productID, err := uuid.Parse(eventItem.ProductID)
		if err != nil {
			c.logger.Error("failed to parse product id", zap.String("product_id", eventItem.ProductID), zap.Error(err))
			_ = delivery.Nack(false, false)
			return
		}
		eventItems = append(eventItems, domain.ReserveStockItem{
			ProductID: productID,
			Quantity:  eventItem.Quantity,
		})
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

	err = c.inventoryService.HandleReserveStockRequested(
		ctx,
		messageID,
		orderID,
		eventItems,
		delivery.Body,
	)
	if err != nil {
		c.logger.Error(
			"failed to handle reserve stock requested event",
			zap.String("message_id", event.MessageID),
			zap.String("order_id", event.OrderID),
			zap.Error(err),
		)
		_ = delivery.Nack(false, true)
		return
	}

	_ = delivery.Ack(false)

	c.logger.Info(
		"reserve stock requested event consumed",
		zap.String("message_id", event.MessageID),
		zap.String("order_id", event.OrderID),
	)
}
