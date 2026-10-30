## Context

See proposal.md for motivation and explicit breaking behavior. Transfer currently calls SwitchStorage at backend/internal/modules/backups/internal/transfer/service.go:141, which updates only storage_id at internal/config/storage_switch_service.go:23. Ordinary logical config saves remove the old catalog inside their transaction at internal/config/service.go:355. Physical config saves instead invoke cleanup after committing at internal/config/service.go:439; internal/physical/config_reaction_service.go:38 immediately cancels a streamer and deletes its row.

The scheduler reads chains by database ID at internal/physical/backuping/scheduler_state.go:34. The WAL manager treats any process for that database as already running at internal/physical/walstreaming/manager.go:154. The transfer API tests deliberately retain old logical backups at backend/internal/modules/backups/transfer_test.go:38, matching the main backup-configs spec that this change replaces.

The storages worker already uses durable deletion obligations, leases, generation checks, backoff and provider timeouts. backend/internal/modules/storages/internal/files/deletion_job.go:190 reschedules failures without a terminal limit. Its pending row disappears on success, so the current queue alone cannot provide durable per-switch completion.

This design follows backend/AGENTS.md and frontend/AGENTS.md, the API-design/testing skills, and docs/architecture.md's module, transaction and stored-file ownership rules. It extends the stored-file architecture only with switch-scoped batch outcomes, documented in the same implementation. New status fields are TEXT validated by code, following the enum storage rule already established by the archived enum-validation-in-code change.

## Goals / Non-Goals

**Goals:** make the local switch atomic, keep new backups independent of old-provider availability, provide finite and durable cleanup status, and reuse existing file deletion machinery.

**Non-Goals:** transactional remote deletion, preserving or migrating the removed backup history, a general workflow engine, or changing retry policy for unrelated deletion causes.

## Decisions

### Switch immediately and delete remotely through the existing worker

One transaction locks the database/config in the established order, re-reads the database and rechecks access, validates the target storage against that locked database's workspace, creates a switch operation, removes the affected backup history and registers its exact files with storages. The workspace read under those locks also scopes the accepted operation. An initial access check does not authorize a save after a concurrent transfer. Commit makes the new config visible. An after-commit hook cancels old physical work; no provider request or process cancellation runs inside the transaction. A persistence failure rolls back the move, config, catalog, obligations and operation together. Use the existing transactional logical removal path, including its in-use refusal while backup/restore execution is active.

Classify assignment under those locks using both the previous config and retained catalog state. A non-null target starts this path when it differs from a configured storage, or when the previous storage is null and retained backup history exists. The latter includes A-to-null-to-B and A-to-null-to-A: both reset the history, record cleanup against the artifacts' actual storage references, and fence old physical publishers before requesting a new FULL. Do not infer artifact locations from the previous config's nullable storage ID. A null-to-storage config save without retained history remains a 200 initial assignment without an operation. Clearing storage alone preserves the history and returns 200 without a switch operation. SWITCH_TO_STORAGE transfers still create an operation even with no retained history.

Rejected: classify a switch solely by comparing two non-null config storage IDs. Clearing the setting would then bypass cleanup and allow reassignment to resume an old physical chain.

For physical data, extend the existing chains/catalog deletion path to enumerate and remove all old FULL/INCR, WAL, history and sidecars in the outer transaction, without losing file identity to chunk boundaries. After removing old FULLs, reconcile remaining WAL and timeline history against the surviving completed FULLs, including when ordinary cascade deletion skipped recovery-artifact cleanup because an old FULL was running. Keep boundary WAL needed by the surviving backup and capture every removed artifact in the operation. Do not call the current post-commit cleanup loop as the switch's atomic removal step.

Rejected: delete remotely before changing storage or long-poll a deletion-first request. An unreachable old provider would block the switch, and deleted remote files cannot be rolled back if a later notifier update fails. Rejected: change only storage_id and let the scheduler discover missing manifests later.

### Persist a backups operation and a storages-owned deletion batch

The switch's deletion_batch_id has a foreign key to storages_deletion_batches with ON DELETE RESTRICT. This durable operation-to-outcome link is distinct from historical database/workspace/storage identities, which do not restrict live-resource deletion.

