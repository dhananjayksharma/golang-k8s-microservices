#!/usr/bin/env bash

# This script must be sourced so exported variables remain available
# in the current shell:
#   source ./setports.sh
#
# Override the env-file location when needed:
#   ENV_FILE=/path/to/.env.local source ./setports.sh

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  echo "ERROR: Run this script with: source ${BASH_SOURCE[0]}" >&2
  exit 1
fi

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${ENV_FILE:-${SCRIPT_DIR}/.env.local}"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "ERROR: Local environment file not found: ${ENV_FILE}" >&2
  echo "Create it from .env.local.example and keep it out of Git." >&2
  return 1
fi

# Export every variable declared in the local environment file.
set -a
# shellcheck disable=SC1090
source "${ENV_FILE}"
set +a

# Non-secret local defaults. Values in .env.local take precedence.
export ORDER_PORT="${ORDER_PORT:-8081}"
export PURCHASE_PORT="${PURCHASE_PORT:-8085}"
export MESSAGE_PORT="${MESSAGE_PORT:-8083}"
export QUEUE_PORT="${QUEUE_PORT:-8084}"
export INVENTORY_PORT="${INVENTORY_PORT:-8914}"
export INVOICE_PORT="${INVOICE_PORT:-8086}"
export REDIS_ADDR="${REDIS_ADDR:-localhost:6379}"

required_variables=(
  MYSQL_DSN
  DATABASE_URL
  RABBITMQ_URL
)

missing_variables=()
for variable_name in "${required_variables[@]}"; do
  if [[ -z "${!variable_name:-}" ]]; then
    missing_variables+=("${variable_name}")
  fi
done

if (( ${#missing_variables[@]} > 0 )); then
  echo "ERROR: Missing required environment variables in ${ENV_FILE}:" >&2
  printf '  - %s\n' "${missing_variables[@]}" >&2
  return 1
fi

echo "Local environment loaded from: ${ENV_FILE}"
echo "Ports: order=${ORDER_PORT}, purchase=${PURCHASE_PORT}, message=${MESSAGE_PORT}, queue=${QUEUE_PORT}, inventory=${INVENTORY_PORT}, invoice=${INVOICE_PORT}"
echo "Redis: ${REDIS_ADDR}"
echo "Database and RabbitMQ credentials loaded (values hidden)."
