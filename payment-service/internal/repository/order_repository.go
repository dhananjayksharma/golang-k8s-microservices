package repository

import "context"

type Order struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"`
}

// OrderRepository defines data access behavior.
// Currently implemented using in-memory dummy data.
type OrderRepository interface {
	GetOrders(ctx context.Context) ([]Order, error)
	GetOrderByID(ctx context.Context, id string) (*Order, error)
}
