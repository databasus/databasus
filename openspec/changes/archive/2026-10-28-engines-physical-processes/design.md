## Context

See proposal.md, "Why", for the motivation, and the spec `physical-backups/client-processes` for the behavior. Paths are under `backend/internal/` unless they start with another top-level directory. `UP` stands for `features/backups/backups/usecases/physical/postgresql`, and `EP` for `engines/postgresql/physical`. This change starts from the code as `engines-physical-probes` leaves it.

**One base backup attempt today.** `streamWithCodecFallback` (`UP/executor.go:64`) loops over `codecFallbackOrder` (`UP/compression.go:14`). For each codec it saves a fresh file name on the backup row and calls `runStream` (`UP/stream.go:91`) with a `buildCmd` closure over `newPgBasebackupCommand` (`UP/pg_basebackup.go:45`). `runStream` then:

1. derives `streamCtx` with a cancel cause and builds the command bound to it, with `cmd.Cancel` sending SIGINT and `WaitDelay` of 10 seconds (`UP/pg_basebackup.go:119-123`);
2. takes `StdoutPipe` and `StderrPipe`, and starts a goroutine that reads stderr to the end (`UP/pg_basebackup.go:206`);
3. builds encryption over an `io.Pipe` into the storage's background write, whose failure cancels `streamCtx` with the save error;
4. wraps the storage branch in a `ByteCounter` that the byte stall watcher and the progress poller read, each from a goroutine of its own (`UP/byte_stall.go:79`, `UP/progress.go:26`);
5. starts the manifest walk on a second pipe in a goroutine (`UP/stream.go:154`);
6. copies stdout into `MultiWriter(counter, manifestWriter)` in a goroutine (`UP/stream.go:196`), kills the process if `streamCtx` ended (`UP/stream.go:204`), drains stderr and waits through `waitCmdWithTimeout`, which abandons `Wait` after 30 seconds (`UP/byte_stall.go:141`);
7. picks the outcome in a fixed order (`UP/stream.go:233-282`): stall, codec rejection, manifest walk failure, storage failure, the tool's failure, and success, which parses the LSNs, serializes the manifest and uploads the encrypted sidecar.

A failure of the tool becomes the fail message `fmt.Sprintf("%v; stderr: %s", streamErr, truncateStderr(stderr))` (`UP/create_full_uc.go:173`, `UP/create_incremental_uc.go:333`), where `streamErr` is the `*exec.ExitError` of `Wait`.

**The receiver today.** `spawnAndSupervise` (`UP/receivewal_process.go:46`) builds the command with SIGINT on cancel, `WaitDelay` of 10 seconds and `Pdeathsig: SIGTERM` (`UP/receivewal_process.go:251`). It runs `Wait` in a goroutine (`:97`) and selects on the parent context, `restartSignal` and the exit. `signalRestart` (`UP/wal_stream.go:550`) puts a token into a channel of one, and the supervision loop drains a stale token before each spawn (`UP/wal_stream.go:402`). The exit is classified from the stderr: `isResumeMismatchError` and `isFatalReceivewalError` (`UP/receivewal_process.go:178`, `:154`).

**`lib/process` today.** `process.Start` (`lib/process/process.go:44`) runs the tool in its own process group with `Pdeathsig: SIGKILL`. When the context ends it sends `StopSignal` to the group, and SIGKILL after `KillAfter`. `WaitDelay` is `KillAfter`, so `Wait` stops waiting for output a child holds open. `Spec.Stdout` and `Spec.Stderr` are writers that `os/exec` copies into from goroutines of its own, and `Wait` returns only after those copies end. `Wait` wraps the error as `run <path>: ...` and keeps the `*exec.ExitError` reachable through `errors.As`.

**Constraints.** `docs/architecture.md`, "Long-running work": external tools only through `lib/process`, goroutines only through `lib` helpers. `backend/AGENTS.md`, "Long-running work", repeats both, and "Where code goes" places engine code without tables in `internal/engines/<engine>`. Engines know nothing of storages, encryption or the backup catalog (`docs/architecture.md`, "Backend layout"). `docs/architecture.md`, "Physical backups", fixes the `pg_basebackup` arguments, the server-side compression with its fallback and the rebuilt manifest. The moved code falls under the full lint of the target layout.

## Goals / Non-Goals

**Goals:**

- `pg_basebackup` and `pg_receivewal` start only from the engine, through `lib/process`.
- The moved code has no bare `go`, no `os/exec` and no wait that is abandoned with its goroutine left running.
- Every outcome and fail message of a backup and every disposition of a receiver exit stays as it is, apart from the process changes the proposal lists.

