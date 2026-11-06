## Context

See `proposal.md` for motivation and scope. This design is needed because merging the lifecycles can accidentally change the order of cancellation, validation, and publication.

- `backend/internal/modules/backups/internal/physical/backuping/backuper.go:135` and `:245` implement separate orchestration methods with repeated common settings, result persistence, logging, and notifications.
- `backend/internal/modules/backups/internal/physical/backuping/create_full_executor.go:21` and `create_incremental_executor.go:27` repeat credentials, attempt naming, slot ownership, streaming, timing, and metadata upload.
- `backend/internal/modules/backups/internal/physical/backuping/backup_run.go:24` already supplies `CommonBackupSpec`; the two enclosing specs add the backup row and incremental inputs.
- `backend/internal/modules/backups/internal/physical/backuping/metadata.go:32` and `:70` repeat metadata fields. Incrementals additionally record chain references and omit the parent-incremental reference when the parent is the root full backup.
- `backend/internal/modules/backups/internal/physical/backuping/scheduler_decision.go:37` calls `decideImmediateReplacementFull`. It is active scheduling behavior, not dead code.
- `backend/internal/modules/backups/internal/physical/backuping/backuper.go:729` already preserves completed results during shutdown. The prerequisite shutdown repair is implemented and has FULL and INCREMENTAL coverage in `physicalapitests/backuping/shutdown_test.go:21`.

The implementation stays inside the backups module, following root `AGENTS.md`, `backend/AGENTS.md`, and `docs/architecture.md` sections "Modules", "Long-running work", "Stored files", and "Physical backups". The lifecycle owner, storage module, engine boundaries, and process supervision remain unchanged. No architecture-document change is needed because this changes internal code sharing without changing those rules.

## Goals / Non-Goals

**Goals:**

- Give shared orchestration and execution steps one owner each.
- Keep kind-specific rules visible and retain their existing order and outcomes.
- Remove the redundant executor objects and obsolete internal specs once their callers use the shared implementation.

**Non-Goals:**

- Generalizing execution beyond the two physical backup kinds.
- Merging scheduler decisions with execution or moving domain rules into the PostgreSQL engine.
- Changing any stored field, serialized metadata shape, error reason, or externally visible backup result.

## Decisions

### 1. Share both orchestration and execution

Replace `runFullBackup` and `runIncrementalBackup` with one `runBackup` method. Keep the notification choice in a named run-options value alongside the backup so the method does not need a long parameter list. Construct `CommonBackupSpec` and its progress callback once. Use the backuper's logger with the backup kind as a structured attribute.

The common executor owns credentials and cleanup, label construction, attempt-name persistence, `WithBackupSlot`, `streamWithCodecFallback`, timestamps, metadata upload, and receipt accumulation. Remove `FullExecutor`, `IncrementalExecutor`, their constructors, and the corresponding service/backuper fields after replacing their internal callers. Reuse `CommonBackupSpec` with a named execution spec carrying the backup and kind-specific inputs. Return the execution result directly: both removed executors returned a nil Go error on every path, so their unused error channel and its unreachable orchestration branch are removed.

```text
MakeBackup
  --> runBackup
      --> open context, prepare run inputs, build common spec
      --> executeBackup
          --> credentials, kind-specific preflight and manifest preparation
          --> backup slot, codec fallback, stream
          --> kind-specific result checks
          --> timing, metadata, receipts
      --> incremental retry classification when applicable
      --> shutdown classification
      --> detached persistence
      --> notification after successful persistence
```

Alternative rejected: merge only orchestration. That leaves duplicate slot, attempt, timing, and publication handling in the executors. Keeping two wrappers around the shared executor also adds no value because these executor objects have no state and only internal callers.

### 2. Use a small internal strategy for kind-specific steps

Select the FULL or INCREMENTAL behavior once per execution. Express the differences through small named functions and a concrete internal strategy value; do not add a registry or exported extension interface. Keep preparation state local to the run and release the downloaded manifest on every exit.

Preserve the following sequence and distinctions:

