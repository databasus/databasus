# Testing

A backup is worth something only if it restores. Proving that takes real database servers. Databasus tests therefore run the real application against servers in containers and check what a caller observes. Most tests create their data through the HTTP API, the same way a user does.

To check your work, run `mise run test`. To repeat one failing test while fixing it, call its runner directly with a command from [Commands](#commands).

This document describes the kinds of tests, what they share and how runs are organized, in that order. The [testing skill](../.agents/skills/testing/SKILL.md) explains how to write one test, and [architecture.md](architecture.md) describes the code the tests rely on.

## Principles

- A test checks what a caller observes: the backend's HTTP API, an engine's behavior against a real database server or a frontend screen.
- Our own database, modules and services are never mocked. Fakes replace only systems we cannot run locally and failures that are unsafe to cause for real.
- Independent tests run in parallel within and across packages, and alongside runs from other worktrees on the same machine. Tests that change process environment settings or share a bounded engine-version fixture stay sequential.
- A test never takes a shared resource under a fixed name.
- Every check runs locally. The review hook tests are the only manual check, and CI repeats all the others on a clean machine.

## Layers

Paths under `internal/` are inside `backend/`.

| Layer                    | What it checks                                                                                                      | Where it lives                                             | What is real                                                                            |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------- | --------------------------------------------------------------------------------------- |
| Module API tests         | the behavior of each module through its HTTP operations, and that every response matches the contract               | `internal/modules/<name>/*_test.go`, package `<name>_test` | the application, the internal database, HTTP; external systems are fakes                |
| Engine tests             | connecting, privileges, dumping and restoring against every supported server version                                | `internal/engines/<engine>/*_test.go`                      | the database servers in containers                                                      |
| Backup and restore tests | a full backup → restore cycle per engine and version, logical and physical, including SSH tunnels, TLS and failover | `internal/modules/backups/e2e/`                            | the application, source and target servers, storages in containers, the recovery script |
| Agent tests              | the verification agent's verification pipeline, and the agent against a real backend                                         | `internal/agent/**`, `internal/agent/e2e/`                 | the backend from the test harness, Docker or Podman                                     |
| Migration tests          | a migration keeps and converts existing rows, on upgrade and on rollback                                            | `backend/migrations/*_test.go`                             | the internal database, empty or fully migrated as the test asks                         |
| Architecture linter tests | each rule of the linter fails on a violation and passes on clean code                                              | `backend/tools/archlint/*_test.go`                         | the linter on fixture modules; no database                                              |
| Upgrade tests            | an instance of the previous release keeps its data after an upgrade to the current code, and its backups restore    | `backend/e2e/upgrade/`                                     | both images, real data written through the API                                          |
| Filesystem tests         | the image on the host filesystems and permission setups that users run                                              | `e2e/docker-storage/`                                      | Docker on the host                                                                      |
| Frontend tests           | screens for every answer the contract allows                                                                        | `frontend/src/**/*.test.tsx`                               | the components; the network is mocked through `mockApi`, typed by the contract          |

- Module API tests cover most behavior. A module whose API tests outlast one package may split them into test-only packages under its directory, such as `internal/modules/backups/physicalapitests/`.
- A unit test without the application is only for pure logic that has no HTTP entry point or is too slow to enumerate through HTTP. Examples are the scheduler's decision function, retention arithmetic, WAL position math, manifest reconstruction and parsers.

## The test application

Backend tests share one idea: `apptest.New(t)` builds the whole backend with `app.New`, the same function the server uses. A test therefore sees the real routing, middleware, modules and wiring, and it owns everything the instance uses:

- its own internal database from `testdb.Open(t)`;
- a clock that the test moves forward, so schedules and deadlines need no real waiting;
- fakes for external systems: an email recorder, an HTTP server for webhook and chat notifiers and a local directory or a MinIO bucket for storages;
- background jobs that never run on their own, so the test runs each pass explicitly with `RunJobPass`.

Each application owns its directories, encryption key, service state and lifecycle. Several can stay alive in one test process, and shutting down one leaves the others usable.

### Talking to the application

- The test talks to the instance over real HTTP through `apiclient.New(t, url)` and creates its data through the API. The testing skill lists the few cases where direct SQL is allowed.
- `contracttest` fails the test when a response differs from `openapi.yaml` in status, content type, body or field formats. The frontend tests mock responses by the contract, so they rely on this check.
- A response with `format: binary` keeps its body unread, so the caller can stream it.
- Tests of agent operations use `agentclienttest`. It checks responses against `agent.yaml` in the same way.

### Live jobs and real time

- `apptest.WithPhysicalBackupLoops()` runs `backups_physical_scheduler`, `backups_physical_request_scheduler` and `backups_physical_wal_stream_manager` for the test application's lifetime. They use the system clock, and `App.Clock` is nil.
- `apptest.WithAgentBinariesDir()` also selects the system clock, because a real agent needs heartbeat timestamps and rate limits to advance with its process.
- `apptest.WithWalStreamTimings` shortens forced WAL rotation, archive staleness and the queue pause size for one application.
- `RunJobPass` runs one pass of either scheduler job in either mode and returns after dispatching its backups, so the test waits for the outcome through the API.
- The physical cleaner runs only through `RunJobPass("backups_physical_cleaner")`.
- An explicit `backups_physical_wal_stream_manager` pass requires the live jobs, because the manager starts long-lived streamers.
- Physical backup execution has a separate system clock for duration, progress and stall detection. This clock follows the real PostgreSQL process even when the schedule clock is fixed.
- Tests with live jobs seed timestamps with margins of minutes. Negative checks use explicit passes, and live-job checks wait for positive outcomes.

### Restarting and cleanup

- `RestartModules` stops the live jobs, cancels the module runs and waits for them, then runs startup recovery and restarts the live jobs.
- Cleanup stops jobs and module runs before closing the internal database. Work that outlives the cleanup fails the test.

## Isolation

Tests from every worktree on a machine share one Docker daemon, so nothing a test creates may collide with another test's resources.

- **Internal database.** All worktrees share one test PostgreSQL server. A test run never stops it.
  - `testdb.Open(t)` creates a fresh database with a unique name from a template with every migration applied. The template is keyed by a hash of the migrations and built once, under an advisory lock.
  - A passing test drops its database. A failing test keeps it and prints its address. Each test process removes databases older than an hour with no connections.
  - The server runs without durability, so a crash of the server or the host can corrupt it. A plain recreation does not help, because it keeps the data volume and reuses the existing template.
  - After such a crash, and only while no test run is active, discard the volume: `docker compose up -d --force-recreate --renew-anon-volumes backend-db-test`. The next test then builds a new template.
- **Database servers for engines.** Tests reuse a source inside its engine-version scope, and each scope stops its container before starting the next version.
  - Booting a server for every scenario is too slow, and keeping every version at once uses too much memory. Independent top-level scopes can overlap.
  - The servers keep data in memory with durability off.
  - Tests sharing a server give a unique name to every database, role, schema and replication slot.
- **Storages.** Each test uses its own bucket, container or directory.
- **Docker networks.** A network a test creates takes a subnet outside Docker's default address pools, through `WithFreeSubnet` in `internal/lib/testcontainers`. The verification agent creates its own networks from those pools.
- **Everything else.** Ports come from `127.0.0.1:0` or `httptest`, and files from `t.TempDir()`. Containers and processes get unique names and labels, and `t.Cleanup` removes them.
- **Shared sources.** Local and CI preparation never reset shared source servers. Seed tests own their logical and physical source containers. Development compose sources stay available to `mise run dev`, and test cleanup never removes their databases, roles or replication slots.

## Waiting

The [testing skill](../.agents/skills/testing/SKILL.md#waiting) gives the rules for waiting, and the linter rejects `time.Sleep` and `Eventually` in tests. Time that drives behavior comes from the clock in `Deps`, so a test of a schedule or a timeout can move it.

## Verification agent end-to-end tests

These tests run the real agent binary against a real backend, so they prove that the agent and the server agree. `mise run agent:e2e` runs `internal/agent/e2e`. `mise run agent:test` excludes this package and needs neither an engine nor the test database.

- **Setup.** `TestMain` builds the agent once with the server's version and once with an older one for the update scenario. Each scenario runs its own copy through `lib/process`.
- **Coverage.** The suite covers PostgreSQL 12–18, TimescaleDB on 17, schema-limited backups, damaged backup bytes, the disk budget, leftovers, restore container privileges, daemon commands and background self-update.
- **The gateway** on the agent's link is the one allowed stand-in. It forwards traffic and can hold claims, hold the backup stream or lower `maxContainerDiskMb` in a claim response. It never creates a verification or accepts a report in the backend's place.
- **Podman.** `DATABASUS_AGENT_E2E_ENGINE_HOST` selects the agent's engine, and an empty value uses Docker. Source servers stay on Docker. `mise run agent:e2e-podman` runs three scenarios on the rootful socket `/run/podman/podman.sock`, overridden by `E2E_ENGINE_SOCKET`.
- **Concurrency.** Test functions run concurrently and separate runs can overlap, because agents have unique IDs, directories, ports and engine labels. The scenarios of `TestVerifyBackup` run one after another, because each agent takes its network from Docker's default address pools.
- **Cleanup.** Every scenario waits for its containers and networks to disappear, and a detached daemon has an explicit PID cleanup. Failed scenarios print the agent output and logs.
- **Runtime.** A run takes about five minutes with cached images, and the package has a 30-minute timeout.
- **CI** runs the Docker suite in the `agent-e2e` backend group and the Podman subset as its own check.

## Frontend tests

Screen tests render real components and mock only the network, typed by the contract, so they cover every answer the contract allows. The [testing skill](../.agents/skills/testing/SKILL.md#frontend-screen-tests) gives the authoring rules.

- **Environment.** Screen tests select happy-dom with `// @vitest-environment happy-dom` on the first line. Pure logic, API and architecture fixtures keep the default node environment.
- **Rendering.** `renderScreen` from `shared/lib/test` supplies the real providers and an in-memory router, and its options select `locale`, `theme`, `route` and `isStrictMode`. Whole-application tests use the app-level renderer, because App supplies its own providers and browser router.
- **HTTP.** `mockApi` provides typed responses, and `inspectApiRequest` checks the declared operation. The harness rejects all-method handlers and pathname wildcards, and it fails teardown on an unexpected request even when the component catches the rejection. Raw deferred handlers use `createPendingApiResponse`.
- **Cleanup** unmounts before cancelling pending responses, then restores storage, document state, dayjs locale, global overrides and fake clocks. Components own their timers, listeners and late-response guards.
- **Limits.** `frontend/vitest.config.ts` holds the one test limit and `shared/lib/test/conditionWait.ts` the one wait limit, so only a hung test fails.
- **Tooling tests.** Node runner and style fixtures live under `frontend/tooling/*.test.ts`. Playwright scenarios stay outside Vitest discovery. Expected-failure fixtures run in isolated child Vitest processes.

## Browser UI tests

These tests exercise the production build in a real browser, for what happy-dom cannot establish: geometry, focus, fonts and images. Screen tests stay the faster layer for overlapping responses and editing-session policy. `pnpm --dir frontend test:browser --project=ui` needs Docker and no backend services, because the HTTP responses are finite synthetic fixtures.

- **Environment.** The entry point builds a cached image from the digest-pinned Playwright image in `frontend/browser/environment.json`. Each run gets a unique temporary checkout, container name and unprivileged user. It removes only its own container and directory.
- **HTTP.** `frontend/browser/HttpFixture.ts` types response builders through generated operations and records undeclared requests and page errors. Teardown fails even if application code catches an HTTP failure. Only same-origin GET navigation and static resources pass through.
- **Scenarios** live in `frontend/browser/scenarios` and need no external provider account. Non-English assertions read the interface dictionaries.
- **Visual references** live in `frontend/browser/references`. Ordinary runs use `updateSnapshots: 'none'`, so a missing or changed image fails. To replace references on purpose, run `pnpm --dir frontend test:browser:update` and inspect the result.
- **Failure artifacts** go to the unique run directory under `frontend/browser-results/`, and CI uploads them.

## Real-API browser tests

These tests run the production browser against the real application, covering sign-in and workspace resources without HTTP fixtures. `pnpm --dir frontend test:browser --project=e2e` needs the pinned tools, frontend dependencies, `.env` copied from `.env.example`, and the shared test database from `docker compose up -d backend-db-test`.

- **Supervisor.** The public Node supervisor runs only `backend/internal/app/browsertests`. That package builds the frontend, constructs `apptest`, creates its account through the API and starts a loopback gateway on a free port. The gateway proxies `/api/`, serves runtime configuration and supplies SPA fallback.
- **Selection.** The Go package belongs exclusively to the `browser-e2e` selection, so ordinary backend testing needs no frontend runtime. Release versioning depends on this CI job.
- **Lifecycle probes.** `pnpm --dir frontend exec vitest run --config vitest.browser-e2e.config.ts` selects `frontend/tooling/runBrowserE2E.test.ts`, which checks exits, diagnostics and cleanup through failure, interruption and overlapping runs. `frontend/vitest.browser-e2e.config.ts` gives these probes their own 180-second limit, and default `pnpm test` excludes them.
- **Evidence.** E2E traces, video and automatic screenshots are disabled. The supervisor keeps only nonsensitive lifecycle fields in `frontend/browser-results/e2e-<runId>/status.json`. Database retention follows the [Isolation](#isolation) rules.

## Checking work

`mise run test` is the one routine local test command, for developers and agents alike. A change that adds or changes tests or the code they cover owes this run, and the OpenSpec change verification of such a change names it. A change that edits only documentation owes no run.

### What `mise run test` runs

After installing the frontend dependencies from the lockfile, the command runs three concurrent processes and waits for all of them, so one failure does not hide others:

- the backend packages listed in `backend/test-slow-packages.txt`, longest first, with a low package limit;
- all other backend packages, with a higher limit, so they do not wait for slow ones;
- the frontend unit and screen tests.

The upgrade test starts after all three pass.

- Output lines carry the name of their process, and the summary names each process that failed.
- The run allows 30 minutes per package and holds a machine-wide lock, so only one full run goes at a time.
- A package missing from the list runs with the fast packages, so a new package needs no entry. A package belongs in the list when it starts containers or networks or runs for over a minute. A script test fails on an entry the run does not select.
- Migration tests and architecture linter tests run here and, in CI, in the `other` group.

### What runs separately

- The browser suites and the filesystem tests (`mise run test-filesystem`) have their own commands.
- The review hook tests in `.agent-hooks/` run by hand, by an agent that changes the hooks.

### CI

- CI runs every suite except the review hook tests on every pull request, split into shards by measured test duration. Physical backup and WAL streaming tests are the slow groups and get their own shards.
- CI runs backend tests with `-race`, and a data race fails the test. Local runs leave it out, because it makes them several times slower. To reproduce a race, add `-race` to the command that repeats the test.

### Upgrade test

- It starts from the newest release tag by version. CI runs it as the check `test-upgrade` against the image built for the commit. `mise run backend:test-upgrade` runs it alone and builds the image from the checkout unless `DATABASUS_IMAGE` names one.
- After the server assertions it runs the actual verification agent from release 3.64.0, pinned by image digest, through a successful self-update and through a manual replacement of the binary (spec `upgrade-test`).
- Both agent scenarios use the local Unix Docker socket and host networking on Linux. Cleanup removes their containers, labeled restore resources and anonymous volumes.

### Speed

On a Fedora machine with the default btrfs, Docker's storage slows a full run the most. [`docs/machine-setup.md`](machine-setup.md) explains the fix.

## Commands

To repeat one failing test, call its runner directly. This takes no lock and builds no image.

| What                                          | Command                                                                                              |
| --------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| Everything routine                            | `mise run test`                                                                                      |
| One backend test, in `backend/`, see below    | `go test -count=1 -run 'TestName/scenario text' ./internal/modules/<module>/...`                     |
| Verification agent unit tests                 | `mise run agent:test`                                                                                |
| Verification agent end-to-end tests on Docker | `mise run agent:e2e`                                                                                 |
| Verification agent subset on rootful Podman   | `mise run agent:e2e-podman`                                                                          |
| Frontend                                      | `pnpm test` in `frontend/`, or `pnpm test -t 'scenario text'`                                        |
| Frontend production browser UI                | `pnpm --dir frontend test:browser --project=ui`                                                     |
| Frontend production browser with real API      | `pnpm --dir frontend test:browser --project=e2e`                                                    |
| Update browser visual references              | `pnpm --dir frontend test:browser:update`                                                          |
| Upgrade test                                  | `mise run backend:test-upgrade`                                                                      |
| Filesystem tests                              | `mise run test-filesystem`                                                                           |
| Review hook tests, by hand                    | `python3 -m unittest discover -s .agent-hooks -p '*_test.py'`                                        |

The bundled PostgreSQL clients need the libedit shim on Fedora, and parallel test binaries collide on the Ryuk reaper name, so the full command that repeats a backend test is:

```sh
cd backend
source ../scripts/fedora.sh
apply_libedit_shim
TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 -run 'TestName/scenario text' ./internal/modules/<module>/...
```
