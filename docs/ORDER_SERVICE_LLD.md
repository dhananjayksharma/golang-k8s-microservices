# Order Service — Low-Level Design (LLD)

## 1. Target package structure

```text
order-service/
├── cmd/
│   ├── api/main.go
│   └── outbox-worker/main.go
├── internal/
│   ├── config/config.go
│   ├── domain/order.go
│   ├── domain/status.go
│   ├── handler/order_handler.go
│   ├── middleware/idempotency.go
│   ├── middleware/request_id.go
│   ├── repository/order_repository.go
│   ├── repository/postgres_order_repository.go
│   ├── service/order_service.go
│   ├── messaging/rabbitmq.go
│   ├── outbox/publisher.go
│   └── observability/metrics.go
├── migrations/
├── loadtest/
├── deploy/
└── scripts/
```

## 2. Domain model

```go
type Order struct {
    ID             uuid.UUID
    CustomerID     uuid.UUID
    Status         OrderStatus
    Currency       string
    Subtotal       int64 // minor units, e.g. paise
    TaxAmount      int64
    ShippingAmount int64
    TotalAmount    int64
    Version        int64
    Items          []OrderItem
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type OrderItem struct {
    ID         uuid.UUID
    OrderID    uuid.UUID
    ProductID  uuid.UUID
    Quantity   int
    UnitPrice  int64
    TotalPrice int64
}
```

Use integer minor currency units instead of `float64` to avoid rounding errors.

## 3. Interfaces

```go
type OrderRepository interface {
    Create(ctx context.Context, order *domain.Order, event outbox.Event) error
    GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error)
    List(ctx context.Context, filter ListFilter) ([]domain.Order, string, error)
    UpdateStatus(ctx context.Context, id uuid.UUID, fromVersion int64, to domain.OrderStatus, reason string, event outbox.Event) error
}

type IdempotencyStore interface {
    Get(ctx context.Context, key string) (*StoredResponse, error)
    Begin(ctx context.Context, key, requestHash string, ttl time.Duration) (bool, error)
    Complete(ctx context.Context, key string, response StoredResponse, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}

type EventPublisher interface {
    Publish(ctx context.Context, routingKey string, body []byte, messageID string) error
}
```

## 4. API contract

### POST `/api/v1/orders`

Required headers:

- `Content-Type: application/json`
- `Idempotency-Key: <unique-client-key>`

Request:

```json
{
  "customer_id": "27e54cb7-16a4-4c15-91e6-7acfab52f8c0",
  "currency": "INR",
  "items": [
    {
      "product_id": "ad835d93-f8bb-40d1-b5ee-e73b226fac43",
      "quantity": 2,
      "unit_price": 149900
    }
  ]
}
```

Response: `201 Created`

### GET `/api/v1/orders/{id}`

Response: `200`, `400`, or `404`.

### GET `/api/v1/orders?customer_id=&status=&cursor=&limit=50`

Use cursor pagination rather than unbounded list retrieval.

### PATCH `/api/v1/orders/{id}/status`

```json
{
  "status": "CANCELLED",
  "reason": "customer requested cancellation",
  "version": 3
}
```

Return `409 Conflict` for stale versions or invalid transitions.

## 5. Validation rules

- Customer and product IDs must be valid UUIDs.
- At least one item is required.
- Maximum 100 items per order.
- Quantity must be 1–1000.
- Price must be non-negative.
- Currency must be a supported ISO-4217 code.
- Total is calculated server-side.
- Request body should have a bounded size, for example 1 MiB.

## 6. Create-order algorithm

```text
1. Validate Idempotency-Key.
2. Decode and validate request.
3. Calculate canonical request hash.
4. Check Redis idempotency entry.
5. If same key + same hash is complete, replay response.
6. If same key + different hash, return 409.
7. Mark key as processing using SET NX with TTL.
8. Calculate totals using integer minor units.
9. Begin PostgreSQL transaction.
10. Insert order.
11. Insert order items in a batch.
12. Insert status-history row.
13. Insert order.created.v1 outbox row.
14. Commit.
15. Store HTTP response in Redis.
16. Return 201.
```

