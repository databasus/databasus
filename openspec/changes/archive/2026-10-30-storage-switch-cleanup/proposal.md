## Why

Switching storage during a workspace transfer keeps the old backup catalog, chain and WAL destination, so subsequent backups can follow a chain whose files are absent from the selected storage. Users also need an explicit deletion warning and a finite cleanup outcome when the previous storage is unreachable.

## What Changes

- **BREAKING**: replace the transfer behavior that preserves old backups with the same fresh-start behavior as an ordinary storage change. Old backup records leave the list when the switch commits; the next physical chain starts with a new FULL.
- Commit the new storage, catalog removal and durable old-file deletion requests together, without contacting the old storage. Stop old physical work after commit and prevent it from publishing into the new history.
- Apply the same cleanup and physical-chain reset when storage is assigned after being cleared and old history remains, including reassignment to the same historical storage. Only initial config assignment without retained history stays a normal 200 save without an operation.
- Reuse the storage deletion worker for background cleanup. Bound this switch's cleanup to five failed attempts per artifact and a 24-hour deadline, persist terminal incomplete outcomes, and leave unrelated cleanup policies unchanged.
- **BREAKING**: storage-changing config updates and SWITCH_TO_STORAGE transfers return an accepted operation instead of their current completed responses. Add a persisted operation status resource; the UI polls it and resumes observation after a reload.
- Warn before confirmation that old backups will become unavailable and their files will be deleted. Explain that the new storage becomes active immediately and physical deletion continues in the background.
- Show pending, complete and incomplete cleanup without blocking new backups or retrying forever.
- Keep operation history discoverable after database deletion. Preserve terminal cleanup outcomes even when an independent obligation later deletes a remaining file.
- Define storage switch operation (StorageSwitch) and deletion batch (DeletionBatch) in docs/glossary.md, including their difference from an individual pending deletion.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `backup-storage-assignment`: ordinary storage changes and transfers atomically reset the backup history and start finite background cleanup.
- `backup-configs`: transferring with another storage removes old backups and returns an observable operation.
- `storage-file-cleanup`: switch-owned deletion attempts have durable attribution, bounded retries and an honest terminal incomplete result.
- `workspace-deletion`: remove the example that switching a logical config deliberately retains its old backup catalog.

## Impact

Constrained by root `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md`, API-design and testing skills, and the transaction/file ownership rules in `docs/architecture.md`.

Affects backups config/transfer/catalog and physical publishing, storages file deletion and its public module interface, a new persisted switch operation, SQL migrations, `backend/api/openapi.yaml`, generated clients/types, transfer/config screens and all six interface dictionaries. Update architectural documentation for bounded switch cleanup. New operations belong to the backups contract tag; storages keeps ownership of provider calls and file obligations.

### Out of scope

- Moving or copying old backup files into the new storage, restoring the removed history, or extending chains across storages.
- Long polling, WebSockets, a generic task platform or a general retry-management UI.
- Changing MOVE_WITH_DATABASE behavior, or treating a save with the same storage as a switch.
- Giving up on ordinary failed-upload, retention or user-deletion cleanup after five failures.
- Replacing rclone, changing R2, or adding backward-compatible response aliases.
