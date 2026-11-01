# Testing

This document describes the kinds of tests Databasus has and what each kind checks. It also explains how tests stay isolated from each other and how runs are organised. How to write a single test is in the [testing skill](../.agents/skills/testing/SKILL.md). The code layout the tests rely on is in [architecture.md](architecture.md).

## Principles

- A test checks what a caller observes: the HTTP API of the backend, the behavior of an engine against a real database server, or a screen of the frontend.
- Our own database, modules and services are never mocked. Fakes replace only systems we cannot run locally and failures that are unsafe to cause for real.
- Independent tests run in parallel within and across packages. Tests that intentionally change process environment settings or share a bounded engine-version fixture remain sequential. Single packages and tests also run alongside runs from other worktrees on the same machine; only full runs take turns through a machine-wide lock. A test never takes a shared resource under a fixed name.
- Every check runs locally, and none of them is manual. CI repeats the same tests on a clean machine.

## Layers

Paths under `internal/` are inside `backend/`.

| Layer                    | What it checks                                                                                                      | Where it lives                                             | What is real                                                                            |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------- | --------------------------------------------------------------------------------------- |
| Module API tests         | the behavior of each module through its HTTP operations, and that every response matches the contract               | `internal/modules/<name>/*_test.go`, package `<name>_test` | the application, the internal database, HTTP; external systems are fakes                |
| Engine tests             | connecting, privileges, dumping and restoring against every supported server version                                | `internal/engines/<engine>/*_test.go`                      | the database servers in containers                                                      |
| Backup and restore tests | a full backup → restore cycle per engine and version, logical and physical, including SSH tunnels, TLS and failover | `internal/modules/backups/e2e/`                            | the application, source and target servers, storages in containers, the recovery script |
| Agent tests              | the verification agent's job pipeline, and the agent against a real backend                                         | `internal/agent/**`, `internal/agent/e2e/`                 | the backend from the test harness, Docker or Podman                                     |
| Upgrade tests            | an instance of the previous release keeps its data after an upgrade to the current code, and its backups restore    | `backend/e2e/upgrade/`                                     | both images, real data written through the API                                          |
| Filesystem tests         | the image on the host filesystems and permission setups that users run                                              | `e2e/docker-storage/`                                      | Docker on the host                                                                      |
| Frontend tests           | screens for every answer the contract allows                                                                        | `frontend/src/**/*.test.tsx`                               | the components; the network is mocked through `mockApi`, typed by the contract          |

### Frontend tests

Screen tests select happy-dom with `// @vitest-environment happy-dom` on the first line. `renderScreen` from `shared/lib/test` supplies the real locale and theme providers, AntD configuration and notification context, and an in-memory router. Its options select `locale`, `theme` and `route`, and `isStrictMode` enables replay at the root; its return value includes a configured user-event instance. Rerendering preserves that context and component state. Whole-application tests use the app-level renderer because App supplies its own providers and browser router. Pure logic, API and architecture fixtures keep the default node environment.

The HTTP boundary is mocked through typed `mockApi` responses. Its helpers cover declared JSON and bodyless responses, Problems, network failures, malformed external bodies, and controlled delayed or pending responses. `inspectApiRequest` from its focused `shared/lib/test/inspectApiRequest` entry point checks the declared operation and exposes typed path/body values and query keys; recorded Requests retain headers. Raw MSW handlers support dynamic scenarios. Malformed and delayed responses take named parameter objects. The harness rejects all-method handlers and pathname wildcards; an origin wildcard and named path parameters remain valid. It matches the pathname separately from an origin wildcard, independently records unexpected requests and fails teardown even when the component catches the rejection.

Cleanup unmounts before cancelling controlled pending responses and disposing HTTP handlers. It clears the request ledger and restores storage, document language/title/classes, dayjs locale, global overrides and fake clocks. Raw deferred HTTP handlers also use `createPendingApiResponse`; an unowned handler hits a bounded real-clock cleanup deadline, and final cleanup still clears handlers, listeners, request records and overrides. Components own their timers, listeners and late-response guards; lifecycle tests verify those subscriptions end on unmount and work after StrictMode replay. Expected-failure harness fixtures run in isolated child Vitest processes, check the diagnostic and exit status, and remove their temporary files.

