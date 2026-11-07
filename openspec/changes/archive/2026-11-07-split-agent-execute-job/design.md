## Context

See `proposal.md` for motivation and scope. A design is needed to settle lifecycle ownership and preserve the different failure paths during extraction.

- `backend/internal/agent/runner/runner.go:176` starts `executeJob`, which ends at line 559. Heartbeat registration precedes version resolution and container creation at lines 191, 194 and 211.
- Container cleanup is registered at `runner.go:244`. The disk watcher starts at line 275, and its cancellation must remain distinguishable from an ordinary abort through the guard at line 263.
- Downloading already has a helper at `runner.go:561`; successful reporting already has one at line 631. Restoration and its verdict branches occupy the middle of the orchestration, with statistics collection at line 513.
- `backend/internal/agent/runner/failure_report.go:24` already builds failure reports and sanitizes diagnostics. The extraction can reuse that implementation.
- Existing coverage includes failure classification and cancellation in `backend/internal/agent/runner/runner_test.go:85`, and real restoration scenarios in `backend/internal/agent/e2e/verify_backup_test.go:20`.

Root `AGENTS.md` and `backend/AGENTS.md` govern naming, logical units, dependency injection, contexts and supervised work. The approach retains the agent boundaries and supervision described in `docs/architecture.md` sections "Verification agent" and "Long-running work"; no architecture rule changes.

## Goals / Non-Goals

**Goals:** Make the execution sequence readable without expanding each stage, and keep each stage's failure classification next to the operation it classifies.

**Non-Goals:** Introduce a reusable execution framework, move decisions into the restorer or backend, or redesign the existing reporting and disk-watcher components. The proposal lists the remaining scope exclusions.

## Decisions

### 1. Keep lifecycle ownership in executeJob

Keep per-job logging, context creation, heartbeat tracking, cleanup registration and disk-watcher ownership in `executeJob`. Extract container preparation into a private function, returning the container to the caller, which immediately registers termination. Preserve the current cleanup order: stop the watcher, terminate the container, untrack the verification, then cancel the job context.

```text
executeJob
  --> create job context and track verification
  --> prepare verification container
  --> register container cleanup and start disk watcher
  --> download backup using the existing helper
  --> restore verification backup
      --> ensure archive owner roles
      --> TimescaleDB pre-restore when applicable
      --> run pg_restore and classify its outcome
      --> TimescaleDB post-restore when applicable
  --> collect verification statistics
  --> check disk verdict and report success
  --> deferred cleanup
```

Keep the heartbeat registration before any operation that can create a container. Stage work uses the cancellable job context. Reports and termination retain their current use of the caller's context; do not introduce detached contexts or new timeouts. Keep the watcher inside `background.Group` and retain outer panic recovery in `Runner.Run`.

Alternative rejected: give every helper its own lifecycle. That could terminate the container or untrack the verification before later stages or reporting finish. A general pipeline would add lifecycle rules for one fixed sequence.

### 2. Extract domain steps and preserve their local verdicts

Use private domain functions for preparing the verification container, restoring the verification backup, and collecting verification statistics. Reuse `downloadBackupIntoContainer`, `reportFailure`, `reportDiskLimitExceeded` and `reportSuccess`. Helpers serving only this execution remain with its logical unit; a dedicated `job_execution.go` may hold that unit if needed to separate it from the claim loop.

Pass shared per-job inputs through small concrete named structs where required by the repository's parameter rules. Keep dependencies on `Runner`; do not duplicate its dependency bundle or add public interfaces. The caller's reporting context is the first function argument; the cancellable job context is explicit in the named job spec, keeping the two lifetimes distinct without adjacent context parameters. A stage that handles a failure through existing reporting functions returns an `isCompleted` predicate alongside any domain result: true means the stage finished and execution may continue; false means the caller must stop, including after a silent abort. The predicate does not mean that a report was accepted.

Keep these distinctions explicit rather than replacing them with one generic error handler:

| Condition | Preserved action |
| --- | --- |
| Unsupported PostgreSQL version or ordinary container creation failure | Failure report without restore result |
| Download abort sentinel, or ordinary cancellation during later work | Stop without a new report, subject to the existing guards |
| Watcher cancels the job because its disk budget was reached | Check the disk verdict before treating cancellation as a silent abort; use the existing terminal disk-limit report |
| Restore command cannot execute | Failure without restore exit code; retain available diagnostic |
| Restore tool reports disk exhaustion | Failure without restore exit code, preserving backend retry classification |
| Restore fails with non-ignorable errors | Failure with restore result and diagnostic |
| Restore has only ignorable errors | Continue to post-restore and statistics |
| TimescaleDB pre-restore fails | Failure without restore result |
| TimescaleDB post-restore fails | Failure with restore result |
| Statistics fail and the verification connection is dead | Failure without restore result |
| Statistics fail while the verification connection is alive | Failure with restore result |

Preserve the current checks at their existing boundaries, including the final disk check before success. Preserve report summaries, diagnostic construction, exit-code presence, log content and retry behavior. Keep a per-job copy of diagnostic secrets and add both container passwords before download or restore work.

Alternative rejected: return only a generic error and construct every failure in `executeJob`. That would leave the long verdict tree in the orchestrator or erase the distinctions above. Splitting by arbitrary line counts would also separate decisions from the operations they explain.

### 3. Use existing behavior tests as the acceptance boundary

Run the existing runner package tests before and after the refactor, then the unfiltered Docker end-to-end suite. The current real-agent suite covers PostgreSQL 12 through 18, TimescaleDB, a broken archive, a disk-budget failure and cleanup of a network from an earlier attempt. Existing runner tests exercise finer cancellation and exit-code distinctions.

Do not add tests of the new private function boundaries or rewrite assertions to fit the extraction. If a separate defect is exposed, apply the testing skill's reproduce-first rule and record the repair in these artifacts.

Alternative rejected: rely only on a successful restore end-to-end scenario. It cannot check the failure classifications that decide whether the backend retries or rejects a backup. Adding duplicate scenarios provides no additional acceptance criterion for this behavior-preserving change.

## Risks / Trade-offs

- [Cancellation hides a disk-limit verdict] -> Keep the disk guard ahead of silent-abort checks at every existing boundary, and retain the final guard before success.
- [Extraction changes cleanup or heartbeat visibility] -> Keep all lifetime ownership in the orchestrator and verify the existing abort, report-giveup and concurrent-restoration tests.
- [Failure information is lost between functions] -> Carry the existing restore result and diagnostic through the relevant stage, preserving the table above and secret sanitization.
- [More function calls make navigation harder] -> Use domain names, keep private helpers with their execution unit and reuse existing components. A specific line count is not an acceptance criterion.

## Migration Plan

No migration or configuration conversion is required. Release and rollback use the normal binary replacement process because no persisted data or public contract changes. Only these planning artifacts need documentation updates; update an existing reference only if the extraction changes a name or path that it actually documents.
