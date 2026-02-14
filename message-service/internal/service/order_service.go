package service

import (
	"context"
	"errors"
	"strings"

	"message-service/internal/domain"
	"message-service/internal/repository"
)

type OrderService struct {
	repo repository.OrderRepository
}

func NewOrderService(repo repository.OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) Create(ctx context.Context, o domain.Order) (domain.Order, error) {
	if strings.TrimSpace(o.OrderID) == "" || strings.TrimSpace(o.UserID) == "" {
		return domain.Order{}, errors.New("order_id and user_id are required")
	}
	return s.repo.Create(ctx, o)
}

func (s *OrderService) ListByUser(ctx context.Context, userID string, limit int64) ([]domain.Order, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user_id is required")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.repo.ListByUser(ctx, userID, limit)
}
