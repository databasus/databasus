#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
source "$root/scripts/fedora.sh"

# Ryuk is disabled, so nothing else removes the containers an interrupted run leaves behind.
# The shared compose fixtures stay up for the next run and for other worktrees.
cleanup() {
    trap - EXIT INT TERM
    pkill -TERM -P $$ 2>/dev/null || true
    local ids
    ids=$(docker ps -aq --filter "label=org.testcontainers")
    if [[ -n "$ids" ]]; then
        # shellcheck disable=SC2086 # one container ID per word
        docker rm -f $ids >/dev/null
    fi
}

wait_for_port() {
    local port=$1
    timeout 60 bash -c "until (exec 3<>/dev/tcp/127.0.0.1/$port) 2>/dev/null; do sleep 2; done"
}

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

if [[ -f "$root/.env" ]]; then
    set -a
    # shellcheck source=/dev/null # .env is local and untracked, so CI has nothing to follow
    source "$root/.env"
    set +a
fi
: "${GOOSE_TEST_DBSTRING:?GOOSE_TEST_DBSTRING must be set, copy .env.example to .env}"
workers=${TEST_PARALLEL_WORKERS:-8}

apply_libedit_shim

cd "$root"
docker compose up -d
timeout 60 bash -c 'until docker exec backend-db-test pg_isready -U postgres >/dev/null; do sleep 2; done'
wait_for_port "${TEST_LOGICAL_POSTGRES_16_PORT:-5004}"
wait_for_port "${TEST_PHYSICAL_POSTGRES_17_PORT:-5007}"
wait_for_port "${TEST_PHYSICAL_POSTGRES_18_PORT:-5008}"

just --justfile "$root/backend/Justfile" clean-testcontainers
just --justfile "$root/backend/Justfile" pull-testcontainers

cd "$root/backend"
TEST_PARALLEL_WORKERS=$workers go run ./cmd/cleanup_test_db
for ((i = 0; i < workers; i++)); do
    dbstring=$(sed -E "s#/([^/?]+)\?#/\1_w$i?#" <<<"$GOOSE_TEST_DBSTRING")
    echo "migrating slot $i"
    GOOSE_DRIVER=postgres GOOSE_DBSTRING="$dbstring" goose -dir ./migrations up
done

TESTCONTAINERS_RYUK_DISABLED=true TEST_PARALLEL_WORKERS=$workers \
    go test -race -tags testhelpers -p="$workers" -count=1 -failfast -timeout 15m ./internal/... ./cmd/...
