package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type ResultConsumer struct {
	db    *sql.DB
	redis *redis.Client
}

func NewResultConsumer(db *sql.DB, r *redis.Client) *ResultConsumer {
	return &ResultConsumer{db: db, redis: r}
}
func (c *ResultConsumer) Run(ctx context.Context) error {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://admin:admin@localhost:5672/"
	}
	conn, err := amqp091.Dial(url)
	if err != nil {
		return fmt.Errorf("rabbitmq dial: %w", err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	if err = ch.ExchangeDeclare(ExchangeOrders, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	q, err := ch.QueueDeclare("order.inventory-results", true, false, false, false, nil)
	if err != nil {
		return err
	}
	for _, key := range []string{RoutingInventoryReserved, RoutingInventoryRejected} {
		if err = ch.QueueBind(q.Name, key, ExchangeOrders, false, nil); err != nil {
			return err
		}
	}
	msgs, err := ch.Consume(q.Name, "order-service", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case m := <-msgs:
			var evt InventoryResult
			if err := json.Unmarshal(m.Body, &evt); err != nil {
				m.Nack(false, false)
				continue
			}
			seenKey := "events:processed:" + evt.EventID
			if c.redis != nil {
				ok, err := c.redis.SetNX(ctx, seenKey, "1", 24*time.Hour).Result()
				if err == nil && !ok {
					m.Ack(false)
					continue
				}
			}
			status := "CONFIRMED"
			if evt.Status == "REJECTED" {
				status = "CANCELLED"
			}
			if _, err := c.db.ExecContext(ctx, "UPDATE orders SET status=$2, version=version+1 WHERE id=$1::uuid", evt.OrderID, status); err != nil {
				m.Nack(false, true)
				continue
			}
			m.Ack(false)
		}
	}
}
