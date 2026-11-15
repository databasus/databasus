## Why

Backend tests remove the Docker networks they create in `t.Cleanup`, which never runs when `go test -timeout` expires, the process is killed or Docker restarts. Ryuk is disabled, and `scripts/test.sh` removes only resources labelled by `mise run test`, so after direct `go test` runs about 190 empty networks piled up on one machine. With Docker on iptables and firewalld, every network is a bridge interface in firewalld's `docker` zone. At 250 interfaces `docker network create` took over 90 seconds, k3s, Patroni and logical tests failed on their timeouts, and compose projects of other repositories hung for over 800 seconds.

## What Changes

- The first network a backend test process creates also starts a background sweep, once per process. The sweep removes networks whose subnet lies in the test network range `10.160.0.0/11` and that were created over an hour ago. Docker itself refuses to remove a network with a running container, and that refusal is ignored like every other removal error.
- The sweep never delays the network the test is creating, and it is bounded by its own time budget. A process that exits mid-sweep leaves the rest to the next process.
- `WithFreeSubnet` takes the caller's context, which the sweep detaches from cancellation so it outlives the first test. Its five callers pass the context they already hold. This changes the signature of an internal test-support function; no shim is kept.
- `docs/testing.md` states in its Isolation section that stale test networks are collected automatically, in parallel with the existing sentence about databases.

## Out of scope

- Enabling Ryuk. Parallel test binaries collide on its reaper name, and it would not cover a Docker restart either.
- `docker network prune` or removing any network outside the test network range. Other projects and the verification agent's restore networks use Docker's default address pools.
- Removing containers or volumes, stopped containers that a killed run or a Docker restart left included. Volumes may follow later with the same approach, as a separate change.
- The shared test PostgreSQL server and the trap in `scripts/test.sh`, which keep their current behavior.
- A sweep from the browser end-to-end supervisor. Its networks lie in the same range, so backend test processes remove them.
- A one-off cleanup of networks that are already on a machine; the first test process after this change does it.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `backend-test-isolation`: adds a requirement that abandoned test networks are removed, alongside the existing requirement for abandoned test databases.

## Impact

- Code: `backend/internal/lib/testcontainers` gets a sweep of abandoned networks and a context parameter on `WithFreeSubnet`. The callers in `containers.go`, `docker_host.go`, `patroni.go`, `k3s.go` and `backend/internal/agent/e2e/engine_test.go` pass their context.
- Docs: `docs/testing.md`, Isolation, item "Docker networks".
- Runtime: test-only. Production code and the application image do not change.
- The change answers to the root `AGENTS.md` and `backend/AGENTS.md`.
