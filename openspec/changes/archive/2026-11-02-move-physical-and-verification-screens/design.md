## Context

See `proposal.md` for the motivation. All paths are under `frontend/src` unless they say otherwise.

- `pages/databases/ui/DatabaseComponent.tsx:5-10`, `DatabaseConfigComponent.tsx:12-19` and `CreateDatabaseComponent.tsx:5` are the only importers of `features/backups/physical` and `features/verification/{config,runs}`. `pages/instance-settings/ui/InstanceSettingsPage.tsx:8` is the only importer of `features/verification/agents`.
- `entities/backups/physical` has two outside consumers: its feature and `CreateDatabaseComponent.tsx:13`. `entities/verification/runs` adds `pages/databases/ui/logical-backups/useLogicalBackupActions.ts:5`. `entities/verification/agents` serves the run list (`features/verification/runs/ui/VerificationsComponent.tsx:18`, agent count) and the agent screen.
- `frontend/fsd-baseline.json` holds two `fsd/public-api` entries for `entities/verification` and `features/verification`. `frontend/lint/checkProjectDependencies.ts:7-20` lists eight nested slices, all of them in this change.
- `frontend/lint/styleChecks.ts:22` allows independent colors in the WAL palette file by its path, and `frontend/tooling/styleChecks.test.ts:107` repeats that path.
- The moved folders hold 158 files and 12 290 lines, including eight `index.ts` files that disappear and the eight agent files that go to `pages/instance-settings` and `entities/verification-agents`. `pages/databases` has 96 files and 15 924 lines today.

The approach is bound by the root `AGENTS.md` (logical units, no shims) and `frontend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- Every moved file reaches its final path with only its imports changed, and the review diff shows renames. The exceptions are `physicalBackupStatus.ts`, which decision 1 renames, and the physical editor and its test, which lose `renderStorageSelection` in task 2.4.
- No test is lost apart from the two architecture cases decision 5 names, and `coverage.md` proves it.
- Only the storage-selection prop changes.

**Non-Goals:**

- Behavior changes, splitting components and repairing style findings.

## Decisions

### 1. The screens go to their pages rather than to flat features

The placement table:

| From | To |
| --- | --- |
| `features/backups/physical/ui/PhysicalBackups*`, `PhysicalBackupTypeCell*`, `PhysicalRestoreComponent` | `pages/databases/ui/physical-backups/` |
| `features/backups/physical/ui/{Edit,Show}PhysicalBackupConfigComponent*`, `pages/databases/ui/EditPhysicalBackupConfigComponent.test.tsx` | `pages/databases/ui/backup-config/` |
| `features/backups/physical/lib/*` | `pages/databases/lib/physical-backups/` |
| `features/backups/physical/model/*` | `pages/databases/model/physical-backups/` |
| `features/backups/physical/monitoring/ui/*` | `pages/databases/ui/wal-stream-monitoring/` |
| `features/backups/physical/monitoring/lib/*` | `pages/databases/lib/wal-stream-monitoring/` |
| `features/backups/physical/monitoring/model/*` | `pages/databases/model/wal-stream-monitoring/` |
| `entities/backups/physical/api/*` | `pages/databases/api/` |
| `entities/backups/physical/model/WalStream*` | `pages/databases/model/wal-stream-monitoring/` |
| other `entities/backups/physical/model/*` | `pages/databases/model/physical-backups/` |
| `features/verification/config/ui/*`, `features/verification/runs/ui/*` | `pages/databases/ui/verification/` |
| `entities/verification/{config,runs}/api/*` | `pages/databases/api/` |
| `entities/verification/{config,runs}/models/*` | `pages/databases/model/verification/` |
| `features/verification/agents/ui/*` | `pages/instance-settings/ui/verification-agents/` |
| `features/verification/agents/lib/*` | `pages/instance-settings/lib/` |
| `features/verification/agents/model/*` | `pages/instance-settings/model/` |
| `entities/verification/agents/*` | `entities/verification-agents/`, with `models` renamed to `model` |

`features/backups/physical/model/physicalBackupStatus.ts` becomes `physicalBackupStatusBadgeStyles.ts`, matching `pages/databases/model/verificationStatusBadgeStyles.ts`, because `PhysicalBackupStatus.ts` lands in the same folder and the two names would differ only by case.

`docs/architecture.md` says to start in the page and to extract a feature only for an independently reusable interaction. These screens have one consumer each, and `move-logical-backup-screens` made the same choice.

Alternative rejected: renaming to flat slices such as `features/physical-backups` and `entities/physical-backups`, as the working plan first suggested. The cost of the move is the same, but single-consumer slices would remain and the render prop would stay, because a feature cannot import the page's storage selection.

### 2. Subfolders inside the page's segments

`pages/databases/model` holds 30 files today, and this move would add about 70. A flat folder would stop telling the reader what the page does, against "Code is split into logical units". The new files go into `physical-backups`, `wal-stream-monitoring` and `verification` subfolders, in the same way `ui` already has `logical-backups` and `restores`.

Alternatives rejected: a flat `model` follows the logical precedent exactly but buries each domain among the others. A separate page per domain is impossible, because all of them render inside one database screen and pages cannot import each other.

### 3. A new `lib` segment for pure calculations

The monitoring calculations and `restoreCommands` keep their `lib` segment, because they are algorithms rather than domain types. This keeps every moved file's folder name and makes the renames easy to review.

Alternative rejected: merging them into `model`, as the logical pure functions did. It would change the segment of 36 files for no gain and mix the calculations with the 43 files `model/wal-stream-monitoring` receives.

### 4. `entities/verification-agents` stays an entity

The agent count on the database page and the management operations on the instance settings page use one facade, so two pages have real need of it. It imports nothing from the pages, and both pages use its public API.

Alternative rejected: moving `getAgentCount` to `pages/databases/api` and the rest to `pages/instance-settings/api` removes the entity but splits one domain API across two pages for a single read.

### 5. Delete the nested-slice mechanism rather than leave it empty

After this change `NESTED_SLICES` is empty and `MIGRATION_GROUPS` does nothing, so both go, with the sentence about them in `docs/architecture.md`. Two cases of `shared/lib/test/frontendArchitecture.test.ts` go with them, `nested group` and `separates new sibling scenarios inside an existing migration group`, because their fixtures now resolve to one slice and the behavior they pinned no longer exists. The scenario of `frontend-code-checks` about nested groups goes with them, through this change's delta spec. The empty `fsd-baseline.json` stays, because `lint:fsd` still reads it until milestone 8 removes the baseline.

Alternative rejected: leaving the list for milestone 8, which would keep dead configuration in an architecture check.

## Risks / Trade-offs

- `pages/databases` becomes the largest page, at about 239 files and 27 000 lines. Subfolders keep it navigable, but the page's size is real, and only a change of page structure would reduce it. That is outside this migration.
- A test could disappear with its file. `coverage.md` lists each test file and its final path, and the test count must equal the baseline minus the two named architecture cases.
- Rollback: reverting the commit restores the old paths. No runtime behavior changes.
