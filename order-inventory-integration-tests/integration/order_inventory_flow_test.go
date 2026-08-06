//go:build integration

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestOrderInventorySuccessfulFlow(t *testing.T) {
	resetState(t, 100)

	request := createOrderRequest{
		CustomerID:     "22222222-2222-4222-8222-222222222222",
		IdempotencyKey: "it-success-" + uuid.NewString(),
		Quantity:       2,
		SKU:            "MOUSE-001",
		UnitPrice:      149900,
		Currency:       "INR",
		TaxAmount:      1000,
		ShippingAmount: 5000,
	}

	created := createOrder(t, request)
	require.Equal(t, "PENDING", created.Status)
	require.Equal(t, int64(299800), created.Subtotal)
	require.Equal(t, int64(305800), created.TotalAmount)

	eventually(t, 15*time.Second, 250*time.Millisecond, func() (bool, string) {
		current := getOrder(t, created.ID)
		stock := getStock(t)
		return current.Status == "CONFIRMED" && stock.Available == 98 && stock.Reserved == 2,
			stateString(current, stock)
	})

	require.Equal(t, 1, reservationCount(t, created.ID))
}

func TestOrderInventoryInsufficientStock(t *testing.T) {
	resetState(t, 1)

	request := createOrderRequest{
		CustomerID:     "22222222-2222-4222-8222-222222222222",
		IdempotencyKey: "it-rejected-" + uuid.NewString(),
		Quantity:       5,
		UnitPrice:      5000,
		Currency:       "INR",
	}

	created := createOrder(t, request)
	require.Equal(t, "PENDING", created.Status)

	eventually(t, 15*time.Second, 250*time.Millisecond, func() (bool, string) {
		current := getOrder(t, created.ID)
		stock := getStock(t)
		return current.Status == "CANCELLED" && stock.Available == 1 && stock.Reserved == 0,
			stateString(current, stock)
	})

	require.Equal(t, 0, reservationCount(t, created.ID))
}

func TestConcurrentOrdersDoNotOversell(t *testing.T) {
	resetState(t, 10)

	const requests = 20
	results := make(chan orderResponse, requests)
	errors := make(chan error, requests)

	for i := 0; i < requests; i++ {
		go func(index int) {
			defer func() {
				if recovered := recover(); recovered != nil {
					errors <- fmt.Errorf("request %d panic: %v", index, recovered)
				}
			}()
			request := createOrderRequest{
				CustomerID:     "22222222-2222-4222-8222-222222222222",
				IdempotencyKey: fmt.Sprintf("it-concurrent-%d-%s", index, uuid.NewString()),
				Quantity:       1,
				UnitPrice:      1000,
				Currency:       "INR",
			}
			// createOrder uses testing helpers and must not be called from a goroutine.
			created, err := createOrderWithoutTesting(request)
			if err != nil {
				errors <- err
				return
			}
			results <- created
		}(i)
	}

	orders := make([]orderResponse, 0, requests)
	for i := 0; i < requests; i++ {
		select {
		case err := <-errors:
			require.NoError(t, err)
		case order := <-results:
			orders = append(orders, order)
		case <-time.After(20 * time.Second):
			t.Fatal("timed out creating concurrent orders")
		}
	}

	eventually(t, 30*time.Second, 500*time.Millisecond, func() (bool, string) {
		confirmed := 0
		cancelled := 0
		for _, created := range orders {
			current := getOrder(t, created.ID)
			switch current.Status {
			case "CONFIRMED":
				confirmed++
			case "CANCELLED":
				cancelled++
			}
		}
		stock := getStock(t)
		complete := confirmed+cancelled == requests
		valid := stock.Available >= 0 && stock.Reserved <= 10 && confirmed <= 10
		return complete && valid,
			fmt.Sprintf("confirmed=%d cancelled=%d available=%d reserved=%d", confirmed, cancelled, stock.Available, stock.Reserved)
	})

	stock := getStock(t)
	require.Equal(t, int64(0), stock.Available)
	require.Equal(t, int64(10), stock.Reserved)
}