**Non-Goals:**

- Changing the outcome order, the thresholds, the arguments or the stderr classification.
- Moving the storage write, the encryption or the sidecar upload, which belong to the backup and not to the engine.

## Decisions

### 1. `lib/process` stays as it is

The four gaps the rewrite plan expected `lib/process` to fill are already covered:

| Need | Today | With `lib/process` |
|---|---|---|
| Stream stdout and stderr | `StdoutPipe`, `StderrPipe` and a goroutine each | `Spec.Stdout` is the counting multi-writer and `Spec.Stderr` a buffer; `os/exec` copies both, and `Wait` returns after the copies end |
| Kill a running process | `cmd.Process.Kill()` after an aborted copy (`UP/stream.go:204`) | the stream context ends, `lib/process` sends SIGINT to the group and SIGKILL after `KillAfter` |
| Bounded wait | `waitCmdWithTimeout` abandons `Wait` after 30 seconds | the SIGKILL after `KillAfter` ends the process, and `WaitDelay` bounds the wait for output a child holds open |
| Replace the tool in tests | `buildCmd` returns `sh -c` (`UP/stream_test.go:109`) | `lib/toolstub` writes a stub at the path `pgshared.ClientTools.GetExecutable` returns, as the logical engine tests do |

The "drain stderr before `Wait`" logic of `stderrCapture.drain` disappears too. `os/exec` finishes its own copy of stderr before `Wait` returns, so a fast-exiting tool's rejection line cannot be cut off.

Alternatives considered:

- *Add `Signal`, `StdoutPipe` and a timed `Wait` to `lib/process`.* That is an API nobody needs once output goes to writers, and a timed `Wait` would bring back the abandoned goroutine.
- *Keep `waitCmdWithTimeout` around `Running.Wait`.* With SIGKILL after 10 seconds, `Wait` can only hang on a process stuck in uninterruptible sleep. The old timeout did not end such a process either: it left the goroutine and the process behind and reported a failure. After this change the backup waits for the kernel instead. A network client in uninterruptible sleep needs a broken disk or NFS on the Databasus host, and the backup recovery at the next start still fails the row.

### 2. The engine streams the base backup into a writer; the backup owns storage and encryption

A base backup is the `pg_basebackup` stream of one full or incremental physical backup; the change adds the term to `docs/glossary.md`. `Source.StreamBaseBackup(ctx, spec BaseBackupSpec, output io.Writer)` takes in `BaseBackupSpec` the client tools, the version, the backup label, the codec, the parent manifest path for an incremental, the system identifier, the clock, the stall thresholds, the progress interval with its callback and a logger. It writes the tar stream into `output` and returns `BaseBackupResult`: start and stop LSN, timeline, the serialized manifest and the streamed size.

Inside, stdout goes into `MultiWriter(counter -> output, manifestPipe)`. The counter sits in front of the caller's writer, so it counts the bytes the storage accepted, as now. The caller's writer is the encryption over the storage pipe, and a pipe write returns only when the upload reads it. The manifest walk, the stall watcher and the progress poller run in a `background.Group` that the call creates and waits for before it returns.

The old code keeps the codec loop, the file name per attempt, the writer chain into storage, the sidecar upload and the mapping to `streamOutcome`. When the storage write fails it cancels the context it passed to the engine with the save error as the cause, as now.

Alternatives considered:

- *The engine owns the whole attempt, storage included.* An engine knows nothing of storages or encryption (`docs/architecture.md`, "Backend layout"), and the logical engine stops at an `io.Writer` too.
- *The engine runs the codec fallback internally.* Each attempt must write its own storage file, because a name whose cleanup is pending cannot be written again (`UP/executor.go:207`). A loop inside the engine would push every attempt into one writer, a change to what lands in storage.
- *Return the manifest entries instead of the serialized manifest.* The format belongs to PostgreSQL, so serializing it is engine work, and the caller only stores the bytes.

### 3. The outcome order spans two layers and stays the same

The engine returns one error type, `BaseBackupError`, with a kind and the tool's stderr. Its kinds are the stalled stream, a rejected codec, a broken manifest (walk, LSN parse, serialization or a zero system identifier, each with today's message), the ended context and the tool's failure. It checks them in today's order: its own stall, the codec rejection from stderr, the manifest walk unless the stream was truncated, the ended context, then the tool's exit status. The old code maps them in the order of `UP/stream.go:233-282`: stall, codec, manifest, its own storage error, then the tool's failure through the classifier of the backup kind.

