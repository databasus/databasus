## Context

See `proposal.md` for motivation and scope. The current implementation has three layers:

- `backend/internal/modules/storages/internal/files/pending_deletion.go:12` stores one pending obligation per file; `store.go:88` registers an obligation before upload, and `store.go:152` consumes matching receipts when publishing. Queue rows therefore also protect files that might still be published.
- `backend/internal/modules/storages/internal/files/deletion_batch_store.go:10` adds a captured file set, counters and a 24-hour deadline. `deletion_batch_repository.go:149` applies the five-failure policy independently of ordinary retries.
- `backend/internal/modules/backups/internal/storageswitch/service.go:32` creates a switch record referring to a batch. `outcome_service.go:36` exposes paged history; `backend/migrations/20261030120000_storage_switch_cleanup.sql:82` creates the history table.
- `frontend/src/pages/databases/ui/DatabasesComponent.tsx:288` renders workspace history above the list, and `backup-config/ShowLogicalBackupConfigComponent.tsx:101` renders database history before the configuration fields. The physical view and editors also render it.
- `backend/internal/modules/storages/internal/storage/deletion_service.go:81` deletes the storage row, drains queued files in the request, settles batches and audits leftovers. Workspace deletion uses that same path at `:50`.
- Configuration saves return a switch operation instead of the saved config in `backend/internal/modules/backups/internal/httpapi/backup_config_handler.go:89`; transfer has a similar branch in `transfer_handler.go:51`.

