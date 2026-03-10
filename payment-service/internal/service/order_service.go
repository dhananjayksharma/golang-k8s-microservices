package service

import (
	"context"

	"github.com/dhananjayksharma/golang-k8s-microservices/payment-service/internal/repository"
)

type OrderService struct {
	repo repository.OrderRepository
}

func NewOrderService(repo repository.OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) ListOrders(ctx context.Context) ([]repository.Order, error) {
	return s.repo.GetOrders(ctx)
}

func (s *OrderService) GetOrder(ctx context.Context, id string) (*repository.Order, error) {
	return s.repo.GetOrderByID(ctx, id)
}
