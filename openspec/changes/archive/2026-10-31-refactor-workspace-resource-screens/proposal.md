## Why

Workspace, storage and notifier screens still mix page state with reusable editors, and database forms directly compose neighboring features. Storage saves can also accept stale connection tests or discard newer edits; this milestone gives those workflows clear owners while preserving their existing capabilities.

## What Changes

- Move workspace settings, members and audit, storage lists/details/transfer/migration and notifier lists/details/transfer into page-owned slices. Keep existing flat domain entities and public API operations.
- Extract `features/edit-storage` and `features/edit-notifier`, preserving notifier lifecycle guarantees and adding revision-aware storage test/save acceptance and submitted-snapshot acknowledgment.
- Share workspace-aware selection and list/detail presentation, leaving loading, filtering and refresh policy with each page. Remove propagated content-height calculations in the migrated screens.
- Compose creation and selection of storages/notifiers above features for every existing consumer: database transfer, database notifier settings, logical/physical config and the repaired Google Drive flow. Remove fake storage IDs and picker remount keys.
- Preserve app-owned workspace selection and creation composition. Publish acknowledged workspace rename/delete changes to app, recover from unavailable workspaces, and replace indefinite loading and browser-native feedback in migrated screens.
- Complete workspace audit reuse, preserving its filters, permissions, page size, timestamps and empty copy.
- **BREAKING (internal imports only)**: delete the old workspace/storage/notifier screen feature exports and editor paths after consumers move; retain no import shims. User-facing tabs, addresses and HTTP shapes remain unchanged.

## Capabilities

### New Capabilities

- `frontend-workspace-screens`: workspace-scoped navigation, list/detail selection, resource editing, composed creation and recoverable screen failures.

### Modified Capabilities

- `frontend-user-screens`: extend the shared audit presentation guarantee to the workspace view while preserving the existing instance and user requirements.

## Impact

Third milestone-4 change; implement after `fix-google-drive-storage-oauth` and `fix-sftp-storage-credentials`, then map coverage against their resulting paths. Root `AGENTS.md` and `frontend/AGENTS.md` apply. Placement follows `docs/architecture.md` and the FSD skill; design, localization and testing follow their repository skills.

Affected areas include app composition, new resource pages, a small database composition boundary, the two reusable editor features, existing domain facades, shared selection/layout, audit presentation, six dictionaries, finite architecture/style baselines, screen tests and browser UI/real-API scenarios. Update `docs/architecture.md` and affected frontend documentation to match the final owners. No new backend behavior, HTTP contract, dependency or global application store is planned.

### Out of Scope

- Database connection forms, full logical/physical config refactoring, schedules, retention, backup controllers, WAL and verification; those retain their later milestones.
- New deep links, router/UI-kit replacement, a universal resource page or a generic form/CRUD engine.
- Reopening the Google Drive transport or SFTP credential contract settled by the prerequisite changes.
- Removing all migration-baseline entries or declaring milestone 4 complete before its required checks and review pass.
