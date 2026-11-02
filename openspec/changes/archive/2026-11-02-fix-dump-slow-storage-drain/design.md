## Context

See `proposal.md` for the motivation and `specs/backup-streaming/spec.md` for the required behavior.

A dump writes through `lib/process`, which assigns the caller's `io.Writer` to `cmd.Stdout` (`backend/internal/lib/process/process.go:51`). For a writer that is not an `*os.File`, `os/exec` creates a pipe and copies from it in its own goroutine. After the process exits, `cmd.Wait` gives that goroutine only `cmd.WaitDelay` (`process.go:54`), then returns `exec.ErrWaitDelay` even for exit status 0.

The writer is the chain that ends in an `io.Pipe` (`backend/internal/modules/backups/internal/pipeline/backup.go:113`). The storage's background upload reads the other end, so a write blocks while the storage is busy. The limit therefore also counts time the storage spends uploading. It is 10 seconds for MySQL, MariaDB, MongoDB and `pg_basebackup` (`backend/internal/engines/mysqlfamily/client_tool.go:13`, `backend/internal/engines/mongodb/client_tool.go:19`, `backend/internal/engines/postgresql/physical/base_backup.go:22`), and one minute for PostgreSQL logical tools (`backend/internal/engines/postgresql/logical/client_tool.go:16`).

`os/exec` cannot tell a writer that is busy from a pipe that a left-behind descendant holds open. Telling them apart is the whole of this fix.

The approach is constrained by `backend/AGENTS.md`: external tools run through `lib/process`, and goroutines are tied to a context and turn a panic into an error. `docs/architecture.md` (Long-running work) is unchanged in substance and gains one sentence.

## Goals / Non-Goals

**Goals:**

- The limit after a tool's exit measures only time spent waiting for the tool's output, never time spent in the writer.
- Every caller of `lib/process` gets this behavior without opting in.

**Non-Goals:**

- A new stall limit for a writer. A logical tool that is still running can already block on a stalled storage until the context ends, so this adds no new kind of hang.
- Any change to `Stdin` or `Stderr` delivery. Both stay with `os/exec` and `WaitDelay`.

## Decisions

### The physical byte-stall limit stays and now also covers the wait after the tool's exit

`watchByteStall` (`backend/internal/engines/postgresql/physical/byte_stall.go:15`) runs until `running.Wait()` returns (`backend/internal/engines/postgresql/physical/base_backup.go:84-90,110,116`). `byteCounter.Write` counts bytes only after the downstream write returns (`backend/internal/engines/postgresql/physical/byte_counter.go:13`), and production sets `FirstByteTimeout` to 10 minutes, `StallTimeout` to 60 seconds and the poll to 10 seconds (`backend/internal/modules/backups/internal/physical/backuping/executor.go:90-92`). The watcher uses `FirstByteTimeout` until the counter leaves zero (`backend/internal/engines/postgresql/physical/byte_stall.go:33-36`). With this fix `Wait` blocks on the drain, so a write held past the budget in force after `pg_basebackup` exits ends the backup as a stalled stream: 60 to 70 seconds once data was accepted, about 10 minutes before that. The spec states both as physical limits.

Alternative rejected: stopping the watcher when the tool exits. A write held that long already trips the watcher while the tool runs, because a slow storage blocks the tool the same way mid-stream. A backup on such a storage fails before it reaches the exit in most runs, so the exemption would help only the last write and would leave a physical backup with no limit at all in that window. Loosening the mid-run limit is a separate product decision.

### Copy standard output ourselves through `os.Pipe`

`Start` creates an `os.Pipe` when `Spec.Stdout` is set and is not an `*os.File`, gives the write end to the child as `cmd.Stdout` and closes that end in the parent once the child has started. `os/exec` starts no copying goroutine for an `*os.File`, so `WaitDelay` no longer covers standard output. A new unit in `lib/process`, `output_drain.go`, owns the read end and the copying goroutine. The goroutine runs through `background.RunRecovering`, so a panic in a writer becomes an error. A failed start closes both ends.

Alternatives rejected:

- Raising `KillAfter` only moves the threshold: at one minute, PostgreSQL still fails below about 2 Mbit/s.
- Removing `WaitDelay` makes a left-behind descendant hang the run until its context ends, and removes the limit for `Stderr` and `Stdin`.
- Reading `StdoutPipe` from the engines leaves four copies of the same logic in the callers, against the single process helper the architecture asks for.

### The limit applies to each read after the tool exits

Once `cmd.Wait` returns, the drain sets a read deadline of `KillAfter` on the read end, which also interrupts a read already blocked. It sets the deadline again before every later read. Kernel pipes support deadlines through the runtime poller. A throwaway program confirmed the three outcomes on Go 1.26 with a limit of 1 second and a writer that takes 3 seconds: a slow writer finished with no error after 3 seconds, a failed tool returned its exit status after 3 seconds, and a held pipe returned `os.ErrDeadlineExceeded` after 1 second.

Refreshing per read is required: `Read` on an expired deadline fails before it looks at the pipe, so a single deadline set at exit would discard data still buffered after a long write.

An expired deadline is reported as `exec.ErrWaitDelay`, so the failure a caller sees stays the same as it is today.

Alternative rejected: copying to EOF and only then calling `Wait`. That order covers a descendant only after cancellation. Without cancellation the backup hangs until its context ends, as it could in v3.64.0, which read before it waited (`backend/internal/features/backups/backups/usecases/logical/mysql/create_backup_uc.go:270-271` at that tag).

### A shared writer shares one pipe

When `Spec.Stdout` and `Spec.Stderr` are the same writer, `Start` assigns the write end to both. `os/exec` shares a descriptor only when the two fields are equal (`childStderr` in `os/exec/exec.go`), and `CombinedOutput` passes one buffer as both (`process.go:110`). Replacing only `Stdout` would make two goroutines write to one `bytes.Buffer`: a data race, and an unreliable output order that an existing test asserts. The comparison recovers from a panic on uncomparable writer types, as `os/exec` does.

Alternative rejected: a locked writer inside `CombinedOutput`. It fixes the race for that one caller, leaves it for any other caller that passes one writer, and still loses the order.

### The default for every caller, with no `Spec` field

Any `Stdout` that is not an `*os.File` goes through the drain. Alternative rejected: an opt-in flag, because a new streaming caller that forgets it would silently bring the defect back.

### What `Wait` returns

The tool's exit error comes first. If the tool succeeded, the drain's error comes next, whether a writer error or `exec.ErrWaitDelay`. With an ended context, the context's error comes first, exactly as now. A writer error closes the read end, so a running tool gets `SIGPIPE`, as it does with the `os/exec` goroutine today.

`cmd.Wait` can itself return `exec.ErrWaitDelay` when a descendant also holds the stderr pipe, which `os/exec` still bounds. That closes the read end at once, so such a run fails after one `KillAfter` rather than two, as soon as a write in progress returns.

A descendant that holds the pipe without cancellation stays a failure even for exit status 0. We cannot prove the output is complete, and for a backup a false success is worse than a failure.

### Once the context ends, kill the group and wait for the writer to return

