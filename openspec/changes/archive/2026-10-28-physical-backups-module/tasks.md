## 1. Shared run context

- [x] 1.1 Move `internal/modules/backups/internal/pipeline/run_context.go` and its test into `internal/modules/backups/internal/runcontext`, with the shutdown cause and an optional time limit as parameters, per design.md decision 6. Logical backups and restores keep their 23-hour limit. Allow 30 minutes per unsharded test package in `backend:test-package` and `scripts/test.sh`, and document the limit; the WAL streaming API package exceeds the previous 15-minute budget. Verify with `mise run backend:test-package ./internal/modules/backups/...` exiting 0, with the logical and restore shutdown tests passing unchanged.

## 2. The lower physical packages in the module

- [x] 2.1 Create `internal/modules/backups/internal/physical/catalog` from `core/physical/{models,enums,dto,repositories}`, `core/enums` and the sidecar naming, per design.md decisions 1, 2 and 4. Every repository holds `*gorm.DB`, takes `ctx`, runs through `database.Conn`, returns `false` for a missing row and takes the time as a parameter, with no SQL `NOW()`. Fold `coordination.go` into `in_flight_backup_repository.go`. Add the repository methods for the queries that design.md decision 2 moves out of services. Move the enum unit test. Verify with `mise run backend:test-package ./internal/modules/backups/internal/physical/...` exiting 0 and `mise run backend:lint` exiting 0.
- [x] 2.2 Create `physical/timelines`, `physical/chains` and `physical/streamhealth` from `usecases/physical/postgresql/timeline.go`, `core/physical/chain_view`, `core/physical/service` and `core/physical/wal_health`, per design.md decisions 1 to 4. `chains.Service` takes its dependencies through `Deps`, opens transactions with the `Transactor`, and keeps the statements and locks of the decision 3 rows it owns. Move their unit tests. Verify with `mise run backend:test-package ./internal/modules/backups/internal/physical/...` exiting 0 and `mise run backend:lint` exiting 0.

## 3. The upper physical packages in the module

- [x] 3.1 Create `physical/backuping` and `physical/retention` from `backuping/physical` (the scheduler, the backuper, the canceller, the cleaner and the backup half of the cancellation listener) and from the backup files of `usecases/physical/postgresql`, per design.md decisions 1 to 6. The scheduler and the cleaner expose `background.Job`s named `backups_physical_scheduler` and `backups_physical_cleaner`. A backup registers for cancellation before it starts in the group of runs, runs on the physical run context, and records a shutdown as decision 6 says. Split `RunStartupWork` into `backuping.FailBackupsLeftInProgress` and `backuping.DropOrphanBackupSlots`, the slot sweep best effort, per design.md decision 7. Move the unit tests, including the pure scheduler decision. Verify with `mise run backend:test-package ./internal/modules/backups/internal/physical/...` exiting 0 and `mise run backend:lint` exiting 0.
- [x] 3.2 Create `physical/walstreaming` from `wal_supervisor.go`, the streamer half of the cancellation listener and the streaming files of `usecases/physical/postgresql`, with `Manager` exposing the job `backups_physical_wal_stream_manager` and `Streamer` per database, per design.md decisions 1, 4 and 6. Each streamer runs in the group of runs, takes `WalQueueDir` and `WalStreamTimings` from its dependencies, and records its own stop when its context ends. Add `walstreaming.FailStaleStreamers` for the startup step of design.md decision 7. Its inner goroutines start through `lib/background`. Move the unit tests. Verify with `mise run backend:test-package ./internal/modules/backups/internal/physical/...` exiting 0 and `mise run backend:lint` exiting 0.
- [x] 3.3 Create `physical/restorestream` from `download/restore_token` and `download/restore_stream`, leaving the recovery script in the controller, and the facade `physical.Service` from `services` and `dto/physical`, per design.md decisions 1 and 12. The facade checks access with `workspaces.Authorize`, writes audit events through `auditing.Recorder` with the old texts, and serves the restore stream with the user stream and a heartbeat in the group of runs. Move the unit tests. Verify with `mise run backend:test-package ./internal/modules/backups/internal/physical/...` exiting 0 and `mise run backend:lint` exiting 0.

## 4. Switch-over

- [x] 4.1 Wire the physical packages into `backups.New` and the module, per design.md decisions 5 to 13:
  - `Deps` gains `WalQueueDir`, `WalStreamTimings` and `PhysicalExecutionClock` and loses `Cancellations`, and the module creates the registry.
  - `GetJobs` returns the three physical jobs, and `Start` runs the three physical startup steps after the logical recovery, in the order of decision 7.
  - The physical reaction joins `config.Deps` in place of `PhysicalBackupConfigSaved`, and the event, its handler type and its subscription are deleted.
  - The physical parts join `OnDatabaseDeleting` and `OnDatabaseBackupTypeChanged` in the old order.
  - The temporary methods, aliases and errors of decision 12 and `GetLatestCompletedPhysicalFullBackup` are added to `module.go`, and `AcquireUserStream`, `IsUserStreamHeld` and the force request methods are deleted.

  Then switch the callers:
  - `internal/app`: wiring, background loops, the shutdown step, the registry, `StartModules`, `RunJobPass` and `getters_test.go`;
  - the controller in `features/backups/backups/controllers/physical`, as an adapter with its old statuses;
  - `features/telemetry`, `cmd/seed_dev` and `features/test_once_protection_test.go`;
  - `apptest`, per decision 13, including the refusal of a manager pass without the live jobs.

  `modules/databases/deletion_test.go` no longer uses the shared registry after task 3.5 of `physical-tests-on-apptest`; confirm it with `rg -n 'GetCancellationRegistry' backend` printing nothing.

  Verify with `mise run backend:test-package ./internal/app/... ./internal/modules/backups ./internal/modules/backups/physicalapitests/... ./internal/modules/databases ./internal/features/telemetry ./cmd/...` exiting 0, and `git diff --stat` against the start of the change listing no modified existing file under `backend/internal/modules/backups/physicalapitests` or `backend/internal/app/apptest/physicaltest` (the new shutdown tests in task 5.1 are added separately).
