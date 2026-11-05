## Context

See `proposal.md` for motivation and scope. The modules from milestones 13a and 13b are already integrated; this change removes the mechanisms that kept their predecessors callable.

The remaining coupling is concrete:

- `backend/internal/app/app.go:113` installs configuration and database bridges, `:132` installs encryption and mail, and `:140` installs the cache. Later in the same constructor it installs module bridges. `backend/internal/app/environment.go:124` installs the logger bridge.
- `backend/internal/app/apptest/apptest.go:56` declares the single-live-application mutex, `:60` allocates a process-wide data directory, and `:105` rejects a second application. Its file helpers still resolve directories through the configuration bridge (`storage_files.go:51`, `damaged_backup.go:23`, `wal_queue.go:14` in the same directory).
- `backend/internal/app/console_test.go:461` reads the global database. `backend/cmd/seed_dev/main_test.go:335` starts a hand-built router; its fixture and `seed_backups_test.go` reconstruct module dependencies through bridges.
- `backend/internal/app/server.go:78` publishes the old shutdown signal and `:117` changes Gin's process-wide mode on every router construction.
- `backend/.golangci.yml:8` selects the helper build tag and `:169` begins two migration-period exclusions. `backend/tools/archlint/archlint.go:28` keeps the corresponding layout pattern; `layers.go:26` classifies only the server command as the command layer.
- `scripts/test.sh:80` runs the destructive source reset under a fixtures lock. Both backend and Podman CI preparation repeat the reset (`.github/workflows/ci-release.yml:260`, `:374`). The seed tests still use fixed source ports (`backend/cmd/seed_dev/main_test.go:414`); production development seeding instead uses its configured logical source and dedicated WAL source (`backend/cmd/seed_dev/main.go:61`).

Root `AGENTS.md`, the repository testing skill and `backend/AGENTS.md` constrain this design. In particular, dependencies belong in `Deps`, wiring belongs in `app`, test setup uses the real API, and production dependency graphs must exclude test support. The relevant architecture sections are Backend layout, Shared library, Long-running work and Automatic checks in `docs/architecture.md`. No layer direction or module ownership changes.

## Goals / Non-Goals

**Goals:** overlapping applications have independent resources; old entry points have no callers before deletion; ordinary builds and tests need no helper tag; lint and review have no migration-period directory exemptions.

**Non-Goals:** no HTTP or schema changes, removal of persisted-data upgrade handling, new application service abstraction, new test scheduler, or change to development source addresses. Existing shard counts, 30-minute package timeouts and 40-minute shard timeouts remain in force.

## Decisions

### 1. Convert consumers before deleting the compatibility graph

Work in dependency order, with compiling and testable slices:

1. Move the reusable utility packages and update their importers.
2. Convert the seed tests to `apptest`, retain the seed command under test, and create their source databases in test-owned containers.
3. Make app tests and apptest file helpers use their own application resources.
4. Remove the bridge installations and the old packages, then remove the single-application guard and shared directory in the same slice. Replace the guard's rejection test with the basic two-live-app API isolation test in that slice, before running its package checks; extend file and shutdown coverage in the following task.
5. Remove the reset and helper-tag tooling, expand checks, then enable independent tests to run concurrently.

The seed tests call `runSeed` against `apptest.App.URL`; their assertions and setup use the generated contract-checking client. Keep all nine existing scenarios: initial creation, repeated seed, deleted database recreation, wrong password, unrelated account, two-factor requirement, workspace permission denial, unreachable backend and unhealthy backend. Preserve the initial logical backup and physical force-full request assertions. Use the existing apptest SQL escape hatch only where the API cannot expose the required state; do not add a public production getter for a test assertion.

App tests in package `app` pass the concrete application or its database into their own SQL helpers. They can access the already-held `App.db` without a new accessor. Getter-installation tests disappear, but migration failure, newer-schema refusal, console commands and master-key upgrade cases remain. The key upgrade test must prove decryption through the app-owned key or an observable API operation after the bridge disappears.

Delete `internal/features`, the old `internal/config` and `internal/storage`, the replaced utility bridges, all lazy `di.go` wiring and self-loading files. Delete `testdb.OpenForProcess` and its obsolete test and narrow any now-unused process-only cleanup entry points. Ordinary per-test database allocation, retained failing databases and safe abandoned-database cleanup remain.

**Alternatives rejected:** deleting bridges first leaves seed and console tests unable to construct their dependencies. Keeping aliases leaves the global dependency graph alive. Building another router in the seed fixture duplicates the application wiring that these tests are meant to verify.

### 2. Preserve utility boundaries during the move

