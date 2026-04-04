package handlers

type CreateInvoiceRequest struct {
	ID            uint64  `json:"id" binding:"required"`
	OrderId       int64   `json:"order_id" binding:"required"`
	InvoiceNumber string  `json:"invoice_number" binding:"required"`
	TotalAmount   float64 `json:"total_amount" binding:"required,gt=0"`
	Status        float64 `json:"status"`
}

type UpdateInvoiceRequest struct {
	InvoiceNumber *string `json:"invoice_number" binding:"required"`
	TotalAmount   float64 `json:"total_amount" binding:"required,gt=0"`
	Status        float64 `json:"status"`
}
