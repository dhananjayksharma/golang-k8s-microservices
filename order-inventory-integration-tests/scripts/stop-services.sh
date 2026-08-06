#!/usr/bin/env bash
set -Eeuo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PID_DIR="$ROOT_DIR/.pids"
for name in order-service inventory-service; do
  file="$PID_DIR/$name.pid"
  if [[ -f "$file" ]]; then
    pid="$(cat "$file")"
    kill -TERM "$pid" 2>/dev/null || true
    rm -f "$file"
  fi
done
