## Context

See `proposal.md` for motivation and scope. A design is needed because the existing ownership overlaps must be resolved before moving files.

- `frontend/src/pages/databases/index.ts:1` exports only `DatabasesPage`. `frontend/src/pages/databases/ui/DatabaseComponent.tsx:7` composes settings, backups, verification, healthcheck and monitoring through direct internal imports.
- WAL's facade lives in `frontend/src/pages/databases/api/physicalWalStreamStatisticsApi.ts:1`, its generated-schema alias in `frontend/src/pages/databases/model/wal-stream-monitoring/WalStreamStatistics.ts:1`, and its UI imports both supporting logic and those models from separate trees (`frontend/src/pages/databases/ui/wal-stream-monitoring/WalStreamMonitoringComponent.tsx:15`). That UI also uses the physical backup facade to request a full backup.
- Physical configuration drafts are currently under the backup list's model (`frontend/src/pages/databases/model/physical-backups/PhysicalBackupConfigDraft.ts:1`), although configuration fields consume them (`frontend/src/pages/databases/ui/backup-config/PhysicalBackupConfigFields.tsx:9`). The wizard consumes the same configuration units (`frontend/src/pages/databases/ui/CreateDatabaseComponent.tsx:18`).
- Small types and label maps are separate from their owners: `frontend/src/pages/databases/model/wal-stream-monitoring/MonitoringText.ts:1` imports its parameter type, and `frontend/src/pages/databases/model/physical-backups/PhysicalBackupStatusLabelKeys.ts:1` imports its enum.
- `frontend/lint/checkProjectDependencies.ts:8` identifies the slice by its layer and first directory. Internal area folders will not introduce additional enforced boundaries.

The design follows root `AGENTS.md` and `frontend/AGENTS.md`, especially generated contract ownership and import paths (`frontend/AGENTS.md:85`, `frontend/AGENTS.md:91`). `docs/architecture.md:342` keeps page-local responsibilities in the page; `docs/architecture.md:344` preserves slice entry points and direct internal imports; `docs/architecture.md:349` keeps small aliases with their owner.

## Goals / Non-Goals

**Goals:** Make the directory tree identify each page-local responsibility, settle ownership of units shared by creation and settings, and reduce needless model files while preserving runtime behavior and existing test scenarios.

**Non-Goals:** See the proposal's explicit exclusions. In particular, folder grouping does not promise architectural isolation between local areas, and there is no numerical file-removal target.

## Decisions

### 1. Group responsibilities inside the existing page slice

Use the following layout. Create a technical subdirectory only when it contains code; the diagram does not require every area to have all four.

```text
pages/databases/
  index.ts
  ui/                         page composition and shared page presentation
  lib/                        supporting logic shared by page areas
  wal-monitoring/
    api/
    model/
    lib/
    ui/
  physical-backups/
  logical-backups/
  verification/
  healthcheck/
  backup-config/
  settings/
  create-wizard/
  restores/
```

Keep `DatabasesPage`, `DatabasesComponent`, `DatabaseComponent` and `DatabaseCardComponent` in root `ui`. Presentation used across areas, such as row action buttons and the backup date popover, may stay there; `formatBackupSize` remains in root `lib`. Keep their existing component composition and callbacks.

The areas are internal concern directories of one slice. Imports between them stay direct and use `@/`; external consumers continue through `@/pages/databases`. Remove obsolete internal paths rather than leaving forwarding files. Keep all existing architecture checks enabled.

**Alternatives:** Keeping the segment-first layout would retain the navigation problem. Extracting widgets would change layer ownership and complicate WAL's physical-backup dependency. Adding public barrels and boundary rules to every area would enlarge the change without being needed for navigation.

### 2. Assign each existing unit to one owner

| Area | Source units and destination responsibility |
| --- | --- |
| `wal-monitoring` | All current `ui`, `model` and `lib` WAL trees, their tests and fixtures, and `physicalWalStreamStatisticsApi`. |
| `verification` | Current verification UI and model trees, `verificationConfigApi`, `verificationRunsApi`, and their tests. |
| `healthcheck` | Current healthcheck UI, its local entry point and tests. Existing entity-owned healthcheck types and operations stay in their entity. |
| `physical-backups` | Current physical backup list, actions, filters, status/type models, restore commands and physical restore UI, `physicalBackupsApi`, and their tests. Configuration-specific models go to `backup-config`. |
| `logical-backups` | Current logical backup UI, `logicalBackupApi`, root `LogicalBackup`, `LogicalBackupPage`, list filter, status and verification-status declarations, `verificationStatusBadgeStyles`, `DownloadToken`, `isLogicalBackupListedByFilter`, and their tests. The badge styles belong with their current logical backup consumer. |
| `backup-config` | Current backup configuration UI and hooks, `BackupStorageSelection`, logical configuration defaults/comparison/validation/retention helpers and their fixtures, and physical configuration drafts/defaults/converters/comparison/validation/retention/notification units currently under `model/physical-backups`. Includes `hasIncrementalBackups`, `getShownPhysicalNotificationTriggers`, and logical retention/notification enums. |
| `settings` | `DatabaseConfigComponent`, connection/notifier/base-info editors, `DatabaseSettingsSection`, `DatabaseTransferDialogComponent`, `show/`, `DatabaseNotifiersFields`, `hasSameNotifierIds`, and their tests. Settings also owns `CreateLimitedUserComponent` and `EditDatabaseBaseInfoComponent`, which the wizard imports directly without duplication. |
| `create-wizard` | `CreateDatabaseComponent`, `ChoosePostgresBackupTypeComponent`, `CreateDatabaseNotifiersStep`, `startFirstBackup`, and their tests. Configuration step components stay with `backup-config`, and reusable settings forms stay with `settings`. |
| `restores` | Current logical restore UI, restore target draft/conversion units and tests under `ui/restores`. Physical restore UI stays with `physical-backups`; entity-owned restore contracts and operations remain untouched. |

