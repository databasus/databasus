## Context

See proposal.md, Why, for the motivation. The facts that shape the approach:

- Every network a test creates goes through `WithFreeSubnet` (`backend/internal/lib/testcontainers/network_subnet.go:23`). Its callers are `newLabelledNetwork` (`containers.go:192`), `DockerHost.CreateNetwork` (`docker_host.go:68`), `StartPatroni` (`patroni.go:76`), `K3s.createNetwork` (`k3s.go:264`) and the agent end-to-end helper `createNetwork` (`backend/internal/agent/e2e/engine_test.go:94`). The browser suite creates its compose network from the same range through its own supervisor (`e2e/browser/network_subnet.py:9`).
- `IsTestNetworkSubnet` (`network_subnet.go:38`) already says whether a subnet lies in `10.160.0.0/11`. `docs/testing.md:73` reserves that range for test-created networks and keeps the verification agent on Docker's default pools.
- The run label `databasus.test-run` is added only when the run ID variable is set (`containers.go:223`), which `scripts/test.sh` does. A direct `go test` creates networks without it.
- `NetworkList` returns `network.Summary`, which embeds `network.Network` with its `IPAM` and `Created` fields (moby `api/types/network/summary.go:11`, `network.go:31`). Its `Containers` map is empty in a list, and inspect fills it only with running endpoints.
- Docker removes a network whose only containers are stopped. This was checked on the development machine with Docker 29.9.2: a network with an exited container showed `Containers = 0` and `docker network rm` succeeded. A network with a running container is refused with "has active endpoints".
- The abandoned-database sweep in `backend/internal/lib/testdb/cleanup.go:33` sets the pattern: `sync.Once` per process, no lock, removal errors ignored, an age threshold of one hour, and a scope type that its test uses to move the clock and narrow the sweep to its own objects.
- `backend/AGENTS.md` constrains the approach. Code takes the caller's context, and only commands and composition create root contexts. Goroutines start only through the `lib` helpers that turn a panic into an error. A package-level variable needs a named `nolint` reason, which process-level test fixture synchronization has.

## Goals / Non-Goals

**Goals:**

- A test process never waits for the sweep before creating its own network. This is a design property, not a spec requirement: it is checked by reading the wiring, because a test of it would need an injectable sweep function only for that check.
- The sweep covers networks from direct `go test` runs, `mise run test`, the agent end-to-end suite and the browser suite, because it selects by subnet instead of by label.

**Non-Goals:**

- Reporting what the sweep removed. Errors and counts are not surfaced; the next process retries whatever remains.
- Coordinating sweeps of concurrent processes.

## Decisions

### Start the sweep from `WithFreeSubnet`, once per process

The first call of `WithFreeSubnet` in a process starts the sweep through a `sync.Once`, then goes on to create its network. Every test network passes through this function, so one call site covers all five callers and any future fixture.

Rejected alternatives:
- A call in each fixture: five call sites, and a new fixture can forget it.
- The trap in `scripts/test.sh`: it misses direct `go test` runs, which are the ones that piled up.
- A manual `mise` task: nobody runs it until the machine is already slow.

### Select by subnet and age, not by label

A network is a candidate when its IPAM config has at least one subnet, every subnet passes `IsTestNetworkSubnet`, and its `Created` time is more than an hour before the sweep's time. Both values come from one `NetworkList` call. Requiring a subnet keeps `host`, `none` and networks without address management out, because "every subnet is in range" holds trivially for a network with none. The selection is a pure predicate over `network.Summary`, tested with a table, because Docker 29 refuses to create a bridge network without a subnet ("ipv4 pool is empty" with a null IPAM driver, "IPv4 or IPv6 must be enabled" without IPv4), so a container-level test cannot produce that case.

Rejected alternatives:
- The `databasus.test-run` label: direct runs do not set it.
- The `org.testcontainers` label: `DockerHost` and the browser suite create networks without testcontainers, and other projects on the machine use testcontainers too.

### The age threshold is longer than any test process lives

The threshold is `abandonedNetworkAge = time.Hour`, and its comment states why: no test process lives that long. The longest `-timeout` in the repository is 40 minutes for the race detector run (`docs/testing.md:399`), the agent end-to-end suite has 30 minutes, and `scripts/test.sh:207` gives 15 minutes. A network older than the longest process therefore belongs to a process that has ended.

