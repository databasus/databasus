#!/bin/bash
set -euo pipefail

source "$(dirname "$0")/lib.sh"

WORK="/tmp/agent-work"
AGENT_ID="11111111-1111-1111-1111-111111111111"

rm -rf "$WORK"
mkdir -p "$WORK"
cd "$WORK"

reset_mock_state
reset_mock_version

cp "$ARTIFACTS/agent-v1" ./databasus-verification-agent
chmod +x ./databasus-verification-agent

INITIAL="$(./databasus-verification-agent version)"
if [ "$INITIAL" != "v1.0.0" ]; then
  echo "FAIL: expected initial version v1.0.0, got $INITIAL"
  exit 1
fi
echo "Initial version: $INITIAL"

./databasus-verification-agent run \
  --databasus-host "$MOCK" \
  --agent-id "$AGENT_ID" \
  --token test-token \
  --max-cpu 4 \
  --max-ram-mb 2048 \
  --max-disk-gb 20 \
  --max-concurrent-jobs 2 \
  --allow-insecure-http > agent.out 2>&1 &
AGENT_PID=$!
trap stop_agent EXIT

# 1) Capacity heartbeat is received (Run sends one immediately).
COUNT=0
for _ in $(seq 1 20); do
  COUNT="$(hb_count || echo 0)"
  [ "${COUNT:-0}" -ge 1 ] && break
  sleep 1
done
if [ "${COUNT:-0}" -lt 1 ]; then
  echo "FAIL: no heartbeat received"
  cat agent.out
  exit 1
fi
echo "Heartbeat received (count=$COUNT)"

# 2) Heartbeat wire shape: flat maxRamGb (2048 MB -> 2 GB), empty job list.
BODY="$(curl -sf "$MOCK/mock/heartbeats")"
if ! echo "$BODY" | grep -q '"maxRamGb":2'; then
  echo "FAIL: heartbeat missing maxRamGb:2"
  echo "$BODY"
  exit 1
fi
if ! echo "$BODY" | grep -q '"currentVerificationIds":\[\]'; then
  echo "FAIL: heartbeat currentVerificationIds not []"
  echo "$BODY"
  exit 1
fi
echo "Heartbeat wire shape OK"

# 3) Bump version -> background self-update downloads v2 and re-execs.
curl -sf -X POST "$MOCK/mock/set-binary-path" \
  -H 'Content-Type: application/json' -d '{"binaryPath":"/artifacts/agent-v2"}'
curl -sf -X POST "$MOCK/mock/set-version" \
  -H 'Content-Type: application/json' -d '{"version":"v2.0.0"}'
echo "Mock bumped to v2.0.0, waiting for background upgrade..."

DEADLINE=$((SECONDS + 90))
while [ $SECONDS -lt $DEADLINE ]; do
  [ "$(./databasus-verification-agent version)" = "v2.0.0" ] && break
  sleep 3
done

FINAL="$(./databasus-verification-agent version)"
if [ "$FINAL" != "v2.0.0" ]; then
  echo "FAIL: expected v2.0.0 after background upgrade, got $FINAL"
  cat agent.out
  exit 1
fi
echo "Binary upgraded to $FINAL"

if ! grep -q "agent binary updated" agent.out; then
  echo "FAIL: upgrade log line missing"
  cat agent.out
  exit 1
fi
if ! grep -q "re-executing after upgrade" agent.out; then
  echo "FAIL: re-exec log line missing"
  cat agent.out
  exit 1
fi

# 4) Same PID is alive after re-exec (syscall.Exec preserves the PID).
if ! kill -0 "$AGENT_PID" 2>/dev/null; then
  echo "FAIL: agent process died during self-update"
  cat agent.out
  exit 1
fi

# 5) Heartbeats resume after re-exec.
BEFORE="$(hb_count || echo 0)"
RESUMED=0
for _ in $(seq 1 45); do
  AFTER="$(hb_count || echo 0)"
  if [ "${AFTER:-0}" -gt "${BEFORE:-0}" ]; then
    RESUMED=1
    break
  fi
  sleep 1
done
if [ "$RESUMED" -ne 1 ]; then
  echo "FAIL: heartbeats did not resume after re-exec ($BEFORE -> ${AFTER:-0})"
  cat agent.out
  exit 1
fi
echo "Heartbeats resumed after re-exec ($BEFORE -> $AFTER)"

stop_agent
trap - EXIT
echo "Verification agent e2e passed"
