## Why

The database screen exposes an empty storage-switch history twice, although users only need to know which storage is active and whether old files still need removal. The backend maintains a file-deletion queue, deletion batches and switch history for that one cleanup task; the extra history also imposes an arbitrary five-failure and 24-hour cutoff.

## What Changes

- Use one durable file-deletion queue for ordinary cleanup and storage changes. Reuse the existing queue, rename it to `storages_pending_file_deletions`, and retain file identity, retry scheduling, errors and write protection. Retry until absence is confirmed or the user explicitly deletes the storage and abandons its cleanup.
- **BREAKING:** remove persisted storage-switch records, deletion batches and their items, the switch-history API and its response schemas. Discard unfinished deletion state from the unreleased implementation; no data migration or restoration of exhausted work is needed. Remove the finite switch-cleanup policy rather than preserving a compatibility layer.
- **BREAKING:** configuration saves always return the saved configuration with 200; database transfers always return 204 after committing. Remove their 202 switch-operation responses and `Location` headers. Read-only cleanup counts on the backup configuration replace operation reads.
- Show only current cleanup beside Storage in the selected database's backup settings, for logical and physical backups, including when backups are disabled. Show remaining-file counts and a retry warning while files remain; remove the block after cleanup. Remove the workspace banner, empty history, completed history and history shown in editors or transfer dialogs.
- **BREAKING:** deleting an unused storage with queued files without explicit abandonment returns 409 `cleanup_pending` and changes nothing. The interface explains that cleanup is still running and offers Wait or Delete anyway. Explicit deletion removes the storage setting and its queue entries atomically, records abandonment in the audit log, and performs no final provider drain. It does not delete the external bucket or directory.
- Keep the existing refusal when a configuration or retained backup references the storage; explicit abandonment never overrides it. Serialize deletion with new queue registration and make late worker results harmless.
- Workspace deletion must not silently abandon cleanup through cascading storage deletion: refuse it with `cleanup_pending` until its affected storages finish cleanup or are explicitly deleted individually. Keep workspace deletion atomic and retain the existing database and referenced-backup restrictions.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `storage-file-cleanup`: one retryable queue and explicit abandonment on storage deletion instead of a final drain or finite switch outcomes.
- `backup-storage-assignment`: preserve atomic reassignment and physical publication protection while replacing switch resources and history with current cleanup counts and authorization.
- `backup-configs`: expose current cleanup with configuration reads and use synchronous configuration-save and transfer acknowledgements.
- `storages`: require explicit acknowledgement before deleting an unused storage with pending file deletions.
- `workspace-deletion`: prevent cascading deletion from bypassing the cleanup warning.
- `frontend-database-screens`: place live cleanup only in the Storage section and preserve settings drafts while polling.

## Impact

This change answers to root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`. It touches the backups, storages and workspace-deletion modules, storage migrations and their tests, `backend/api/openapi.yaml`, generated servers and clients, frontend backup configuration/storage/workspace screens, all six interface dictionaries, and affected screen and browser UI tests. `docs/architecture.md` and `docs/glossary.md` must describe the queue and explicit abandonment without switch-history entities. No new dependency is needed, and `backend/api/agent.yaml` does not change.

### Out of scope

- Copying existing backups between storages or preserving old backup history after reassignment.
- Restoring the earlier global theme, workspace selector, database-card colors or tab design.
- A cleanup-history screen, per-switch percentages, an administrative queue browser, a manual retry action or an export of abandoned files.
- A bulk force-delete option for workspaces or bypassing storage-in-use checks.
- Persisting provider credentials in cleanup rows, retaining a deleted storage solely for retries, or keeping old API aliases.
- Committing, publishing or deploying the implementation.
