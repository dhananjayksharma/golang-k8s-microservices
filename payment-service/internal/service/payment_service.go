package service

import (
	"context"

	"go-gin-mysql-k8s/internal/repository"
)

type PaymentService struct {
	repo repository.PaymentRepository
}

func NewPaymentService(repo repository.PaymentRepository) *PaymentService {
	return &PaymentService{repo: repo}
}

func (s *PaymentService) ListPayments(ctx context.Context) ([]repository.Payment, error) {
	return s.repo.GetPayments(ctx)
}

func (s *PaymentService) GetPayment(ctx context.Context, id string) (*repository.Payment, error) {
	return s.repo.GetPaymentByID(ctx, id)
}
