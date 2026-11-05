## Context

See `proposal.md` for the motivation and the measurements. These facts about the current code shape the approach:

- `scripts/test.sh:68` runs one `go test -race -p=8 -failfast` over every package, then the upgrade test. It holds a machine-wide lock (`scripts/test.sh:40-44`).
- CI runs its groups through its own `go test -race` line (`.github/workflows/ci-release.yml:428`) and does not call `scripts/test.sh`, `backend:test-package` or `backend:test-group`. Other CI checks call shared commands: `mise run agent:test` (`:256`), `mise run agent:e2e-podman` (`:318`), `mise run backend:test-upgrade` (`:623`) and the real-API browser command (`:218`). CI needs `backend:test-group-packages` and `backend:test-group-tests`, which stay.
- The review hook lists accepted commands in `VERIFICATION_FORMS` (`.agent-hooks/review_gate.py:240`), the prompts (`.agent-hooks/proposal-prompt.md`, `.agent-hooks/implementation-prompt.md`) forbid a full run, and so do the task rules in `openspec/config.yaml` and `AGENTS.md:76`.
- The shared test server starts with only `max_connections=300` (`docker-compose.yml:49`), while engine test containers already turn durability off (`backend/internal/lib/testcontainers/postgres.go:53`).
- Every `apptest.New` calls `App.NewHandler`, and `RequireContractOperation` parses the contract in `NewContractRouter` and again in `checkAccessMarks` (`backend/internal/lib/httpserver/contract_guard.go:21`, `:26`, `:55`, `:76`). `NewHandler` builds a guard for each of the two contracts (`backend/internal/app/routes.go:15`, `:54`). The test client already parses once per process (`backend/internal/lib/apiclient/apiclient.go:18`).
- The `tparallel` linter reports 315 top-level tests today, and `backend/.golangci.yml` does not enable it.

The approach answers to the testing skill, `docs/testing.md` and these `backend/AGENTS.md` rules: no package-level variable holds a dependency, no package does work at import time, and wiring goes into `internal/app`.

## Goals / Non-Goals

**Goals:**

- One local command that checks the backend and frontend suites, with a backend part shorter than today's 594 seconds.
- The other packages do not wait for slow ones, and a long waiting test overlaps other work.
- CI's selection, checks and race detector stay as they are. Its disposable test server gets the same durability flags, since it starts from the same compose file.

**Non-Goals:**

- A fixed time target. The measured critical path is known, but how far it shrinks depends on what the slow packages share, which implementation measures.
- Any change to production behavior.

## Decisions

### Delete the narrow commands and let everything run through `mise run test`

`backend:test-package` and `backend:test-group`, the commands that take packages or shards as arguments, go away. The fixed suites stay, because nobody chooses their contents by judgment: `agent:test`, `agent:e2e-podman` and `backend:test-upgrade` because CI calls them, and `agent:e2e` because the `local-development-workflow` capability requires it. `agent:e2e` delegates to `backend:test-package` today, so it gets its own `go test` line with the same shim, timeout and `TESTCONTAINERS_RYUK_DISABLED=true`. Direct `go test -run` stays documented for repeating one failing test; it needs the libedit shim on Fedora, so `docs/testing.md` gives the full command.

Rejected alternatives:

- Keeping the narrow commands beside it: agents would keep picking packages by judgment, which is the choice the user wants gone.
- An "affected packages" command: it decides from imports what a change can break, and it misses behavior reached through the API.

### Run slow packages and all others in two concurrent `go test` processes

`scripts/test.sh` installs the frontend dependencies as `scripts/dev.sh:58` does, then starts three processes and waits for all of them: slow backend packages, other backend packages and `pnpm --dir frontend test`. It exits non-zero if any fails, prints each process's output under its name and runs the upgrade test only when all pass. It drops `-failfast`, so one failure does not hide others. `cleanup` signals only direct children today (`scripts/test.sh:19`), so it must reach all three processes and their children.

The run selects packages as `scripts/test.sh:66` does today. Slow packages are the entries of `backend/test-slow-packages.txt` found in that selection, in the list's order; the other process takes the rest, so a new package joins it automatically. A stale or excluded entry therefore costs nothing at run time, and a script test, which CI runs, fails and names it. The list holds the packages measured at over 60 seconds and the packages whose tests start containers or networks, such as `internal/modules/databases` and `internal/lib/sshtunnel` (`backend/internal/lib/testcontainers/postgres_physical.go:169`, `ssh_bastion.go:86`).