| Current unit | Destination |
| --- | --- |
| Stream readers, writers, range parsing and retry policy in `util/io` | `lib/iox`, retaining the existing stream behavior and tests |
| Container launchers and their testdata | `lib/testcontainers` |
| SSH settings adapter in `util/testing/bastion` | `lib/testcontainers/bastion` |
| HTTP tunnel proxy and real-client timeout support | `lib/httpclienttest`, one file per concept |
| Mailpit client | `lib/mailpittest` |
| Counting TCP forwarder | `modules/storages/internal/providers/providertest`, its sole consumer's test support |
| Portable filename sanitization | `lib/filename`, preserving exact replacements and tests |
| Directory creation and startup cleanup | private units in `app` |
| Legacy HTTP request helpers, pubsub, period and old Turnstile | deleted after their final consumer disappears |

Keep the bastion adapter outside the root container package. Its dependency on `lib/sshtunnel` would otherwise create an import cycle through that package's own container-based tests; the constraint is already recorded at `backend/internal/util/testing/bastion/bastion.go:3`.

Test support remains test-only through imports rather than a build tag. Extend depguard's test-support permissions to the chosen packages without allowing production packages to import them. Keep the existing transitive production-binary checks. Moving container testdata must preserve SSH image build contexts and key lookup relative to the source file.

**Alternatives rejected:** renaming all of `util` into one shared utility package keeps unrelated concepts together. Combining the bastion adapter with launchers introduces an import cycle. Generalizing the single-consumer forwarder adds a shared package without another user.

### 3. Give each test application ownership of all mutable resources

`apptest.New` creates its root under `tb.TempDir()`. Store its data, temporary and WAL queue paths on that `apptest.App`; every file helper resolves through those fields. Keep a distinct key file, mail recorder, cache, rate limiter, clock, runner and module lifecycle for each instance. Do not use `t.Setenv` or change the working directory to configure an instance.

Retain the existing cleanup ordering: stop jobs and module work, wait for backups and notification delivery, close HTTP and database resources, then allow the temporary directory and test database cleanups to run. Test a neighbouring app surviving the first app's complete cleanup through a nested test scope, so cleanup runs at a deterministic point without adding a second public stop API.

Remove per-router `gin.SetMode`. The production command selects release mode once before starting the application. A test package that needs an explicit mode sets it once before running tests; other test processes can use Gin's default mode. No constructor mutates it, and no global once guard is added to production code.

Replace the existing second-app rejection test with overlapping-app cases. Create the same account email through both APIs, write different bytes under the same local filename through their real file stores, inspect their distinct staging and WAL paths and key files, and prove one app's shutdown leaves the other usable. Add independent `t.Parallel` calls after the isolation changes. Keep intentional process-environment tests and sequences sharing one external fixture sequential; do not parallelize engine-version fixture lifetimes that were deliberately bounded for memory.

**Alternatives rejected:** serializing construction while retaining global bridges still lets the last constructed app redirect the first app's helpers. Changing process environment to point at each directory is incompatible with parallel tests. Parallelizing every subtest without considering fixture ownership would introduce source-server and memory contention.

### 4. Delete shared-source reset infrastructure

The final fixed-source consumer is the seed fixture; moving it to test-owned sources removes the reset's purpose. Audit remaining fixed-port and shared-role/permission/slot accesses before deletion, including package helpers and CI selectors. Repair a remaining test-owned object name or cleanup in this change rather than retaining a destructive reset for it.

Remove `cmd/cleanup_test_db`, the fixtures lock and reset timeout, reset calls and obsolete source-readiness waits in scripts and both CI test preparation paths. Keep the lock that serializes full local runs and the run ID used to clean up only an interrupted run's labeled containers, networks and volumes. Keep `backend-db-test` available and preserve its per-test databases. Development sources used by the real seed command stay available; their existence does not require automated tests to reset them.

Update the Python runner tests to assert the absence of a reset and fixed-source waits, preservation of a concurrent package run, full-run serialization, interruption cleanup and upgrade-test ordering. Add a workflow-preparation assertion to the same tests so both CI paths cannot reintroduce resets or obsolete fixed-source readiness. These checks use subprocess stubs and workflow inspection, not the full backend suite.

**Alternatives rejected:** porting the reset command to the new configuration package keeps a command that deletes other sessions' sources. Removing all compose sources would break development seeding and goes beyond this teardown.

### 5. Enforce checks on every backend package

Remove both `path-except` migration exclusions, their layout anchor, `archlint`'s layout pattern and filtering calls, its config-pattern test, and its helper-build-tag rule and tests. Keep generated-code handling, production/test distinctions and the existing rule-specific permissions. Classify every non-agent `cmd` package as the command layer; the verification-agent case remains first. Add regression fixtures outside the former path list and for the seed command's boundaries. Import restrictions that do not depend on the importer's layer, including the agent-only import rule and module boundaries, also apply to unclassified production packages.

Enable `gochecknoglobals` for the same backend scope as the other checks. A diagnostic probe found 50 findings before deleting legacy code; this is an inventory aid, not an acceptance count. Apply these dispositions:

- Application dependencies and mutable application state move into the owning instance or arrive through `Deps`; no suppression is accepted for a bridge or app singleton.
- Fixed tables, regular expressions and values that can be local constants or local values become such values. Share an immutable table only with a specific explanation of its immutability and purpose.
- Linker-populated agent version metadata and read-only binary paths initialized by `TestMain` may have narrowly explained declaration-level suppressions. They must be initialized before tests and never hold an app or database dependency.
- Existing test-only coordination for database template creation, once-per-process abandoned-database cleanup and external-driver TLS registration may retain one narrowly scoped synchronization state with a specific explanation. They coordinate process fixtures and must never select an application's database, configuration or services. Keep their existing safety tests and synchronization. Do not replace the database advisory lock or disable safe cleanup to satisfy the linter.

Audit the whole resulting check output, including the agent, tools, engine privilege tables, backup notification triggers and workspace access tables. Every retained suppression names the exact linter and explains the constraint; no package-wide global-variable exclusion is allowed. Normal exception-free dependency wiring remains the architecture rule.

**Alternatives rejected:** blanket suppression or excluding the agent merely hides findings. Rebuilding immutable lookup tables or binaries for every request/test wastes work. Replacing the already-safe test-database coordination just to remove its synchronization variables increases scope and can weaken concurrency safety.

### 6. Remove the tag and update current documentation together

Delete the helper tag from mise package/shard commands, package and test selectors, the Podman task, local runner, CI, lint build options and agent/editor environment. Remove only the gopls wrapper. Keep the shared mise PATH entry and shellcheck enumeration because the TypeScript wrapper still supplies frontend/node_modules to the pinned server; verify its TypeScript resolution from the repository root. Check `.claude/settings.json`, `.codex/config.toml`, `.vscode/settings.json` and `.zed/settings.json` explicitly. Existing processes may retain an inherited `GOFLAGS`; local verification runs with that override cleared so success cannot depend on the removed tag.

Keep runner orchestration and selector coverage as distinct tests. The former may stub selector outputs to exercise process handling. Keep all three shard stubs behind one release condition until they have all started, so sequential execution fails the test. Signal only the runner PID in the interruption test, then prove that its cleanup stops its shards while the neighbouring invocation and its shard remain alive. The latter runs both real package/test selector scripts against temporary fixture package directories and test sources, with only package-discovery and command-dispatch boundaries replaced. Its independently derived inventory includes weighted and unweighted tests and repeated names across packages, excludes `TestMain`, and must match the union of all three nonempty physical API selections exactly once per package/test pair. Record discovery arguments to prove the scripts add no helper tag. The existing physical API shard verification then exercises the selected real tests.

Remove transition allowances from root and backend agent instructions, architecture/testing documents and all three review prompts. Update prompt tests to require universal checking. Update backend layout documentation to include the seed command and the resulting shared test packages, and remove stale explanatory comments in compose and scripts. The normal rules belong in their existing documents; do not add a parallel set of agent instructions.

The five delta specifications carry the requirement changes. During implementation, clean up the transitional Purpose of the main `backend-code-checks` specification; requirement synchronization follows the normal OpenSpec archive flow. `openspec/config.yaml` needs no change. Historical archived changes and `temp/refactor` remain historical evidence.

Two existing documentation mismatches are resolved without runtime changes: the shard spec's 15-minute timeout is aligned with the current 40-minute command, and testing documentation describes the filesystem suite as a separate command and CI check, as the current tasks actually run it. No filesystem suite is added to `mise run test`.

**Alternatives rejected:** updating only prose leaves reviewer tests and active requirements demanding the deleted exemptions. Rewriting archives destroys useful historical evidence. Removing the filesystem caveat while retaining an inaccurate claim about the full runner would leave the documentation inconsistent.

## Risks / Trade-offs

- **Parallel tests expose state beyond the obvious bridges.** Use overlapping-app regression tests and race checking before adding parallel calls across module tests. Preserve sequential scopes that intentionally share an external fixture.
- **Additional concurrent source containers use more memory and database connections.** Keep version fixtures bounded, retain existing test-group limits and use package/shard verification rather than introducing a new concurrency policy.
- **A deletion loses an existing assertion.** Map the nine seed scenarios and app startup, console, key-upgrade and lifecycle cases to their replacements in the relevant tasks; delete only tests whose sole behavior is installing a bridge or rejecting a second app.
- **New lint coverage reveals pre-existing defects in moved test support or commands.** Fix violations in the touched code and preserve behavior; do not reintroduce path exemptions or weaken checks.
- **Broad import changes affect many packages.** Preserve utility tests, run importing packages through the targeted commands in `tasks.md`, and keep the final Change verification list equal to those package selections.

## Migration Plan

This is a source and tooling migration with no database migration or operator action. Apply the slices above in one change, validate ordinary untagged production builds and targeted race-checked tests, then finish mandatory review before the tree is ready to commit. Rollback before release is reverting the change; no data transformation needs reversing. Master-key upgrade handling and historical migrations remain intact.
