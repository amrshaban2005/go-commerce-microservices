#!/usr/bin/env bash
set -uo pipefail

readonly ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

readonly CHECK_NAMES=(
    "Formatting"
    "Lint"
    "Unit tests"
    "Integration tests"
    "Vet"
    "E2E fixtures"
    "E2E tests"
)

readonly CHECK_TARGETS=(
    "fmt-check"
    "lint"
    "test"
    "test-integration"
    "vet"
    "seed-e2e"
    "test-e2e"
)

declare -a CHECK_RESULTS=()
declare -a CHECK_DURATIONS=()
passed=0
failed=0

cd "$ROOT_DIR"

for index in "${!CHECK_TARGETS[@]}"; do
    name="${CHECK_NAMES[$index]}"
    target="${CHECK_TARGETS[$index]}"
    started_at=$SECONDS

    printf '\n============================================================\n'
    printf 'CHECK: %s\n' "$name"
    printf 'COMMAND: make %s\n' "$target"
    printf '============================================================\n'

    if make --no-print-directory "$target"; then
        CHECK_RESULTS+=("PASS")
        passed=$((passed + 1))
    else
        CHECK_RESULTS+=("FAIL")
        failed=$((failed + 1))
    fi

    CHECK_DURATIONS+=("$((SECONDS - started_at))s")
done

printf '\n============================================================\n'
printf 'DEV CHECK SUMMARY\n'
printf '============================================================\n'
printf '%-22s %-8s %s\n' "Check" "Result" "Duration"
printf '%-22s %-8s %s\n' "----------------------" "--------" "--------"

for index in "${!CHECK_TARGETS[@]}"; do
    printf '%-22s %-8s %s\n' \
        "${CHECK_NAMES[$index]}" \
        "${CHECK_RESULTS[$index]}" \
        "${CHECK_DURATIONS[$index]}"
done

printf '\nPassed: %d  Failed: %d\n' "$passed" "$failed"

if ((failed > 0)); then
    exit 1
fi
