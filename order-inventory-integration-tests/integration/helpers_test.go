//go:build integration

package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var (
	orderBaseURL     = envOr("ORDER_BASE_URL", "http://localhost:18081")
	inventoryBaseURL = envOr("INVENTORY_BASE_URL", "http://localhost:8914")
	postgresURL      = envOr("POSTGRES_URL", "postgres://order_user:order_dummy@localhost:15432/order_db?sslmode=disable")
	mysqlDSN         = envOr("MYSQL_DSN", "root:root@tcp(localhost:13306)/appdb?parseTime=true")
	rabbitMQURL      = envOr("RABBITMQ_URL", "amqp://admin:admin@localhost:15672/")
)

type createOrderRequest struct {
	CustomerID     string `json:"customer_id"`
	IdempotencyKey string `json:"idempotency_key"`
	SKU            string `json:"sku"`
	Quantity       int    `json:"quantity"`
	UnitPrice      int64  `json:"unit_price"`
	Currency       string `json:"currency"`
	TaxAmount      int64  `json:"tax_amount"`
	ShippingAmount int64  `json:"shipping_amount"`
}

type orderResponse struct {
	ID             string `json:"id"`
	CustomerID     string `json:"customer_id"`
	IdempotencyKey string `json:"idempotency_key"`
	SKU            string `json:"sku"`
	Quantity       int    `json:"quantity"`
	UnitPrice      int64  `json:"unit_price"`
	Status         string `json:"status"`
	Subtotal       int64  `json:"subtotal"`
	TotalAmount    int64  `json:"total_amount"`
	Version        int64  `json:"version"`
}

type stockItem struct {
	SKU       string `json:"sku"`
	Available int64  `json:"available"`
	Reserved  int64  `json:"reserved"`
	Version   int64  `json:"version"`
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func openPostgres(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", postgresURL)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func openMySQL(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", mysqlDSN)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func resetState(t *testing.T, available int64) {
	t.Helper()
	pg := openPostgres(t)
	_, err := pg.Exec(`TRUNCATE TABLE order_status_history, outbox_events, orders RESTART IDENTITY CASCADE`)
	require.NoError(t, err)

	my := openMySQL(t)
	_, err = my.Exec(`DELETE FROM reservations`)
	require.NoError(t, err)
	_, err = my.Exec(`DELETE FROM stock_items`)
	require.NoError(t, err)
	_, err = my.Exec(`
		INSERT INTO stock_items (sku, available, reserved, version, created_at, updated_at)
		VALUES ('DEFAULT', ?, 0, 1, NOW(), NOW())`, available)
	require.NoError(t, err)
}

func createOrder(t *testing.T, request createOrderRequest) orderResponse {
	t.Helper()
	body, err := json.Marshal(request)
	require.NoError(t, err)

	response, err := http.Post(orderBaseURL+"/orders/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode, string(responseBody))

	var created orderResponse
	require.NoError(t, json.Unmarshal(responseBody, &created))
	return created
}

func getOrder(t *testing.T, id string) orderResponse {
	t.Helper()
	response, err := http.Get(orderBaseURL + "/orders/" + id)
	require.NoError(t, err)
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode, string(body))

	var order orderResponse
	require.NoError(t, json.Unmarshal(body, &order))
	return order
}

func getStock(t *testing.T) stockItem {
	t.Helper()
	response, err := http.Get(inventoryBaseURL + "/v1/inventory")
	require.NoError(t, err)
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode, string(body))

	var items []stockItem
	require.NoError(t, json.Unmarshal(body, &items))
	for _, item := range items {
		if item.SKU == "DEFAULT" {
			return item
		}
	}
	t.Fatalf("DEFAULT stock not found: %s", string(body))
	return stockItem{}
}

func eventually(t *testing.T, timeout, interval time.Duration, condition func() (bool, string)) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var detail string
	for time.Now().Before(deadline) {
		ok, current := condition()
		detail = current
		if ok {
			return
		}
		time.Sleep(interval)
	}
	t.Fatalf("condition not met within %s; last state: %s", timeout, detail)
}

func waitForHTTP(t *testing.T, endpoint string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		require.NoError(t, err)
		response, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				return
			}
		}
		select {
		case <-ctx.Done():
			t.Fatalf("service not ready: %s: %v", endpoint, ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func reservationCount(t *testing.T, orderID string) int {
	t.Helper()
	db := openMySQL(t)
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM reservations WHERE order_id = ?`, orderID).Scan(&count))
	return count
}

func stateString(order orderResponse, stock stockItem) string {
	return fmt.Sprintf("order_status=%s order_version=%d available=%d reserved=%d stock_version=%d",
		order.Status, order.Version, stock.Available, stock.Reserved, stock.Version)
}

func createOrderWithoutTesting(request createOrderRequest) (orderResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return orderResponse{}, err
	}
	response, err := http.Post(orderBaseURL+"/orders/", "application/json", bytes.NewReader(body))
	if err != nil {
		return orderResponse{}, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return orderResponse{}, err
	}
	if response.StatusCode != http.StatusCreated {
		return orderResponse{}, fmt.Errorf("create order status=%d body=%s", response.StatusCode, string(responseBody))
	}
	var created orderResponse
	if err := json.Unmarshal(responseBody, &created); err != nil {
		return orderResponse{}, err
	}
	return created, nil
}
