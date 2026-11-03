Backend commands below run in `backend/` after `source ../scripts/fedora.sh` and `apply_libedit_shim`, as `docs/testing.md` says. `PIPELINE_TEST` stands for `go test -count=1 -run 'TestLogicalBackupProgress' ./internal/modules/backups/internal/pipeline/`.

## 1. Failing tests

- [x] 1.1 Add `ProgressInterval time.Duration` to `BackupSpec`, with no change in behavior. Set it to one hour in `newTestBackupSpec` and to a new constant `logicalBackupProgressInterval = 5 * time.Second` in `logical/backup_run_service.go`. Verify that `mise run backend:lint` passes and that `rg -n 'logicalBackupProgressInterval' backend/internal/modules/backups/internal/logical` shows the constant as `5 * time.Second` and as the value of `ProgressInterval`.
- [x] 1.2 In `pipeline/backup_progress_test.go`, add `TestLogicalBackupProgress` with the scenario "reports progress at most once per interval": a 3 MB incompressible stub dump with the one-hour interval produces exactly one report. Verify that `PIPELINE_TEST` fails with more than one report.
- [x] 1.3 Add the scenario "keeps copying while the progress listener is blocked", with a one-millisecond interval: the stub dump writes 2 MB, then waits for a gate file, then writes the rest; the test waits until the listener is entered and blocked, opens the gate, and requires the storage to read the end of the file while the listener is still blocked. Verify that `PIPELINE_TEST` with `-timeout 30s` fails on the current code because the storage never reaches the end of the file.

## 2. Reporter

- [x] 2.1 Add `pipeline/backup_progress.go` with the reporter that calls the listener with the stored size on every tick until its context ends, and start it from `streamDumpToStorage` through a `background.Group`, with a stop function under `sync.OnceFunc` that cancels and waits. Defer the stop, and call it after `finishBackupStream` and before the final `OnProgress`. In the same task delete `progressReportingWriter` and `progressReportIntervalMb`, pass the dump output as the chain's input, and update the comments on `OnProgress` and `backupWriterChain`. Verify that `PIPELINE_TEST` passes, that `rg -n 'progressReportingWriter|progressReportIntervalMb' backend` finds nothing, and that `mise run backend:lint` passes.

## 3. Remaining scenarios

- [x] 3.1 Add the pipeline scenario "records the final size when the backup completes between ticks": with the one-hour interval, the last report equals `StoredSizeMb`. Verify with `PIPELINE_TEST`.
- [x] 3.2 Add scenarios in a `testing/synctest` bubble on the reporter alone: one call per interval, a call on each interval while the size is unchanged, a tick skipped while the previous call is still running, and no call after the stop. Verify with `PIPELINE_TEST`.
- [x] 3.3 Add a pipeline scenario where the dump fails while a listener call is in flight, with a one-millisecond interval, requiring that no call starts after `streamDumpToStorage` returns. Verify with `PIPELINE_TEST -race`.
- [x] 3.4 Add the pipeline scenario "completes the backup when a periodic report panics", with a one-millisecond interval: the listener panics on its first call only, the stub dump waits at a gate until that call happened, the backup returns no error, and a capturing log handler holds the panic's record. Only the first call panics because the final report runs on the backup's own goroutine, where a panic does fail the backup. Verify with `PIPELINE_TEST`.
- [x] 3.5 Extend "returns 202 with a listed backup in progress that later completes with its size" in `logical_backup_start_test.go`: download the completed backup and require the read `SizeMb` to match the file's length, so a size left from a periodic refresh fails it. Verify with `TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 -run 'TestStartLogicalBackup' ./internal/modules/backups/`.
- [x] 3.6 In the same test function, add "advances the duration and keeps the size while the dump produces no output": start the backup with a new fixture `startQuietDumpBackup`, which locks a view that references no table, so the dump tool waits while the size query before it does not, and returns once the tool waits on the lock, so `startedAt` is already taken. `startBlockedSourceBackup` cannot serve here because the size query waits on its lock before the stream starts. Before moving the clock, read the backup and require `IN_PROGRESS`, `SizeMb` 0 and `DurationMs` 0. Then advance `tm.app.Clock` by one minute and wait with `testwait.Until` until the read `DurationMs` reaches one minute, and require `SizeMb` 0. The wait takes up to one real 5-second tick. Verify with the command of task 3.5.
- [x] 3.7 In `TestCancelLogicalBackup`, add "keeps the canceled status and size 0 after a cancel while the dump streams": start with `startQuietDumpBackup`, wait for a refresh as in task 3.6, cancel, wait for `CANCELED`, and require size 0. This pins what the user reads after a cancel. That a late refresh cannot change an ended backup is pinned by task 3.3 together with the `IN_PROGRESS` guard in `updateProgress`, because a quiet dump has size 0 with or without a refresh. Verify with `TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 -run 'TestCancelLogicalBackup' ./internal/modules/backups/`.

## 4. Formatting and lint

- [x] 4.1 Run `mise run backend:format` and verify that `mise run backend:lint` passes.

## Change verification

```sh
mise run test
```
