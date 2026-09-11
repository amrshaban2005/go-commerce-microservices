#!/usr/bin/env bash
set -euo pipefail

readonly ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly INFRA_ENV_FILE="${INFRA_ENV_FILE:-deployments/.env.infrastructure}"

cd "$ROOT_DIR"

set -a
source "$INFRA_ENV_FILE"
set +a

docker compose --env-file "$INFRA_ENV_FILE" \
    -f deployments/docker-compose.infrastructure.yml \
    exec -T postgres \
    psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d inventory_db \
    < services/test/e2e/fixtures/inventory.sql