Backups owns backups_storage_switches: operation ID, database ID, workspace at acceptance, old/target storage IDs as historical identity, creation time and deletion batch ID. A missing previous storage is nullable: omit previousStorageId in the response when the accepted config named no storage, even if its retained artifacts identify old storages. Each batch item independently records its artifact's actual original storage. Storages owns the deletion batch's deadline, status, completion time, aggregate counts and per-artifact outcomes; only storages reaches providers or its pending deletion tables. Backups stores no second mutable copy of cleanup state.

Keep the database ID as historical identity so deleting the database neither cascades into operation history nor becomes blocked by that history. Authorize operation reads through the saved acceptance workspace without requiring a live database lookup. A later database transfer leaves earlier operations in their acceptance workspace; members of the later destination gain no access to earlier operations through the moved database.

Add typed module operations to request an attributed batch and read durable outcomes for a set of batch IDs. After authorizing an operation or a page of operations, backups composes its identity/authorization metadata with those batched outcomes directly in the response. This avoids per-item queries and a second background job. No storages dependency on backups is introduced. The existing deletion worker settles deadline expiry as well as provider results; file deletion success or exhaustion updates the durable batch outcome in the same transaction that removes or stops its pending obligation. Preserve counts after the transient pending row is gone. A storage-module read failure fails that status read through the normal safe Problem response rather than serving a potentially stale copied status.

Historical operation/batch storage IDs must not create a live RESTRICT dependency that makes the old storage permanently undeletable. Before a storage deletion drops pending rows/credentials, settle associated batch items as deleted or unresolved. Keep only historical IDs and sanitized metadata in the outcome record; do not copy credentials or complete provider configuration. Apply the existing provider-delete leftover report, and do not claim successful deletion for skipped/live or failed files.

Rejected: infer progress only from the number of remaining pending rows, because success, deleted storage and lost association become indistinguishable. Rejected: let backups query storages tables directly, which violates module ownership. Rejected: duplicate credentials in a generic task payload.

Rejected: copy batch status, counts and completion time into the backups operation through a reconciler. The deletion worker already owns durable outcomes, so another mutable copy adds synchronization and restart cases without changing observable behavior.

Add two definitions to docs/glossary.md in this change: **Storage switch operation**, StorageSwitch in code/API, is the durable record of an already-committed switch whose old-file cleanup may still be pending; **Deletion batch**, DeletionBatch in code/module interfaces, groups the artifact outcomes and finite policy for one switch. A batch is distinct from a **Pending deletion**, PendingDeletion, the existing individual file obligation. Link their lifecycle rules to the architecture rather than duplicating those rules in the glossary.

### Bound only this switch's cleanup

Use five failed attempts per artifact, the existing exponential delay/jitter, and an absolute deadline 24 hours after acceptance. Both limits are implementation defaults, not user settings. Each attempt is bounded by the smaller of the existing lease and the remaining deadline. On the first pass at or after the deadline, settle every unresolved item without another provider call. The application must be running for the deletion worker to settle the final state; restart preserves the original deadline and counts.

Track the switch's attribution separately from independent ordinary obligations for the same file. Giving up must deactivate only the switch's ownership, not suppress required cleanup for another cause. A terminal operation never becomes successful later merely because another obligation eventually deletes its file. Repeated requests cannot reset a switch's deadline or attempts, or attach newer-storage artifacts to an older operation. Generation/lease checks reject stale completions.

A provider failure settles its attributed batch attempt even when a repeated deletion request changed the pending queue's generation. A new claim first settles expired attempts and does not overwrite an unsettled active attempt for the same file. Queue generation still guards ordinary rescheduling and successful removal of the pending obligation.

Preserve live-write exclusion. A writer still active when the deadline expires yields an unresolved outcome; it is not a reason to delete from under that writer. File obligations from a canceled write continue under their independently recorded cleanup cause.

Rejected: apply a global five-attempt cap to the queue, which weakens the existing failed-upload and retention guarantees. Rejected: mark every failed file deleted or silently discard its identity. Rejected: retain switch-owned retries forever.

### Fence old physical publishers and restart from FULL