## 7. Concurrency control

The current `len(slice)+1` ID creation is unsafe and can reuse IDs after deletion. PostgreSQL-generated UUIDs remove this issue.

Status updates use optimistic locking:

```sql
UPDATE orders
SET status = $1,
    version = version + 1,
    updated_at = now()
WHERE id = $2
  AND version = $3;
```

Zero affected rows means stale version or missing order.

## 8. Transaction boundaries

A create transaction contains:

- one `orders` insert;
- one batch `order_items` insert;
- one `order_status_history` insert;
- one `outbox_events` insert.

No external HTTP or RabbitMQ call may occur inside the transaction.

## 9. Outbox publisher

Worker behavior:

1. Select pending rows using `FOR UPDATE SKIP LOCKED`.
2. Mark a batch as processing or hold row locks in a short transaction.
3. Publish persistent RabbitMQ messages with message ID equal to outbox ID.
4. Mark successfully published records.
5. Increment retry count and save last error on failure.
6. Move to `DEAD` after configured maximum retries.

Suggested batch size: 100. Suggested polling interval: 250 ms at this traffic level.

## 10. Error model

```json
{
  "error": {
    "code": "ORDER_NOT_FOUND",
    "message": "order was not found",
    "request_id": "01J..."
  }
}
```

Recommended mappings:

| Condition | HTTP |
|---|---:|
| Invalid input | 400 |
| Missing authentication | 401 |
| Forbidden operation | 403 |
| Not found | 404 |
| Idempotency mismatch / stale version | 409 |
| Rate limit | 429 |
| Dependency unavailable | 503 |
| Unexpected failure | 500 |

## 11. Configuration

| Variable | Example |
|---|---|
| `ORDER_PORT` | `8081` |
| `DATABASE_URL` | `postgres://order:order@postgres:5432/orderdb?sslmode=disable` |
| `REDIS_ADDR` | `redis:6379` |
| `RABBITMQ_URL` | `amqp://admin:admin@rabbitmq:5672/` |
| `RABBITMQ_EXCHANGE` | `orders.events` |
| `DB_MAX_OPEN_CONNS` | `15` |
| `DB_MAX_IDLE_CONNS` | `5` |
| `REQUEST_TIMEOUT` | `3s` |
| `IDEMPOTENCY_TTL` | `24h` |

Configuration must be validated at startup.

## 12. HTTP server design

Use `http.Server`, not `gin.Engine.Run`, to configure timeouts and graceful shutdown:

```go
srv := &http.Server{
    Addr:              ":8081",
    Handler:           router,
    ReadHeaderTimeout: 2 * time.Second,
    ReadTimeout:       5 * time.Second,
    WriteTimeout:      10 * time.Second,
    IdleTimeout:       60 * time.Second,
}
```

## 13. Database pool

Starting values per pod:

```go
db.SetMaxOpenConns(15)
db.SetMaxIdleConns(5)
db.SetConnMaxIdleTime(5 * time.Minute)
db.SetConnMaxLifetime(30 * time.Minute)
```

With two pods, the application can use up to 30 connections. Reserve additional PostgreSQL capacity for migrations, monitoring, and administration.

## 14. Testing design

### Unit tests

- total calculation
- transition validation
- handler validation
- idempotency conflict behavior
- service error mappings

### Integration tests

- repository CRUD with PostgreSQL
- rollback when an item insert fails
- outbox row created atomically
- optimistic-lock conflict
- Redis idempotency replay
- RabbitMQ publisher retry

### Load tests

- constant 10 RPS for 10 minutes
- 20-RPS burst for 1 minute
- duplicate idempotency-key scenario
- read-heavy scenario
- pod restart during traffic

## 15. Current-code changes in priority order

1. Replace in-memory slice with repository abstraction.
2. Add PostgreSQL migrations and repository implementation.
3. Return `201` from create and validate path IDs.
4. Require idempotency key.
5. Move RabbitMQ publication to transactional outbox worker.
6. Add health, readiness, metrics, and structured logs.
7. Add Dockerfile and Kubernetes resources.
8. Add automated tests and k6 acceptance test.