- [x] 4.2 Run the end-to-end scenarios against the switched code. Verify with `mise run backend:test-package ./internal/modules/backups/e2e/postgresqlphysical/...` exiting 0 with 14 top-level scenarios passing in pg17 and 12 in pg18.
- [x] 4.3 Delete everything under `backend/internal/features/backups/backups` except `controllers/physical`. Delete `features/backups/config` parts that only the physical code used, and any unused tagged physical helpers left in `features/storages`, `features/databases` and `features/audit_logs`; retain the shared helpers still used by healthcheck, verification or audit-log tests. Verify with `rg -n 'features/backups/backups/(backuping|core|usecases|services|download|dto)' backend` printing nothing, `rg -n 'storage\.GetDb|config\.GetEnv|_bridge\.' backend/internal/modules/backups/internal/physical backend/internal/modules/backups/module.go` printing nothing, and `mise run backend:test-package ./internal/features/... ./internal/app/... ./cmd/...` exiting 0.

## 5. Shutdown behavior and lint

- [x] 5.1 Add the API tests of the `application-shutdown` delta and the lost-cancel check of design.md decision 14 to `physicalapitests/backuping` and `physicalapitests/walstreaming`. Cover:
  - a full and an incremental backup interrupted by `RestartModules`, failed with `backup interrupted by application shutdown` and not cancelled, with their stored files gone after a pass of `storage_file_deletion`, one failure webhook to a notifier subscribed to failures and none to a notifier that is not;
  - a WAL stream that stops at the restart, keeps the same replication slot on the source, and is reported running by the stream statistics within 30 seconds of the restart;
  - a user cancel of a full and of an incremental backup while `pg_basebackup` streams it, each listed as cancelled with `context canceled` as its fail message, not the shutdown message;
  - a cancel right after a scheduler pass that ends the backup;
  - a restart of the modules while a full backup runs, which lists it as failed with `backup interrupted by application shutdown`;
  - with the live jobs, backups disabled and one pass of `backups_physical_wal_stream_manager`, after which the statistics report the streamer as not started and the slot is still on the source.

  Add the `api-contract` scenarios "Live jobs run on the system clock", "The cleaner does not run with the live jobs" and "A manager pass needs the live jobs" to `app/apptest/apptest_test.go`, the first also checking that `App.Clock` is nil. Verify with `mise run backend:test-package ./internal/modules/backups/physicalapitests/backuping ./internal/modules/backups/physicalapitests/walstreaming ./internal/app/apptest ./internal/lib/shutdown` exiting 0.
- [x] 5.2 Check that the physical code leaves every exception. Verify with `rg -n 'context\.Background|time\.Now\(|NOW\(\)' backend/internal/modules/backups/internal/physical --glob '!*_test.go'` printing nothing, `rg -n '^\s*go\s' backend/internal/modules/backups/internal/physical --glob '!*_test.go'` printing nothing, and `mise run backend:lint` exiting 0 with `go run ./tools/archlint` in `backend/` reporting nothing.

## 6. Documents

- [x] 6.1 Delete the `PhysicalBackupConfigSaved` paragraph from "Modules" in `docs/architecture.md`. Rewrite the `WithPhysicalBackupLoops` paragraph of "The test application" in `docs/testing.md` per design.md decision 13. In `docs/glossary.md`, add the term "WAL stream manager" with `walstreaming.Manager`, add `walstreaming.Streamer` to the code column of the existing "WAL streamer" row, and add `WalQueueDir` to the code column of "Local WAL queue". Update the rewrite plan at `/home/rostislav/development/project/databasus/temp/refactor/plan.md`: mark the 10c2b line, update the relaxation rows about the bridges, `PhysicalBackupConfigSaved`, `backups_bridge`, the temporary `module.go` methods and the live loops, and add a relaxation row for the moved physical code: the four rules of "Physical backups" in `docs/architecture.md` that wait for milestone 11, per design.md Non-Goals. Search the repository for every changed name and bring documents and comments in line. Verify with `rg -n 'PhysicalBackupConfigSaved|PhysicalWalStreamSupervisor|WalStreamSupervisor|GetCancellationRegistry' backend docs AGENTS.md backend/AGENTS.md .agents/skills` printing nothing, `rg -n 'walstreaming\.Manager|walstreaming\.Streamer' docs/glossary.md` showing both rows, `rg -n 'milestone 11|вех[аи] 11' /home/rostislav/development/project/databasus/temp/refactor/plan.md` showing the new relaxation row, and `mise run lint` exiting 0.

## Change verification

```sh
mise run backend:test-package ./internal/modules/backups/...
mise run backend:test-package ./internal/app/...
mise run backend:test-package ./internal/modules/databases ./internal/features/telemetry ./cmd/...
mise run backend:test-package ./internal/features/...
mise run backend:test-package ./internal/lib/shutdown
```
