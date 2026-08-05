package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"golang-k8s-microservices/inventory-service/internal/inventory"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

const (
	inventoryQueueName = "inventory.order-created"
	consumerName       = "inventory-service"

	processingKeyPrefix = "inventory:events:processing:"
	processedKeyPrefix  = "inventory:events:processed:"

	processingTTL = 5 * time.Minute
	processedTTL  = 24 * time.Hour
)

type Consumer struct {
	inventoryService *inventory.Service
	redisClient      *redis.Client
}

func NewConsumer(
	inventoryService *inventory.Service,
	redisClient *redis.Client,
) *Consumer {
	return &Consumer{
		inventoryService: inventoryService,
		redisClient:      redisClient,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://admin:admin@localhost:5672/"
	}

	connection, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return fmt.Errorf("connect to RabbitMQ: %w", err)
	}
	defer connection.Close()

	channel, err := connection.Channel()
	if err != nil {
		return fmt.Errorf("open RabbitMQ channel: %w", err)
	}
	defer channel.Close()

	if err := channel.ExchangeDeclare(
		ExchangeOrders,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare exchange %q: %w", ExchangeOrders, err)
	}

	queue, err := channel.QueueDeclare(
		inventoryQueueName,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange": "orders.dlx",
		},
	)
	if err != nil {
		return fmt.Errorf("declare queue %q: %w", inventoryQueueName, err)
	}

	if err := channel.QueueBind(
		queue.Name,
		RoutingOrderCreated,
		ExchangeOrders,
		false,
		nil,
	); err != nil {
		return fmt.Errorf(
			"bind queue %q to routing key %q: %w",
			queue.Name,
			RoutingOrderCreated,
			err,
		)
	}

	if err := channel.Qos(10, 0, false); err != nil {
		return fmt.Errorf("configure consumer QoS: %w", err)
	}

	deliveries, err := channel.Consume(
		queue.Name,
		consumerName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("start RabbitMQ consumer: %w", err)
	}

	log.Printf(
		"inventory consumer started: queue=%s routing_key=%s",
		queue.Name,
		RoutingOrderCreated,
	)

	for {
		select {
		case <-ctx.Done():
			log.Println("inventory consumer stopping")
			return nil

		case message, ok := <-deliveries:
			if !ok {
				return errors.New("RabbitMQ delivery channel closed")
			}

			if err := c.handleMessage(ctx, channel, message); err != nil {
				log.Printf(
					"inventory event processing failed: message_id=%s error=%v",
					message.MessageId,
					err,
				)
			}
		}
	}
}