`Wait` selects on the drain and on the context. Once the context has ended, it kills the group (`process.go:88` already does so after a tool's exit), so a descendant in the group releases the pipe, and it keeps waiting for the copying goroutine. A descendant that left the group is covered by the per-read deadline. `Wait` never returns while the goroutine can still use the caller's writer.

`os/exec` behaves the same way: `awaitGoroutines` closes the pipes and then waits for its goroutines even after `WaitDelay` expires. A writer blocked at cancellation is released by its owner. The logical pipeline already closes its `io.Pipe` on an ended context (`backend/internal/modules/backups/internal/pipeline/backup.go`).

A `KillAfter` of zero keeps its current meaning of no limit for the read deadline.

Alternative rejected: abandoning the goroutine after `KillAfter`. A caller could then touch its writer chain, for example close the zstd encoder, while the goroutine is still writing into it, which `os/exec` never allows today.

### A stalled physical stream releases its output through its owner

A stall cancels only the context inside `StreamBaseBackup`, and the writer is an `io.Pipe` that `runStream` owns (`backend/internal/modules/backups/internal/physical/backuping/stream.go`). A write blocked there would hold `Wait` until the storage returned, so the stall would never be reported. `BaseBackupSpec` gains `ReleaseOutput`, which `StreamBaseBackup` calls when its stream context ends before the tool's run finishes. `runStream` closes the pipe with the cause and cancels its own stream context, which also ends an upload that never returns. The call is stopped once the run has finished, so a successful backup keeps its upload.

Alternatives rejected:

- A counted writer that returns when the context ends would need its own goroutine around the blocked write, and that goroutine could still be using the writer after `Wait` returned, which the decision above forbids.
- Closing the pipe only on a stall would leave a manifest failure, which cancels the stream the same way, with the same hang.

### The last encrypted chunk is flushed inside the stream

`EncryptingWriter` buffers up to 1 MiB and writes the last chunk only in `Close`. `BaseBackupSpec` gains `FlushOutput`, which `StreamBaseBackup` runs after the tool succeeds and before it stops the watcher and `ReleaseOutput`, so a storage that stops there fails as a stalled stream. `runStream` no longer closes the encrypting writer itself. After a failed tool nothing flushes that chunk, because the backup is already failed and the write could block on the same storage.

Alternative rejected: a timeout around the close in `runStream`, which would be a second stall limit with its own value next to the watcher.

The wait for the upload to finish after all data is delivered stays without a limit. The storage's own timeouts and a cancellation end it, and a limit there would need to know how long each provider takes to complete a file. The spec states this.

### PostgreSQL logical tools return to 10 seconds

`clientToolKillAfter` in `postgresql/logical` becomes 10 seconds and its comment is replaced by the one the other engines use. The minute worked around this defect. Alternative rejected: keeping the minute, which would leave one engine different with no reason left to give.

### Pin the behavior in `lib/process`, plus the physical stream tests

`postgresql/logical/dump.go:56`, `mysqlfamily/dump.go:57` and `mongodb/dump.go:42` pass the caller's output unchanged to `process.Run`, so `lib/process` alone decides the behavior of a logical dump. `TestRun` pins it with a short `KillAfter` and a private fake writer, as the existing test at `process_test.go:91` does with 200 ms. `pg_basebackup` is the one caller with a different writer and a stall watcher, so `TestStreamBaseBackup` has two scenarios of its own, both after the writer has accepted data: one hold past the 10-second limit, which is real time because the limit is a constant there, and one output that never accepts more, with a manual clock. `TestRunStream` covers the owner's side with a storage that never returns: plain and encrypted output, and a user's cancellation after the tool exited.

The holds are what the fakes simulate rather than waits for a condition, so `testwait` does not apply to them.

Alternatives rejected:

- A test per engine at `Source.Dump` adds about 12 seconds of real time per engine and exercises no engine-specific path.
- A pipeline test against a real database, like `storage_failure_test.go`, passes on the broken code. MySQL and MariaDB compress with zstd in the chain (`backend/internal/modules/backups/internal/pipeline/mysql_engine.go:20`), and the encoder buffers a small dump, so no write reaches the stalled storage before the tool exits.

## Risks / Trade-offs

- [Every caller of `lib/process` changes its output path, including the verification agent] → The agent's and `clienttools` tests are in the verification set, and scenarios pin the output order and the race under `-race`.
- [A writer that never returns after cancellation blocks `Wait`] → `os/exec` already behaves this way, and the pipeline releases its writer when the context ends.
- [`pg_restore` loses the extra 50 seconds on `Stdin`: a tool that succeeds without reading its input to the end while the storage read is stuck fails after 10 seconds instead of a minute] → MySQL and MongoDB restores already run with 10 seconds, and the restore tests are in the verification set.
- [A real dump against a real slow storage is not exercised by a test] → The dump path through the chain has its existing tests, and the new tests pin the exact point of failure in `lib/process`.
- [An engine could stop passing its output straight to `lib/process` without a test noticing] → The engines' existing dump tests pin that the output arrives, and a wrapper with its own limit would have to be added deliberately.
- [A physical backup on a storage that holds one write past the stall budget still fails as a stalled stream] → The spec states this, and a test pins it. The fix helps physical backups only for holds longer than 10 seconds and shorter than that budget: 60 seconds once data was accepted, 10 minutes before.
