#!/usr/bin/env bash
set -euo pipefail
: "${DATABASE_URL:=postgres://order_user:order_any@localhost:5432/order_db?sslmode=disable}"
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 <<'SQL'
SELECT current_database(), current_user, now();
SELECT status, count(*) FROM orders GROUP BY status ORDER BY status;
SELECT status, count(*), min(created_at) AS oldest FROM outbox_events GROUP BY status ORDER BY status;
SELECT count(*) AS expired_idempotency_records FROM idempotency_records WHERE expires_at < now();
SQL