func (c *Consumer) handleMessage(
	ctx context.Context,
	channel *amqp.Channel,
	message amqp.Delivery,
) error {
	var event OrderCreated

	if err := json.Unmarshal(message.Body, &event); err != nil {
		_ = message.Nack(false, false)

		return fmt.Errorf("decode order.created event: %w", err)
	}

	if err := validateOrderCreated(event); err != nil {
		_ = message.Nack(false, false)

		return err
	}

	processed, err := c.isEventProcessed(ctx, event.EventID)
	if err != nil {
		_ = message.Nack(false, true)

		return fmt.Errorf("check processed event: %w", err)
	}

	if processed {
		log.Printf(
			"duplicate event ignored: event_id=%s sku=%s order_id=%s",
			event.EventID,
			event.SKU,
			event.OrderID,
		)

		return message.Ack(false)
	}

	lockAcquired, err := c.acquireEventLock(ctx, event.EventID)
	if err != nil {
		_ = message.Nack(false, true)

		return fmt.Errorf("acquire event lock: %w", err)
	}

	if !lockAcquired {
		log.Printf(
			"event is already being processed: event_id=%s",
			event.EventID,
		)

		return message.Nack(false, true)
	}

	success := false

	defer func() {
		if !success {
			if err := c.releaseEventLock(context.Background(), event.EventID); err != nil {
				log.Printf(
					"release event lock failed: event_id=%s error=%v",
					event.EventID,
					err,
				)
			}
		}
	}()
	if event.SKU == "" {
		_ = message.Nack(false, false)
		return errors.New("order.created sku is required")
	}

	reservation, reserveErr := c.inventoryService.Reserve(
		ctx,
		event.OrderID,
		event.SKU,
		int64(event.Quantity),
	)

	result := InventoryResult{
		EventID:    uuid.NewString(),
		OccurredAt: time.Now().UTC(),
		OrderID:    event.OrderID,
		SKU:        event.SKU,
	}

	routingKey := RoutingInventoryReserved

	if reserveErr != nil {
		routingKey = RoutingInventoryRejected

		result.EventType = RoutingInventoryRejected
		result.Status = "REJECTED"
		result.Reason = reserveErr.Error()

		if errors.Is(reserveErr, inventory.ErrInsufficientStock) {
			log.Printf(
				"inventory rejected order: order_id=%s reason=insufficient_stock",
				event.OrderID,
			)
		} else {
			log.Printf(
				"inventory reservation failed: order_id=%s error=%v",
				event.OrderID,
				reserveErr,
			)
		}
	} else {
		result.EventType = RoutingInventoryReserved
		result.Status = "RESERVED"
		result.ReservationID = reservation.ID

		log.Printf(
			"inventory reserved: order_id=%s reservation_id=%s quantity=%d",
			event.OrderID,
			reservation.ID,
			event.Quantity,
		)
	}

	body, err := json.Marshal(result)
	if err != nil {
		_ = message.Nack(false, true)

		return fmt.Errorf("encode inventory result: %w", err)
	}

	if err := channel.PublishWithContext(
		ctx,
		ExchangeOrders,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    result.EventID,
			Timestamp:    result.OccurredAt,
			Type:         result.EventType,
			Body:         body,
		},
	); err != nil {
		_ = message.Nack(false, true)

		return fmt.Errorf("publish %s event: %w", routingKey, err)
	}

	if err := c.markEventProcessed(ctx, event.EventID); err != nil {
		_ = message.Nack(false, true)

		return fmt.Errorf("mark event processed: %w", err)
	}

	if err := message.Ack(false); err != nil {
		return fmt.Errorf("acknowledge RabbitMQ message: %w", err)
	}

	success = true

	return nil
}

func validateOrderCreated(event OrderCreated) error {
	if event.EventID == "" {
		return errors.New("order.created event_id is required")
	}

	if event.OrderID == "" {
		return errors.New("order.created order_id is required")
	}

	if event.Quantity <= 0 {
		return fmt.Errorf(
			"order.created quantity must be greater than zero: %d",
			event.Quantity,
		)
	}

	return nil
}

func (c *Consumer) isEventProcessed(
	ctx context.Context,
	eventID string,
) (bool, error) {
	if c.redisClient == nil {
		return false, nil
	}

	key := processedKeyPrefix + eventID

	count, err := c.redisClient.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (c *Consumer) acquireEventLock(
	ctx context.Context,
	eventID string,
) (bool, error) {
	if c.redisClient == nil {
		return true, nil
	}

	key := processingKeyPrefix + eventID

	return c.redisClient.SetNX(
		ctx,
		key,
		"1",
		processingTTL,
	).Result()
}

func (c *Consumer) releaseEventLock(
	ctx context.Context,
	eventID string,
) error {
	if c.redisClient == nil {
		return nil
	}

	return c.redisClient.Del(
		ctx,
		processingKeyPrefix+eventID,
	).Err()
}

func (c *Consumer) markEventProcessed(
	ctx context.Context,
	eventID string,
) error {
	if c.redisClient == nil {
		return nil
	}

	pipeline := c.redisClient.TxPipeline()

	pipeline.Del(
		ctx,
		processingKeyPrefix+eventID,
	)

	pipeline.Set(
		ctx,
		processedKeyPrefix+eventID,
		"1",
		processedTTL,
	)

	_, err := pipeline.Exec(ctx)

	return err
}
