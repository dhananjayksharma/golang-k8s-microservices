package service

import (
	"context"

	"github.com/dhananjayksharma/golang-k8s-microservices/invoice-service/internal/repository"
)

type InvoiceService struct {
	repo repository.InvoiceRepository
}

func NewInvoiceService(repo repository.InvoiceRepository) *InvoiceService {
	return &InvoiceService{repo: repo}
}

func (s *InvoiceService) ListInvoices(ctx context.Context) ([]repository.Invoice, error) {
	return s.repo.GetInvoices(ctx)
}

func (s *InvoiceService) GetInvoice(ctx context.Context, id string) (*repository.Invoice, error) {
	return s.repo.GetInvoicesByID(ctx, id)
}
