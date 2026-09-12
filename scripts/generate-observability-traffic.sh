#!/usr/bin/env bash
set -uo pipefail

readonly BASE_URL="${BASE_URL:-http://localhost:8080}"
readonly DURATION_SECONDS="${DURATION_SECONDS:-60}"
readonly CONCURRENCY="${CONCURRENCY:-5}"
readonly REQUEST_DELAY_SECONDS="${REQUEST_DELAY_SECONDS:-0.1}"
readonly REQUEST_TIMEOUT_SECONDS="${REQUEST_TIMEOUT_SECONDS:-10}"

if ! [[ "$DURATION_SECONDS" =~ ^[1-9][0-9]*$ ]]; then
    echo "DURATION_SECONDS must be a positive integer" >&2
    exit 2
fi
if ! [[ "$CONCURRENCY" =~ ^[1-9][0-9]*$ ]]; then
    echo "CONCURRENCY must be a positive integer" >&2
    exit 2
fi

readonly RESULTS_DIR="$(mktemp -d)"
readonly RESULTS_FILE="$RESULTS_DIR/results"

cleanup() {
    rm -rf "$RESULTS_DIR"
}
trap cleanup EXIT

readonly ROUTES=(
    "/api/v1/products"
    "/api/v1/products/search?q=keyboard"
    "/api/v1/orders"
    "/api/v1/products/search"
    "/api/v1/not-found"
)

if ! curl --silent --show-error --fail --max-time "$REQUEST_TIMEOUT_SECONDS" \
    "$BASE_URL/swagger/index.html" >/dev/null; then
    echo "API gateway is not reachable at $BASE_URL" >&2
    exit 1
fi

request() {
    local route="$1"
    local result

    if result="$(curl --silent --output /dev/null \
        --write-out '%{http_code} %{time_total}' \
        --max-time "$REQUEST_TIMEOUT_SECONDS" \
        "$BASE_URL$route" 2>/dev/null)"; then
        printf '%s\n' "$result" >>"$RESULTS_FILE"
    else
        printf '000 0\n' >>"$RESULTS_FILE"
    fi
}

printf 'Generating API gateway traffic for %ss with concurrency %s...\n' \
    "$DURATION_SECONDS" "$CONCURRENCY"

deadline=$((SECONDS + DURATION_SECONDS))
request_number=0

while ((SECONDS < deadline)); do
    for ((worker = 0; worker < CONCURRENCY; worker++)); do
        route="${ROUTES[$((request_number % ${#ROUTES[@]}))]}"
        request "$route" &
        request_number=$((request_number + 1))
    done

    wait
    sleep "$REQUEST_DELAY_SECONDS"
done

printf '\nTraffic summary\n'
printf '%-12s %s\n' "HTTP status" "Requests"
awk '{counts[$1]++} END {for (status in counts) print status, counts[status]}' \
    "$RESULTS_FILE" | sort -n | while read -r status count; do
        printf '%-12s %s\n' "$status" "$count"
    done

awk '
    {
        total_requests++
        total_duration += $2
    }
    END {
        average = total_requests == 0 ? 0 : total_duration / total_requests
        printf "\nTotal requests: %d\nAverage client duration: %.4fs\n", total_requests, average
    }
' "$RESULTS_FILE"

printf '\nPrometheus scrapes every 15s. Wait for the next scrape, then refresh Grafana.\n'
