package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	ErrOrderNotFound = errors.New("order not found")
	ErrInvalidOrder  = errors.New("invalid order")
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

type Order struct {
	ID             string    `json:"id"`
	CustomerID     string    `json:"customer_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	SKU            string    `json:"sku"`
	RequestHash    string    `json:"request_hash"`
	Quantity       int       `json:"quantity"`
	UnitPrice      int64     `json:"unit_price"`
	Status         string    `json:"status"`
	Currency       string    `json:"currency"`
	Subtotal       int64     `json:"subtotal"`
	TaxAmount      int64     `json:"tax_amount"`
	ShippingAmount int64     `json:"shipping_amount"`
	TotalAmount    int64     `json:"total_amount"`
	Version        int64     `json:"version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type OrderRepository interface {
	Create(ctx context.Context, order Order) (Order, error)
	GetAll(ctx context.Context) ([]Order, error)
	GetByID(ctx context.Context, id string) (Order, error)
	Update(ctx context.Context, id string, order Order) (Order, error)
	Delete(ctx context.Context, id string) error
}

type OrderService interface {
	Create(ctx context.Context, order Order) (Order, error)
	GetAll(ctx context.Context) ([]Order, error)
	GetByID(ctx context.Context, id string) (Order, error)
	Update(ctx context.Context, id string, order Order) (Order, error)
	Delete(ctx context.Context, id string) error
}

type orderService struct {
	repository OrderRepository
}

func NewOrderService(repository OrderRepository) OrderService {
	return &orderService{repository: repository}
}

func (s *orderService) Create(ctx context.Context, order Order) (Order, error) {
	order = normalizeForCreate(order)
	if err := validateOrder(order); err != nil {
		return Order{}, err
	}
	return s.repository.Create(ctx, order)
}

func (s *orderService) GetAll(ctx context.Context) ([]Order, error) {
	return s.repository.GetAll(ctx)
}

func (s *orderService) GetByID(ctx context.Context, id string) (Order, error) {
	if !uuidPattern.MatchString(id) {
		return Order{}, ErrOrderNotFound
	}
	return s.repository.GetByID(ctx, id)
}

func (s *orderService) Update(ctx context.Context, id string, order Order) (Order, error) {
	if !uuidPattern.MatchString(id) {
		return Order{}, ErrOrderNotFound
	}
	order = normalizeForUpdate(order)
	if err := validateOrder(order); err != nil {
		return Order{}, err
	}
	return s.repository.Update(ctx, id, order)
}

func (s *orderService) Delete(ctx context.Context, id string) error {
	if !uuidPattern.MatchString(id) {
		return ErrOrderNotFound
	}
	return s.repository.Delete(ctx, id)
}

func normalizeForCreate(order Order) Order {
	order.Currency = strings.ToUpper(strings.TrimSpace(order.Currency))
	if order.Currency == "" {
		order.Currency = "INR"
	}
	order.Status = strings.ToUpper(strings.TrimSpace(order.Status))
	if order.Status == "" {
		order.Status = "PENDING"
	}
	order.Subtotal = int64(order.Quantity) * order.UnitPrice
	order.TotalAmount = order.Subtotal + order.TaxAmount + order.ShippingAmount
	if order.Version == 0 {
		order.Version = 1
	}
	if order.RequestHash == "" {
		sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%d|%d|%d|%d|%s",
			order.CustomerID,
			order.IdempotencyKey,
			order.SKU,
			order.Quantity,
			order.UnitPrice,
			order.TaxAmount,
			order.ShippingAmount,
			order.Currency,
		)))
		order.RequestHash = hex.EncodeToString(sum[:])
	}
	return order
}

func normalizeForUpdate(order Order) Order {
	return normalizeForCreate(order)
}

func validateOrder(order Order) error {
	if !uuidPattern.MatchString(order.CustomerID) {
		return fmt.Errorf("%w: customer_id must be a valid UUID", ErrInvalidOrder)
	}
	if strings.TrimSpace(order.IdempotencyKey) == "" || len(order.IdempotencyKey) > 128 {
		return fmt.Errorf("%w: idempotency_key is required and must be at most 128 characters", ErrInvalidOrder)
	}
	if len(order.RequestHash) != 64 {
		return fmt.Errorf("%w: request_hash must be a 64-character SHA-256 value", ErrInvalidOrder)
	}
	if order.Quantity <= 0 {
		return fmt.Errorf("%w: quantity must be greater than zero", ErrInvalidOrder)
	}
	if order.UnitPrice <= 0 {
		return fmt.Errorf("%w: unit_price must be greater than zero", ErrInvalidOrder)
	}
	if order.TaxAmount < 0 || order.ShippingAmount < 0 {
		return fmt.Errorf("%w: tax_amount and shipping_amount cannot be negative", ErrInvalidOrder)
	}
	if len(order.Currency) != 3 || order.Currency != strings.ToUpper(order.Currency) {
		return fmt.Errorf("%w: currency must be a three-letter uppercase code", ErrInvalidOrder)
	}
	if order.Subtotal != int64(order.Quantity)*order.UnitPrice {
		return fmt.Errorf("%w: subtotal is inconsistent", ErrInvalidOrder)
	}
	if order.TotalAmount != order.Subtotal+order.TaxAmount+order.ShippingAmount {
		return fmt.Errorf("%w: total_amount is inconsistent", ErrInvalidOrder)
	}
	return nil
}
