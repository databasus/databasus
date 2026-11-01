## Why

The logical backup editor, backup list and restore screens are about to be rewritten by `extract-schedule-editor` and `refactor-logical-backup-screens`. A change lands as one commit, so moving and rewriting them together would hide the behavior changes among moved lines. Only `pages/databases` uses these screens, yet they live in features. That placement forces a baseline exception for the import of `features/restores` and two render props whose only purpose is to get around the ban on imports between features. This change moves the code to its final owners first.

## What Changes

- Move `LogicalBackupsComponent` and its filter panel from `features/backups/logical` to `pages/databases/ui/logical-backups`.
- Move `EditLogicalBackupConfigComponent` and `ShowLogicalBackupConfigComponent` to `pages/databases/ui/backup-config`, and merge their two test files of the same name into one.
- Move `features/restores/ui/*` to `pages/databases/ui/restores`.
- Split `entities/backups/{config,logical,shared}` by who uses each unit:
  - `config` and `BackupEncryption` from `shared` become `entities/backup-configs`, because `features/backups/physical` and `pages/storages` use them too;
  - `logicalBackupApi` moves to `pages/databases/api`, because the page is its only consumer after the move;
  - the logical backup types, statuses, list filter, download token, logical retention policy, retention period and notification trigger move to `pages/databases/model`, with the two `model` files of `features/backups/logical`, for the same reason;
  - `entities/backups/physical` stays until milestone 7, and its consumers change only their imports.
- Remove `renderRestoreTargetForm` and the `renderStorageSelection` prop of the logical editor, along with `RestoreTargetFormOptions`. The page's own files import `DatabaseConnectionForm` and `BackupStorageSelection` directly. These are the only prop changes. The physical editor keeps `renderStorageSelection` until milestone 7.
- Update `fsd-baseline.json`: remove the restore `fsd/forbidden-imports` entry and the `entities/backups` `fsd/public-api` entry, because neither finding exists after the move.
- Update the 41 milestone 6 entries of `style-baseline.json` to the new paths and the owner `pages/databases`, without changing their constructs.
- Remove the four departed slices from `NESTED_SLICES` in `frontend/lint/checkProjectDependencies.ts`.
- **BREAKING (internal imports only)**: delete `features/backups/logical`, `features/restores` and `entities/backups/{config,logical,shared}` with no forwarding exports. Screens, addresses and HTTP requests stay the same.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This change moves code without changing any behavior, so `.openspec.yaml` sets `skip_specs: true`.

## Impact

This is the second change of milestone 6, after `fix-monthly-schedule-day`. Root `AGENTS.md` and `frontend/AGENTS.md` apply. Placement follows `docs/architecture.md#frontend` and the feature-sliced-design skill, and changed tests follow the testing skill.

Affected areas:

- `frontend/src/features/backups/logical` and `frontend/src/features/restores`, which are deleted;
- `frontend/src/entities/backups/{config,logical,shared}`, which are deleted, and the new `frontend/src/entities/backup-configs`;
- `frontend/src/pages/databases`, which receives the screens and new `api` and `model` segments and drops the two render props at its call sites;
- imports in `frontend/src/features/backups/physical` and `frontend/src/pages/storages`, which use the config entity;
- `frontend/fsd-baseline.json`, `frontend/style-baseline.json` and `frontend/lint/checkProjectDependencies.ts`;
- `docs/architecture.md`, whose "Database pages compose reusable editing" states what the page owns, how it reaches backups, why draft types are entity types and which editors receive render props, and `frontend/README.md`, which describes the restore render prop;
- `temp/refactor/frontend-plan.md`, whose item 6.2 still names `entities/logical-backups`.

There is no backend, HTTP contract, generated code, dictionary or dependency change.

### Out of Scope

- Any other change to the props, state or behavior of the moved screens, including `isSaveToApi`, `isShowBackButton` and `isShowCancelButton`.
- The defects listed for `refactor-logical-backup-screens`, such as the wizard draft reset and the restore dialog crash when a backup disappears.
- Splitting the 877-line editor or the list, and repairing the 41 style findings. `refactor-logical-backup-screens` does both.
- The shared schedule editor and `entities/schedules`, which belong to `extract-schedule-editor`. `Schedule` moves with the config entity until then.
- `features/backups/physical` and `entities/backups/physical`, apart from imports.
- The dictionary keys under `backups.logical` and `restores`, which stay as they are.
