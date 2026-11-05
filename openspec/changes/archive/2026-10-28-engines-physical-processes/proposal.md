## Why

The old physical code starts `pg_basebackup` and `pg_receivewal` itself with `exec.CommandContext`. Neither runs in a process group of its own. A stop sends SIGINT to the tool alone, and nothing kills it when it ignores that signal: `WaitDelay` only stops waiting for its output. `pg_basebackup` has no parent-death signal, so a crash of Databasus leaves it running against the source. `runStream` in `stream.go` handles the pipes and waits for the process by hand, and seven bare `go` statements in the moved code hold that together, among them a `Wait` that is abandoned after 30 seconds while its goroutine stays behind. `docs/architecture.md`, "Long-running work", requires every external tool to run through `lib/process` and every goroutine to start through a `lib` helper. The dumps and restores of logical backups already do since `engines-dump-restore`.

This is the second half of the `engines-physical` step of milestone 10, after `engines-physical-probes`. It moves the two tools into the physical engine and onto `lib/process`, together with what only they need: the byte stall watcher, the progress poller, the codec order and the rebuilt `backup_manifest`. The diff carries every change to how these processes live and die, and nothing else. The 14 physical end-to-end scenarios that `physical-e2e-on-apptest` put on `apptest` run unchanged and show that backups, WAL streaming and restores still work.

## What Changes

- The physical engine streams a base backup, the `pg_basebackup` stream of one full or incremental backup; the glossary gains the term. It starts `pg_basebackup` for a full or an incremental backup with a given codec and writes the tar stream into the caller's writer. Along the way it rebuilds the backup manifest from that stream, watches for a stalled stream and reports progress. It returns the start and stop LSN, the timeline, the manifest and the streamed size, or an error that says what went wrong, with the tool's error output.
- The codec fallback order and the `--compress` value move into the engine with a codec type of its own, `pgphysical.Codec`. The old code keeps the loop over the codecs, because each attempt writes a storage file of its own, and maps the engine's codec to the stored value.
- The physical engine receives WAL into a directory through `pg_receivewal` and reports how the WAL receiver ended: stopped by its context, restarted on request, a failure a respawn may fix, a fatal failure, or a resume point the server has recycled. The supervision loop, the back pressure and the rebuilds stay in the old code.
- Both tools run through `lib/process`. That brings these accepted changes in how the processes behave:
  - a stop reaches the whole process group, not the tool alone;
  - a tool still running 10 seconds after SIGINT gets SIGKILL;
  - when Databasus dies, the kernel sends SIGKILL to `pg_basebackup` and `pg_receivewal`. Today `pg_receivewal` gets SIGTERM and `pg_basebackup` gets nothing;
  - the wait for an exited `pg_basebackup` is no longer abandoned after 30 seconds; the kill after 10 seconds bounds it instead.
- The fail message of a failed backup and the error of a fatal WAL receiver exit, which the "WAL streaming stopped" notification quotes, keep today's text. Both quote the tool's exit status and error output, without the `run <path>:` prefix of `lib/process`.
- The stall deadlines and the progress timing read the time from a `lib/clock` source, as `docs/architecture.md` requires for time that drives behavior.
- `lib/process` needs no new API: its `Spec` already streams output into a writer, stops the group on the end of the context and bounds the wait.
- The bare `go` statements of `stream.go`, `pg_basebackup.go`, `byte_stall.go`, `progress.go` and `receivewal_process.go` are gone. The output copies run inside `os/exec`. The manifest walk, the stall watcher and the progress poller run in a `lib/background` group of the operation. The receiver's restart request cancels the receiver's context instead of racing a wait goroutine.
- The tests that start a fake tool through `sh -c` switch to `lib/toolstub`. `lib/process` gains the one test it lacks: a tool dies when its parent process dies. The engine tests pin the 10-second kill of a tool that ignores the interrupt, and the old code's tests pin the cancelled outcome, the fail message and the back-pressure restart.

## Capabilities

### New Capabilities

- `physical-backups/client-processes`: the client tools that physical backups and WAL streaming run on the Databasus host end together with their backup or stream, and together with Databasus, and a failed backup's message keeps naming what the tool reported.

### Modified Capabilities

None. What a backup, a WAL stream or a restore produces, and every outcome they record, stays as the specs `physical-backups/backup-method`, `physical-backups/wal-archiving` and `physical-backups/wal-slot-rebuild` state.

## Out of scope

- Everything `engines-physical-probes` moves: `walmath`, the SQL probes and the `pg_receivewal` directory reading. This change builds on it.
- Moving state, loops, repositories or the WAL uploader into the `backups` module, and ending the bare `go` statements outside the moved code: the scheduler, the startup cleanup, the WAL upload dispatcher and the WAL stream supervisor's loops. They are `physical-backups-module`.
- `pg_combinebackup`, which runs in the recovery script on the user's host, not in Databasus.
- Changing the stall thresholds, the progress interval, the codec order, the `pg_basebackup` and `pg_receivewal` arguments or the classification of their error output.
- Remembering the working codec per database. The spec `physical-backups/backup-method` promises it, but the code tries zstd first on every backup. That is a separate fix.
- Recording a running physical backup as failed at shutdown. Today it is failed by the recovery at the next start, and `physical-backups-module` changes that.
- The end-to-end scenarios under `internal/modules/backups/e2e/postgresqlphysical` and `internal/app/apptest/physicaltest`, which must not change.

## Impact

- **Backend, new and changed:** `backend/internal/engines/postgresql/physical` (the base backup, the manifest rebuild, the codecs, the stall watcher, the progress poller and the WAL receiver), and one new test in `backend/internal/lib/process`.
- **Backend, reduced:** `backend/internal/features/backups/backups/usecases/physical/postgresql` loses `pg_basebackup.go`, `byte_stall.go`, `progress.go`, `manifest.go`, `compression.go` and the process code of `stream.go` and `receivewal_process.go`. The storage write, the encryption, the manifest sidecar upload, the codec loop and the mapping to stored outcomes stay.
- **Not affected:** the schema, the contract, the frontend and every stored outcome and fail message.
- **Documents:** `docs/architecture.md`, "Engines and providers", where the list of what an engine knows gains streaming a physical base backup and receiving WAL. `docs/glossary.md` gains "Base backup". `temp/refactor/plan.md` (git-ignored, main checkout only) records that `lib/process` needed no extension and checks off 10b2.
- **Rules the change answers to:** the root `AGENTS.md` and `backend/AGENTS.md`.
