## Context

See proposal.md for motivation. The state this design starts from, with paths relative to `frontend/src`:

- `pages/databases/ui/DatabaseComponent.tsx:5`, `DatabaseConfigComponent.tsx:12` and `CreateDatabaseComponent.tsx:5` are the only production importers of `features/backups/logical`. `features/backups/logical/ui/LogicalBackupsComponent.tsx:45` is the only production importer of `features/restores`.
- `features/restores/ui/RestoresComponent.tsx:34` and `features/backups/logical/ui/EditLogicalBackupConfigComponent.tsx:60` receive the connection form and the storage selection as render props. The comment at `features/restores/ui/RestoreTargetFormOptions.ts:5` gives the ban on imports between features as the reason.
- `entities/backups/{config,logical,shared}` import only the `shared` layer and the top-level `constants`, and never each other, so the split creates no import between entities. `entities/backups/logical` is imported only by `features/backups/logical`, `features/restores` and `pages/databases`, for example `CreateDatabaseComponent.tsx:14`. The logical retention, period and notification trigger types are used only by the logical editor and show view, and `LogicalBackupSettings` does not import them.
- `pages/storages/ui/StorageTransferDialogComponent.tsx:8` and five files in `features/backups/physical` use the config or encryption exports.
- `frontend/fsd-baseline.json` holds the restore import entry and an `entities/backups` `fsd/public-api` entry. `frontend/lint/fsd-baseline.ts:44` fails lint on a stale entry.
- `frontend/style-baseline.json` holds 41 milestone 6 entries: `LogicalBackupsComponent` 16, `RestoresComponent` 13, `EditLogicalBackupConfigComponent` 6, `ShowLogicalBackupConfigComponent` 4, `LogicalBackupsFiltersPanelComponent` 2.
- `frontend/lint/checkProjectDependencies.ts:7` enumerates nested slices, including the four this change deletes.

The approach is bound by root `AGENTS.md` (logical units, naming, no compatibility shims, comments) and `frontend/AGENTS.md` (verification, skills). Placement follows `docs/architecture.md#frontend`: the main logic of one screen lives in its page, and each slice is reached through its public API.

## Goals / Non-Goals

**Goals:**

- Git follows each moved production file as a rename, because its content changes only in imports and, for `RestoresComponent`, `LogicalBackupsComponent` and `EditLogicalBackupConfigComponent`, in the few lines of the slot.
- No feature or entity remains for logical backups and restores, and the baselines record only findings that still exist.

**Non-Goals:**

- Any behavior or style change. `extract-schedule-editor` and `refactor-logical-backup-screens` make those on top of this move.

## Decisions

### 1. The screens move into the page, in three folders

`logical-backups`, `backup-config` and `restores` under `pages/databases/ui` name the three concerns.

Rejected: keeping the screens in one flat `ui` folder, because 12 more files there would hide what the page consists of.

### 1a. Units with one consuming slice leave the entity layer

`docs/architecture.md` ("Start in the page") extracts an entity for a domain model with real consumers, and "Keep domain operations with their owner" allows a page to hold a domain API facade. After the move, `pages/databases` is the only consumer of the logical backup types, `logicalBackupApi` and the logical retention, period and notification trigger types. So `logicalBackupApi` goes to `pages/databases/api`, and the types, their label keys and the two `model` files go to `pages/databases/model`. `entities/backup-configs` remains an entity because `features/backups/physical` and `pages/storages` also use it.

These units leave `entities/backups/logical` last, after the screens, because a feature cannot import a page while the screens are still features.

Rejected: a new `entities/logical-backups`, as the milestone plan first said. No second consumer exists or is planned, so the slice would add a boundary and a public API around code with one owner. Moving the retention types to `entities/backup-configs` was rejected for the same reason. Placing everything inside `ui/logical-backups` was also rejected: `api` and `model` are the segments that say what these files are, and the retention types belong to the config editor rather than the list.

### 2. New page folders have no `index.ts`

Consumers import the file directly, as `pages/databases/ui/show` does.

Rejected: a barrel like `pages/databases/ui/healthcheck/index.ts`. The folder is private to one page, so a barrel protects no boundary and adds a file to keep in sync.

