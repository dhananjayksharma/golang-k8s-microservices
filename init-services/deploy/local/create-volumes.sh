#!/usr/bin/env bash

set -Eeuo pipefail

VOLUMES=(
  "order-postgres-data"
  "order-mysql-data"
  "order-redis-data"
  "order-rabbitmq-data"
)

echo "Creating persistent Docker volumes..."

for volume in "${VOLUMES[@]}"; do
  if docker volume inspect "$volume" >/dev/null 2>&1; then
    echo "EXISTS  : $volume"
  else
    docker volume create "$volume" >/dev/null
    echo "CREATED : $volume"
  fi
done

echo
echo "Persistent volumes:"
for volume in "${VOLUMES[@]}"; do
  docker volume inspect \
    --format='{{.Name}} -> {{.Mountpoint}}' \
    "$volume"
done

echo
echo "All database volumes are ready."