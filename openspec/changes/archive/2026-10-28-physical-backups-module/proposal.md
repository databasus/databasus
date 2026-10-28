## Why

This is the last state change of milestone 10. `physical-backups-schema` renamed the physical tables to `backups_physical_*`. `physical-tests-on-apptest` rewrote the old package tests as API tests of the running application. The physical code itself still lives outside the `backups` module, in `internal/features/backups/backups`, and it breaks the rules of `docs/architecture.md` that the logical side already follows:

- **It reaches everything through process-wide state.**
  - Repositories call `storage.GetDb()` and take no context.
  - Settings come from `config.GetEnv()`.
  - Databases, storages, notifiers and access come through the bridges `databases_bridge`, `storages_bridge`, `notifiers_bridge` and `workspaces_bridge`.
- **The module keeps temporary seams for it.**
  - `module.go` publishes `PhysicalBackupConfigSaved` after a save commits, and the handler's error becomes the answer of a save that has already committed. `docs/architecture.md` allows this one event only until the physical backups move.
  - `module.go` also lends the old code `AcquireUserStream`, `IsUserStreamHeld` and the force request methods.
  - `Deps` carries a process-wide cancellation registry for it.
- **Its background work is hand-written.** The scheduler, the cleaner and the WAL stream supervisor are ticker loops in a `background.Group` of `internal/app`, not jobs of `background.Runner`. The scheduler starts each backup with a bare `go` on `context.Background()`. A stop of Databasus therefore does not end the backup: it runs until the process dies, and only the recovery at the next start records it as failed. A logical backup at the same moment is recorded as failed before the process exits.

## What Changes

- **The physical code moves into `internal/modules/backups/internal/physical`**, split into subpackages by meaning:
  - `catalog`: models and repositories;
  - `chains`: chains, restore sets, the list and deletion;
  - `timelines`: timeline checks and history files;
  - `streamhealth`: the stream health verdict and statistics;
  - `backuping`: the scheduler and the backuper;
  - `walstreaming`: the WAL streaming;
  - `retention`: the cleaner;
  - `restorestream`: restore tokens and the restore stream;
  - a facade over them.

  The old `backuping/physical`, `core/physical`, `core/enums`, `services`, `usecases/physical/postgresql`, `download` and `dto/physical` are deleted.
- **Names.**
  - `PhysicalWalStreamSupervisor`, one per installation, becomes `walstreaming.Manager`, the "WAL stream manager" in prose, a new glossary term.
  - `WalStreamSupervisor`, one per database, becomes `walstreaming.Streamer`, the existing glossary term "WAL streamer".
  - The two types called `PhysicalBackupService` become the facade `physical.Service` and `chains.Service`.
- **Dependencies arrive through `Deps`.**
  - Repositories run their queries through `database.Conn(ctx)` and take a context, and services open transactions with the module's `Transactor`.
  - The queries outside repositories move into them.
  - The client tools directory, the WAL queue directory and the WAL stream timings come from `Deps`.
  - Databases, storages and notifiers are read through their modules' `module.go`, access is checked with `workspaces.Authorize`, and audit events go through the module's recorder.
- **Background work runs like the logical side's.**
  - The scheduler (every second), the cleaner (every minute) and the reconciliation of the WAL stream manager (every 15 seconds) become jobs of `background.Runner`.
  - Running backups and WAL streamers start in the module's group of runs, from the context `Module.Start` keeps, and `WaitForRuns` waits for them.
  - The startup work that `physical-tests-on-apptest` gathered into one step runs in `Module.Start`.
  - The remaining bare `go` statements start through `lib/background`.
  - Every time read comes from the module's clock and reaches SQL as a parameter.
  - The cancellation registry becomes the module's own.
- **A stop of Databasus ends a running physical backup with a recorded outcome.** The backup is recorded as failed with the message `backup interrupted by application shutdown` before the process exits, as a logical backup is. It gets the failure notification any failed backup gets. WAL streamers stop and keep their replication slots, as today, and each records that it stopped, so the next start resumes the stream at once instead of after 90 seconds.
- **Two defects of the old wiring go away.**
  - A cancel or a delete that came right after the scheduler dispatched a backup could miss it, because the backup registered for cancellation only after it started. The backup ran on and later recorded its result. The backup now registers before it starts.
  - A WAL streamer whose goroutine had already ended before a stop kept its row `RUNNING` until its heartbeat went stale. Each streamer now records its own stop.

  Apart from these, the shutdown outcome and the error text of a refused access (see Impact), nothing a user sees changes.
