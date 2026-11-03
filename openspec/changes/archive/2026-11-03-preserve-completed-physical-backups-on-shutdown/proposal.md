## Why

Stopping Databasus after a physical backup has finished its files but before its successful outcome is recorded can turn that success into an error and delete the files. A full backup can take hours, so preserving the completed result avoids an expensive repeat before release.

## What Changes

- Preserve a completed physical FULL or INCREMENTAL result when application shutdown cancels its context before publication.
- Continue recording backups interrupted before completion as `ERROR` with reason `APPLICATION_RESTART` and message `backup interrupted by application shutdown`, including their file cleanup and failure notifications.
- Add deterministic regression coverage for completion during shutdown and retain the existing interrupted-stream coverage.
- Clarify the physical-backup shutdown requirement to distinguish completed work awaiting publication from interrupted work.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `application-shutdown`: Preserve successful physical backups whose execution has completed when shutdown reaches result classification, while keeping the failure behavior for interrupted execution.

## Impact

- Backend physical backup result classification and the API tests under `backend/internal/modules/backups/physicalapitests/backuping/`; a small test-application helper may be needed for shutdown synchronization.
- Governed by root `AGENTS.md` and `backend/AGENTS.md`, including the testing skill's requirement to reproduce the defect before fixing it.
- Uses the existing publication transaction and file-write confirmation. No new dependencies, HTTP operations, migrations, or compatibility changes.

### Out of scope

- Changing shutdown deadlines, startup recovery, WAL streaming, logical backups, or restores.
- Changing user cancellation or storage-publication validity rules.
- Preserving an archive whose executor still reports an incomplete or failed result, including a failed metadata upload.
- Broad backup lifecycle refactoring or new production synchronization hooks.