### 3. The `entities/backups` public API entry leaves the baseline now

Steiger treats a folder with a child named `config` as a slice, because `config` is a conventional segment name. That is the only reason `entities/backups` is reported as a slice with no public API. With `config`, `logical` and `shared` gone, `entities/backups` holds only `physical` and counts as a group of slices. Running `checkFrontendArchitecture` on a minimal fixture confirmed it: the finding is reported with `config` present and absent without it.

Rejected: keeping the entry until milestone 7, as the milestone plan first said. The entry would be stale and fail lint.

### 4. Both render props go in this change

Once the components live in the page, nothing prevents the direct import, and the slots separate nothing. The diff in the production files is a few lines, so git still detects the renames.

Rejected: keeping the slots until `refactor-logical-backup-screens`. The tests would stay untouched, but `docs/architecture.md`, `frontend/README.md` and the comment on `RestoreTargetFormOptions` would give a false reason for two more changes, and `extract-schedule-editor` would edit an editor around a slot with no purpose.

### 5. Tests that observe a slot switch to the real collaborator

Removing the slots is the only reason a test changes beyond its imports. Each affected guarantee stays pinned:

- `features/restores/ui/RestoresComponent.test.tsx:55` keeps its check that a physical database offers no target button and drops only the call count of the slot.
- `features/restores/ui/RestoresComponent.test.tsx:71` captured the draft passed to the slot and is removed, because its guarantees are already pinned on the real form and the actual requests. `pages/databases/ui/RestoreTargetComposition.test.tsx:252` enters a MySQL target, asserts the exact `POST /restores` body and waits for the listed restore. `:423` opens the form without the source settings, and `:469` asserts that no request names the source database. `coverage.md` records this mapping and the suite's count falls by exactly this one test.
- `features/backups/logical/ui/EditLogicalBackupConfigComponent.test.tsx:57` asserted on the offered storage ID. It now answers the storage list request and asserts that the selection shows that storage.
- `pages/databases/ui/EditLogicalBackupConfigComponent.test.tsx:290` passed an empty slot. The real selection requests the list, so the test declares that response.

Rejected: mocking `DatabaseConnectionForm` or `BackupStorageSelection` with `vi.mock`. The testing rules mock only the HTTP boundary, and a module mock would preserve the slot's blind spot under another name.

### 6. The two editor test files merge

`features/backups/logical/ui/EditLogicalBackupConfigComponent.test.tsx` and `pages/databases/ui/EditLogicalBackupConfigComponent.test.tsx` would collide in `backup-config`, and a test file mirrors the file it tests. They merge into one with every scenario kept, so the suite has one file fewer and these scenarios keep their number.

Rejected: renaming one of them after its scenario. Two files for one component would break the mirror rule, and the split only reflected the old layer boundary.

### 7. `NESTED_SLICES` loses the four departed entries

The list states which slices exist, so stale names mislead the next reader. `MIGRATION_GROUPS` still derives `entities/backups` and `features/backups` from the physical entries, so the fixtures in `shared/lib/test/frontendArchitecture.test.ts` keep their meaning and stay unchanged.

Rejected: leaving the entries until milestone 7. They would describe code that no longer exists.

### 8. Style baseline entries follow the files

The 41 entries get the new source and the owner `pages/databases`, with their constructs and removal milestone unchanged.

Rejected: repairing the colors here. Those edits touch lines that `refactor-logical-backup-screens` rewrites, and they reduce git's rename detection.

## Risks / Trade-offs

- [A real storage selection or connection form in a test sends a request the old slot did not, and the mock server records it as unexpected] → Each affected test declares its responses, and the suite runs in full before handover.
- [Removing the restore draft test could lose a guarantee] → `coverage.md` maps each of its assertions to the composition test that holds it, on request bodies rather than on the draft.
- [Merged tests could lose a scenario unnoticed] → `coverage.md` records the test count before the move, and the final count must be lower only by the one removed restore test.
- [Re-pathed style entries could look like hiding findings] → Their count, constructs and milestone are unchanged, and `refactor-logical-backup-screens` removes them.