The approach follows root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`: domain logic stays in its owning module, cross-module calls use `module.go`, contracts precede generated code, and frontend page state and effects stay with their consumers. `docs/architecture.md` sections Modules, Stored files, Storage switches, HTTP API and Frontend define those boundaries. This change updates the Storage switches section and glossary entries because switch records and batches cease to exist.

## Goals / Non-Goals

**Goals:** one durable source of unfinished deletion work; truthful current cleanup counts; safe explicit abandonment; fewer response variants and no operation history.

**Non-Goals:** reconstructing completed cleanup, discovering unknown provider objects, or changing receipt validation, storage-generation checks, retention, backup-chain selection or provider deletion semantics. Refer to the proposal for product exclusions.

## Decisions

### 1. Reuse the file queue and remove batches and switch records

Rename `storages_pending_deletions` to `storages_pending_file_deletions`, including its constraints and indexes. Keep uniqueness on `(storage_id, file_name)`, the storage foreign key, row IDs, generation, scheduling, attempt count and sanitized last error. Remove `is_ordinary`: every obligation has the same retry policy. Keep the existing background job, bounded provider calls, backoff, jitter, cross-instance claims and advisory write protection.

Add nullable `cleanup_database_id` and `cleanup_workspace_id` together, with a structural both-or-neither constraint and an index for current-scope aggregation. These identify cleanup requested by storage reassignment; they are not foreign keys, so deleting or transferring the database cannot cancel outstanding work. Set them through the storages module's public deletion-request struct in the same transaction as reassignment. The workspace is the database's authorized workspace after the change, the target workspace for a transfer.

Ordinary write registrations remain unscoped. A normal upload must not make its database display a storage-change cleanup notice. When reassignment takes an existing row away from a writer, increment its generation and attach scope if it has none. Once attributed, do not move that row to a later workspace or duplicate it for a second switch. Its one deletion satisfies the underlying obligation; no independent per-switch outcome is promised. Preserve the first scope when ordinary requests later touch that file.

Reject keeping an invisible switch-history layer: it retains bookkeeping with no user-facing purpose. Reject a separate `pending_backups_to_cleanup` table: existing obligations include WAL, metadata, partial uploads and canceled writes, and a second queue would duplicate claims and race handling. Reject deriving visible cleanup by counting every upload registration: these rows do not necessarily represent discarded files.

### 2. Keep reassignment atomic and make every file retryable

Replace `acceptStorageSwitch` and its batch creation with ordinary scoped file-deletion requests. The config/workspace change, old catalog removal and every known file obligation still commit together. Preserve the physical storage generation and last-selected-storage tracking, publication fencing, old-writer cancellation after commit, target-FULL preservation, new-FULL scheduling and WAL restart behavior. A late old writer's newly discarded file must join the same current cleanup when its originating reassignment scope is still available; if it cannot be attributed safely, it remains an ordinary obligation and is still covered by storage-deletion checks.

All failures retain their queue row and retry with bounded exponential backoff until absence is confirmed or storage deletion explicitly abandons work. Restarting preserves attempts and scheduling. Claiming a batch reserves its rows without increasing attempt counts; increment each count only when dispatching that file while its lease is valid. Rows skipped after an earlier deletion exhausts the batch lease retain their previous count and retry delay. Remove cutoff, expiry, batch settlement, terminal status and completed-item retention code. Worker success deletes only a matching generation; a late failure updates only the row it claimed, never inserts or recreates an abandoned row.

Reject preserving five attempts or 24 hours: reaching an arbitrary limit would silently abandon the queue the user expects us to finish. Reject changing physical backup concurrency to simplify deletion: these guards prevent stale backup publication and serve a different purpose.

Late physical writers discard their files through the same queue. If outstanding rows for that database and source storage still carry the accepting scope, discarded artifacts inherit it; lookup batches all source storages in one read. If no scope remains or multiple workspaces make the origin ambiguous, the files are ordinary deletion obligations. Timeline history artifacts and their metadata sidecars follow the same discard path after failed publication, in a transaction that survives producer cancellation. No separate history is retained to reconstruct attribution after completion or abandonment.

### 3. Return current cleanup with the existing configuration resource

Add required read-only `cleanup` to `BackupConfig`: `pendingFileCount` and `failedFileCount`, nonnegative integers, with failures a subset of pending files. Pending counts all scoped outstanding rows, including delayed or currently claimed work; failures count queued rows with a last deletion error. There is no total, percentage, deadline or completion record. A zero count means no known queued files in this scope, not a scan of the provider.

`GET /backup-configs/{databaseId}` authorizes the current database, then aggregates rows matching that database and its current workspace. The existing config list returns the same shape, using batched aggregation instead of one query per config. Summary composition checks current database-workspace access again, caching authorization once per workspace, so a concurrent transfer or a storage-filtered list cannot expose a scope the caller cannot view. Unsaved/default configs also return a summary without writing anything. Backups obtains summaries through the storages module's public API, never its tables. Input schemas do not accept `cleanup`.

`PUT /backup-configs/{databaseId}` always returns 200 with the acknowledged config and summary. `POST /databases/{databaseId}/transfer` always returns 204 once the move commits. Remove their 202/Location branches, `/storage-switches`, `/storage-switches/{storageSwitchId}` and all old switch schemas and clients. The request finishes the config change or transfer; file deletion is an independently queued effect, as with backup removal. `agent.yaml` is unaffected.

Current config viewers see only aggregate counts for the current database/workspace scope. They do not gain access to prior-workspace cleanup on a later transfer. No provider error, file path, source-storage identifier or credential is returned. Storage owners remain able to abandon all obligations for a storage even if their database was deleted or moved. Ordinary storage reads need no new cleanup fields: guarded deletion is authoritative.

Reject a new cleanup-history or queue-list endpoint: counts on the existing resource answer the UI's question with fewer API operations. Reject using queue disappearance as a saved successful result: explicit abandonment also removes rows.

### 4. Require explicit abandonment at storage deletion

Extend `DELETE /storages/{storageId}` with optional boolean `shouldAbandonCleanup`, default false. Authenticate, authorize editing and hold the storage row lock. First enforce current config and retained-backup references, returning 409 `in_use` even with abandonment. If any queue row exists and abandonment is false, return 409 `cleanup_pending` without side effects or provider calls. Add that code to `ProblemCode` and the six error dictionaries.

With no obligations, return 204 after ordinary deletion. With `shouldAbandonCleanup=true`, delete the storage and all its obligations in one transaction and audit the abandonment count and at most the existing bounded number of file names. Acknowledge possible remaining files, not confirmed existence. Do not perform a final synchronous drain and do not retain credentials elsewhere. A worker attempt already in progress may still finish, but no new attempt starts for the deleted storage and late results cannot recreate rows.

Queue creation must synchronize with storage deletion for every path: pre-upload registration, explicit deletion requests and reassignment. Use the existing storage row/foreign-key locking relationship consistently, so registration either wins and is seen by the guard or deletion wins and the producer handles the missing storage without starting a new upload. Never drop an obligation while the storage still exists. Retain advisory file locks and stale-generation guards; abandonment does not grant another writer permission to publish or reuse a file name.

The first delete after the ordinary remove confirmation omits abandonment. On `cleanup_pending`, replace that confirmation with: "We are still cleaning up this storage. Wait for cleanup to finish. If you delete it now, automatic cleanup will stop and any remaining files must be removed manually." Offer Wait and a destructive Delete anyway action. Only that second deliberate action sends `shouldAbandonCleanup=true`. Waiting closes the warning and changes nothing. A stale empty precheck cannot bypass the server guard. Disable duplicate submissions, retain the warning on failure, and reset acknowledgement on storage/session changes.

Reject silently draining before removal: it delays deletion and makes the promise depend on provider timing. Reject frontend-only warning checks: another request can enqueue files after a check. Reject an acknowledgement token tied to an exact queue count: the user explicitly abandons all remaining cleanup for this storage; the server still checks references and ownership at the final request.

### 5. Cascading workspace deletion cannot waive the warning

Workspace deletion uses the same non-abandoning guard inside its existing transaction. A workspace with databases still returns `workspace_not_empty`; retained references still return `in_use`. Otherwise any affected queue row returns `cleanup_pending` and rolls back the whole deletion. The translated workspace message directs the user to wait or delete affected storages individually with their explicit warning. Once they finish cleanup or are individually removed, the workspace can be deleted normally. A queue in another workspace is untouched even when its attribution names the deleted workspace.

Reject silently treating workspace deletion as abandonment: its existing confirmation does not explain the lost cleanup. Reject adding bulk abandonment to workspace deletion: it expands the flow beyond the requested storage action and adds a second force-delete contract.

### 6. Keep the live notice inside the Storage section

The database page owns a focused cleanup view and its polling state under `pages/databases`, composed by the logical and physical configuration sections. `entities/backup-configs` owns the generated summary alias and request facade, without importing storages or database-page code. Keep Storage and its notice available when scheduled backups are disabled, no current storage is selected, or either backup editor is open. The notice and read-retry action remain beside the Storage selection while editing. Render the acknowledged storage name/icon; below it show a small contextual progress icon, remaining count and muted explanation. Show a retry warning when `failedFileCount` is nonzero. Do not manufacture percentages from a moving queue.

Load the summary through configuration reads on entry and accept the configuration returned by a successful save as the acknowledged state, including its cleanup summary. Do not discard that acknowledgement for another read. Poll every two seconds while a known count is nonzero. Extract only cleanup from polling responses; never overwrite an editor draft or turn a background read into a settings save. Stop on zero, unmount, scope/session change or a read failure; keep the last known notice on failure with Retry status beside it. Retry only the read. Re-entry and reload re-read current state. Polling the existing config is sufficient; detecting a change from another tab while the current summary is zero is deferred until the next normal read/re-entry.

Remove `StorageSwitchHistoryComponent`, its API/models/exports, every workspace/editor/config/transfer caller and obsolete dictionary keys. A successful transfer closes its dialog and updates the list through the existing callback; pending cleanup appears when the target database settings are opened. Use existing themed AntD primitives and semantic tokens, live translations in all six locales, keyboard interaction and responsive wrapping. Preserve the single page scroll owner.

Reject placing the notice above the database list or in an always-open disclosure: both distract from normal browsing. Reject a shared generic task framework: this UI has one page owner and a small summary. Reject theme or navigation repairs here: they are outside the agreed change.

## Risks / Trade-offs

- **Unreachable storage keeps retries alive:** use the existing capped backoff and sanitized errors; explicit storage deletion is the user-controlled exit. There is no terminal failed queue state to hide work.
- **Active writes also have queue rows:** only attributed deletion requests contribute to the database notice. Every queue row still triggers the storage-deletion guard because deleting access can strand an upload too.
- **Abandonment can leave remote objects:** the second confirmation explains it, an audit event records the snapshot, and neither the UI nor the response claims the external storage was emptied. Provider attempts already running may still remove some objects.
- **Unreleased cleanup state is discarded:** unfinished deletions from the unreleased implementation are not migrated. Durability and retry guarantees apply to work registered by the new implementation.
- **A single queue no longer gives historical progress percentages:** show an outstanding count. New requests and late old writes can increase it; completion means the known queue is empty.
- **Source storage is removed during cleanup:** abandonment removes its queue rows, so a subsequent database summary becomes zero; the audit records why. Current summaries are not historical proof of successful deletion.
- **Old clients break:** release backend and frontend together. Remove old operations and response variants completely; no compatibility shims.

## Migration Plan

1. Add a goose migration that discards the unreleased pending deletion state, removes switch records, batch items, batches and `is_ordinary`, renames the queue and its schema objects, and adds nullable scope columns with structural constraints. Do not copy or reconstruct old obligations, attribution, attempts, claims or completed history.
2. Run the new server exclusively against the new schema; mixed old/new processes are unsupported. Stop old instances before migration. Retain physical storage generations and publication fencing.
3. Regenerate the strict servers, test clients and frontend types from `openapi.yaml`, then ship the updated backend and frontend together. Update `docs/architecture.md` and `docs/glossary.md`, and search active documentation, fixtures and tests for obsolete operation-history assumptions. Historical archived changes remain historical evidence.
4. Test the resulting queue schema, paired attribution constraint and removal of obsolete tables. No migration-preservation tests are needed for unreleased cleanup data. Continue checking settings and credentials written by the last released API in the upgrade test.

Validate the queue and mutation contracts through real module HTTP tests, the physical reassignment/publication and cross-instance tests, and affected backup/restore end-to-end scenarios. Screen tests cover polling, cancellation and explicit abandonment; browser UI scenarios cover the Storage placement, warning focus, both themes, narrow layouts and live language changes. The tasks name the concrete commands.
