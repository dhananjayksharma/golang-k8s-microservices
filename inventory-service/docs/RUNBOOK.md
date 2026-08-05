# Local Runbook

## Start infrastructure
```bash
cd deploy/local
docker compose up -d postgres mysql redis rabbitmq
```
RabbitMQ UI: `http://localhost:15672` (`admin` / `admin`).

## Inventory Service
```bash
export MYSQL_DSN='root:root@tcp(localhost:3306)/appdb?parseTime=true'
export REDIS_ADDR='localhost:6379'
export RABBITMQ_URL='amqp://admin:admin@localhost:5672/'
go mod tidy
go run ./cmd/api
```
GORM creates `stock_items` and `reservations`. Seed stock:
```bash
docker compose -f deploy/local/docker-compose.yml exec -T mysql \
  mysql -uroot -proot appdb < migrations/001_inventory_seed.sql
```

## Order Service
```bash
export DATABASE_URL='postgres://order_user:order_password@localhost:5432/order_db?sslmode=disable'
export REDIS_ADDR='localhost:6379'
export RABBITMQ_URL='amqp://admin:admin@localhost:5672/'
export ORDER_PORT=8081
go mod tidy
go run .
```

## Verify
```bash
curl http://localhost:8914/healthz
curl http://localhost:8914/v1/inventory
curl http://localhost:8081/health/ready
```

## Create order
```bash
curl -X POST http://localhost:8081/orders/ -H 'Content-Type: application/json' -d '{
  "customer_id":"22222222-2222-4222-8222-222222222222",
  "idempotency_key":"demo-001",
  "quantity":2,
  "unit_price":149900,
  "currency":"INR",
  "tax_amount":0,
  "shipping_amount":0
}'
```
Expected order status flow: `PENDING -> CONFIRMED` when inventory is reserved, otherwise `PENDING -> CANCELLED`.
