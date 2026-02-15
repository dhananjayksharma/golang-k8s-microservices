package service

import (
	"context"
	"errors"
	"testing"

	"golang-k8s-microservices/invoice-service/internal/repository"
)

type fakeInvoiceRepo struct {
	getInvoicesFn    func(ctx context.Context) ([]repository.Invoice, error)
	getInvoicesByIDFn func(ctx context.Context, id string) (*repository.Invoice, error)
}

func (f *fakeInvoiceRepo) GetInvoices(ctx context.Context) ([]repository.Invoice, error) {
	if f.getInvoicesFn == nil {
		return nil, nil
	}
	return f.getInvoicesFn(ctx)
}

func (f *fakeInvoiceRepo) GetInvoicesByID(ctx context.Context, id string) (*repository.Invoice, error) {
	if f.getInvoicesByIDFn == nil {
		return nil, nil
	}
	return f.getInvoicesByIDFn(ctx, id)
}

func TestNewInvoiceService(t *testing.T) {
	repo := &fakeInvoiceRepo{}
	svc := NewInvoiceService(repo)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.repo != repo {
		t.Fatal("expected repository to be assigned in service")
	}
}

func TestListInvoices(t *testing.T) {
	t.Run("returns repository data", func(t *testing.T) {
		expected := []repository.Invoice{
			{ID: "inv-1", Amount: 10.5, Status: "PAID"},
			{ID: "inv-2", Amount: 15, Status: "PENDING"},
		}

		var gotCtx context.Context
		repo := &fakeInvoiceRepo{
			getInvoicesFn: func(ctx context.Context) ([]repository.Invoice, error) {
				gotCtx = ctx
				return expected, nil
			},
		}
		svc := NewInvoiceService(repo)

		ctx := context.WithValue(context.Background(), "trace_id", "abc123")
		got, err := svc.ListInvoices(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotCtx != ctx {
			t.Fatal("expected context to be passed through to repository")
		}
		if len(got) != len(expected) {
			t.Fatalf("expected %d invoices, got %d", len(expected), len(got))
		}
		for i := range expected {
			if got[i] != expected[i] {
				t.Fatalf("expected %v at index %d, got %v", expected[i], i, got[i])
			}
		}
	})

	t.Run("returns repository error", func(t *testing.T) {
		expectedErr := errors.New("db unavailable")
		repo := &fakeInvoiceRepo{
			getInvoicesFn: func(ctx context.Context) ([]repository.Invoice, error) {
				return nil, expectedErr
			},
		}
		svc := NewInvoiceService(repo)

		got, err := svc.ListInvoices(context.Background())
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected error %v, got %v", expectedErr, err)
		}
		if got != nil {
			t.Fatalf("expected nil invoices on error, got %v", got)
		}
	})
}

func TestGetInvoice(t *testing.T) {
	t.Run("returns repository invoice by id", func(t *testing.T) {
		expected := &repository.Invoice{ID: "inv-7", Amount: 99.99, Status: "PAID"}

		var gotCtx context.Context
		var gotID string
		repo := &fakeInvoiceRepo{
			getInvoicesByIDFn: func(ctx context.Context, id string) (*repository.Invoice, error) {
				gotCtx = ctx
				gotID = id
				return expected, nil
			},
		}
		svc := NewInvoiceService(repo)

		ctx := context.WithValue(context.Background(), "request_id", "req-1")
		got, err := svc.GetInvoice(ctx, "inv-7")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotCtx != ctx {
			t.Fatal("expected context to be passed through to repository")
		}
		if gotID != "inv-7" {
			t.Fatalf("expected id inv-7, got %s", gotID)
		}
		if got != expected {
			t.Fatalf("expected pointer %p, got %p", expected, got)
		}
	})

	t.Run("returns repository error", func(t *testing.T) {
		expectedErr := errors.New("invoice not found")
		repo := &fakeInvoiceRepo{
			getInvoicesByIDFn: func(ctx context.Context, id string) (*repository.Invoice, error) {
				return nil, expectedErr
			},
		}
		svc := NewInvoiceService(repo)

		got, err := svc.GetInvoice(context.Background(), "missing")
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected error %v, got %v", expectedErr, err)
		}
		if got != nil {
			t.Fatalf("expected nil invoice on error, got %v", got)
		}
	})
}
