package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

const (
	orderID    = "11111111-1111-4111-8111-111111111111"
	customerID = "22222222-2222-4222-8222-222222222222"
)

type fakeOrderRepository struct {
	createFn func(context.Context, Order) (Order, error)
	orders   map[string]Order
}

func newFakeOrderRepository() *fakeOrderRepository {
	return &fakeOrderRepository{orders: make(map[string]Order)}
}

func (r *fakeOrderRepository) Create(ctx context.Context, order Order) (Order, error) {
	if r.createFn != nil {
		return r.createFn(ctx, order)
	}
	order.ID = orderID
	order.CreatedAt = time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)
	order.UpdatedAt = order.CreatedAt
	r.orders[order.ID] = order
	return order, nil
}

func (r *fakeOrderRepository) GetAll(context.Context) ([]Order, error) {
	orders := make([]Order, 0, len(r.orders))
	for _, order := range r.orders {
		orders = append(orders, order)
	}
	return orders, nil
}

func (r *fakeOrderRepository) GetByID(_ context.Context, id string) (Order, error) {
	order, ok := r.orders[id]
	if !ok {
		return Order{}, ErrOrderNotFound
	}
	return order, nil
}

func (r *fakeOrderRepository) Update(_ context.Context, id string, update Order) (Order, error) {
	current, ok := r.orders[id]
	if !ok {
		return Order{}, ErrOrderNotFound
	}
	update.ID = id
	update.CreatedAt = current.CreatedAt
	update.UpdatedAt = current.UpdatedAt.Add(time.Minute)
	update.Version = current.Version + 1
	r.orders[id] = update
	return update, nil
}

func (r *fakeOrderRepository) Delete(_ context.Context, id string) error {
	if _, ok := r.orders[id]; !ok {
		return ErrOrderNotFound
	}
	delete(r.orders, id)
	return nil
}

func validOrder() Order {
	return Order{
		CustomerID:     customerID,
		IdempotencyKey: "create-order-001",
		Quantity:       2,
		UnitPrice:      149900,
		Currency:       "inr",
		TaxAmount:      53964,
		ShippingAmount: 5000,
	}
}

func TestOrderService_Create_Success(t *testing.T) {
	repo := newFakeOrderRepository()
	svc := NewOrderService(repo)

	created, err := svc.Create(context.Background(), validOrder())
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	if created.ID != orderID {
		t.Fatalf("Create() ID = %q, want %q", created.ID, orderID)
	}
	if created.Currency != "INR" || created.Status != "PENDING" || created.Version != 1 {
		t.Fatalf("Create() defaults not applied: %+v", created)
	}
	if created.Subtotal != 299800 {
		t.Errorf("Create() subtotal = %d, want 299800", created.Subtotal)
	}
	if created.TotalAmount != 358764 {
		t.Errorf("Create() total = %d, want 358764", created.TotalAmount)
	}
	if len(created.RequestHash) != 64 {
		t.Errorf("Create() request hash length = %d, want 64", len(created.RequestHash))
	}
}

func TestOrderService_Create_ForwardsNormalizedOrder(t *testing.T) {
	var received Order
	repo := newFakeOrderRepository()
	repo.createFn = func(_ context.Context, order Order) (Order, error) {
		received = order
		order.ID = orderID
		return order, nil
	}

	svc := NewOrderService(repo)
	_, err := svc.Create(context.Background(), validOrder())
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	if received.Currency != "INR" || received.Status != "PENDING" {
		t.Fatalf("repository received unnormalized order: %+v", received)
	}
	if received.Subtotal != 299800 || received.TotalAmount != 358764 {
		t.Fatalf("repository received incorrect totals: %+v", received)
	}
}

func TestOrderService_Create_RepositoryError(t *testing.T) {
	wantErr := errors.New("insert failed")
	repo := newFakeOrderRepository()
	repo.createFn = func(context.Context, Order) (Order, error) {
		return Order{}, wantErr
	}

	svc := NewOrderService(repo)
	got, err := svc.Create(context.Background(), validOrder())
	if !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want %v", err, wantErr)
	}
	if got != (Order{}) {
		t.Fatalf("Create() = %+v, want empty Order", got)
	}
}

func TestOrderService_Create_Validation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Order)
	}{
		{name: "invalid customer UUID", mutate: func(o *Order) { o.CustomerID = "bad-id" }},
		{name: "missing idempotency key", mutate: func(o *Order) { o.IdempotencyKey = "" }},
		{name: "zero quantity", mutate: func(o *Order) { o.Quantity = 0 }},
		{name: "zero unit price", mutate: func(o *Order) { o.UnitPrice = 0 }},
		{name: "negative tax", mutate: func(o *Order) { o.TaxAmount = -1 }},
		{name: "invalid currency", mutate: func(o *Order) { o.Currency = "RUPEE" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := validOrder()
			tt.mutate(&order)

			repositoryCalled := false
			repo := newFakeOrderRepository()
			repo.createFn = func(context.Context, Order) (Order, error) {
				repositoryCalled = true
				return Order{}, nil
			}

			svc := NewOrderService(repo)
			got, err := svc.Create(context.Background(), order)
			if !errors.Is(err, ErrInvalidOrder) {
				t.Fatalf("Create() error = %v, want ErrInvalidOrder", err)
			}
			if got != (Order{}) {
				t.Fatalf("Create() = %+v, want empty Order", got)
			}
			if repositoryCalled {
				t.Fatal("repository.Create() called for invalid order")
			}
		})
	}
}

func TestOrderService_CRUD(t *testing.T) {
	repo := newFakeOrderRepository()
	svc := NewOrderService(repo)
	ctx := context.Background()

	created, err := svc.Create(ctx, validOrder())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := svc.GetByID(ctx, created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("GetByID() = %+v, %v; want %+v", got, err, created)
	}

	update := validOrder()
	update.Quantity = 3
	update.Status = "CONFIRMED"
	updated, err := svc.Update(ctx, created.ID, update)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.ID != created.ID || updated.Quantity != 3 || updated.Version != 2 {
		t.Fatalf("Update() = %+v", updated)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := svc.GetByID(ctx, created.ID); !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("GetByID(deleted) error = %v, want ErrOrderNotFound", err)
	}
}
