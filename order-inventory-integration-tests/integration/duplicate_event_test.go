//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

type orderCreatedEvent struct {
	EventID        string    `json:"event_id"`
	EventType      string    `json:"event_type"`
	OccurredAt     time.Time `json:"occurred_at"`
	OrderID        string    `json:"order_id"`
	CustomerID     string    `json:"customer_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	Quantity       int       `json:"quantity"`
	UnitPrice      int64     `json:"unit_price"`
	Currency       string    `json:"currency"`
	TotalAmount    int64     `json:"total_amount"`
}

func TestDuplicateOrderCreatedEventReservesOnce(t *testing.T) {
	resetState(t, 20)

	event := orderCreatedEvent{
		EventID:        uuid.NewString(),
		EventType:      "order.created",
		OccurredAt:     time.Now().UTC(),
		OrderID:        uuid.NewString(),
		CustomerID:     "22222222-2222-4222-8222-222222222222",
		IdempotencyKey: "it-duplicate-" + uuid.NewString(),
		Quantity:       3,
		UnitPrice:      1000,
		Currency:       "INR",
		TotalAmount:    3000,
	}

	publishOrderCreated(t, event)
	publishOrderCreated(t, event)

	eventually(t, 15*time.Second, 250*time.Millisecond, func() (bool, string) {
		stock := getStock(t)
		count := reservationCount(t, event.OrderID)
		return count == 1 && stock.Available == 17 && stock.Reserved == 3,
			stateString(orderResponse{}, stock)
	})
}

func publishOrderCreated(t *testing.T, event orderCreatedEvent) {
	t.Helper()
	connection, err := amqp.Dial(rabbitMQURL)
	require.NoError(t, err)
	defer connection.Close()

	channel, err := connection.Channel()
	require.NoError(t, err)
	defer channel.Close()

	require.NoError(t, channel.ExchangeDeclare("orders.events", "topic", true, false, false, false, nil))
	body, err := json.Marshal(event)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, channel.PublishWithContext(ctx, "orders.events", "order.created", false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    event.EventID,
		Timestamp:    event.OccurredAt,
		Body:         body,
	}))
}
