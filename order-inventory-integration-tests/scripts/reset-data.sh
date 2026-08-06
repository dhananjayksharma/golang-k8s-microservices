#!/usr/bin/env bash
set -Eeuo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE="$ROOT_DIR/compose/docker-compose.integration.yml"

docker compose -f "$COMPOSE" exec -T postgres psql -U order_user -d order_db <<'SQL'
TRUNCATE TABLE order_status_history, outbox_events, orders RESTART IDENTITY CASCADE;
SQL

docker compose -f "$COMPOSE" exec -T mysql mysql -uroot -proot appdb <<'SQL'
SET FOREIGN_KEY_CHECKS=0;
TRUNCATE TABLE reservations;
TRUNCATE TABLE stock_items;
SET FOREIGN_KEY_CHECKS=1;
SQL

docker compose -f "$COMPOSE" exec -T mysql mysql -uroot -proot appdb < "$ROOT_DIR/fixtures/mysql_seed.sql"
docker compose -f "$COMPOSE" exec -T redis redis-cli FLUSHALL >/dev/null

echo "Integration data reset complete."