A storage failure before the tool starts arrives as an ended context, and the caller's storage error then wins. That keeps the special case at `UP/stream.go:187` without code of its own.

Alternative considered: *the engine takes a storage error callback and does the whole order itself.* The engine would then decide about a failure it cannot see, only to keep the order in one place. The tests of decision 8 pin the order across the two layers.

### 4. The fail message keeps the exit error, not the `lib/process` wrapping

`BaseBackupError` keeps the `*exec.ExitError` of the run. The classifiers format it as today: `exit status N; stderr: <the last 2048 bytes>`. This follows `clientToolFailure.getExitError` of the logical engine (`engines/postgresql/logical/client_tool_failure.go:33`). The engine also keeps the `exec.LookPath` check with today's message, `pg_basebackup binary not found at <path>`. The client tools are checked at startup, so that message only appears on a broken installation.

Alternative considered: *accept the `run <path>:` prefix.* The fail message reaches the user, and the prefix puts the install path of the client tools into it, which the spec forbids.

### 5. The receiver: a synchronous wait and a restart that cancels

`Source.ReceiveWal(ctx, spec WalReceiverSpec)` starts `pg_receivewal` with today's arguments and environment, `PGAPPNAME` included. It waits for it and returns `WalReceiverExit`: how it ended, the run time and, for a fatal exit, the error. The classification of stderr moves with it. The fatal error keeps today's text, `pg_receivewal fatal error: exit status N; stderr: ...` (`UP/receivewal_process.go:139`): it is built from the `*exec.ExitError`, as in decision 4, not from the `lib/process` wrapping. That error travels through the WAL stream supervisor into the "WAL streaming stopped" notification (`features/backups/backups/backuping/physical/wal_supervisor.go:337-343`), so the wrapping would put the install path of the client tools into a message users read. The engine exports the application name prefix, which the rebuild's ownership rule of `engines-physical-probes` reads.

`spawnAndSupervise` derives a cancellable context for each run and registers its cancel function under a mutex. `signalRestart` sets a "restart requested" flag and cancels the registered run, if any. The supervision loop clears the flag where it drains the channel today. Right after registering a new run it checks the flag, so a request that arrived between the clear and the registration still restarts that run at once, as the token in the channel does now. A run whose context ended through the restart reports an internal restart, and one whose parent context ended reports a stop.

Alternatives considered:

- *Keep the channel and wait in a `background.Group` goroutine.* That keeps a goroutine whose only job is to turn the channel into a cancellation.
- *A `Signal` method on `process.Running`.* SIGINT to the group is what the context's end already sends, so a second path to the same signal adds nothing.

### 6. The watcher threads run in a group the operation owns

The stall watcher and the progress poller keep their polling: every 10 seconds for the stall and every 5 seconds for progress. The stall deadlines and the elapsed time of a progress report are measured with the `clock.Source` from `lib/clock` that the spec carries, because `docs/architecture.md`, "Long-running work", and `backend/AGENTS.md` require time that drives behavior to come from a clock. The old code passes the system clock. The poll cadence stays a real ticker, as in `background.Runner`, because it only decides how often the clock is read. The progress callback writes to the database, and the comment at `UP/progress.go:15-24` requires that a slow write never blocks the stream. A poller off the write path keeps that. The two pollers and the manifest walk run in a `background.Group` from `lib/background`, which recovers a panic into an error and lets the call wait for them.

The thresholds and the cadences become fields of the spec instead of package variables: `FirstByteTimeout`, `StallTimeout`, `StallPollInterval` and `ProgressInterval`. The old code passes its constants. A test passes a `clock.Manual` and a short poll cadence and advances the clock past a deadline, which replaces the reassigned `progressReportInterval` (`UP/progress.go:13`).

Alternatives considered:

- *Reset a `time.AfterFunc` on every write for the stall.* It needs no poller, but the progress poller still needs one, and the stall would fire up to 10 seconds earlier than now.
- *A new stream helper package in `lib`.* `docs/architecture.md` lists `iox` for stream helpers, but no such package exists yet. Two pollers used by one engine do not justify creating it here.

### 7. Codec type of the engine

`pgphysical.Codec` has the values zstd, gzip and none, with `GetCodecFallbackOrder()` and `GetCompressArgument()` for the `--compress` value. One word names the concept, as in `docs/architecture.md`, "Physical backups". The stored `physical_enums.PhysicalBackupCompression` stays the column type. The old code maps one to the other with an exhaustive switch.

