package repository

import (
	"context"
)

// DummyPaymentRepository is a temporary in-memory implementation.
// This will be replaced with DB-backed implementation later.
type DummyPaymentRepository struct {
	data []Payment
}

func NewDummyPaymentRepository() *DummyPaymentRepository {
	return &DummyPaymentRepository{
		data: []Payment{
			{
				ID:     "pay_001",
				Amount: 499.99,
				Status: "SUCCESS",
			},
			{
				ID:     "pay_002",
				Amount: 199.00,
				Status: "PENDING",
			},
		},
	}
}

func (r *DummyPaymentRepository) GetPayments(ctx context.Context) ([]Payment, error) {
	return r.data, nil
}

func (r *DummyPaymentRepository) GetPaymentByID(ctx context.Context, id string) (*Payment, error) {
	for _, p := range r.data {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, nil
}
