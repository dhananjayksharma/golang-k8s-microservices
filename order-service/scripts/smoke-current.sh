#!/usr/bin/env bash
set -euo pipefail
BASE_URL="${BASE_URL:-http://localhost:8081}"
created=$(curl -fsS -X POST "$BASE_URL/orders/" -H 'Content-Type: application/json' \
  -d '{"quantity":2,"price":1499,"date":"2026-08-02T00:00:00Z"}')
echo "$created"
id=$(printf '%s' "$created" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
curl -fsS "$BASE_URL/orders/$id"; echo
curl -fsS "$BASE_URL/orders/"; echo