Use a persisted storage generation on the config, incremented with a switch, and capture it when physical work starts. Publication checks the current generation inside the transaction under the same config/database lock order as the switch. An old run cannot publish a base/manifest/WAL/history result after commit. A rejected publication leaves its write receipt available for cleanup.

After commit cancel old backup and WAL contexts, drain them through the existing supervised module runs and retain the per-database busy reservation until the old backup finishes. Do not let an old streamer's exit delete or release a newer streamer's ownership; compare its existing claim identity or the generation when settling streamer state. Reconciliation starts WAL for the new generation once the old execution drains. Persist the force-FULL request for enabled physical configs, clear stale incremental requests and reset current backup-success/error indicators so the database is visibly starting a new history. Disabled configs stay disabled and require a FULL when later enabled.

Request exact deletion for all catalog-owned and known in-flight old artifacts at acceptance. A sidecar created by an old writer after acceptance cannot join the new catalog; its write receipt remains an independent cleanup obligation. The switch's deadline and counts describe its captured artifact set, never a claim that unknown external files were discovered or deleted.

Rejected: identify a streamer only by database ID, which preserves the old destination. Rejected: rely on cancellation alone; a late remote write can complete before its process observes cancellation. Rejected: start a second writer immediately and release the busy reservation prematurely.

### Use normal polling of a persisted resource

The internal storage/catalog slices keep the existing generated HTTP contract compilable, adapting internal method call sites together when necessary. In the HTTP integration slice, edit the contract first, regenerate with mise run generate-api, and implement the complete generated interface and dependency wiring before running backend lint or API tests. Do not regenerate the new GET operations in a separate earlier task while their handlers are absent. That integration slice introduces:

- PUT /backup-configs/{databaseId} returns 202 plus StorageSwitch and Location when changing one configured storage to another or assigning storage while the previous config names none but retained backup history exists. Ordinary edits, clearing storage and initial assignment without retained history retain 200 plus BackupConfig.
- POST /databases/{databaseId}/transfer returns 202 plus StorageSwitch for SWITCH_TO_STORAGE, including unsaved/default configs with no old artifacts; MOVE_WITH_DATABASE retains 204.
- GET /storage-switches/{storageSwitchId} reads one operation.
- GET /storage-switches?workspaceId=...&databaseId=...&limit=...&after=... discovers operation history, ordered by createdAt descending and then id descending with cursor pagination over both values. workspaceId is required and databaseId is optional; filtering uses the historical database ID and does not require the database to exist.

All new operations use the backups tag. Resources expose id, databaseId, workspaceId, previousStorageId when present, targetStorageId, status, createdAt, completedAt, cleanupDeadline, and artifact counts (total, pending, deleted, remaining). The counts satisfy total = pending + deleted + remaining; remaining means a terminal unresolved outcome. States are CLEANUP_PENDING, COMPLETED and COMPLETED_WITH_REMAINDERS. completedAt is absent while pending and present in either terminal state, which has zero pending artifacts. A zero-file operation completes immediately but still answers 202 on a switch request. Both accepted mutations provide Location pointing to GET /storage-switches/{storageSwitchId}.

Reading/listing calls workspaces.Authorize with ActionView in the operation's workspace at acceptance, preserving its instance-admin bypass: an instance admin may read and list without membership; only a non-admin nonmember gets 404. Do not implement a separate membership-only guard. Existing edit checks protect initiation, including both workspaces for a transfer and the existing instance-admin permissions. Return the normal Problem errors and never expose raw provider errors, file names, signed URLs or secrets. Deleting a database does not erase unfinished cleanup; the historical operation remains scoped to its acceptance workspace.

The UI polls every two seconds while pending, stops at a terminal state or unmount, and refetches on remount/reload. A polling error keeps the last state and displays a retry action; it never repeats the destructive request. Discovery works from the target database/workspace after transfer and from workspace operation history if the database is subsequently deleted.

Rejected: long polling, WebSockets or holding the mutation open; the already accepted switch does not need a persistent connection. Rejected: an ephemeral toast alone, which loses cleanup state on reload.

### Make destructive consequences visible before submission

Both ordinary storage changes and workspace transfer show localized copy before confirmation: existing backups will become unavailable, old files will be deleted, the new storage becomes active immediately, and cleanup runs in the background and may leave files if the old storage is unavailable. The existing submit button confirms this warned action; no second approval dialog is needed.

