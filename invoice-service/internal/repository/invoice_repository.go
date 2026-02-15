package repository

import "context"

type Invoice struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"`
}

// PaymentRepository defines data access behavior.
// Currently implemented using in-memory dummy data.
type InvoiceRepository interface {
	GetInvoices(ctx context.Context) ([]Invoice, error)
	GetInvoicesByID(ctx context.Context, id string) (*Invoice, error)
}
