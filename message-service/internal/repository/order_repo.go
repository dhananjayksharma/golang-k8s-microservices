package repository

import (
	"context"

	"message-service/internal/domain"
)

type OrderRepository interface {
	Create(ctx context.Context, o domain.Order) (domain.Order, error)
	ListByUser(ctx context.Context, userID string, limit int64) ([]domain.Order, error)
}