Include assignments from a cleared setting when old history remains, even if the user selects the same storage that originally held those backups. A missing currently configured storage must not suppress the deletion warning or accepted-operation handling.

After acceptance display the active new storage and a separate cleanup notice with progress. The terminal warning says switching succeeded and some old files could not be confirmed deleted; it does not suggest that a failed operation should be retried by switching again. Add complete translations in all six dictionaries.

Rejected: promise deletion before activation when the selected implementation activates first. Rejected: hold the screen in a generic loading state until every old file is deleted.

## Risks / Trade-offs

- Large physical histories can make the local catalog transaction expensive -> batch SQL statements within the same outer transaction; never wait on provider I/O, and test with multiple chains.
- Old execution races new history -> generation-fence every physical publication, preserve busy reservations and test delayed uploads/rollback.
- Provider outages leave remote files -> five failures or the 24-hour deadline produces an explicit durable warning and stops switch-owned retries.
- A storage can be deleted before cleanup finishes -> settle attributed batch items before pending rows disappear and preserve the historical outcome.
- Multiple switches overlap -> immutable batch attribution and per-attempt names isolate A-to-B cleanup from B-to-C cleanup.
- Terminal operation/outcome history grows -> retain compact historical records for this change; no silent retention policy or generic task-history feature is introduced.
- HTTP response changes break clients -> mark them BREAKING, regenerate clients and update all frontend callers/tests in this change without aliases.

## Migration Plan

Add transactional goose migrations for the operation, batch/outcome state, config generation and any required attempt attribution. Follow the established TEXT enum storage rules and existing ownership prefixes. Existing unowned pending deletions retain their unlimited policy. No existing file is retroactively declared a successful switch cleanup.

Update docs/architecture.md for attributed batch outcomes and the explicit switch-only finite-cleanup exception. Update the affected main specs through the normal delta sync at archive. Run API, storage-worker, physical publishing, frontend and migration regressions; no deployment or full backend suite is required to complete the change. Deployment follows the existing one-way schema policy.

### Unfinished physical writes after catalog deletion

Physical FULL and INCREMENTAL object keys use the same database-UUID prefix as WAL and history artifacts. The storages-owned pending file reference then retains the database attribution even when a backup is deleted while its writer drains. The config also persists last_selected_storage_id independently of the current nullable storage_id. A non-null selection updates both IDs; clearing changes only storage_id. The migration backfills the historical ID from the current selection, and the historical column has no live-resource foreign key. Switch acceptance queries the stable file prefix and uses this last selected ID to capture known unfinished writes without depending on surviving catalog rows. Files from earlier storage selections remain under their independent obligations. The HTTP previousStorageId continues to describe the current config before acceptance and stays absent after clearing. Display names remain in the backup label after the UUID prefix.

Rejected: look up unfinished FULL files by the database name, which can change and does not uniquely identify a database. Rejected: keep a second backup-owned file-obligation table, which duplicates the storages-owned durable registrations.

Cross-instance write protection uses PostgreSQL session advisory locks on storage ID plus file name. Acquire the lock before registering the pending row, and keep it until provider writing and receipt preparation finish. One dedicated connection per file store holds every active file's lock; it returns to the pool after the last write. Deletion claims acquire the matching transaction lock only when no writer holds it, and storage drains use the same guard even after the pending row is cascaded away. Queue-generation changes and cleanup deadlines do not release a writer's protection. A detected connection failure cancels all writers on that session and prevents receipt publication.

Rejected: reserve one connection per upload, which can exhaust the application pool. Rejected: use a write lease's expiry as proof that a slow provider has stopped writing. A shared session protects writes through cancellation and long pauses without a renewal job; its cost is one checked-out connection per file store while uploads run. Process death releases the locks, leaving durable obligations for cleanup.

Deletion of a storage or workspace also enumerates durable REMAINING batch items, even after their switch-owned pending rows have been released. It drains their files with the last available credentials and includes unconfirmed absences in the audit report, while preserving terminal batch counts and timestamps. Ordinary deletion timeouts persist their error and exponential backoff when their generation still owns the row; an expired successful result cannot complete a batch or release its current obligation.