Alternative considered: *the engine takes `physical_enums`.* That would make the engine import the backup catalog's enums, which the layer direction forbids.

### 8. Tests

The process guarantees come from `process.Start`, and the linter keeps `os/exec` out of the engine, so the engine cannot bypass them. Each guarantee is tested once, where it is implemented:

- `lib/process` already tests that cancelling the context kills a child the tool started (`lib/process/process_test.go:27`) and that a tool ignoring the stop signal is killed after the grace period (`:79`). It gains the one test it lacks: the test binary starts itself as a helper parent that runs a stub tool and reports its PID, the test kills the helper with SIGKILL and waits with `testwait.Until` until the PID is gone.
- Engine tests with `lib/toolstub` replace the `sh -c` seam and cover what the engine decides:
  - a stub `pg_basebackup` that writes a tar and the verbose LSN lines yields the manifest and LSNs;
  - a stub that stops writing trips the stall watcher once a `clock.Manual` passes the deadline; before the first byte, the test observes the watcher's stall decision directly so delayed process teardown cannot hide an early cancellation;
  - a stub that prints the codec rejection returns that kind;
  - a stub that exits 1 keeps its exit status and stderr in the error and not the tool path;
  - a missing tool keeps today's message;
  - a stub that traps SIGINT and blocks is killed about 10 seconds after the context ends and not before, and `StreamBaseBackup` returns only after it exited. This pins the engine's own `KillAfter`. The same test runs for `ReceiveWal`.
- The receiver tests move: arguments and environment, the application name, the stderr classification, and the authentication failure that ends as fatal (`UP/receivewal_process_test.go:125`). A new stub test pins the fatal error's text without the tool path.
- The old code's tests, against a stub tool chosen through the client tools root that `CommonBackupSpec` and `WalStreamSpec` now carry instead of reading `config.GetEnv()`:
  - `stream_test.go` keeps the storage failure that wins over the rest, the attempt that leaves no receipt to publish and the sidecar upload, and drops the cases that moved. The `manifest_test.go` and `pg_basebackup_test.go` cases move to the engine;
  - a full backup attempt against a stub that starts a child and blocks is cancelled mid-run. The attempt returns, and the stub's and the child's PIDs are gone within 10 seconds;
  - a full backup attempt against a stub that exits 1 with a server rejection on stderr records a fail message with the exit status and the rejection, and without the stub's directory;
  - a WAL stream with a stub receiver that writes its PID and blocks gets segment files past a small queue limit. The receiver and its child delay their exit after SIGINT. The test drains the queue as soon as the interrupt is observed, before either process exits. The next receiver checks both previous PIDs for overlap, and the slot is still in `pg_replication_slots` of the source;
  - a restart requested before a run registers still restarts that run.
- The active-slot deletion test waits for the receiver application name joined to the slot's active PID. Slot creation can itself appear active briefly, so the slot's active flag alone does not establish that a receiver attached.
- The legacy physical usecase test binary holds a session network until its tests finish. This keeps its connection to the shared testcontainers reaper alive while tests use the external source, so a late bastion test cannot race a reaper retiring after the other packages ended.
- The 14 end-to-end scenarios run unchanged, among them the bastion that dies mid-stream and the slot lifecycle.

## Risks / Trade-offs

- [The `pg_basebackup` path carries every backup, and a mistake here fails all of them] → The end-to-end scenarios run unchanged, the stub tests cover each outcome kind, and the outcome order has a test per branch.
- [SIGKILL on Databasus's death stops `pg_receivewal` without flushing] → `--synchronous` already writes a segment before it is renamed into place, and the next start removes `.partial` files and lets the slot resend them (`UP/wal_stream.go:323`).
- [`Pdeathsig` fires when the OS thread that started the process exits, not only the process] → `lib/process` already accepts this for every dump and restore, and Go's runtime does not end such threads unless a goroutine locked them. This change makes it no worse than the logical tools.
- [A restart request lost between clearing the flag and registering a run] → The check right after registration closes that window, and a test pins it.
- [The parent death test is slow or flaky] → It waits for the PID with `testwait.Until` and a deadline, and kills nothing but its own helper.
- [The tests of the 10-second kill make the suite slower] → Two tests of about 10 seconds each, one per tool, which run in parallel with the rest of the package.

## Migration Plan

Nothing to deploy: no schema, configuration or stored data changes. A backup or WAL stream running during the upgrade ends as it does today when Databasus stops. Rollback is a revert of the change.
