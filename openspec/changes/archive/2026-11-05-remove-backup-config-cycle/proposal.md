## Why

Backup configuration reads and save orchestration share one service, making it depend on logical and physical backup services that also depend on it. Module construction hides this cycle behind callbacks to services assigned later, and the physical save combines validation, cleanup, persistence and reactions in one long method.

## What Changes

- Introduce `backups/internal/configsave` above `config`, `logical` and `physical` to own configuration saves and storage switches, including switches during workspace transfers.
- Keep configuration models, validation, reads, locks, publication checks, backup requests and database/storage event handlers in `config`.
- Reuse the existing configuration repository for persistence, exposing only the operations the new package needs.
- Split physical save orchestration into steps for preparing the config, changing storage, persisting state and registering the reaction after commit.
- Construct services in dependency order and remove the history-removal and saved-config callback adapters and their late-bound service variables.
- Preserve access checks, validation errors, transaction boundaries, lock order, storage generations, cleanup attribution and reactions after commit.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This is an internal refactor that preserves the behavior specified by `backup-configs`, `backup-storage-assignment` and `storage-file-cleanup`, so `.openspec.yaml` sets `skip_specs: true`.

## Impact

- Affected code: the backups module constructor, configuration service and repository, new configuration-save service, configuration HTTP handler dependencies and database-transfer service dependencies. Logical and physical backup services keep their behavior and receive direct calls from the new service.
- The change follows root `AGENTS.md`, `backend/AGENTS.md` and the module and storage-switch rules in `docs/architecture.md`.
- HTTP contracts, database schemas, stored configuration values and the public `backups.Module` contract remain unchanged. Existing internal save entry points move to the new service; no forwarding methods or callback compatibility adapters remain.
- Verification uses the existing backups API and internal tests plus all `physicalapitests`, backend formatting and backend lint. New behavioral scenarios are unnecessary.

## Out of Scope

- Changes to backup scheduling, retention, WAL streaming, publication rules or remote file deletion behavior.
- New HTTP operations, response fields, migrations, frontend changes or product requirements.
- Moving database/storage event handling, manual backup requests or configuration models into the new package.
- Reorganizing other backups packages or changing repository-wide architecture conventions.
