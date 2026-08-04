package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"order-service/service"
)

type PostgresOrderRepository struct {
	db *sql.DB
}

func NewPostgresOrderRepository(db *sql.DB) *PostgresOrderRepository {
	return &PostgresOrderRepository{db: db}
}

func (r *PostgresOrderRepository) Create(ctx context.Context, order service.Order) (service.Order, error) {
	const query = `
		INSERT INTO orders (
			customer_id, idempotency_key, request_hash, quantity, unit_price,
			status, currency, subtotal, tax_amount, shipping_amount, total_amount, version
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, customer_id, idempotency_key, request_hash, quantity, unit_price,
			status::text, currency, subtotal, tax_amount, shipping_amount, total_amount,
			version, created_at, updated_at`

	var created service.Order
	if err := r.db.QueryRowContext(ctx, query,
		order.CustomerID,
		order.IdempotencyKey,
		order.RequestHash,
		order.Quantity,
		order.UnitPrice,
		order.Status,
		order.Currency,
		order.Subtotal,
		order.TaxAmount,
		order.ShippingAmount,
		order.TotalAmount,
		order.Version,
	).Scan(
		&created.ID,
		&created.CustomerID,
		&created.IdempotencyKey,
		&created.RequestHash,
		&created.Quantity,
		&created.UnitPrice,
		&created.Status,
		&created.Currency,
		&created.Subtotal,
		&created.TaxAmount,
		&created.ShippingAmount,
		&created.TotalAmount,
		&created.Version,
		&created.CreatedAt,
		&created.UpdatedAt,
	); err != nil {
		return service.Order{}, fmt.Errorf("insert order: %w", err)
	}
	return created, nil
}

func (r *PostgresOrderRepository) GetAll(ctx context.Context) ([]service.Order, error) {
	const query = `
		SELECT id, customer_id, idempotency_key, request_hash, quantity, unit_price,
			status::text, currency, subtotal, tax_amount, shipping_amount, total_amount,
			version, created_at, updated_at
		FROM orders
		ORDER BY created_at DESC, id DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	orders := make([]service.Order, 0)
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders: %w", err)
	}
	return orders, nil
}

func (r *PostgresOrderRepository) GetByID(ctx context.Context, id string) (service.Order, error) {
	const query = `
		SELECT id, customer_id, idempotency_key, request_hash, quantity, unit_price,
			status::text, currency, subtotal, tax_amount, shipping_amount, total_amount,
			version, created_at, updated_at
		FROM orders
		WHERE id = $1::uuid`

	order, err := scanOrder(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.Order{}, service.ErrOrderNotFound
		}
		return service.Order{}, fmt.Errorf("get order %s: %w", id, err)
	}
	return order, nil
}

func (r *PostgresOrderRepository) Update(ctx context.Context, id string, order service.Order) (service.Order, error) {
	const query = `
		UPDATE orders
		SET quantity = $2,
			unit_price = $3,
			status = $4,
			currency = $5,
			subtotal = $6,
			tax_amount = $7,
			shipping_amount = $8,
			total_amount = $9,
			version = version + 1
		WHERE id = $1::uuid
		RETURNING id, customer_id, idempotency_key, request_hash, quantity, unit_price,
			status::text, currency, subtotal, tax_amount, shipping_amount, total_amount,
			version, created_at, updated_at`

	updated, err := scanOrder(r.db.QueryRowContext(ctx, query,
		id,
		order.Quantity,
		order.UnitPrice,
		order.Status,
		order.Currency,
		order.Subtotal,
		order.TaxAmount,
		order.ShippingAmount,
		order.TotalAmount,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.Order{}, service.ErrOrderNotFound
		}
		return service.Order{}, fmt.Errorf("update order %s: %w", id, err)
	}
	return updated, nil
}

func (r *PostgresOrderRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM orders WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete order %s: %w", id, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted rows: %w", err)
	}
	if rowsAffected == 0 {
		return service.ErrOrderNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOrder(row rowScanner) (service.Order, error) {
	var order service.Order
	err := row.Scan(
		&order.ID,
		&order.CustomerID,
		&order.IdempotencyKey,
		&order.RequestHash,
		&order.Quantity,
		&order.UnitPrice,
		&order.Status,
		&order.Currency,
		&order.Subtotal,
		&order.TaxAmount,
		&order.ShippingAmount,
		&order.TotalAmount,
		&order.Version,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	return order, err
}
