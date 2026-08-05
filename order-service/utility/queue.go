package utility

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

func PublishJSON(ctx context.Context, exchange, routingKey string, payload any) error {
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
		return fmt.Errorf("rabbitmq channel: %w", err)
	}
	defer ch.Close()
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return ch.PublishWithContext(ctx, exchange, routingKey, false, false, amqp091.Publishing{
		ContentType: "application/json", DeliveryMode: amqp091.Persistent, Timestamp: time.Now().UTC(), Body: body,
	})
}

// Kept for compatibility with older controller code.
func PublishOrderEvent(message string) error {
	return PublishJSON(context.Background(), "orders.events", "order.legacy", map[string]string{"message": message})
}
