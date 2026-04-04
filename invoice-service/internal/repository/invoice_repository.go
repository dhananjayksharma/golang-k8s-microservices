package repository

import "context"

type Invoice struct {
	ID            string  `json:"id"`
	OrderId       int64   `json:"order_id"`
	InvoiceNumber string  `json:"invoice_number"`
	TotalAmount   float64 `json:"total_amount"`
	Status        string  `json:"status"`
}

// InvoiceRepository defines data access behavior.
type InvoiceRepository interface {
	GetInvoices(ctx context.Context) ([]Invoice, error)
	GetOrderByID(ctx context.Context, id string) (*Invoice, error)
}
