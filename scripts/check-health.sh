#!/usr/bin/env bash
set -uo pipefail

readonly ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly HEALTH_TIMEOUT_SECONDS="${HEALTH_TIMEOUT_SECONDS:-120}"
readonly POLL_INTERVAL_SECONDS="${HEALTH_POLL_INTERVAL_SECONDS:-2}"
readonly SCOPE="${1:-all}"

cd "$ROOT_DIR"

compose() {
    local project="$1"
    shift

    case "$project" in
        infrastructure)
            docker compose --env-file deployments/.env.infrastructure \
                -f deployments/docker-compose.infrastructure.yml "$@"
            ;;
        services)
            docker compose --env-file deployments/.env.services \
                -f deployments/docker-compose.services.yml "$@"
            ;;
    esac
}

print_diagnostics() {
    local project="$1"

    echo "Container status for $project:"
    compose "$project" ps --all || true
    echo "Recent logs for $project:"
    compose "$project" logs --no-color --tail 100 || true
}

wait_for_project() {
    local project="$1"
    shift
    local deadline=$((SECONDS + HEALTH_TIMEOUT_SECONDS))

    for service in "$@"; do
        echo "Waiting for $service to become healthy..."

        while true; do
            local container_id
            container_id="$(compose "$project" ps --all --quiet "$service" 2>/dev/null || true)"

            local status="not-created"
            if [[ -n "$container_id" ]]; then
                status="$(docker inspect \
                    --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' \
                    "$container_id" 2>/dev/null || true)"
            fi

            case "$status" in
                healthy)
                    echo "$service is healthy"
                    break
                    ;;
                unhealthy|exited|dead)
                    echo "$service entered state: $status" >&2
                    print_diagnostics "$project"
                    return 1
                    ;;
            esac

            if ((SECONDS >= deadline)); then
                echo "Timed out after ${HEALTH_TIMEOUT_SECONDS}s waiting for $project services" >&2
                print_diagnostics "$project"
                return 1
            fi

            sleep "$POLL_INTERVAL_SECONDS"
        done
    done
}

case "$SCOPE" in
    infrastructure)
        wait_for_project infrastructure postgres rabbitmq mongo redis elasticsearch
        ;;
    services)
        wait_for_project services order-service catalog-write-service inventory-service catalog-read-service api-gateway
        ;;
    all)
        wait_for_project infrastructure postgres rabbitmq mongo redis elasticsearch
        wait_for_project services order-service catalog-write-service inventory-service catalog-read-service api-gateway
        ;;
    *)
        echo "Usage: $0 [infrastructure|services|all]" >&2
        exit 2
        ;;
esac
