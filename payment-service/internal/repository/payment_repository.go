package repository

import "context"

type Payment struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"`
}

// PaymentRepository defines data access behavior.
// Currently implemented using in-memory dummy data.
type PaymentRepository interface {
	GetPayments(ctx context.Context) ([]Payment, error)
	GetPaymentByID(ctx context.Context, id string) (*Payment, error)
}
