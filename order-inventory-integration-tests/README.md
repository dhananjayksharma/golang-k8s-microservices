# Order + Inventory Integration Tests

This package validates the asynchronous workflow across:

- Order Service
- PostgreSQL
- RabbitMQ
- Inventory Service
- MySQL/GORM
- Redis

Place this folder beside both services:

```text
workspace/
├── order-service/
├── inventory-service/
└── order-inventory-integration-tests/
```

## Test scenarios

1. Service readiness
2. Successful order and inventory reservation
3. Insufficient stock and order cancellation
4. Duplicate `order.created` event reserves stock once
5. Twenty concurrent orders cannot oversell ten stock units

## Prerequisites

- Go 1.24+
- Docker and Docker Compose
- `curl`
- Order Service and Inventory Service source folders

## 1. Start infrastructure

```bash
cd order-inventory-integration-tests
./scripts/up.sh
```

Integration ports:

| Dependency | Port |
|---|---:|
| PostgreSQL | 15432 |
| MySQL | 13306 |
| Redis | 16379 |
| RabbitMQ AMQP | 15672 |
| RabbitMQ UI | 25672 |

RabbitMQ UI credentials: `admin/admin`.

## 2. Run Order Service migrations

```bash
./scripts/migrate-order.sh
```

When the service directories are elsewhere:

```bash
ORDER_SERVICE_DIR=/path/to/order-service ./scripts/migrate-order.sh
```

## 3. Start both services

```bash
./scripts/run-services.sh
```

The script uses:

- Order Service: `http://localhost:18081`
- Inventory Service: `http://localhost:8914`

Logs are written under `.logs/`.

Custom directories:

```bash
ORDER_SERVICE_DIR=/path/to/order-service \
INVENTORY_SERVICE_DIR=/path/to/inventory-service \
./scripts/run-services.sh
```

## 4. Reset and seed test data

Run this after Inventory Service starts, because GORM creates `stock_items` and `reservations`:

```bash
./scripts/reset-data.sh
```

## 5. Run all integration tests

```bash
./scripts/test.sh
```

Equivalent command:

```bash
go test -tags=integration ./integration/... -count=1 -p=1 -v
```

`-p=1` is intentional because the tests share the `DEFAULT` stock row.

Run one scenario:

```bash
go test -tags=integration ./integration/... \
  -run TestOrderInventorySuccessfulFlow \
  -count=1 -v
```

## 6. Stop services and infrastructure

```bash
./scripts/stop-services.sh
./scripts/down.sh
```

`down.sh` removes integration database volumes.

## Expected results

Successful flow:

```text
Order: PENDING -> CONFIRMED
Stock: available 100 -> 98
Stock: reserved 0 -> 2
Reservations: 1
```

Insufficient stock:

```text
Order: PENDING -> CANCELLED
Stock unchanged
Reservations: 0
```

Duplicate event:

```text
Two identical order.created events
One reservation
One stock reduction
```

Concurrent flow:

```text
Initial stock: 10
Requests: 20 x quantity 1
Confirmed: no more than 10
Available: never negative
Reserved: exactly 10
```

## Environment overrides

```bash
ORDER_BASE_URL=http://localhost:18081
INVENTORY_BASE_URL=http://localhost:8914
POSTGRES_URL='postgres://order_user:order_dummy@localhost:15432/order_db?sslmode=disable'
MYSQL_DSN='root:root_dummy@tcp(localhost:13306)/appdb?parseTime=true'
RABBITMQ_URL='amqp://admin:admin@localhost:15672/'
```

## Troubleshooting

Check logs:

```bash
tail -f .logs/order-service.log
tail -f .logs/inventory-service.log
```

Check RabbitMQ queues:

```bash
docker compose -f compose/docker-compose.integration.yml exec rabbitmq \
  rabbitmqctl list_queues name messages_ready messages_unacknowledged
```

Check PostgreSQL:

```bash
docker compose -f compose/docker-compose.integration.yml exec postgres \
  psql -U order_user -d order_db \
  -c 'SELECT id,status,quantity,version FROM orders ORDER BY created_at DESC;'
```

Check MySQL:

```bash
docker compose -f compose/docker-compose.integration.yml exec mysql \
  mysql -uroot -proot appdb \
  -e 'SELECT * FROM stock_items; SELECT * FROM reservations;'
```

Check Redis:

```bash
docker compose -f compose/docker-compose.integration.yml exec redis \
  redis-cli KEYS '*events*'
```
