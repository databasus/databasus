# Testing

A backup is worth something only if it restores. Proving that takes real database servers. Databasus tests therefore run the real application against servers in containers and check what a caller observes. Most tests create their data through the HTTP API, the same way a user does.

Choose checks for the scope of the change as [the project rules](../AGENTS.md#tests) require. [Checking work](#checking-work) describes the selection. To repeat one failing test while fixing it, call its runner directly with a command from [Commands](#commands).

This document describes the kinds of tests, what they share and how runs are organized, in that order. The [testing skill](../.agents/skills/testing/SKILL.md) explains how to write one test, and [architecture.md](architecture.md) describes the code the tests rely on.

## Principles

- A test checks what a caller observes: the backend's HTTP API, an engine's behavior against a real database server or a frontend screen.
- Our own database, modules and services are never mocked. Fakes replace only systems we cannot run locally and failures that are unsafe to cause for real.
- Independent tests run in parallel within and across packages, and alongside runs from other worktrees on the same machine. Tests that change process environment settings or share a bounded engine-version fixture stay sequential.
- A test never takes a shared resource under a fixed name.
- Every check runs locally. CI repeats every suite except the review hook tests and the interruption test of the browser end-to-end suite on a clean machine; `mise run test-full` includes both locally.

## Layers

Paths under `internal/` are inside `backend/`. The table distinguishes the test surfaces and the real systems each needs.

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
| Browser end-to-end suite | the critical path, sign-in and workspace resources through the screens of the application image | `e2e/browser/` | the image, its internal database, a PostgreSQL server, Chromium |
| Frontend tests           | screens for every answer the contract allows                                                                        | `frontend/src/**/*.test.tsx`                               | the components; the network is mocked through `mockApi`, typed by the contract          |

- Module API tests cover most behavior. A module whose API tests outlast one package may split them into test-only packages under its directory, such as `internal/modules/backups/physicalapitests/`.
- A unit test without the application is only for pure logic that has no HTTP entry point or is too slow to enumerate through HTTP. Examples are the scheduler's decision function, retention arithmetic, WAL position math, manifest reconstruction and parsers.

## The test application

Backend tests share one idea: `apptest.New(t)` builds the whole backend with `app.New`, the same function the server uses. A test therefore sees the real routing, middleware, modules and wiring, and it owns everything the instance uses:

- its own internal database from `testdb.Open(t)`;
- a clock that the test moves forward, so schedules and deadlines need no real waiting;
- fakes for external systems: an email recorder, an HTTP server for webhook and chat notifiers and a local directory or a MinIO bucket for storages;
- background jobs that stay stopped by default, so the test runs each pass explicitly with `RunJobPass`. Scheduling tests can start selected jobs with `StartJobRunner`, which the harness stops before closing the application. `WithStorageFileTimings` lets a cleanup latency test retain the production interval, so a short test interval cannot hide a missing wake signal.

Each application owns its directories, encryption key, service state and lifecycle. Several can stay alive in one test process, and shutting down one leaves the others usable.

### Talking to the application

- The test talks to the instance over real HTTP through `apiclient.New(t, url)` and creates its data through the API. The testing skill lists the few cases where direct SQL is allowed.
- `contracttest` fails the test when a response differs from `openapi.yaml` in status, content type, body or field formats. The frontend tests mock responses by the contract, so they rely on this check.
- A response with `format: binary` keeps its body unread, so the caller can stream it.
- Tests of agent operations use `agentclienttest`. It checks responses against `agent.yaml` in the same way.

### Live jobs and real time

Live physical backup jobs and tests serving a real agent use the system clock rather than the movable test clock. The agent's heartbeats and rate limits must advance with its process. Physical backup duration, progress and stall detection also follow real time, even when the schedule clock is fixed. The [test application options](../backend/internal/app/apptest/options.go) select these modes.

- A scheduler pass returns after dispatching backups; wait for their outcome through the API.
- The physical cleaner runs only through an explicit pass. Streamer-management passes require live-job mode, because they start long-lived streamers.
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
- **Shared sources.** Local and CI preparation never reset shared source servers. Seed tests own their logical and physical source containers. Development compose sources stay available to `mise run run`, and test cleanup never removes their databases, roles or replication slots.

## Waiting

The [testing skill](../.agents/skills/testing/SKILL.md#waiting) defines condition-wait limits and their exceptions; fixed sleeps and `Eventually` are rejected by lint. Move the injected clock when time drives behavior. Go's timeout covers the whole package, so a call that hangs outside a bounded wait can still consume that timeout.

## Verification agent end-to-end tests

These tests run the real agent binary against a real backend, proving that they agree on the contract and can restore a backup. The separate agent unit-test command needs neither a container engine nor the test database. [Commands](#commands) lists both runs and the rootful Podman subset; source servers still use Docker in that subset.

- **Allowed stand-in.** A forwarding gateway may delay traffic or reduce a claimed resource budget. It never creates verifications or accepts reports in the backend's place.
- **Concurrency.** Tests and separate runs may overlap through unique resources. The scenarios of `TestVerifyBackup` stay sequential because each agent allocates networks from Docker's default address pools.
- **Cleanup.** Each scenario waits for its containers and networks to disappear and explicitly cleans up detached daemon PIDs. Failed scenarios print agent output and logs.

## Frontend tests

Screen tests render real components and mock only the network, typed by the contract, so they cover every answer the contract allows. The [testing skill](../.agents/skills/testing/SKILL.md#frontend-screen-tests) gives the authoring rules.

- **Environment.** Screen tests select happy-dom with `// @vitest-environment happy-dom` on the first line. Pure logic, API and architecture fixtures keep the default node environment.
- **Rendering.** `renderScreen` from `shared/lib/test` supplies real providers, an in-memory router and options for locale, theme, route and Strict Mode. Whole-application tests use the app-level renderer, because App supplies its own providers and browser router.
- **HTTP.** The harness supplies `mockApi` for contract-typed responses and helpers for request inspection. It rejects all-method handlers and pathname wildcards, and fails teardown on an unexpected request even when the component catches the rejection. Use its controlled responses for overlapping requests.
- **Cleanup** unmounts before cancelling pending responses, then restores storage, document state, dayjs locale, global overrides and fake clocks. Components own their timers, listeners and late-response guards.
- **Limits.** Test and condition-wait limits are shared across the suite, rather than selected by each test.
- **Tooling tests.** Node runner and style fixtures live under `frontend/tooling/*.test.ts`. Playwright scenarios stay outside Vitest discovery. Expected-failure fixtures run in isolated child Vitest processes.

## Browser UI tests

These tests exercise the production build in a real browser, for what happy-dom cannot establish: geometry, focus, fonts and images. Screen tests stay the faster layer for overlapping responses and editing-session policy. The [browser UI command](#commands) needs Docker and no backend services, because HTTP responses are finite synthetic fixtures.

- **Environment.** Each run uses a digest-pinned browser image, an unprivileged user and its own temporary checkout and container. Cleanup removes only that run's resources.
- **HTTP.** Fixtures follow generated operations and fail teardown on undeclared requests or page errors, even when application code catches the HTTP failure. Only same-origin GET navigation and static resources pass through.
- **Scenarios** live in `frontend/browser/scenarios` and need no external provider account. Non-English assertions read the interface dictionaries.
- **Visual references** live in `frontend/browser/references`. A missing or changed image fails an ordinary run. Use the explicit update command from [Commands](#commands) to replace references, then inspect the result.
- **Failure artifacts** go to the unique run directory under `frontend/browser-results/`, and CI uploads them.

## Browser end-to-end suite

These tests exercise the application image through its screens, without importing backend or frontend code or using paths into those trees from scenarios and fixtures. They need Docker and the pinned tools, but no `.env`, running application or shared test database. The standalone command tests `DATABASUS_IMAGE` when set, or builds an image from the checkout; CI requires an explicit image.

- **Isolation.** Each run has its own application, source database and browser environment. Runs may overlap without taking a lock or conflicting with development ports.
- **Evidence.** The critical path creates a database connection, takes a backup and restores it. It compares source and restored rows with the rows it wrote; screen status alone cannot prove success.
- **Cleanup.** Every outcome removes all resources the run created, including its image if it built one. Removal failures fail the run and name the resource. Startup also removes leftovers whose supervisor has died. Failed runs retain no resources and print container log tails with passwords and bearer tokens masked.
- **Checks of the suite.** Separate commands cover its unit logic and lifecycle failures, interruption and overlapping runs. CI runs these checks against the candidate image and skips probes requiring an image build. Interrupting the lifecycle command itself remains a manual check, described in [Checking work](#checking-work).

## Checking work

Follow [the project rules](../AGENTS.md#tests) when deciding whether a change needs `mise run test`. For a localized functional change, run the affected package or screen tests directly. For a presentation change, run the frontend checks and verify the affected layout in a browser. Documentation-only changes need no test run. OpenSpec change verification names the commands selected for the change.

`mise run test-full` runs every test that needs only the usual Docker development environment. It includes the routine run, the end-to-end packages of the verification agent and of backups, the upgrade test, frontend browser UI, the browser end-to-end suite and its unit, lifecycle and interruption checks, and the script and review hook tests. Podman and filesystem tests remain separate because they require additional host setup. An agent [does not start the full command on its own](../AGENTS.md#tests).

Whether checking a change with targeted tests or with `mise run test`, an agent also runs by hand the specific long or end-to-end suites its change touches:

- A change to scenarios in `e2e/browser/scenarios` or to the `Dockerfile` runs `mise run e2e:browser`, and the change verification names a scenario change's command.
- A change to tests in `backend/internal/agent/e2e` or to the verification agent's code runs `mise run agent:e2e`, or the Go test command for the affected tests.
- A change to tests in `backend/internal/modules/backups/e2e` runs the Go test command for the affected tests.
- A change to `backend/e2e/upgrade` runs `mise run backend:test-upgrade`.

A break in the backup pipeline can pass `mise run test`. When a change touches that pipeline, run the affected tests in `backend/internal/modules/backups/e2e` by hand.

A change to the browser end-to-end suite's own code owes its commands too:

- A change to its code outside the scenarios runs `mise run e2e:unit`.
- A change to how a run starts or cleans up also runs `mise run e2e:lifecycle`.
- A change to the lifecycle command also runs `uv run --project e2e/browser pytest e2e/browser/lifecycle_interruption`.

### What `mise run test` runs

After installing the frontend dependencies from the lockfile, the command runs three concurrent processes and waits for all of them, so one failure does not hide others:

- the backend packages listed in `backend/test-slow-packages.txt`, longest first, with a low package limit;
- all other backend packages, with a higher limit, so they do not wait for slow ones;
- the frontend unit and screen tests.

`mise run test-full` adds the end-to-end packages to the backend selection. After the backend and frontend processes pass, it runs these checks in order and stops if one fails:

1. Script tests, review hook tests and browser e2e unit tests.
2. Frontend browser UI.
3. Upgrade test.
4. Browser end-to-end scenarios.
5. Browser e2e lifecycle checks, then their interruption test.

The image and browser suites run sequentially because they disturb each other's timing under load.

- Output lines carry the name of their process, and the summary names each process that failed.
- The run allows 15 minutes per package and holds a machine-wide lock, so only one `mise run test` or `mise run test-full` goes at a time.
- A package missing from the list runs with the fast packages, so a new package needs no entry. A package belongs in the list when it starts containers or networks or runs for over a minute. A script test fails on an entry that neither `mise run test` nor `mise run test-full` selects.
- Migration tests and architecture linter tests run here too.

### What runs separately

- The rootful Podman subset (`mise run agent:e2e-podman`) needs its socket configured.
- The filesystem suite (`mise run test-filesystem`) needs host support for NFS/CIFS and their mount clients.
- Every suite can also run directly when checking a change that touches it; the [commands](#commands) below include hidden CI tasks, which remain callable by name.

### CI

- CI runs every suite except the review hook tests and the interruption test of the browser end-to-end suite on every pull request, split into shards by measured test duration. Physical backup and WAL streaming tests are the slow groups and get their own shards.
- CI runs backend tests with `-race`, and a data race fails the test. Local runs leave it out, because it makes them several times slower. To reproduce a race, add `-race` to the command that repeats the test.

### Upgrade test

The upgrade test starts from the newest release tag by version and checks the image built from the checkout, unless `DATABASUS_IMAGE` names one. After the server assertions, it checks self-update and manual replacement using an actual older agent pinned by image digest. The `upgrade-test` spec defines that fixture and its acceptance criteria.

The agent scenarios require a local Unix Docker socket and Linux host networking. Cleanup removes their containers, labeled restore resources and anonymous volumes.

#### Manual agent recovery

Self-update needs write access to the directory holding the agent's executable. Permission or read-only filesystem errors in its update logs, together with no recent accepted heartbeat on the server, indicate a failed replacement. The upgrade test verifies this recovery procedure:

1. Stop the agent. For its built-in daemon, run the installed binary's `stop` command by absolute path, from its existing working directory as its normal account. For a service or container, stop its supervisor instead.
2. Replace the executable using an account permitted to write to its directory. Download from the connected server to a temporary file, choosing `arch=amd64` or `arch=arm64` for the agent's machine. For example, replacing the server address and installed path:

   ```sh
   curl --fail --location --output /opt/databasus-agent/verification-agent.new \
     "https://your-databasus-host/api/v1/system/verification-agent?arch=amd64" && \
   chmod 755 /opt/databasus-agent/verification-agent.new && \
   mv -f /opt/databasus-agent/verification-agent.new /opt/databasus-agent/verification-agent
   ```

   For a container, deploy the updated image instead.
3. Preserve `databasus-verification.json`, its token and resource settings. Restart the daemon with the installed binary's absolute path and `start` command, as the same account from the existing working directory, without resupplying setup credentials or resource flags. Restart a service or container through its supervisor with the existing settings.

Confirm accepted server contact and a completed backup verification. A successful local `version` or `status` command alone does not prove recovery, and no new token is needed.

### Speed

Container creation and temporary-file I/O can limit local test speed. [Preparing a machine](machine-setup.md) describes storage considerations without requiring a particular host setup.

## Commands

To repeat one failing test, call its Go or frontend runner directly. This takes no lock and builds no image.

| What                                          | Command                                                                                              |
| --------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| Backend and frontend test suites              | `mise run test`                                                                                      |
| All tests for the usual Docker development environment | `mise run test-full`                                                                                      |
| One backend test, in `backend/`, see below    | `go test -count=1 -run 'TestName/scenario text' ./internal/modules/<module>/...`                     |
| Verification agent unit tests                 | `mise run agent:test`                                                                                |
| Verification agent end-to-end tests on Docker | `mise run agent:e2e`                                                                                 |
| Verification agent subset on rootful Podman   | `mise run agent:e2e-podman`                                                                          |
| Frontend                                      | `pnpm test` in `frontend/`, or `pnpm test -t 'scenario text'`                                        |
| Frontend production browser UI                | `pnpm --dir frontend test:browser --project=ui`                                                     |
| Update browser visual references              | `pnpm --dir frontend test:browser --project=ui --update-references`                                                          |
| Upgrade test                                  | `mise run backend:test-upgrade`                                                                      |
| Filesystem tests                              | `mise run test-filesystem`                                                                           |
| Browser end-to-end suite alone                | `mise run e2e:browser`                                                                               |
| Its parts that need no Docker daemon          | `mise run e2e:unit`                                                                                  |
| Its lifecycle probes                          | `mise run e2e:lifecycle`                                                                             |
| Review hook tests                    | `python3 -m unittest discover -s .agent-hooks -p '*_test.py'`                                        |

The bundled PostgreSQL clients need the libedit shim on Fedora, and parallel test binaries collide on the Ryuk reaper name, so the full command that repeats a backend test is:

```sh
cd backend
source ../scripts/fedora.sh
apply_libedit_shim
TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 -run 'TestName/scenario text' ./internal/modules/<module>/...
```
