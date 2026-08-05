package events

import "time"

const (
	ExchangeOrders           = "orders.events"
	RoutingOrderCreated      = "order.created"
	RoutingInventoryReserved = "inventory.reserved"
	RoutingInventoryRejected = "inventory.rejected"
)

type OrderCreated struct {
	EventID        string    `json:"event_id"`
	EventType      string    `json:"event_type"`
	OccurredAt     time.Time `json:"occurred_at"`
	OrderID        string    `json:"order_id"`
	CustomerID     string    `json:"customer_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	SKU            string    `json:"sku"`
	Quantity       int       `json:"quantity"`
	UnitPrice      int64     `json:"unit_price"`
	Currency       string    `json:"currency"`
	TotalAmount    int64     `json:"total_amount"`
}

type InventoryResult struct {
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	OccurredAt    time.Time `json:"occurred_at"`
	OrderID       string    `json:"order_id"`
	ReservationID string    `json:"reservation_id,omitempty"`
	Status        string    `json:"status"`
	Reason        string    `json:"reason,omitempty"`
}