Move tests with their production unit. Page-level composition tests remain in root `ui`, including `NotifierComposition.test.tsx`; update only their module references. Shared page presentation does not become a new `shared` slice or a `common` directory.

Preserve existing collaborations: the wizard and settings use `backup-config`; backup lists may use configuration presentation; WAL uses the physical backup facade; logical backups can render verification status and logical restores. These are legal imports inside one page slice. Do not introduce callback adapters, dependency injection or API duplication merely to hide them.

**Alternatives:** Leaving configuration models in the backup-list area preserves the current dispersion. Moving entity contracts into these folders would duplicate ownership and break lower-layer consumers. Moving reusable settings forms into the wizard would make their owning settings flow depend on creation-specific organization.

### 3. Consolidate declarations according to the concept they describe

Within `wal-monitoring/model`, combine related generated response-shape aliases in `walStreamContract.ts`. Logical backups and verification use `logicalBackupContract.ts` and `verificationContract.ts` in their respective owners. The single physical backup row alias keeps its own file; `FullBackupsRetention` stays with the physical configuration draft that owns that nested shape. Keep aliases indexed from `ApiComponents`; do not copy generated shapes or replace page-local drafts with HTTP request types.

Keep independently meaningful enum units, but put their existing label mappings in the same file. Preserve enum values, exported identifiers and translation keys. Keep tiny view declarations beside their owning unit: `MonitoringTextParam` with `MonitoringText`, `VerdictIcon` with `VerdictBanner`, and `RestoreTimelineSegmentKind` with `RestoreTimelineSegment`. `AmountProgress` belongs with its sole consumer in `lib/pathStages.ts`, and `MissingDataRange` with its sole consumer in `lib/chartRows.ts`. This avoids putting unrelated presentation, lifecycle and API declarations into one contract. Do not combine independently meaningful functions or controllers.

Move declarations verbatim apart from import/export cleanup, formatting and removing obsolete self-imports. Audit the resulting runtime import graph: type-only dependencies stay type-only, and consolidation must not introduce circular runtime dependencies. Existing entity contracts remain owned by their entities.

**Alternatives:** One mandatory contract file per area would mix concepts in larger areas. Keeping every alias and label mapping in its own file would retain unnecessary navigation. Importing every generated type directly at call sites would erase the existing domain names.

### 4. Separate relocation from declaration consolidation

First move files and update all literal module references while leaving declarations intact. Validate that intermediate layout with frontend lint, type checking and the page's existing tests. Then consolidate declarations and remove the redundant files. Keep runtime function bodies, JSX, polling intervals, query keys, permissions and test assertions unchanged in both passes.

These are implementation passes, not instructions to create commits. The request's single isolated structural change can contain both; its diff includes import updates and declaration grouping and must not be represented as rename-only.

**Alternatives:** Combining ownership decisions, moves and runtime cleanup at once makes regressions difficult to attribute. Mandatory separate commits would impose a Git workflow beyond this planning request; the tasks instead end with a reviewable working tree.

### 5. Record the local layout without changing agent conventions

Update the Frontend section of `docs/architecture.md` with a short description of the database page's responsibility-first layout and remaining composition modules. Search repository references to moved files and update any affected documentation or tooling references without changing their behavior. The FSD skill already delegates ownership to that document, so neither it nor `AGENTS.md` needs an edit.

**Alternatives:** Editing the skill to inventory this page's directories would duplicate implementation documentation. Omitting the architecture update would leave the chosen layout unexplained next to the existing placement principles.

## Risks / Trade-offs

- **Concurrent edits conflict with moves** -> Inspect `git status`, `git worktree list` and locally available branches before applying. Record touched files, leave others' uncommitted edits alone, and perform the move against the latest agreed base. Remote or unpublished work cannot be inferred from the local inventory. If overlapping edits prevent preserving another contributor's work, stop before changing those files and report the overlap.
- **Folders suggest stronger boundaries than exist** -> Document that areas belong to one slice and preserve direct imports. Do not weaken lint configuration or claim that it checks area isolation.
- **Import paths appear outside ordinary imports** -> Search re-exports, type imports, dynamic imports, stylesheet imports and test/tooling references. Keep source-fixture strings and CSS resource paths under their existing semantics, as `frontend/AGENTS.md:93` requires.
- **Grouping changes module initialization** -> Preserve type-only imports and runtime constants, inspect new dependencies and use the existing tests and production build to detect breakage. Avoid unrelated import cleanup or changes to module side effects.
- **A file-count goal encourages unrelated consolidation** -> Restrict changes to this page's owned units and report actual removals after implementation. Do not broaden the refactor to meet the earlier 100-150 estimate.

## Migration Plan

No deployment, database migration or HTTP migration is needed. Apply the ownership moves, validate, consolidate declarations, update documentation and run final checks. Preserve the public page export, URLs, mounted component relationships and existing scenario assertions. The final review must find only structural changes and documentation; any behavioral defect discovered during verification follows the repository's repair rules and is explicitly reported.

The final checks are frontend formatting and lint, explicit type checking, production build, the existing database page tests and the routine test suite required when moving test files. Do not run the full test suite command reserved for an explicit user request. Reversal, if needed later, is an ordinary reversal of the structural change; no compatibility layer is introduced.
