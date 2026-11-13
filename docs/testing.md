# Testing

A backup is worth something only if it restores. Proving that takes real database servers. Databasus tests therefore run the real application against servers in containers and check what a caller observes. Most tests create their data through the HTTP API, the same way a user does.

Choose checks for the scope of the change as [the project rules](../AGENTS.md#tests) require. [Checking work](#checking-work) describes the selection. To repeat one failing test while fixing it, call its runner directly with a command from [Commands](#commands).

This document describes the kinds of tests, what they share and how runs are organized, in that order. The [testing skill](../.agents/skills/testing/SKILL.md) explains how to write one test, and [architecture.md](architecture.md) describes the code the tests rely on.

## Principles

- A test checks what a caller observes: the backend's HTTP API, an engine's behavior against a real database server or a frontend screen.
- Our own database, modules and services are never mocked. Fakes replace only systems we cannot run locally and failures that are unsafe to cause for real.
- Independent tests run in parallel within and across packages, and alongside runs from other worktrees on the same machine. Tests that change process environment settings or share a bounded engine-version fixture stay sequential. Independent suites of one run start together. Runs aim to saturate the machine, as [Resource usage and parallelism](#resource-usage-and-parallelism) describes.
- A test that passes alone but fails beside other suites has a defect in the test. It waits for the condition it depends on or gets a time limit that matches the operation; it is not moved out of the concurrent run or retried.
- A test never takes a shared resource under a fixed name.
- Every check runs locally. CI repeats every suite except the review hook tests and the interruption test of the browser end-to-end suite on a clean machine; `mise run test-full` includes both locally.

## Layers

Paths under `internal/` are inside `backend/`. The table distinguishes the test surfaces and the real systems each needs.

| Layer                     | What it checks                                                                                                      | Where it lives                                             | What is real                                                                                             |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| Module API tests          | the behavior of each module through its HTTP operations, and that every response matches the contract               | `internal/modules/<name>/*_test.go`, package `<name>_test` | the application, the internal database, HTTP; external systems are fakes                                 |
| Engine tests              | connecting, privileges, dumping and restoring against every supported server version                                | `internal/engines/<engine>/*_test.go`                      | the database servers in containers                                                                       |
| Backup and restore tests  | a full backup → restore cycle per engine and version, logical and physical, including SSH tunnels, TLS and failover | `internal/modules/backups/e2e/`                            | the application, source and target servers, storages in containers, the recovery script                  |
| Agent tests               | the verification agent's verification pipeline, and the agent against a real backend                                | `internal/agent/**`, `internal/agent/e2e/`                 | the backend from the test harness, Docker or Podman                                                      |
| Migration tests           | a migration keeps and converts existing rows, on upgrade and on rollback                                            | `backend/migrations/*_test.go`                             | the internal database, empty or fully migrated as the test asks                                          |
| Architecture linter tests | each rule of the linter fails on a violation and passes on clean code                                               | `backend/tools/archlint/*_test.go`                         | the linter on fixture modules; no database                                                               |
| Upgrade tests             | an instance of the previous release keeps its data after an upgrade to the current code, and its backups restore    | `backend/e2e/upgrade/`                                     | both images, real data written through the API                                                           |
| Filesystem tests          | the image on the host filesystems and permission setups that users run                                              | `e2e/docker-storage/`                                      | Docker on the host                                                                                       |
| Browser end-to-end suite  | the critical path, sign-in and workspace resources through the screens of the application image                     | `e2e/browser/`                                             | the image, its internal database, PostgreSQL 18, Chromium, an isolated recovery runner and Docker daemon |
| Frontend tests            | screens for every answer the contract allows                                                                        | `frontend/src/**/*.test.tsx`                               | the components; the network is mocked through `mockApi`, typed by the contract                           |

- Module API tests cover most behavior. A module whose API tests outlast one package may split them into test-only packages under its directory, such as `internal/modules/backups/physicalapitests/`.
- A unit test without the application is only for pure logic that has no HTTP entry point or is too slow to enumerate through HTTP. Examples are the scheduler's decision function, retention arithmetic, WAL position math, manifest reconstruction and parsers.

## The test application

Backend tests build the same application composition as the server, so they exercise real routing, middleware, modules and wiring. Each instance owns an isolated internal database, directories, encryption key, service state and lifecycle. External systems use the substitutes permitted by the [testing skill](../.agents/skills/testing/SKILL.md#unit-tests-and-fakes).

A movable clock makes schedules deterministic. Background jobs stay stopped by default: tests advance time and run individual passes explicitly. Tests of live scheduling can start selected jobs without replacing the production mechanism or shortening an interval whose latency is under test. The [test application](../backend/internal/app/apptest) owns these setup and lifecycle options.

### Talking to the application

Tests create data and observe outcomes through real HTTP. Contract-checking clients validate status, content type, bodies and field formats for both the user and agent APIs. Binary responses remain streams rather than being read eagerly. The testing skill defines the few cases where direct SQL is allowed.

### Live jobs and real time

Live physical backup jobs and tests serving a real agent use system time for execution, heartbeats and rate limits. Physical duration, progress and stall detection follow real time even when the schedule clock is fixed.

A scheduler pass dispatches work; tests wait for completion through the API. Physical cleanup still requires explicit passes, while streamer management needs live-job mode. Seed timestamps with enough margin for live processing. Use explicit passes for negative checks and wait for positive outcomes from live jobs.

### Restarting and cleanup

A restart drains jobs and module work before startup recovery and resumption. Cleanup stops all instance-owned work before closing its dependencies; work that outlives cleanup fails the test. Several test applications can coexist, and stopping one must leave the others usable.

## Isolation

Tests from every worktree on a machine share one Docker daemon, so nothing a test creates may collide with another test's resources.

- **Internal database.** All worktrees share one test PostgreSQL server. A test run never stops it.
  - Each test gets a uniquely named database from a template matching its migrations. Template creation is coordinated across concurrent runs.
  - Passing tests drop their databases; failing tests retain them and print their addresses. Stale, unused databases are collected automatically.
  - The server runs without durability, so a crash of the server or the host can corrupt it. A plain recreation does not help, because it keeps the data volume and reuses the existing template.
  - After such a crash, and only while no test run is active, discard the volume: `docker compose up -d --force-recreate --renew-anon-volumes backend-db-test`. The next test then builds a new template.
- **Database servers for engines.** Tests reuse a source inside its engine-version scope, and each scope stops its container before starting the next version.
  - Booting a server for every scenario is too slow, and keeping every version at once uses too much memory. Independent top-level scopes can overlap.
  - The servers keep data in memory with durability off.
  - Tests sharing a server give a unique name to every database, role, schema and replication slot.
- **Storages.** Each test uses its own bucket, container or directory.
- **Docker networks.** Every network a test creates itself takes a random `/24` subnet from the test network range `10.160.0.0/11`, outside Docker's default address pools, and retries on an overlap. This holds for backend tests, the upgrade test and browser end-to-end runs. The verification agent cannot choose a range and allocates its restore networks from the default pools, so those pools stay free for it. The backend [container fixtures](../backend/internal/lib/testcontainers) and the browser suite's [supervisor](../e2e/browser) each follow the convention, because the browser suite imports no backend code.
- **Images.** An image a run builds has a tag no concurrent run uses, and the run removes it when it ends. The full run builds `databasus-full-test:local`; full runs take turns through their lock, so the tag cannot collide, and the next full run replaces an image a killed run left. The direct upgrade command builds `databasus-upgrade-test:<run ID>`. An image named through `DATABASUS_IMAGE` is used as given and never removed.
- **Everything else.** Ports come from `127.0.0.1:0` or `httptest`, and files from `t.TempDir()`. Containers and processes get unique names and labels, and `t.Cleanup` removes them.
- **Shared sources.** Local and CI preparation never reset shared source servers. Seed tests own their logical and physical source containers. Development compose sources stay available to `mise run run`, and test cleanup never removes their databases, roles or replication slots.

## Resource usage and parallelism

Tests should keep the machine busy. A run that waits while CPU, memory and disk sit idle has a defect to fix.

### Reference machine

Local runs assume a developer machine with:

- 64 GB of memory, with up to 32 GB available exclusively for tests across the entire run, whether it runs selected packages, `mise run test` or `mise run test-full`; the rest serve compilation, the editor and the desktop;
- 32 logical CPUs (vCPUs), which set Go's default parallel test limit;
- free disk space that grows with concurrency, both on Docker's storage and in the temporary directory.

Disk space needs a wide margin rather than an exact figure. Durable fixtures write data and WAL, and tests keep backups in temporary directories. When Docker keeps its data in an image file on the root filesystem, that file grows into the root filesystem's free space. If that space runs out, the run fails and the host can stall or restart.

### What bounds a run

- **CPU** does not. Load above the core count is acceptable: most tests wait on containers and the network, so the cores stay busy only when many tests run at once.
- **Free memory and disk space** do.
- **No test or package caps its own concurrency with a constant.** The runner's limits decide. When a resource requires a limit, the runner sets it from a measurement and states the reason beside it.

### Time targets

| On the reference machine | Time |
|---|---|
| Target for one package, or a few packages run together | about 5 minutes |
| Limit | 10 minutes |
| Runner timeout per package | 15 minutes, so an overrun fails instead of hiding |
| `mise run test-full`, with test images present and a warm build cache | about 10 minutes; limit 20 minutes |

A test that exists to observe real time may take longer. It runs beside the others, so it does not stretch the critical path.

### Parallelism in local runs

Go runs tests marked with `t.Parallel()` concurrently inside one test binary, up to `-parallel` at once. By default that is the number of logical CPUs. Every test owns its internal database, containers, ports and files, as [Isolation](#isolation) describes, so tests need no coordination beyond the shared engine-version scopes.

Each command adds its own layer on top:

- **`mise run test`** runs three suites at once: the slow backend packages, the other backend packages and the frontend tests.
  - Slow packages mostly wait on containers and get a lower package limit. The group already lasts as long as its longest package, so a higher limit only adds memory.
  - Slow packages also get a parallel test limit. Their tests start containers, and the Docker daemon they share is the bottleneck: with one test per logical processor, dozens of Patroni clusters start at once, container stops and network removals outlast their deadlines, and the group fails and leaks containers.
  - Other packages mostly compute and get a higher limit.
  - The [test runner](../scripts/test.sh) owns the numbers.
- **`mise run test-full`** adds the end-to-end packages to the slow group and starts every other suite beside them, each as soon as what it needs is ready:
  - the [crashed-run probes](#browser-end-to-end-suite), script tests, review hook tests, the browser suite's unit tests and browser UI start at once;
  - the application image builds once at the start, then the upgrade test runs against it;
  - the crashed-run probes and then the lifecycle interruption checks run while no other browser run of the command is active. They run one probe after another and no other browser run may overlap them, so they go first, beside the backend packages and the suites that start no browser run;
  - after them, the browser end-to-end scenarios and the other lifecycle probes run together, the probes split into parts that several processes run, because one process runs its probes one after another.

  A failed suite stops no other suite; only a failed image build keeps the suites that run the image from starting. The run ends with a summary of each suite's outcome and duration, the build's duration and the total.
- **Browser UI** takes its worker count from its runner: half the host's logical processors rounded up, and at least two, locally, and two per shard in CI.
- **A direct package run** uses Go's defaults and gets the whole machine.

### Parallelism in CI

CI combines parallel jobs with parallel tests:

- Each group or shard runs on its own runner. The longest check decides the wait, not the sum of all checks, and a slow runner holds back only its own shard.
- Runners are available up to 16 vCPUs and 32 GB RAM. There is substantial aggregate concurrency available; use a 16-vCPU runner when a suite benefits from it.
- Small jobs use small runners so they consume less of the paid runners' shared vCPU concurrency quota. Larger runners and higher test parallelism are chosen from measured runtime and peak memory, within the runner's 32 GB budget. Two concurrent tests is not a general limit.
- A long package is split into several top-level tests. Their [recorded weights](../backend/test-shard-weights.txt) let the shards balance it.

[CI](#ci) describes groups and shards.

### Storage speed

Container creation and small synchronous writes also limit speed. [Preparing a machine](machine-setup.md) describes storage considerations without requiring a particular host setup.

## Waiting

The [testing skill](../.agents/skills/testing/SKILL.md#waiting) defines condition-wait limits and their exceptions; fixed sleeps and `Eventually` are rejected by lint. Move the injected clock when time drives behavior. Go's timeout covers the whole package, so a call that hangs outside a bounded wait can still consume that timeout.

## Physical schema recovery tests

Physical recovery tests start the recovered server and verify data, schema and generated-ID writes. They cover full and incremental chains and point-in-time recovery across transaction boundaries. PostgreSQL permits sequence gaps, so recovery checks prevent ID reuse without assuming consecutive allocations. The [physical suites](../backend/internal/modules/backups/e2e/postgresqlphysical) and their [shared fixtures](../backend/internal/app/apptest/physicaltest) own the detailed scenario matrix.

Run the schema and sequence scenarios from the repository root with the pinned tools and the [Fedora preparation](#commands):

```sh
source scripts/fedora.sh
apply_libedit_shim
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -run '^TestRestorePhysicalSchemaAndSequences' \
  ./internal/modules/backups/e2e/postgresqlphysical/pg17 \
  ./internal/modules/backups/e2e/postgresqlphysical/pg18
```

## Patroni physical backup tests

The Patroni suite exercises physical backup and recovery through a real highly available PostgreSQL source. Unlike the reusable engine servers, these sources use disk-backed data and durable settings. Every scenario owns its resources and tests cleanup on failure and interruption. Scenarios are grouped into top-level tests by family and PostgreSQL major, so CI can spread them across shards. The runner's parallel limit alone decides how many clusters run at once.

Prepare Docker, the shared test PostgreSQL server, pinned tools and client bundles as described in [Commands](#commands). The initial fixture build needs network access; later runs can reuse its build cache. Missing tools and build failures fail the suite rather than skip scenarios. The [fixture README](../backend/internal/lib/testcontainers/testdata/patroni/README.md) owns dependency and build details.

Run the complete matrix from the repository root:

```sh
source scripts/fedora.sh
apply_libedit_shim
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout 15m \
  ./internal/modules/backups/e2e/postgresqlphysical/patroni
```

Use Go's `-run` filter to repeat an individual scenario. [PostgreSQL with Patroni](postgresql-patroni.md) explains endpoint routing, external slot ownership and recovery gaps.

## CloudNativePG physical backup tests

The CloudNativePG (CNPG) suite runs the Patroni scenario families on PostgreSQL 17 and 18 against sources that the CNPG operator manages in Kubernetes. A package run starts its own disposable single-node Kubernetes clusters, each a k3s server in a privileged Docker container, and its scenarios share them. Every scenario creates a namespace with a CNPG cluster of two PostgreSQL nodes. Databasus reaches the primary through a NodePort Kubernetes Service with the selector of CNPG's `-rw` Kubernetes Service, using a role declared in the cluster manifest over TLS verified with `verify-full`. Like the Patroni sources, these use disk-backed data and durable settings. Scenarios are grouped into top-level tests by family and PostgreSQL major, so CI can spread them across shards. The runner controls scenario parallelism; fixture-control scenarios may create several CNPG clusters within one test, so cluster count is measured separately.

The suite drives Kubernetes only through `kubectl` inside its k3s containers. It publishes no Kubernetes API port, writes no kubeconfig on the host and never reads or changes the developer's Kubernetes configuration.

The suite needs:

- rootful Docker on Linux, because scenarios connect from the host to the k3s container's address, and only rootful Docker on Linux makes that address reachable;
- a Docker daemon that allows privileged containers, which k3s requires;
- raised inotify limits, because every k3s container's kubelet and containerd take inotify instances and watches from the same per-user limits; [Preparing a machine](machine-setup.md#inotify-limits-for-cloudnativepg-tests) gives the values;
- Docker, the shared test PostgreSQL server, pinned tools and client bundles as described in [Commands](#commands).

The k3s, operator and CNPG PostgreSQL images are pulled through Docker and imported into k3s, which pulls nothing from a registry. A missing prerequisite, such as an unreachable Docker daemon, a refused privileged container, an unreachable container address or an image that cannot be obtained, fails the suite with a message that names it; no scenario is skipped. The [fixture README](../backend/internal/lib/testcontainers/testdata/cnpg/README.md) owns the dependency records.

Run the complete matrix from the repository root:

```sh
source scripts/fedora.sh
apply_libedit_shim
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout 15m \
  ./internal/modules/backups/e2e/postgresqlphysical/cnpg
```

Use Go's `-run` filter to repeat an individual scenario. [PostgreSQL with CloudNativePG](postgresql-cloudnativepg.md) explains the source configuration, slot synchronization and recovery limits.

A run removes its k3s containers, networks, volumes and temporary directories after success, a failure or cancellation. When the test process of `mise run test-full` is killed, the test runner still removes its labelled resources. Child runs can use distinct `databasus.test-run` values while retaining the enclosing run in `databasus.test-root-run`; the runner removes resources matched by either ownership label with its own run ID. A killed direct package run has no run label and leaves its resources behind. Every k3s container, network and volume carries the fixture label `databasus.k3s-fixture`, whose value is that k3s fixture's ID:

```sh
docker ps -a --filter label=databasus.k3s-fixture
docker network ls --filter label=databasus.k3s-fixture
docker volume ls --filter label=databasus.k3s-fixture
```

The fixture label also matches the k3s fixtures of CNPG runs still active in other worktrees. Before removing anything, make sure no other CNPG run is active, or filter by the IDs of the killed run with `label=databasus.k3s-fixture=<fixture ID>`.

## Verification agent end-to-end tests

These tests run the real agent binary against a real backend, proving that they agree on the contract and can restore a backup. The separate agent unit-test command needs neither a container engine nor the test database. [Commands](#commands) lists both runs and the rootful Podman subset; source servers still use Docker in that subset.

- **Allowed stand-in.** A forwarding gateway may delay traffic or reduce a claimed resource budget. It never creates verifications or accepts reports in the backend's place.
- **Concurrency.** Tests and separate runs may overlap through unique resources. The scenarios of `TestVerifyBackup` stay sequential because each agent allocates networks from Docker's default address pools.
- **Cleanup.** Each scenario waits for its containers and networks to disappear and explicitly cleans up detached daemon PIDs. Failed scenarios print agent output and logs.

## Frontend tests

Screen tests render real components and mock only the network, typed by the contract, so they cover every answer the contract allows. The [testing skill](../.agents/skills/testing/SKILL.md#frontend-screen-tests) gives the authoring rules.

- **Environment.** Screen tests select happy-dom with `// @vitest-environment happy-dom` on the first line. Pure logic, API and architecture fixtures keep the default node environment.
- **Rendering.** The [shared test infrastructure](../frontend/src/shared/lib/test) supplies real providers, isolated query state, routing, locale, theme and Strict Mode. Whole-application tests use the app-level renderer because the application supplies its own providers and browser router.
- **HTTP.** Contract-typed mocks declare finite methods and paths and fail teardown on unexpected requests, even if the component catches the error. Controlled responses expose overlapping work without timers. A cancelled request is a settled transport outcome; Strict Mode setup counts do not imply an equal number of HTTP requests.
- **Cleanup.** Unmount before settling pending responses, then restore browser state, globals and clocks. Session changes and unmount clear owned query state. Components own timers, listeners and late-response guards. Wait for observable dispatch before changing a pending action's session; polling tests assert reads and cleanup rather than library internals.
- **Limits.** Test and condition-wait limits are shared across the suite.
- **Tooling.** Expected-failure fixtures run in isolated child processes. Browser scenarios stay outside unit-test discovery.

## Browser UI tests

These tests exercise the production build in a real browser, for what happy-dom cannot establish: geometry, focus, fonts and images. Screen tests stay the faster layer for overlapping responses and editing-session policy. The [browser UI command](#commands) needs Docker and no backend services, because HTTP responses are finite synthetic fixtures.

- **Isolation.** Each run uses a pinned browser environment and an unprivileged container with its own temporary checkout and private preview server. It publishes no host port and cleans up only its own resources.
- **Architectures.** Native amd64 and arm64 use the same committed references and exact pixel comparisons. Other architectures fail before scenarios; reference updates require the explicit command below.
- **HTTP.** Finite contract fixtures fail teardown on undeclared requests or page errors, even if the application catches the failure. Only same-origin GET navigation and static resources pass through.
- **References and artifacts.** [Browser fixtures](../frontend/browser) own scenarios, visual references and runner probes. Missing or changed images fail an ordinary run. Inspect changes after the explicit reference-update command. Failed runs save artifacts under `frontend/browser-results/` for local inspection and CI upload.

## Browser end-to-end suite

These tests exercise the application image through its screens, without importing backend or frontend code or using paths into those trees from scenarios and fixtures. They need Docker with `image save --platform` support and the pinned tools, but no `.env`, running application or shared test database. The standalone command tests `DATABASUS_IMAGE` when set, or builds an image from the checkout; CI requires an explicit image.

- **Isolation.** Each run has its own application, source database, browser, recovery runner and private Docker daemon. Runs may overlap without taking a lock or conflicting with development ports. The private engine requires privileged containers; its Unix socket and writable data volumes belong only to that run. The host engine socket is never mounted, and only the browser publishes a host port.
- **Evidence.** Success requires actual restored data in running databases, not screen status alone. Scenarios execute the recovery commands the interface presents and validate selection together with the canonical URL. The [suite](../e2e/browser) owns scenario coverage and recovery-runner setup.
- **Prerequisites.** Recovery setup needs network access for its packages and any missing pinned images; nested recovery uses a locally supplied image. It supports Host tools and Docker recovery without rewriting the displayed commands.
- **Cleanup.** Success, failure, interruption and crashed-run recovery remove only the run's resources, including nested containers and any image it built. Removal failures fail the run. Failed runs print bounded diagnostics with credentials and tokens masked; diagnostic failure cannot prevent cleanup.
- **Checks of the suite.** Separate commands exercise unit logic and lifecycle failures, interruptions and overlapping runs. [Checking work](#checking-work) selects these checks, including the standalone lifecycle-command interruption check.
- **Crashed-run probes.** Every browser run first removes what crashed runs left behind, anywhere on the machine. A crashed-run probe kills a run or prevents its cleanup and then expects its own later run to remove the remains, so it gives a reliable result only while its caller starts no other run. Such a probe registers its teardown through the lifecycle suite's shared fixture, which marks it automatically. `mise run e2e:lifecycle` takes `--only-crashed-run-probes` or `--without-crashed-run-probes`, alone or with a group, so a caller can run the other probes concurrently and these alone; `--shard INDEX/COUNT` splits a selection into parts for several processes. A browser run started from another worktree meanwhile can still remove a crashed run first; the suite takes no machine-wide lock against that race.

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

The routine command prepares locked frontend dependencies and runs backend tests outside the end-to-end packages plus frontend unit and screen tests. Independent groups finish even if another fails, and the summary identifies failures. Migration and architecture-linter tests are included.

The full command adds the Docker end-to-end, browser, upgrade, script and review-hook suites described above, scheduled as [Parallelism in local runs](#parallelism-in-local-runs) describes. Both commands share a machine-wide lock; a direct package test takes no such lock.

The [test runner](../scripts/test.sh) owns process limits, package timeouts and full-run ordering. [Slow-package selection](../backend/test-slow-packages.txt) bounds container load: register a package there when it starts containers or networks or runs for over a minute. Unlisted packages still run in the fast group.

### What runs separately

- The rootful Podman subset (`mise run agent:e2e-podman`) needs its socket configured.
- The filesystem suite (`mise run test-filesystem`) needs host support for NFS/CIFS and their mount clients.
- Every suite can also run directly when checking a change that touches it; the [commands](#commands) below include hidden CI tasks, which remain callable by name.

### CI

CI repeats local verification on clean runners. The [release workflow](../.github/workflows/ci-release.yml) owns triggers, matrix groups, shard counts, resource limits and artifact transfer. All required results gate release version selection; manual development runs do not publish releases.

- Native amd64 and arm64 jobs exercise supported test surfaces. An architecture matrix or a local amd64 pass does not establish a successful native ARM run.
- Sharding cannot exclude unweighted or newly added tests. Failures do not cancel other matrix legs, and each selected package runs once.
- Unsupported server images follow the [shared exclusion policy](../backend/internal/lib/testcontainers/unavailable_images.json). Those versions still run where images exist, and container-free refusal checks remain on ARM. Other image failures are fatal.
- Image-based suites consume candidates built for the tested commit and architecture. Host, engine and candidate architecture are checked before scenarios.
- Podman and network-filesystem tests need suitable hosts. Local fixture checks do not prove mounted-filesystem behavior.
- Backend CI uses race detection. Add `-race` to a direct local command when reproducing a reported race; routine local runs omit it for speed.
- Python quality and dependency checks gate both release and development publication. Offline dependency checks do not establish release age or replace locked installation.

### Python quality checks

`mise run lint` includes every repository-owned Python file. `mise run format` applies Python formatting and safe lint fixes. The hidden `python:lint` and `python:format` tasks run those checks alone, from any repository directory. Tool versions are pinned in `mise.toml`; root `pyproject.toml` defines lint, annotation, formatting and type rules.

Lint, formatting checks and type checks cover repository-owned Python, including new and hidden directories. Generated environments and deleted files are excluded. A failed stage does not hide later diagnostics; checks never modify source, the index or the stash. Explicit formatting makes safe fixes without rewriting types. The [quality runner](../scripts/python_quality.py) owns file discovery and tool invocation.

Type checking uses the browser project's locked dependencies and pinned Python. It needs neither Docker nor a running application; initial dependency preparation may need network access, and setup failures fail lint. The [annotation convention](../AGENTS.md#functions-methods-and-types) applies at function and external-data boundaries.

The pre-commit hook checks the full Python selection when Python or its quality configuration changes, without automatic fixes or stashing. Unrelated prose does not select it. For deletion-only changes or a rename out of Python, explicitly run the full hook step below because automatic path selection omits deleted and old paths. CI checks the full selection.

Run the checks directly with:

```sh
mise run python:lint
mise run python:format
hk check --check --all --step python-lint
```

The runner and hook integration tests use the real pinned tools in temporary Git repositories. Prepare their shared dependency environment before running the script tests:

```sh
mise exec -- uv sync --locked --project e2e/browser
mise exec -- python3 -m unittest discover -s scripts -p '*_test.py'
```

### Upgrade test

The upgrade test starts from the newest release tag by version and checks the image built from the checkout, unless `DATABASUS_IMAGE` names one. It builds that image as `databasus-upgrade-test:<run ID>` and removes it when the test ends. After the server assertions, it checks self-update and manual replacement using an actual older agent pinned by image digest. The `upgrade-test` spec defines that fixture and its acceptance criteria.

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

## Commands

To repeat one failing test, call its Go or frontend runner directly. This takes no lock and builds no image.

| What                                                   | Command                                                                          |
| ------------------------------------------------------ | -------------------------------------------------------------------------------- |
| Backend and frontend test suites                       | `mise run test`                                                                  |
| All tests for the usual Docker development environment | `mise run test-full`                                                             |
| One backend test, in `backend/`, see below             | `go test -count=1 -run 'TestName/scenario text' ./internal/modules/<module>/...` |
| Verification agent unit tests                          | `mise run agent:test`                                                            |
| Verification agent end-to-end tests on Docker          | `mise run agent:e2e`                                                             |
| Verification agent subset on rootful Podman            | `mise run agent:e2e-podman`                                                      |
| Frontend                                               | `pnpm test` in `frontend/`, or `pnpm test -t 'scenario text'`                    |
| Frontend production browser UI                         | `pnpm --dir frontend test:browser --project=ui`                                  |
| One frontend browser UI shard                         | `pnpm --dir frontend test:browser --project=ui --shard=1/3`                       |
| Update browser visual references                       | `pnpm --dir frontend test:browser --project=ui --update-references`              |
| Upgrade test                                           | `mise run backend:test-upgrade`                                                  |
| Filesystem tests                                       | `mise run test-filesystem`                                                       |
| Browser end-to-end suite alone                         | `mise run e2e:browser`                                                           |
| Its parts that need no Docker daemon                   | `mise run e2e:unit`                                                              |
| Its complete lifecycle probes                          | `mise run e2e:lifecycle`                                                         |
| One lifecycle group                                    | `mise run e2e:lifecycle failures` (or `interruption`, `isolation`)                 |
| One backend remainder shard                            | See the race-enabled shard command below |
| Review hook tests                                      | `python3 -m unittest discover -s .agent-hooks -p '*_test.py'`                    |

The bundled PostgreSQL clients need the libedit shim on Fedora, and parallel test binaries collide on the Ryuk reaper name, so the full command that repeats a backend test is:

```sh
cd backend
source ../scripts/fedora.sh
apply_libedit_shim
TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 -run 'TestName/scenario text' ./internal/modules/<module>/...
```

To repeat one backend remainder shard with CI concurrency and race detection, prepare the Fedora shim as above, then run from the repository root:

```sh
packages=$(mise run --quiet backend:test-group-packages other-1-of-3)
pattern=$(mise run --quiet backend:test-group-tests other-1-of-3)
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -race -p=8 -parallel=2 -count=1 -timeout 40m -run "$pattern" $packages
```

Use `other-2-of-3` and `other-3-of-3` for the remaining selections. The same commands repeat a shard of another sharded group by its name, such as `patroni-2-of-5`. For a `cnpg` shard, omit `-parallel=2` to use Go's default parallelism, as its CI runner does. The provider root runs separately, without overlapping test work:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -race -p=1 -parallel=1 -count=1 -timeout 40m ./internal/modules/storages/internal/providers
```
