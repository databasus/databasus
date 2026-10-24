#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
source "$root/scripts/fedora.sh"

readonly backend_port=4005
readonly test_fixtures=(test-logical-postgres-16 test-physical-postgres-17 test-physical-postgres-18)
readonly own_services=(backend-db wal-backup-postgres-18 wal-backup-loader)

# `stop`, not `down`: the dev database keeps its data in an anonymous volume that `down` would drop.
# The test fixtures stay up, because the backend suite and other worktrees share them.
stop_services() {
    docker compose -f "$root/docker-compose.yml" stop "${own_services[@]}"
}

# air keeps watching after the backend it started dies, so a watcher left by a killed run of this
# worktree would rebuild a backend next to the new run and fight it for the port. Watchers of other
# worktrees work in their own backend/ and are left alone.
stop_stale_backend_watchers() {
    local air_pid
    for air_pid in $(pgrep -x air); do
        [[ "$(readlink "/proc/$air_pid/cwd" 2>/dev/null)" == "$root/backend" ]] || continue
        echo "stopping stale backend watcher pid $air_pid"
        kill "$air_pid" 2>/dev/null || true
        timeout 10 tail --pid="$air_pid" -f /dev/null || kill -9 "$air_pid" 2>/dev/null || true
    done
}

# A stale backend keeps holding the port and answers with the previously built code; a new one that
# fails to bind only logs the error, so requests silently hit the old binary. Its watcher is stopped
# first by stop_stale_backend_watchers, because air would otherwise start the backend again.
free_backend_port() {
    local port_hex inode process_dir pid
    port_hex=$(printf ':%04X' "$backend_port")
    inode=$(awk -v p="$port_hex" '$4 == "0A" && $2 ~ p"$" {print $10}' /proc/net/tcp /proc/net/tcp6 | head -1)
    [[ -n "$inode" ]] || return 0
    for process_dir in /proc/[0-9]*; do
        pid=${process_dir#/proc/}
        if [[ -n "$(find "$process_dir/fd" -lname "socket:\[$inode\]" -print -quit 2>/dev/null)" ]]; then
            echo "freeing port $backend_port: killing stale backend pid $pid"
            kill -9 "$pid" 2>/dev/null || true
        fi
    done
}

[[ -f "$root/.env" ]] || {
    echo "$root/.env is missing, copy .env.example to .env" >&2
    exit 1
}

apply_libedit_shim

trap stop_services EXIT
docker compose -f "$root/docker-compose.yml" up -d "${own_services[@]}" "${test_fixtures[@]}"
timeout 60 bash -c 'until docker exec backend-db pg_isready -U postgres >/dev/null; do sleep 2; done'

pnpm --dir "$root/frontend" install --frozen-lockfile
stop_stale_backend_watchers
free_backend_port

cd "$root"
process-compose --config process-compose.yaml --no-server --disable-dotenv up "$@"
