#!/usr/bin/env bash
set -Eeuo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ORDER_SERVICE_DIR="${ORDER_SERVICE_DIR:-$ROOT_DIR/../order-service}"
INVENTORY_SERVICE_DIR="${INVENTORY_SERVICE_DIR:-$ROOT_DIR/../inventory-service}"
LOG_DIR="$ROOT_DIR/.logs"
PID_DIR="$ROOT_DIR/.pids"
mkdir -p "$LOG_DIR" "$PID_DIR"

(
  cd "$INVENTORY_SERVICE_DIR"
  MYSQL_DSN='root:root@tcp(localhost:13306)/appdb?parseTime=true' \
  REDIS_ADDR='localhost:16379' \
  RABBITMQ_URL='amqp://admin:admin@localhost:15672/' \
  go run ./cmd/api >"$LOG_DIR/inventory-service.log" 2>&1 &
  echo $! > "$PID_DIR/inventory-service.pid"
)

(
  cd "$ORDER_SERVICE_DIR"
  DATABASE_URL='postgres://order_user:order_password@localhost:15432/order_db?sslmode=disable' \
  REDIS_ADDR='localhost:16379' \
  RABBITMQ_URL='amqp://admin:admin@localhost:15672/' \
  ORDER_PORT='18081' \
  go run . >"$LOG_DIR/order-service.log" 2>&1 &
  echo $! > "$PID_DIR/order-service.pid"
)

echo "Services started. Logs: $LOG_DIR"
for endpoint in http://localhost:8914/healthz http://localhost:18081/health/ready; do
  for _ in $(seq 1 60); do
    if curl -fsS "$endpoint" >/dev/null 2>&1; then
      echo "Ready: $endpoint"
      break
    fi
    sleep 2
  done
done