The package limit of the slow process stays low because of two measurements: with `-p=32` the same tests took 3 to 7 times longer at low CPU, and Docker ran out of network address pools. Implementation measures slow limits of 6, 8 and 12 and other limits of 16 and 32, and keeps the fastest pair that passes. The other limit must also respect the shared server, where each application can open 10 connections against a limit of 1000.

Rejected alternatives:

- A single `go test -p=32`: measured at 532 seconds for 174 packages, with a failed and hung `internal/agent/e2e`.
- A single `go test -p=8`: this is today's 594 seconds, and `storages/internal/providers` waits until second 343.
- Classifying packages automatically through their imports: `apptest` reaches the container helpers, so nearly every module API package would count as slow even when it starts no container.
- `go test -shuffle=on` to spread long tests: long tests in the measured shard already start in the first seconds, so order within a package is not the problem. `TestIdlePhysicalWalSource` ends at about 228 seconds of a run that lasts longer.
- Skipping long tests in short mode: the user prefers that every test runs. Long tests that start early overlap other work and do not set the length of a full run.

### Shorten the critical path through parallel tests first, shards second

`engines/postgresql/physical` sets the run's length, and its top-level tests add up to about its whole time. Implementation finds why they do not overlap. If they can overlap, they run in parallel. If a shared engine-version scope prevents that, `scripts/test.sh` runs the package as several local `go test -run` processes. It reads the top-level test names from source and deals them by the weights in `backend/test-shard-weights.txt`, as `mise-tasks/backend/test-group-tests` does, so a test without a weight still lands in exactly one shard. The selection stays inside `scripts/test.sh` and outside `backend:test-group-packages`, because each package belongs to exactly one CI group and this package stays in `other`. The same check then applies to the next package on the critical path, such as `storages/internal/providers`, whose `TestSaveFile` and `TestFileOperations` take almost all of its 250 seconds. The work stops when the next gain is under 30 seconds.

Rejected alternative: more shards everywhere. Each shard adds containers and networks against a limit already reached, so shards go only where a measurement shows a gain.

### Add `-race` to shared commands only when `CI` is set

Every repository test command uses `go test ${CI:+-race} ...`, including `scripts/test.sh` and `agent:e2e`, which CI does not call today, so a later CI step that calls them keeps race checks. `runBrowserE2E.ts` adds the flag when `process.env.CI` is set; GitHub Actions sets `CI` on every step. `.github/workflows/ci-release.yml:428` stays as it is. A script test pins this rule, because no test run can notice a missing race detector by failing.

Rejected alternatives:

- `GOFLAGS=-race` in the workflow: the agent end-to-end tests build binaries with `CGO_ENABLED=0` (`backend/internal/agent/e2e/binaries_test.go:67`), and `-race` requires cgo.
- A `--race` flag on tasks: a new CI step that omits it silently loses race checks, and CI's workflow would have to change.
- Race checks during OpenSpec verification only: the user decided that OpenSpec runs without them too, because CI catches races before release.

### Turn durability off with server flags in the compose file

`backend-db-test` gets `-c fsync=off -c synchronous_commit=off -c full_page_writes=off`, the same flags as the engine containers.

Rejected alternatives:

- Data directory on tmpfs: 102 seconds against 105 on the four packages, for a real cost in memory.
- `STRATEGY FILE_COPY` in `CREATE DATABASE`: with durability off, 32 concurrent create and drop pairs take 14 ms against 23 ms per pair, too small to matter.

### Parse each contract once per process, in the wiring package

`httpserver` gains `NewAccessCheckedContractRouter(contract []byte) (routers.Router, error)`: with one parse, it validates the contract, checks its access marks and builds the router. `RequireContractOperation(contractRouter routers.Router, logger *slog.Logger)` takes that router, so it can no longer fail. `internal/app` holds `loadUserContractRouter` and `loadAgentContractRouter`, each a `sync.OnceValues`, the same pattern and `gochecknoglobals` reason as `apiclient.go:18`. The value is immutable and is not a dependency. `NewHandler` still returns a load error, so an unknown mark still fails at startup. `NewContractRouter` stays for test clients, whose contracts may carry no marks.

Rejected alternatives:

- A cache keyed by contract bytes inside `httpserver`: a general-purpose global in a library for a problem with two fixed inputs.
- Building the guard in `apptest` and passing it through the environment: production and tests would wire it differently.
- Leaving it as is because local runs drop the race detector: it still costs about 100 ms per test locally and 846 ms per test in CI, and the user agreed to this fix.

