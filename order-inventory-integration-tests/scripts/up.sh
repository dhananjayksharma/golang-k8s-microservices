#!/usr/bin/env bash
set -Eeuo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
docker compose -f "$ROOT_DIR/compose/docker-compose.integration.yml" up -d

echo "Waiting for infrastructure health..."
for service in postgres mysql redis rabbitmq; do
  for _ in $(seq 1 60); do
    status="$(docker inspect --format='{{if .State.Health}}{{.State.Health.Status}}{{else}}running{{end}}' "$(docker compose -f "$ROOT_DIR/compose/docker-compose.integration.yml" ps -q "$service")" 2>/dev/null || true)"
    if [[ "$status" == "healthy" || "$status" == "running" ]]; then
      echo "$service: $status"
      break
    fi
    sleep 2
  done
done
