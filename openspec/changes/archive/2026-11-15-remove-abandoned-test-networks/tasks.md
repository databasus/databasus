## 1. Context on network creation

- [x] 1.1 Add a `ctx context.Context` first parameter to `WithFreeSubnet` in `backend/internal/lib/testcontainers/network_subnet.go` and pass the context each caller already holds in `containers.go`, `docker_host.go`, `patroni.go`, `k3s.go`, `network_subnet_test.go` and `backend/internal/agent/e2e/engine_test.go`; verify with `go -C backend vet ./internal/lib/testcontainers/ ./internal/agent/e2e/` and `go -C backend test -count=1 -run '^TestWithFreeSubnet$' ./internal/lib/testcontainers/`.

## 2. Sweep of abandoned networks

- [x] 2.1 Add `backend/internal/lib/testcontainers/network_cleanup.go` with `abandonedNetworkAge`, whose comment says it exceeds the longest test process timeout, the `networkCleanupScope{now, networkNames}` type and `removeAbandonedNetworks(ctx, docker, scope)`. The predicate `isAbandonedTestNetwork(summary, now)` holds for a network that has at least one IPAM subnet, whose every subnet passes `IsTestNetworkSubnet` and whose `Created` is older than the threshold. The function lists networks once, keeps those the predicate accepts and whose name is in scope when the scope lists names, shuffles them and removes each one, ignoring removal errors; verify with `mise run backend:lint`.
- [x] 2.2 Add `removeAbandonedNetworksOncePerProcess(ctx)` to the same file: a `sync.Once` with a `//nolint:gochecknoglobals` reason, a Docker client from `client.New(client.FromEnv)` closed when the sweep ends, and a `background.Group` with a discard logger that runs the sweep under `context.WithoutCancel(ctx)` with a 15-minute timeout. Call it at the start of `WithFreeSubnet`, before the creation loop, and keep the comments that explain why the sweep runs in the background and why it needs no lock; verify with `mise run backend:lint` and `go -C backend test -count=1 -run '^TestWithFreeSubnet$' ./internal/lib/testcontainers/`.
- [x] 2.3 Add `backend/internal/lib/testcontainers/network_cleanup_test.go` with `TestRemoveAbandonedNetworks` and the scenario "removes only old test networks without running containers". The test creates in the test range an empty network, a network with a running `alpine:3.21` container from the image catalog and a network whose only `alpine:3.21` container has exited, plus a network from Docker's default pools. It then calls `removeAbandonedNetworks` with `now` an hour and a margin ahead and the scope narrowed to these names, and asserts that the empty and the stopped-container test-range networks are gone while the other two remain. A second call with the real time keeps a fresh empty test-range network. Every network and container is removed in `t.Cleanup`. Docker 29 refuses to create a bridge network without a subnet, so a second test, `TestIsAbandonedTestNetwork`, checks the predicate on `network.Summary` values in a table: it rejects an old network with an empty IPAM config, an old network with one subnet in the range and one outside it, and a network in the range that is younger than the threshold, and accepts an old network whose only subnet is in the range; verify with `cd backend && source ../scripts/fedora.sh && apply_libedit_shim && TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 -run '^(TestRemoveAbandonedNetworks|TestIsAbandonedTestNetwork)$' ./internal/lib/testcontainers/`.

## 3. Documentation

- [x] 3.1 In `docs/testing.md`, Isolation, item "Docker networks", add one or two sentences, parallel to "Stale, unused databases are collected automatically", saying that every backend test process removes test-range networks older than an hour that have no running container, and that leftover networks slow firewalld with each interface; verify with `rg -n "Docker networks" docs/testing.md` and by reading the item.

## 4. Checks of the touched code

- [x] 4.1 Run `mise run backend:format` and `mise run backend:lint` and fix every finding.
- [x] 4.2 Repeat the agent end-to-end suite, whose network helper changed in 1.1; verify with `mise run agent:e2e`.

## Change verification

```sh
mise run test
```
