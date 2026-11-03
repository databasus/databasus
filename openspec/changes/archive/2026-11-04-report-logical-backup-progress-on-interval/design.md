## Context

See `proposal.md` for the motivation. The behavior is specified in `specs/logical-backups/spec.md`.

`progressReportingWriter` sits at the entry of the writer chain (`backend/internal/modules/backups/internal/pipeline/backup.go:274`, `:327`). After each write it compares the stored size against `progressReportIntervalMb` (`backup.go:22`, `:339`) and calls the listener on the copying goroutine. The listener runs `updateProgress` synchronously (`backend/internal/modules/backups/internal/logical/backup_run_service.go:53`, `:92`).

Three facts shape the approach:

- The stored size is already safe to read from another goroutine, because `CountingWriter` is atomic for the sake of zstd's internal goroutines (`backend/internal/lib/iox/counting_writer.go:11`).
- `updateProgress` changes only a row still in progress (`backend/internal/modules/backups/internal/logical/backup_repository.go:184`). `complete` writes the final size (`backup_repository.go:198`), and `fail` and `cancel` reset it to zero (`:213`, `:219`).
- The logical list shows the duration the server stored (`frontend/src/pages/databases/ui/logical-backups/LogicalBackupsTable.tsx:78`), and the list refreshes every second (`frontend/src/pages/databases/ui/logical-backups/LogicalBackupsComponent.tsx:23`). The physical list computes a running duration in the browser (`frontend/src/pages/databases/lib/physical-backups/getPhysicalBackupDuration.ts:19`).

