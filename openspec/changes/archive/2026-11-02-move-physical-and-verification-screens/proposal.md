## Why

The physical backup, WAL stream monitoring and verification screens are about to be reworked by the next changes of frontend milestone 7. A change lands as one commit, so moving and rewriting them together would hide the behavior changes among moved lines.

These screens sit in nested folders that only look like slices: `features/backups/physical`, `features/verification/{agents,config,runs}` and their entity counterparts. Each has one consumer page, except the agent API, which two pages use. The nesting needs two entries in `fsd-baseline.json` and a special list in the dependency check, and the physical editor needs a render prop only to get around the ban on a feature importing a page. This change moves the code to its final owners first, as `move-logical-backup-screens` did for logical backups.

## What Changes

- Move the physical backup list, filters, type cell and restore screen to `pages/databases/ui/physical-backups`, and the physical config editor and show view to `pages/databases/ui/backup-config`.
- Move WAL stream monitoring to `pages/databases/ui/wal-stream-monitoring`, `lib/wal-stream-monitoring` and `model/wal-stream-monitoring`.
- Move the verification config and run screens to `pages/databases/ui/verification`, and the agent screen to `pages/instance-settings/ui/verification-agents`.
- Split the entities by who uses each unit:
  - `entities/verification/agents` becomes `entities/verification-agents`, because both pages use it;
  - the physical and WAL APIs and the verification config and run APIs move to `pages/databases/api`, and their types and label keys move to `pages/databases/model`, because that page is their only consumer.
- Remove `renderStorageSelection` from the physical editor, which imports `BackupStorageSelection` directly. This is the only prop change. `StorageSelectionOptions` then has one consumer, so it becomes the `Props` of `BackupStorageSelection` and leaves `entities/storages`.
- Update `fsd-baseline.json`, which becomes empty, and the milestone 7 entries of `style-baseline.json` to the new paths and owners, without changing their constructs.
- Rename `physicalBackupStatus.ts` to `physicalBackupStatusBadgeStyles.ts` as it moves, because it would otherwise differ only by case from `PhysicalBackupStatus.ts` in one folder, and it holds badge styles. This is the only file rename beyond the moves.
- Delete `NESTED_SLICES` and `MIGRATION_GROUPS` from `frontend/lint/checkProjectDependencies.ts`, because nothing is nested any more, along with the two architecture test cases that exercised them, and point the WAL palette exception in `frontend/lint/styleChecks.ts` at its new path.
- **BREAKING (internal imports only)**: delete `features/backups`, `features/verification`, `entities/backups` and `entities/verification` with no forwarding exports. Screens, addresses, HTTP requests and dictionary keys stay the same.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `frontend-code-checks`: the baseline requirement is replaced by one under a new name that no longer names nested domain groups and drops their scenario, because frontend lint no longer knows such groups. A folder under a layer is now always one slice.

The screens themselves change no behavior.

## Impact

This is the second change of frontend milestone 7, after `fix-wizard-partial-create`. The root `AGENTS.md` and `frontend/AGENTS.md` apply. Placement follows `docs/architecture.md#frontend` and the feature-sliced-design skill, and changed tests follow the testing skill.

Affected areas:

- `frontend/src/features/backups`, `features/verification`, `entities/backups` and `entities/verification`, which are deleted, and the new `frontend/src/entities/verification-agents`;
- `frontend/src/pages/databases`, which receives the screens and a new `lib` segment;
- `frontend/src/pages/instance-settings`, which receives the agent screen and new `lib` and `model` segments;
- `frontend/src/entities/storages`, which loses `StorageSelectionOptions`;
- `frontend/fsd-baseline.json`, `frontend/style-baseline.json`, `frontend/lint/checkProjectDependencies.ts`, `frontend/lint/styleChecks.ts`, `frontend/tooling/styleChecks.test.ts` and `frontend/src/shared/lib/test/frontendArchitecture.test.ts`;
- `openspec/specs/frontend-code-checks/spec.md`, through the delta spec of this change;
- `docs/architecture.md`, whose "Database pages compose reusable editing" names the page's content and the render prop, and whose paragraph on nested domain groups describes the deleted list.

There is no backend, HTTP contract, generated code, dictionary or dependency change.

### Out of Scope

- Any other change to props, state or behavior, including `isSaveToApi`, `isShowBackButton`, `toUnsavedDatabase`, every `alert` and every defect listed for the next changes.
- Splitting the moved components and repairing the style findings.
- Moving the logical files of `pages/databases/model` into a subfolder.
- Renaming dictionary keys, which `refactor-verification-config` does.