Testing Library and user-event drive controls by accessible roles and names. Tests wait for visible state, an observed request or response completion. Race tests release individual responses in a chosen order; polling tests advance a local fake clock. They never use a fixed pause to give a request time to finish. `frontend/vitest.config.ts` holds the one test limit and `shared/lib/test/conditionWait.ts` the one wait limit, set from the setup file, so a healthy test takes as long as the machine needs and only a hung one fails. The [testing skill](../.agents/skills/testing/SKILL.md#frontend-screen-tests) gives the authoring rules.

### Browser UI tests

`pnpm --dir frontend test:browser --project=ui` builds the production application and exercises its actual routes, providers and domain API facades with finite synthetic HTTP responses. It needs Docker, not backend services. The separate real-API project is described below.

The entry point builds a cached dependency image from the digest-pinned Playwright Linux/Chromium/font image in `frontend/browser/environment.json`. Node and pnpm match mise; Node's archive is hash-checked and dependency installation uses the frozen lockfile. Each run gets a unique temporary checkout, copied dependencies, build/cache outputs, container name and unprivileged runtime user. The production preview binds a free internal port, passes readiness and stops with its owning container. Success, failure and interruption remove only that run's container and temporary directory. The reusable immutable dependency image remains cached; no shared database or container is stopped.

`frontend/browser/HttpFixture.ts` types API response builders through generated operations. It matches explicit methods and finite paths and independently records undeclared requests and page errors. Teardown fails even if application code catches an HTTP failure. Only same-origin GET navigation and static resource requests pass through; fetch/XHR requests require fixtures regardless of their path or file extension. Browser scenarios may share pure response builders with screen tests, but do not replace domain APIs or providers. Non-English assertions read the interface dictionaries.

The suite checks shell/auth/profile/users/instance-settings behavior, retained drafts and errors across language/theme changes, narrow drawer containment and focus restoration, SMTP state and requests, keyboard/disclosure/confirmation focus, scoped accessibility, local fonts and fallback, 320 CSS-pixel reflow, actual text enlargement and WAL plot geometry. Screen tests remain the faster layer for overlapping responses and editing-session policy.

Visual references live in `frontend/browser/references`. Ordinary runs use `updateSnapshots: 'none'`: missing or changed images fail without updating expected images. To deliberately replace references, run `pnpm --dir frontend test:browser:update` in the same pinned environment and inspect the resulting images. Locale, theme, viewport, data, time, reduced motion and font readiness are fixed by the scenarios. Synthetic traces and actual/diff images are copied into the unique run directory under `frontend/browser-results/`; CI uploads failures.

### Real-API browser tests

`pnpm --dir frontend test:browser --project=e2e` runs the production browser against the real application. Install the pinned Go/Node/pnpm tools and frontend dependencies, copy `.env.example` to `.env` in a fresh checkout, and start the shared test database with `docker compose up -d backend-db-test`. Docker supplies the same pinned Chromium environment as the UI suite; no development server is needed.

The public Node supervisor runs only `backend/internal/app/browsertests`. That Go package builds the frontend in its run's temporary directory, constructs `apptest`, creates its account through the API and starts a loopback gateway on a free port. The gateway proxies `/api/`, serves runtime configuration before the application bundle and supplies SPA fallback. Separate browser scenarios check sign-in/sign-out and workspace resources. The resource scenario signs in, creates uniquely named workspaces and a local storage, reads it back through the API, reloads the selected workspace/storage preferences and switches to an empty workspace to verify resource isolation. Only external GitHub metadata is synthetic.

The Go package also runs directly through `mise run backend:test-package ./internal/app/browsertests`. It belongs exclusively to the `browser-e2e` selection and is excluded from `other` and the broad default backend run. Ordinary backend testing requires no frontend runtime. CI provisions this browser job independently, and release versioning depends on its success.

`pnpm --dir frontend exec vitest run --config vitest.browser-e2e.config.ts` selects `frontend/tooling/runBrowserE2E.test.ts`, which drives the actual public command for scenario/readiness failure, interruption and overlapping runs. It checks nonzero exits and diagnostics, gateway/process/container/build-directory cleanup and unrelated-service survival. Process shutdown waits for owned descendants after the driver exits. Container cleanup verifies absence and rejects an unresolved runtime failure. Interruption during cleanup keeps both the public exit and retained status nonzero, even after the test driver succeeds. The E2E CI job runs these probes after the pilot. Default `pnpm test` excludes them and needs no backend. Database retention follows the existing failure policy below. E2E traces, video and automatic screenshots are disabled; temporary Playwright output is deleted. The public supervisor retains only nonsensitive lifecycle fields in `frontend/browser-results/e2e-<runId>/status.json`. Failure probes put synthetic credential, token and authorization-code markers into the browser and check that retained evidence contains none of them.

`browser/scenarios/{storageResources,notifierResources,workspaceSettings}.spec.ts` checks every storage/notifier provider in light French and dark Russian presentation, keyboard selection, accessible controls, retained input and actual 200 percent text at 320 CSS pixels. It also checks workspace read-error recovery. Existing Google authorization and SFTP scenarios retain nested cancel/create/select, credential and live locale/theme coverage. These UI scenarios use contract-shaped HTTP fixtures and require no external provider account.

Node runner/style fixtures live under `frontend/tooling/*.test.ts` and are explicitly included in Vitest. Playwright scenarios remain outside Vitest discovery. The frontend lint task also runs `lint:styles`, which checks parsed presentation against its exact shrinking baseline. The [design document](frontend-design.md#enforcement-and-migration) explains ownership.

A module whose API tests outlast one package may split them into test-only packages under its directory, such as `internal/modules/backups/physicalapitests/`.

Module API tests cover most behavior. A unit test without the application is used only for pure logic that has no HTTP entry point, or that is too slow to enumerate through HTTP. Examples are the scheduler's decision function, retention arithmetic, WAL position math, manifest reconstruction and parsers.

## The test application

`apptest.New(t)` builds the whole backend with `app.New`, the same function the server uses. A test therefore sees the real routing, middleware, modules and wiring. The test owns everything the instance uses:

- its own internal database from `testdb.Open(t)`;
- a clock that the test moves forward, so schedules and deadlines need no real waiting;
- fakes for external systems: an email recorder, an HTTP server for webhook and chat notifiers, and a local directory or a MinIO bucket for storages;
- background jobs that never run on their own; the test runs each pass of a job explicitly.

`apptest.WithPhysicalBackupLoops()` runs the `backups_physical_scheduler` and `backups_physical_wal_stream_manager` jobs for the lifetime of the test application. Both use the system clock, and `App.Clock` is nil in this mode. Without the option, neither job runs automatically. The physical cleaner runs only through `RunJobPass("backups_physical_cleaner")` in either mode. `RunJobPass("backups_physical_scheduler")` runs one scheduler pass in either mode, returning after dispatching its backups; the test waits for their outcome through the API. An explicit `backups_physical_wal_stream_manager` pass requires the live jobs, because the manager starts long-lived streamers.

`RestartModules` stops the live jobs, cancels the module runs and waits for them before startup recovery. An interrupted physical backup records an application shutdown failure; a WAL streamer records its stop and preserves its replication slot and local queue. Startup fails backups left in progress, fails stale streamers, then sweeps orphan backup slots on reachable sources, before restarting the live jobs. Cleanup stops jobs and module runs and waits for them before closing the internal database. Work that outlives the cleanup fails the test.

`apptest.WithAgentBinariesDir()` also selects the system clock and leaves `App.Clock` nil. Tests that run a real agent need heartbeat timestamps and rate limits to advance with that process.

Physical backup execution has a separate injected system clock for elapsed duration, progress, stall detection and the bounded summarizer wait. These follow the real PostgreSQL process even when a test holds the schedule clock fixed.

`apptest.WithWalStreamTimings` shortens forced WAL rotation, archive staleness and the queue pause size for one application. The server keeps the production values. Tests with live physical jobs seed timestamps with margins of minutes. Negative checks use explicit scheduler or cleaner passes; live-job checks wait for positive outcomes.

Each application owns its data, staging and WAL queue directories, encryption key, service state and lifecycle. Several applications can remain alive in one test process, and shutting down one leaves the others usable.

The test talks to the instance over real HTTP through `apiclient.New(t, url)`. The shared response check `contracttest` checks every response against `openapi.yaml` and fails the test on any mismatch. It checks the status, the content type, the body and the field formats. A response whose contract schema has `format: binary` keeps its body unread, so the caller can stream it; the client checks its status, content type and headers. JSON errors from downloads still get full body validation. The frontend tests mock responses by the contract, so they rely on this check. The verification module's tests of agent operations use `agentclienttest`, which checks responses against `agent.yaml` in the same way.

A test creates its data through the API, the same way a user does. The testing skill lists the few cases where direct SQL is allowed.

## Verification agent end-to-end tests

`mise run agent:e2e` runs `internal/agent/e2e` against a real backend from `apptest`. `TestMain` builds the agent once with the server's version and once with an older version for the background update scenario. Each scenario copies its binary into its own temporary directory and runs it through `lib/process`. The backend takes the backup from a source PostgreSQL server, decrypts it on the agent's stream request and stores the agent's report. The tests create users, workspaces, storages, databases and backups and call verification user operations through the generated API client. The agent binary makes its protocol requests through the generated `agentclient`.

The suite covers PostgreSQL 12–18, TimescaleDB on 17, a backup limited to the public schema, damaged backup bytes, the disk budget, leftovers before a retry and at startup, restore container privileges, daemon commands and background self-update. The damaged backup is unencrypted; a helper overwrites its local stored bytes without changing its completed status or file-store records. The disk-budget scenario also checks that PostgreSQL 18's data directory counts toward the limit.

A gateway on the agent's link is the one allowed stand-in in this suite. It forwards requests and responses to the backend, with three opt-in behaviors: hold claims, hold the backup stream, or lower `maxContainerDiskMb` in a successful claim response. These let a scenario arrange leftovers, inspect a running container or exercise a budget below the backend's minimum. It never creates a job or accepts a report in the backend's place. Held requests end when their request or test context ends.

`DATABASUS_AGENT_E2E_ENGINE_HOST` selects the engine for the agent and the test's engine checks; an empty value uses Docker. Only the agent process receives it as `DOCKER_HOST`, so source servers stay on Docker. `mise run agent:e2e-podman` checks access to the rootful socket at `/run/podman/podman.sock` (overridden by `E2E_ENGINE_SOCKET`) and runs three scenarios: an encrypted backup on PostgreSQL 18, a broken backup and restore container privileges. It fails before testing when the socket cannot be opened and prints the command that grants access.

A measured Docker run with race checks took 4 minutes 50 seconds (289.770 seconds), including both binary builds, with the required images already cached. Cold pulls and host contention change that duration; the package keeps the 30-minute timeout of `backend:test-package`.

Independent scenarios run concurrently. Separate runs can overlap: agents have unique IDs, directories, ports and engine labels. Every scenario waits for its containers and networks to disappear after the outcome; cleanup also stops its process and removes labeled resources on failure. A detached daemon has an explicit PID cleanup. Failed scenarios print the agent output and logs. `mise run agent:test` excludes the e2e package and requires neither an engine nor the test database. CI runs the Docker suite in the `agent-e2e` backend group and the Podman subset as its own check.

## Isolation

- **Internal database.** Databases for tests live on one test PostgreSQL server, shared by every worktree on the machine and never stopped by a test run. `testdb.Open(t)` creates a fresh database with a unique name from a template that has every migration applied. The template is built once per set of migrations, under an advisory lock, and it is keyed by a hash of the migrations. A passing test drops its database. A failing test keeps it and prints its address. Each test process removes databases that are older than an hour and have no connections.
- **Database servers for engines.** Tests reuse a source inside its engine-version scope. Each scope stops its container before starting the next version; independent top-level scopes can overlap. Booting a server for every scenario is too slow, and retaining every version at once uses too much memory. The servers keep their data in memory with durability turned off. Tests that share a server give a unique name to every object they create: database, role, schema and replication slot.
- **Storages.** Each test uses its own bucket, container or directory.
- **Everything else.** Ports come from `127.0.0.1:0` or `httptest`, and files come from `t.TempDir()`. Containers and processes that a test starts get unique names and labels, and `t.Cleanup` removes them.

## Waiting

Tests never wait for a fixed time. They wait for a condition with `testwait.Until`, up to the test's overall timeout, and the linter rejects `time.Sleep` and `Eventually` in tests. Time that drives behavior comes from the clock in `Deps`. A test of a schedule or a timeout therefore moves the clock instead of sleeping. The testing skill describes how to write a negative check.

## Choosing the tests to run

While working, run the package or the test you need right now. Before handing the work over, run the tests of every package the change touched and of the packages that test the same feature, with `mise run backend:test-package <packages>` and `pnpm test` in `frontend/`. A package that tests the same feature is one whose tests exercise the changed behavior through the API, such as the backup and restore tests for a change in an engine.

`mise run backend:test-package` and the full run allow 30 minutes per package, because an unsharded physical API package can exceed 15 minutes. The full run is `mise run test`. It holds a machine-wide lock, so only one full run goes at a time and the others wait. Single packages and single tests run without the lock. CI runs everything on every pull request.

`mise run backend:test-group physical-api-1-of-3 physical-api-2-of-3 physical-api-3-of-3` runs the physical API suite with the same selections as CI. The selected shards run concurrently in separate processes, each with race checks and a 40-minute timeout. Each process stops after its first failed test, as CI does. Use a single shard name to repeat that selection. The task also accepts the `physical` and `logical` shards; it rejects unsharded groups and `other`.

## Runs

- Backend tests run with `-race`, and a data race fails the test.
- CI runs the same groups as the full run, split into shards by the measured duration of tests. Physical backup tests and WAL streaming tests are the slow groups and get their own shards.
- The upgrade test is part of `mise run test`. Filesystem tests run separately through `mise run test-filesystem`. CI runs both on every pull request.
- The upgrade test starts from the newest release tag by version. It runs in `mise run test` after the backend packages pass, and in CI as the check `test-upgrade` against the image CI built for the commit. `mise run backend:test-upgrade` runs it alone and builds the image from the checkout unless `DATABASUS_IMAGE` names one.

Local and CI preparation never reset shared source servers. Seed tests own their logical and physical source containers. Development compose sources remain available to `mise run dev`; test cleanup never removes their databases, roles or replication slots.

## Commands

| What                                          | Command                                                                                              |
| --------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| Given backend packages                        | `mise run backend:test-package <packages>`                                                           |
| Selected CI shards, concurrently              | `mise run backend:test-group <shard>...`                                                             |
| Default backend and upgrade suite             | `mise run test`                                                                                      |
| One backend package                           | `go test -race -count=1 ./internal/modules/<module>/...` in `backend/`                               |
| One backend test                              | `go test -race -count=1 -run 'TestName/scenario text' ./internal/modules/<module>/...` in `backend/` |
| Verification agent unit tests                 | `mise run agent:test`                                                                                |
| Verification agent end-to-end tests on Docker | `mise run agent:e2e`                                                                                 |
| Verification agent subset on rootful Podman   | `mise run agent:e2e-podman`                                                                          |
| Frontend                                      | `pnpm test` in `frontend/`, or `pnpm test -t 'scenario text'`                                        |
| Frontend production browser UI                | `pnpm --dir frontend test:browser --project=ui`                                                     |
| Frontend production browser with real API      | `pnpm --dir frontend test:browser --project=e2e`                                                    |
| Update browser visual references              | `pnpm --dir frontend test:browser:update`                                                          |
| Upgrade test                                  | `mise run backend:test-upgrade`                                                                      |
| Filesystem tests                              | `mise run test-filesystem`                                                                           |