### Enforce parallel subtests with `tparallel`

The linter is added after the existing tests are fixed. It reports three things, and each has its own outcome:

- A parallel test whose direct subtests are all sequential (307 of today's reports) gets `t.Parallel()` in its independent subtests, or keeps them sequential with `//nolint:tparallel // <shared state or dependency>`. A subtest stays sequential when it shares an application, a moving clock, a source server, an engine-version fixture or a process environment setting, or when it depends on an earlier subtest.
- A sequential test with parallel subtests (the other 8 reports) gets `t.Parallel()` at the top level, or a `//nolint:tparallel` reason naming what blocks the top level.
- A test with parallel subtests that uses `defer` and calls no `t.Cleanup` moves the release to `t.Cleanup`.

The linter stops at the first direct subtest that calls `t.Parallel`, does not look into nested `t.Run` calls, reports a `defer` only in a test that calls no cleanup, and cannot resolve every helper, so these cases are left to review. `nolintlint` has `allow-unused: false` in `backend/.golangci.yml`, so a test the linter does not report carries no `//nolint:tparallel`. A plain comment states the reason there when one is needed.

Rejected alternatives:

- `paralleltest`: it also demands `t.Parallel()` in all top-level tests, including pure unit tests that gain nothing.
- No linter: the current state arose because only the testing skill asks for parallel subtests.
- A mechanical rewrite of every subtest: parallel runs of subtests that share state would create flaky tests.

### Replace this change's own verification block when the hook changes

The hook in force while this change is planned rejects `mise run test` in the verification section, and the hook this change delivers rejects every narrow command. The block therefore holds the narrow commands until the task that changes the hook, and that task replaces them with `mise run test`.

Rejected alternative: leaving the section with browser commands only. It would not name the backend packages this change touches, and the proposal review requires that.

## Risks / Trade-offs

- [Each agent's check waits for other agents' full runs] → Full runs take turns through the existing lock, so concurrent agents queue. The user accepted a full run by default. Direct `go test -run` during fixing does not take the lock.
- [A data race reaches CI before anyone sees it locally] → This is the agreed trade-off. CI blocks the release, and the testing skill says how to reproduce the report with `go test -race`.
- [Docker runs out of network address pools] → It happened at `-p=32`, and the bounded limit avoids it in the measured `-p=8` run. Other projects hold about 20 networks on this machine, which leaves little headroom. A failure must be fast: `TestAgentStartup` hung after this error and is fixed in this change. The final report tells the user about `default-address-pools` and the leftover networks, which this change does not touch.
- [The bottleneck at high concurrency is not what it seems] → Docker is the likely cause and was not proved. Implementation chooses limits by measurement, so the choice holds whatever the cause.
- [A crash corrupts the shared test server] → Its contents are disposable, but a plain recreation keeps the anonymous data volume, and an existing template is reused rather than rebuilt (`backend/internal/lib/testdb/template.go:68`). Recovery therefore discards the volume: `docker compose up -d --force-recreate --renew-anon-volumes backend-db-test`, which `docs/testing.md` documents as a step for after a crash, while no test run is active. A test on an empty throwaway server covers this path without discarding the shared server's databases.
- [The compose change recreates a running server] → The first `docker compose up -d` recreates `backend-db-test` and breaks runs active at that moment. With the user's authorization, a task recreates the server with `docker compose up -d backend-db-test` once no test run is active. This happened twice: once for the durability flags and once for the raised connection limit. This command keeps the data volume.
- [Parallel subtests expose shared state] → Fixing that state or keeping the subtests sequential with a reason is part of each package's task.
- [Parallel subtests hit the connection limit] → Each application can open 10 connections. Four module packages with parallel subtests passed on the limit of 300 under a temporary overlay, but a full run with parallel subtests failed with `too many clients`. The compose line therefore sets `max_connections=1000`; a measured run at package limits 8 and 16 peaked at 212 connections.
- [Parallel subtests add load to the CI runners] → `-parallel` defaults to the CPU count, so a 4-CPU runner still runs 4 tests at a time per package, and the 8-CPU runner of the `other` group runs 8 per package across its 8 packages.

## Migration Plan

A task recreates the shared test server on the project's only development machine. CI starts a fresh server on every job and needs nothing. To roll back, revert the commit: the narrow commands, the hook forms and the race flags return together.