- **No temporary event.** `PhysicalBackupConfigSaved` is deleted. The config service calls the physical side inside the module after the save commits, as the event's handlers do today, and the physical side's error still becomes the answer of the save. The three old `DatabaseDeleting` handlers and the old `DatabaseBackupTypeChanged` handler become part of the module's handlers, in the same order. Milestone 11 merges them.
- **`module.go`.**
  - `AcquireUserStream`, `IsUserStreamHeld`, `SubscribeToPhysicalBackupConfigSaved`, the force request methods and `Deps.Cancellations` are deleted.
  - Temporary methods serve the old controller, one per route: the list, a backup request, cancel, delete, the two restore tokens, the restore stream, the stream statistics and the live state of the stream. The response types and the errors the controller maps to statuses are aliased beside them. `physical-backups-api` deletes all of them.
  - A temporary method serves telemetry, which reads the latest completed full backup, until telemetry moves in milestone 13.
- **The old controller stays as an adapter** in `features/backups/backups/controllers/physical` over those methods. Its routes, its request and response bodies and its status codes stay as they are, including the 400 it answers for a refused access. The frontend, the end-to-end scenarios and the API tests from `physical-tests-on-apptest` do not change.
- **Tests.**
  - The unit tests move with their code.
  - The API tests and the end-to-end scenarios keep their code.
  - `apptest` keeps its methods and changes what is behind them. `WithPhysicalBackupLoops` runs the module's physical scheduler and WAL stream manager live on the system clock; the cleaner runs only through `RunJobPass`. `RunJobPass` reaches the real jobs. `RestartModules` and the cleanup need nothing physical of their own.
  - New API tests cover the shutdown of a running physical backup and of a WAL streamer.
- **`internal/app`** loses the physical loops and their test seams, the physical event wiring, the shutdown step for the shared registry and the physical getters of its test. `cmd/seed_dev`, `features/telemetry` and `features/test_once_protection_test.go` switch to the module.
- **The physical code leaves the linter and `archlint` exceptions** by moving into the target layout.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `application-shutdown`: a stop signal also ends a running physical backup with a recorded failure before the process exits, WAL streamers stop with their replication slots kept and record their stop, and the 8-second limit covers both.
- `api-contract`: the live physical loops of a test application become the backups module's physical jobs and run on the system clock.

## Out of scope

- Every simplification of milestone 11:
  - one owner of the backup status;
  - one table for full and incremental backups;
  - merging the scheduler's two checks for an immediate replacement;
  - one stream health verdict;
  - one deletion handler;
  - remembering the working codec.

  Four rules of "Physical backups" in `docs/architecture.md` wait for these: one owner of the status, one table, one health verdict and the remembered codec. The rewrite plan records them as a temporary relaxation of the moved code.
- The physical operations in the API contract, the generated client in the frontend and in the tests, and the removal of the old controller, its recovery script and the temporary methods. That is `physical-backups-api`.
- Telemetry, the system health check and the verification runs, which move in milestones 12 and 13.
- Any schema change. `physical-backups-schema` made them all.
- Changing what a user cancel records. A cancelled physical backup keeps today's statuses and reasons.

## Impact

- **Backend, new:** `backend/internal/modules/backups/internal/physical/...` with the unit tests that move, and the shutdown tests in `backend/internal/modules/backups/physicalapitests`.
- **Backend, changed:**
  - `backend/internal/modules/backups/module.go`, `internal/module`, `internal/config` (the call after a physical save);
  - `internal/pipeline`: its run context moves to a package the physical side shares;
  - `backend/internal/features/backups/backups/controllers/physical` (an adapter over `module.go`);
  - `backend/internal/app`, `backend/internal/app/apptest`;
  - `backend/internal/features/telemetry`, `backend/cmd/seed_dev`, `backend/internal/features/test_once_protection_test.go`.
- **Backend, deleted:** everything under `backend/internal/features/backups/backups` except the controller, with the physical helpers behind the `testhelpers` tag that `physical-tests-on-apptest` kept.
- **Not affected:** the schema, the contract, the frontend, the old physical routes and their status codes and bodies, and every stored backup. The one exception is the error text of a refused access, which now comes from `workspaces.Authorize`; its status stays 400.
- **Running instances:** nothing changes on upgrade. A physical backup that runs while Databasus stops is recorded as failed at that moment instead of at the next start. The local WAL queues keep their directory.
- **Documents:**
  - `docs/architecture.md`, "Modules": the paragraph about `PhysicalBackupConfigSaved` is deleted;
  - `docs/testing.md`, "The test application": the paragraph about `WithPhysicalBackupLoops` describes the module's jobs;
  - `docs/glossary.md`: the new term "WAL stream manager", and the code name of the existing "WAL streamer";
  - the rewrite plan `temp/refactor/plan.md`, which lives only in the main checkout: the 10c2b line and the table of temporary relaxations, which gains a row for the four rules of "Physical backups" in `docs/architecture.md` that wait for milestone 11.
- **Rules the change answers to:** the root `AGENTS.md` and `backend/AGENTS.md`.