| Stage | FULL | INCREMENTAL |
| --- | --- | --- |
| Before execution | Best-effort source-size measurement | Resolve the parent row and storage reader; derive incremental cadence |
| Executor preflight | Verify full timeline compatibility | Verify root timeline compatibility, check WAL summarizer readiness, then download/decrypt the parent manifest |
| Stream | No incremental manifest path; full error classifier | Parent manifest path; incremental error classifier |
| Failed stream | Recheck against the full preflight timeline | Recheck against the root full timeline |
| Successful stream | Validate start LSN against history; discard invalid artifacts; best-effort history upload on timelines above 1 | Keep existing incremental success handling |
| Before shutdown classification | No incremental retry logic | Exhausted retry budget changes ERROR to CHAIN_BROKEN only when the cause is not application shutdown |
| Result persistence | Include measured raw size when available | Keep raw size absent |

Retain existing statuses, error reasons, and messages for preflight refusals, missing manifests, stream failures, storage failures, and slot failures. Preserve the current outer panic recovery. Keep cancellation-sensitive work on the original context; detach only where the existing implementation already detaches it.

Alternative rejected: a single large method with repeated kind checks at every shared step. It hides the genuine differences inside the lifecycle. A general strategy framework is also rejected because two fixed kinds do not need registration or runtime extensibility.

### 3. Build common metadata once and preserve publication boundaries

Build the shared `catalog.BackupMetadata` fields once, then fill the backup type and incremental chain references according to the backup kind. Preserve the existing normalization that omits `ParentIncrementalBackupID` when the parent equals `RootFullBackupID`. Reuse the existing `uploadMetadata` implementation and its independent timeout.

Keep a unique object key for every codec attempt and persist it under publication validation before streaming bytes. Keep write receipts with the result until `persistResult` publishes through `BackupLifecycle`. Do not publish inside the executor. Preserve timestamp placement: execution duration includes preflight and streaming but excludes metadata upload; `CompletedAt` is populated on unsuccessful exits as well.

Keep `classifyShutdown` after incremental retry classification and before detached persistence. Completed execution remains completed during shutdown, while failed required accompanying writes remain failures. A persistence failure or superseded backup must not send a success notification.

Alternative rejected: sharing only metadata upload. Upload is already shared; the duplication is in metadata construction. Moving publication into the executor is rejected because it would split lifecycle ownership and change the storage transaction boundary described in `docs/architecture.md`.

### 4. Keep immediate replacement scheduling intact

Leave `decideImmediateReplacementFull` and its caller unchanged. A full backup interrupted by failover or an incremental chain broken by a timeline switch must retain its current immediate replacement behavior.

Alternative rejected: remove the helper as part of executor cleanup. Its active call precedes cadence checks, so removal would change when a usable chain becomes available. Scheduler redesign is outside this refactor.

### 5. Verify existing observable behavior through the current suites

Run the existing internal backuping tests and physical API tests, including shutdown, retries, slots, storage publication, and metadata failures. Run both PostgreSQL 17 and 18 physical end-to-end packages without a scenario filter: they exercise full-only recovery, incremental chains, target-time recovery, and promotion behavior with real clusters. Run the routine suite as the final broad regression check.

Alternative rejected: add tests tied to the new internal strategy or rewrite existing scenarios to fit the implementation. The behavior is unchanged and already exercised through the API and actual recovery. If implementation exposes a separate defect, follow the testing skill's reproduce-first rule and update the change artifacts for that repair.

## Risks / Trade-offs

- [Different preflight paths become one] -> Keep the kind-specific order explicit, especially summarizer checks before parent-manifest download and full-only post-stream validation.
- [Shutdown or cancellation changes file ownership] -> Preserve context boundaries, receipt accumulation, and lifecycle publication; rerun the existing shutdown and stale-publication scenarios.
- [Metadata changes break restore-chain discovery] -> Preserve field values and root-parent normalization; require successful real recovery for both supported PostgreSQL test versions.
- [Abstraction replaces duplication with indirection] -> Keep one concrete strategy local to this package, with small functions and no extension framework. A specific line reduction is not an acceptance criterion.

## Migration Plan

No migration or data conversion is required. Existing archives and metadata remain readable without modification. Deploy through the usual application release; rollback uses the previous binary because this refactor changes no persisted format or schema.

Planning artifacts are the only documentation additions. Main specs and agent instructions remain unchanged; update a reference in existing documentation only if implementation actually removes or renames something it mentions.
