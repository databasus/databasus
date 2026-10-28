## Why

Milestone 10c2 moves about 19,000 lines of old physical backup code into the `backups` module. Its plan also rewrote about 12,000 lines of old package tests in the same diff, so the move would have been checked by tests written alongside it. The rewrite plan forbids exactly that for 10a and 10b1: safety tests must not change in the same diff as the code under them. This change does the test half first, against the old code. `physical-backups-module` then moves the code under API tests that stay as they are.

The old package tests cannot survive the move as written:

- They build services through package getters and `storage.GetDb()`.
- About 2,000 lines of helpers behind the `testhelpers` build tag write the catalog through the old repositories.
- WAL streaming tests replace the storages module with a fake file store.
- The controller test builds its own router.

`docs/testing.md` allows module API tests, and unit tests only for pure logic. None of these old tests is either.

## What Changes

- **API tests of physical backups on `apptest`.** The old tests that need the internal database, a getter or a source server become API tests against the running application.
  - They call the old routes through `apptest.CallOldAPI` and its public siblings.
  - They seed with SQL only the catalog states the API cannot create.
  - They read internal tables only for data no route returns, such as the typed error reason.
  - They live in five test-only packages under `internal/modules/backups/physicalapitests/`: the catalog, the restore set, the stream statistics, backing up and WAL streaming. `apptest` allows one application per process, and these tests start source servers and wait on real time, so one package would outlast the run limit.
- **Unit tests for pure logic.** Tests of pure functions stay where they are and move with their code in `physical-backups-module`. Where a test reaches pure logic only through a getter or the database, the logic is extracted into a pure function in the old code without changing behavior, and the test becomes a unit test of it. The scheduler's choice between a full backup, an incremental and nothing is the largest such extraction.
- **Dropped tests.** A test is dropped only when it checks a race no request can reproduce, a branch no caller reaches, dead code, or something the engine tests or the end-to-end scenarios already check. Every drop is listed with the coverage it loses. Dead code found this way is deleted with its test: the dependents summary preview, the history file presence check and the protected-slot branch of the startup slot sweep.
- **What the test application gains.**
  - `RunJobPass` runs one pass of the physical scheduler and the physical cleaner by the job names they take in `physical-backups-module`. `RunPhysicalBackupCleanerPass` is deleted.
  - `RestartModules` also restarts the live physical loops and runs the physical startup work: failing backups left in progress, recovering stale streamers and sweeping orphan backup slots.
  - The option `WithWalStreamTimings` shortens the forced WAL rotation interval, the archive staleness threshold and the local queue pause size.
  - New helpers: a physical database inserted without a source server, catalog seeders, a published storage file, an open restore stream, a held SQL lock and a check for the WAL queue directory.
- **Small production changes in the old code, without visible behavior change.**
  - The scheduler gains a single pass.
  - The startup work of the three loops moves out of their `Run` into one step that `internal/app` runs when the modules start.
  - The WAL stream supervisor takes its timings from `backups.WalStreamTimings`, a new type in `module.go` with production defaults.
  - The server builds the scheduler, the cleaner and the supervisor from its environment instead of singletons. The live state route reads the supervisor that runs, so in the test application it no longer reads a singleton that never started.
- **The rest of the old test code goes.** The old package tests that moved are deleted, along with every tagged physical helper nothing else uses. `internal/modules/databases/deletion_test.go` no longer registers a fake WAL streamer in the shared cancellation registry; the WAL streaming package checks the same deletion with a real streamer.
- **CI.** The `wal-usecases` group, which ran the old use case package, becomes `physical-api` and runs the new test packages in three shards.
- **Nothing a user sees changes.** Backups, WAL streaming, restores, routes and their answers stay as they are.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `api-contract`: the test application can run one pass of the physical scheduler and cleaner, restarts the live physical loops with the modules, and lets a test shorten the WAL stream timings.
- `ci-pipeline`: the `wal-usecases` group becomes `physical-api` and runs the API tests of physical backups.

## Out of scope

- Moving any physical production code into the module, the Runner, the WAL stream manager and the per-database WAL streamers. That is `physical-backups-module`.
- The physical operations in the API contract and the generated client. The new tests call the old routes, as the end-to-end scenarios do; `physical-backups-api` switches both.
- Every simplification of milestone 11, including one owner of the backup status and the removal of the scheduler's second staleness check. The pure decision function extracted here keeps today's behavior exactly.
- Fixing defects the inventory found: a cancel lost between the scheduler's dispatch and the backup's registration, and a streamer row left `RUNNING` after a shutdown. They are described in the design of `physical-backups-module`, which removes their cause.
- Changing the end-to-end scenarios beyond replacing `RunPhysicalBackupCleanerPass` with `RunJobPass`.

## Impact

- **Backend, new:**
  - `backend/internal/modules/backups/physicalapitests/{catalog,restoring,streamstatistics,backuping,walstreaming}` (test-only packages);
  - `backend/internal/app/apptest`: the physical catalog seeders and helpers.
- **Backend, changed:**
  - `backend/internal/app` (`background_loops.go`, `app.go`, `environment.go`);
  - `backend/internal/app/apptest` (`RunJobPass`, `RestartModules`, the options, cleanup);
  - `backend/internal/modules/backups/module.go` (`WalStreamTimings`);
  - the old physical packages under `backend/internal/features/backups/backups`: the scheduler pass, the startup step, the timings, the live supervisor, the pure extractions and dead-code removal;
  - `backend/internal/app/apptest/physicaltest`, `backend/internal/modules/databases/deletion_test.go`.
- **Backend, deleted:** the old physical package tests outside the pure ones, `usecases/physical/postgresql/testing.go` and the other tagged physical helpers nothing uses any more, `mocks_test.go` and the `setup_test.go` files.
- **CI:** `mise-tasks/backend/test-group-packages`, `.github/workflows/ci-release.yml`, `backend/test-shard-weights.txt`.
- **Documents:** `docs/testing.md` (the test application, the layers table), `docs/glossary.md` (forced WAL rotation, archive staleness threshold, local WAL queue) and the rewrite plan `temp/refactor/plan.md`, which lives only in the main checkout and gets the 10c2a and 10c2b lines.
- **Not affected:** the schema, the contract, the frontend, every route and answer, and stored backups.
- **Running instances:** nothing changes.
- **Rules the change answers to:** the root `AGENTS.md` and `backend/AGENTS.md`.

The local CI-shard runner also makes these longer tests usable by implementation review: selected shards run concurrently in separate processes, with the existing timeout and failure checks. The review command allowlist gains only these named shards; it still refuses full-suite runs and shell syntax.