The age is the real protection, because Docker treats a network with only stopped containers as unused. A live test that stops its container for a while, which `DockerHost` lets the upgrade and recovery suites do, would otherwise lose its network.

Rejected alternatives:
- Keeping networks that have stopped containers, through `ContainerList` with a `network` filter. After a Docker restart the killed run's containers stay stopped, so their networks would never be removed, and the restart case is part of the requirement.
- A shorter threshold, such as 15 minutes: it would remove the networks of a live 40-minute race run whose containers are stopped at the moment of the sweep.

### Remove without inspecting first

The sweep calls `NetworkRemove` for every candidate and ignores the error. Docker refuses to remove a network with a running container, and a network another process removed first returns "not found".

Rejected alternative: `NetworkInspect` before each removal to check `Containers`. It doubles the requests and adds a window between inspect and remove. It protects nothing beyond what Docker's refusal already protects.

### Run the sweep in the background with its own budget

The sweep runs in a goroutine started through `background.Group`, with a discard logger as `postgres_proxy.go:103` does, so a panic turns into an error instead of killing the test process. The sweep's context is `context.WithoutCancel(ctx)` of the first caller with a 15-minute timeout. `WithFreeSubnet` returns to its creation loop at once. When the process exits mid-sweep, the goroutine ends with it, and the next process removes the rest.

Rejected alternatives:
- A synchronous sweep: removing 190 networks behind a slow firewalld takes minutes, and the first test in the process fails on its own timeout.
- A synchronous sweep limited to N networks: it still delays the first test by N removals of up to tens of seconds each, and it takes many processes to drain a large pile.

The background sweep competes with the process's own network creations on firewalld. Each removal shrinks the firewalld zone, so creations become faster as the sweep proceeds.

The timeout bounds a sweep in a process whose life no test timeout bounds, such as `-timeout 0`. It matches the default package timeout of the test runner; a sweep that does not finish leaves the rest for the next process.

### `WithFreeSubnet` takes the caller's context

`WithFreeSubnet(ctx, create)` replaces `WithFreeSubnet(create)`. All five callers already hold a context (`t.Context()` or a fixture's `ctx`) and pass it.

Rejected alternatives:
- `context.Background()` inside the sweep: `backend/AGENTS.md` keeps root contexts in commands and composition.
- Passing the caller's context as it is: the first caller is usually a test whose `t.Context()` ends with the test, and the sweep would stop with it.

### No lock between processes; each sweep shuffles its candidates

`go test -p=8` starts up to eight sweeps at once over the same list. They need no lock, because a duplicate removal fails harmlessly. Each sweep shuffles its candidates, so concurrent sweeps work on different networks instead of all trying the same one first.

Rejected alternative: a file lock or a Docker-side marker. It adds failure modes, such as a stale lock after a kill, for a race that costs only a few duplicate requests.

### Test the sweep directly through its scope

`removeAbandonedNetworks(ctx, docker, scope)` takes a `networkCleanupScope{now, networkNames}`. The test creates its own networks, sets `now` an hour and a margin ahead and lists only its own network names, so it never removes another run's network. The once-per-process sweep of the test binary runs with the real time and cannot remove the test's fresh networks.

Rejected alternative: a process-wide scope override and a child-process test through `sync.Once`, as `testdb` has for `TestOpenAsChildProcess` (`backend/internal/lib/testdb/testdb_test.go:165`). For networks the wiring is one call in `WithFreeSubnet`, and checking it would need a child process and a wait for an asynchronous removal. The override variable is therefore not added.

## Risks / Trade-offs

- [A process with `-timeout 0` or a debugger session runs over an hour while its container is stopped, and another process removes its network] → The constant's comment states the assumption. Such sessions are interactive, and the failure shows up as a missing network when the container starts again.
- [Right after this change, several processes sweep a large pile at once and briefly add Docker load] → It happens once per accumulated pile. The shuffle spreads the work, and every removal makes firewalld faster.
- [Docker is unreachable or the list fails] → The sweep returns silently. The process's own network creation fails with its own error, which already reports the cause.
- [A remote `DOCKER_HOST` whose clock is ahead of the host] → `Created` is compared with the local clock, and a skew above an hour could remove a live empty network. The project's tests run against the local daemon, so this is accepted.

## Migration Plan

No migration. The first test process after the change removes networks that are already on a machine. Rollback is a revert of the change.
