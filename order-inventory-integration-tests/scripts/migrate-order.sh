#!/usr/bin/env bash
set -Eeuo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ORDER_SERVICE_DIR="${ORDER_SERVICE_DIR:-$ROOT_DIR/../order-service}"
[[ -d "$ORDER_SERVICE_DIR/migrations" ]] || { echo "Order migrations not found: $ORDER_SERVICE_DIR/migrations"; exit 1; }
docker run --rm --network host \
  -v "$ORDER_SERVICE_DIR/migrations:/migrations:ro" \
  migrate/migrate:v4.19.1 \
  -path=/migrations \
  -database='postgres://order_user:order_password@localhost:15432/order_db?sslmode=disable' \
  up