The approach is constrained by "Long-running work" in `backend/AGENTS.md`, which requires goroutines to start through the `lib` helpers, gives periodic jobs to `background.Runner` and takes the current time from the clock in `Deps`, and by "Code is split into logical units" in the root `AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- The copy of a dump never waits for a progress write.
- No progress write outlives the backup run that started it.

**Non-Goals:**

- Refreshing the duration outside the stream. Before it, `runBackup` opens the dump source and reads its sizes (`pipeline/backup.go:79-86`). After it, `publishBackup` writes the sidecar for up to `metadataWriteTimeout` and runs the publishing transaction. The duration holds still in those phases, which is what it does today, and the spec states it.
- Measuring the commit rate or the table growth before and after. The change follows from the code path, and the estimate in the proposal stays an estimate.

## Decisions

### Report from a goroutine with a ticker

`streamDumpToStorage` starts a reporter before the dump. Each tick reads the stored size and calls `OnProgress`. `progressReportingWriter` and `progressReportIntervalMb` are deleted, and the chain's input becomes the dump output directly. The reporter is a unit of its own in `pipeline/backup_progress.go`.

Alternative rejected: a time check inside `Write`. It needs about 5 lines and no concurrency and reduces the load equally, but the copy would still wait for one `UPDATE` every 5 seconds, a stuck `UPDATE` would stop the dump because `updateProgress` has no timeout of its own, and the duration would freeze while the dump is quiet. The user chose the goroutine after comparing the two.

### Start the reporter through `background.Group`

The reporter runs in a `background.Group` of its own, the same way as the physical reporter (`backend/internal/engines/postgresql/physical/base_backup.go:98`). A panic in a tick is logged and does not fail the backup, and a failed save is logged by `recordProgress` (`logical/backup_run_service.go:98`), which matches the spec. The final report still runs on the backup's own goroutine, where a panic fails the backup, as it does today.

Alternative rejected: a bare `go` statement. `backend/AGENTS.md` forbids it, and a panic would kill the process along with every other running backup.

### Stop and wait on every exit path, before the final report

The stop function cancels the reporter's context and waits for the group with a context that cannot end, so the wait covers an `UPDATE` already in flight. It is wrapped in `sync.OnceFunc`. `streamDumpToStorage` defers it right after the start, which covers errors and panics, and calls it explicitly once `finishBackupStream` returns, before the final `OnProgress`.

The reporter's context stops only the ticking. The `UPDATE` itself runs on `runCtx`, captured by the listener in `logical`, so a storage failure that ends `backupCtx` does not abort a write in flight, while a stop of the run does.

Alternative rejected: stop without waiting and rely on the `IN_PROGRESS` guard in `updateProgress`. The guard protects the outcome of the row, but the goroutine could still reach the database after the run has ended. `docs/testing.md:79` fails a test whose work outlives its cleanup, and at shutdown such a write would race the closing of the pool. An unordered late tick could also overwrite the final report with a smaller size in the window before `complete`.

### Write on every tick, also when the size is unchanged

Unlike the physical reporter, which skips an unchanged size (`backend/internal/engines/postgresql/physical/progress.go:24`), the logical reporter always calls the listener. The listener computes the duration, so a skip would freeze the duration the list shows while the dump is quiet, for example while zstd buffers or the tool waits for a lock. The physical list has no such problem because the browser computes its duration.

Alternative rejected: skip an unchanged size to match the physical reporter. It saves at most one `UPDATE` per 5 seconds per backup and keeps the frozen duration.

### Pass the interval through `BackupSpec.ProgressInterval`

`logical` sets it from a constant of 5 seconds, as `backend/internal/modules/backups/internal/physical/backuping/executor.go:93` does for physical backups. Tests pass their own interval.

Alternative rejected: a constant inside `pipeline`. A pipeline test could not separate the report made at completion from a tick without depending on how fast the machine runs the dump.

### Tick on a real timer, not on the clock from `Deps`

The reporter uses `time.NewTicker`, like the physical reporter. The duration keeps coming from `s.clock` in `logical`. `lib/clock` has no timers, and the pipeline's dump is a real process that moves in real time. `backend/AGENTS.md` asks for the current time from the clock, and the reporter reads no current time. Its rule that a periodic job goes to `background.Runner`, with no ticker loop of its own, and the rule in `docs/architecture.md` that time driving behavior comes from a clock in `Deps` cover named application jobs. This reporter is part of one backup run and stops with it, the same pattern as the physical reporter and `walstreaming/wal_rotation.go`. A slow save makes `time.Ticker` drop ticks, so a refresh due during a save is skipped, as the spec says.

Alternative rejected: compare `s.clock.Now()` against the last report. Module tests hold that clock still, so no tick would ever fire there, and it needs a time check in the path the change is removing.

### Keep the final report

`complete` writes the final size too, but the sidecar is written first and may take up to `metadataWriteTimeout`, 30 seconds (`backup_run_service.go:21`). The final `OnProgress` (`backup.go:160`) keeps the shown size correct during that wait.

Alternative rejected: delete the final report. It removes three lines, `sync.OnceFunc`, the explicit stop and the spec's exception for the final refresh, but during the sidecar write the list would show a size up to 5 seconds stale, or 0 for a backup shorter than one interval, for up to 30 seconds.

### Test each behavior at the layer that observes it

A dump is a real process, which a `synctest` bubble cannot wait on, so the cadence is tested on the reporter function alone. The repository already uses `testing/synctest` for such tests. The report count, the blocked listener and the contained panic run `streamDumpToStorage` with a stub dump tool and an interval of one hour or one millisecond. What a user reads, the completed size, the advancing duration and the cancelled state, goes through module API tests. They move `tm.app.Clock` and hold the dump by locking a view that references no table. `server.lockItems` cannot hold it here, because the size query reads `items` before the stream starts (`backend/internal/engines/postgresql/logical/size.go:17`) and would wait on that lock, so no refresh would happen. Locking a view that references `items` locks `items` too. The duration test waits for one real 5-second tick.

A refresh whose save returns an error has no test of its own. Failing only that `UPDATE` from a module test would take a trigger on the table and a wait for a tick with no signal to wait on. The behavior follows from the types: `OnProgress` returns nothing, so the pipeline cannot fail on it, and `recordProgress` logs the error (`logical/backup_run_service.go:98`). The panic scenario covers the failed tick.

Alternative rejected: only module API tests. A slow-database or panicking listener cannot be caused through the API, and exact counts would need many real intervals.

## Risks / Trade-offs

- [The wait in the stop function blocks on an `UPDATE` that hangs.] → It ends when `runCtx` ends, and the publishing transaction that follows would hang on the same database, so this adds no new way to hang.
- [A failing internal database logs one error per tick.] → This is less noise than the per-megabyte logging of today.
- [Backups shorter than 5 seconds show no running size.] → They show size 0 and duration 0 until the final report, about what a backup under one megabyte shows today, and only for those few seconds.
- [The stored size of an encrypted backup leaves out the encryption overhead.] → `CountingWriter` sits before encryption (`pipeline/backup.go:257`), today and after this change. The spec speaks of the size the dump reached, and the size check in tests uses an unencrypted backup.
- [Backups slower than 0.2 MB/s write slightly more often than today.] → The rate stays bounded at one `UPDATE` per 5 seconds per backup.
