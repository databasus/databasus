## Why

Databasus reads PostgreSQL timeline history through a SQL server-file function that its own limited backup role is not granted. After promotion, this can prevent history archiving even though physical backups and WAL reception work; the permission failure is supported by code inspection but has not yet been reproduced with the provisioned role.

## What Changes

- Reproduce the permission failure on PostgreSQL 17 and 18 using credentials created through the public limited-user API on the primary and replicated to its standby.
- Replace SQL server-file reads with one replication-protocol history acquisition path shared by WAL reception and FULL completion.
- Verify the source cluster on the same connection used to request history, validate the requested timeline and returned history, and leave the provisioned role's privileges unchanged.
- Preserve atomic artifact/metadata publication, duplicate handling, encryption and the existing storage-switch protection.
- Require published history for archive-based recovery on a timeline above 1, while keeping an otherwise valid FULL usable through explicit per-backup restore. Restore availability and downloads must use the same rule. Existing reported WAL coverage may shrink until missing history is archived.
- Prove recovery by starting PostgreSQL from a downloaded post-promotion restore bundle and checking rows written after the replacement FULL.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `physical-backups/replication-credentials`: History acquisition works with the Databasus-provisioned role without granting server-file or administrative access.
- `physical-backups/wal-archiving`: Timeline history is verified, published once and retried safely under storage changes.
- `postgresql-physical-backup-failover`: Promotion with provisioned credentials preserves history and produces a recoverable replacement FULL and subsequent WAL.
- `physical-backups/operations`: Archive-based restores on promoted timelines require published history; explicit backup restores remain available independently.
- `physical-backups/wal-stream-monitoring`: Reported recovery coverage uses the same history prerequisite as restore selection.

## Impact

The change follows root `AGENTS.md` and `backend/AGENTS.md`. It affects the physical PostgreSQL engine, the backups module's timeline upload and restore selection, API tests, and shared physical recovery scenarios. No new dependency, HTTP operation, response field, database migration or privilege grant is planned. The existing SQL history reader is removed without a fallback.

Update `docs/architecture.md` to describe history acquisition and the recovery prerequisite. Existing API contracts keep their shapes and error codes; their descriptions are clarified where needed. User documentation must not prescribe additional server-file privileges as a workaround.

Out of scope:

- Recovery that replays WAL across timelines.
- Patroni, CloudNativePG and cloud-provider integration or fixtures.
- A broader audit or redesign of backup-role privileges.
- A second local-file acquisition mechanism, a new monitoring reason or frontend controls.
- CI runner changes, deployment, commits and publishing.
