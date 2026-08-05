# Order Service with PostgreSQL

The API stores orders in PostgreSQL `order_db` using UUID order IDs, idempotency keys, paise-based monetary values, and optimistic versioning.

## Start dependencies

```bash
cd deploy/local
docker compose up -d postgres rabbitmq
cd ../..
```

## Configure

```bash
export ORDER_PORT=8081
export DATABASE_URL='postgres://order_user:order_password@localhost:5432/order_db?sslmode=disable'
```

## Reset an incompatible local schema

The fixed migration uses the UUID production schema. If an older/simple migration has already been applied locally, reset the development volume:

```bash
cd deploy/local
docker compose down -v
docker compose up -d postgres rabbitmq
cd ../..
```

This deletes local development data.

## Run migrations

```bash
migrate -path migrations -database "$DATABASE_URL" up
migrate -path migrations -database "$DATABASE_URL" version
```

Rollback one migration:

```bash
migrate -path migrations -database "$DATABASE_URL" down 1
```

## Start the API

```bash
go mod tidy
go run .
```

Readiness:

```bash
curl -i http://localhost:8081/health/ready
```

## Create an order

All monetary values are in paise. The service calculates `subtotal`, `total_amount`, defaults `currency/status/version`, and generates `request_hash`.

```bash
curl -X POST http://localhost:8081/orders/ \
  -H 'Content-Type: application/json' \
  -d '{
    "customer_id": "22222222-2222-4222-8222-222222222222",
    "idempotency_key": "mouse-order-005",
    "sku": "MOUSE-001",
    "quantity": 2,
    "unit_price": 2225,
    "currency": "INR",
    "tax_amount": 145,
    "shipping_amount": 500
  }'
```

## API endpoints

- `POST /orders/`
- `GET /orders/`
- `GET /orders/:id`
- `PUT /orders/:id`
- `DELETE /orders/:id`
- `GET /health/live`
- `GET /health/ready`

## Tests

```bash
go test ./service -v
go test ./... -cover
```

Coverage report:

```bash
go test ./... -coverpkg=./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Inventory integration
See `docs/SERVICE_COMMUNICATION.md`. Order Service keeps `database/sql` and does not use GORM.
