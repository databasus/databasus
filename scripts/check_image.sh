#!/usr/bin/env bash
set -euo pipefail

image=$1
container_name="databasus-image-check-$$"

data_directory=$(mktemp -d)

# shellcheck disable=SC2329 # Runs from the EXIT trap below.
remove_container_and_data_directory() {
    docker rm -f "$container_name" >/dev/null 2>&1 || true
    rm -rf "$data_directory" || true
}
trap remove_container_and_data_directory EXIT

show_container_logs() {
    docker logs "$container_name" || true
}

# 0770 keeps the traversal bit off for others, so the smoke run only passes
# while the in-container postgres account reaches PGDATA through its group.
chmod 770 "$data_directory"

docker run -d --name "$container_name" \
    -p 4005:4005 \
    -v "$data_directory:/databasus-data" \
    "$image"

for attempt in $(seq 1 60); do
    health_body=$(mktemp)
    code=$(curl -s -o "$health_body" -w '%{http_code}' http://localhost:4005/api/v1/system/health || true)

    if [ "$code" = "200" ] && grep -q '"status"' "$health_body"; then
        echo "healthcheck OK after ${attempt} attempt(s)"
        rm -f "$health_body"
        break
    fi

    rm -f "$health_body"

    if [ "$attempt" -eq 60 ]; then
        echo "healthcheck never returned valid JSON (last code: ${code})"
        show_container_logs
        exit 1
    fi

    sleep 2
done

body=$(curl -sf http://localhost:4005/)
if ! echo "$body" | grep -q 'id="root"'; then
    echo "index.html did not contain React root div"
    echo "$body"
    exit 1
fi
echo "index.html OK"

for agent_arch in amd64 arm64; do
    echo "Checking verification agent download for $agent_arch..."
    agent_path=$(mktemp)
    http=$(curl -sS -o "$agent_path" -w '%{http_code}' "http://localhost:4005/api/v1/system/verification-agent?arch=$agent_arch")

    if [ "$http" != "200" ]; then
        echo "verification agent ($agent_arch) download failed: HTTP $http"
        head -c 200 "$agent_path" || true
        show_container_logs
        rm -f "$agent_path"
        exit 1
    fi

    magic=$(head -c 4 "$agent_path" | od -An -t x1 | tr -d ' ')
    if [ "$magic" != "7f454c46" ]; then
        echo "verification agent ($agent_arch) is not an ELF binary (first 4 bytes: $magic)"
        head -c 200 "$agent_path" || true
        show_container_logs
        rm -f "$agent_path"
        exit 1
    fi

    size=$(stat -c%s "$agent_path")
    echo "verification agent ($agent_arch) OK (${size} bytes)"

    rm -f "$agent_path"
done

# A console command is the only user of the credential start.sh publishes for
# processes that miss PID 1's environment, so nothing else here notices when
# publishing breaks.
echo "Checking that a console command reaches the metadata database..."
console_output=$(mktemp)
if ! docker exec "$container_name" ./main --list-admins >"$console_output" 2>&1; then
    echo "console command could not reach the metadata database"
    cat "$console_output"
    show_container_logs
    rm -f "$console_output"
    exit 1
fi

rm -f "$console_output"
echo "console command OK"

# A broken non-PostgreSQL bundle only logs a warning and disables that engine,
# so the container stays healthy. The image ships every bundle, so any such
# warning here means one of them cannot run in the image.
echo "Checking that every bundled client tool runs in the image..."
if docker logs "$container_name" 2>&1 | grep -F "client tools check failed"; then
    echo "a bundled client tool cannot run in the image"
    exit 1
fi
echo "client tools OK"

critical_scan_status=0
trivy image --severity CRITICAL --ignore-unfixed --pkg-types os,library --scanners vuln --exit-code 1 "$image" ||
    critical_scan_status=$?

trivy image --severity HIGH --ignore-unfixed --pkg-types os,library --scanners vuln --exit-code 0 "$image"

exit "$critical_scan_status"
