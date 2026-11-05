## Context

See proposal.md for motivation. The state this design starts from, with paths relative to `frontend/src`:

- `features/databases/index.ts` exports four forms. `pages/databases/ui/CreateDatabaseComponent.tsx:6` imports all four, `pages/databases/ui/DatabaseConfigComponent.tsx:20` imports the connection form, and `features/restores/ui/RestoresComponent.tsx:26` imports the connection form across the feature layer.
- `features/databases/ui/edit/EditDatabaseSpecificDataComponent.tsx:8` imports `CreateReadOnlyComponent` and renders it as a dialog at line 102. The wizard renders the same form at `CreateDatabaseComponent.tsx:178`.
- `features/databases/ui/edit/toDatabaseSettingsTarget.ts` is used by every connection form and by `CreateReadOnlyComponent.tsx:19`, so all its consumers stay together in this move.
- Five form files import a parser from a private path of `entities/databases`, for example `EditMySqlSpecificDataComponent.tsx:15` and `EditPostgreSqlLogicalSpecificDataComponent.tsx:16`.
- `frontend/fsd-baseline.json` holds six milestone 5 entries: those five imports and the restore import. `frontend/lint/fsd-baseline.ts` identifies a finding by rule, source and target.
- `frontend/style-baseline.json` holds 63 `direct-color` entries for `features/databases`. `frontend/lint/styleBaseline.ts:9` identifies a finding by rule, source and construct, so a moved file needs its entries updated.

The approach is bound by root `AGENTS.md` (logical units, naming, no compatibility shims) and `frontend/AGENTS.md` (verification, skills). Placement follows `docs/architecture.md#frontend`: start in the page, extract a feature for an independently reusable interaction, and use each slice's public API.

## Goals / Non-Goals

**Goals:**

- Git follows each moved file as a rename, because its content changes only in import paths.
- `features/databases` no longer exists, and no consumer imports a private entity file.

**Non-Goals:**

- Any behavior, prop or style change. `refactor-database-connection-form` makes those on top of this move.

## Decisions

### 1. The connection forms become `features/database-connection`

The forms appear in the creation wizard, in the settings section and inside restore, which is a separate domain. That makes them a reusable interaction with its own public API.

Rejected: moving them into `pages/databases`. Today `features/restores` imports the form directly, and a feature cannot import a page. The next change replaces that import with a slot, but even then the form would serve another domain's dialog from inside one page's private files.

### 2. Only the base-info and backup type forms move to the page

Only the wizard uses `EditDatabaseBaseInfoComponent` and `ChoosePostgresBackupTypeComponent`, so `docs/architecture.md` places them in the page.

`CreateReadOnlyComponent` stays with the connection form for now and is exported for the wizard, because the connection form renders it as a dialog. The next change moves it to the page when the settings section takes over that dialog.

Rejected: moving `CreateReadOnlyComponent` to the page now. The feature would then import a page, which `docs/architecture.md` forbids. Passing the dialog through a slot was also rejected, because it changes props, and this change allows only import path edits. Keeping the base-info and backup type forms in the feature was rejected too, because a feature named after the connection would own a wizard's name step and engine picker.

### 3. `toDatabaseSettingsTarget` stays with its consumers

Every consumer stays in `features/database-connection` during this move, so the function moves with them unchanged. The next change moves it to `entities/databases` and restricts its input by draft type, when a page-owned limited user form starts using it.

Rejected: moving it to the entity here. No consumer outside the feature needs it yet, and its signature changes in the next change anyway.

### 4. Parsers are public entity exports, and the PostgreSQL one names its engine

Parsing a connection string is a domain rule with direct tests, so it stays in the entity. Its generic name cannot stand next to `MySqlConnectionStringParser` in one public API, and the root `AGENTS.md` naming rules forbid it.

Rejected: keeping `ConnectionStringParser` until the rewrite. The public name would be wrong from its first export, and the rename is mechanical.

### 5. Baseline entries follow the files

The 63 style entries get the new source and owner, with their constructs and removal milestone unchanged. The restore import entry gets the new target.

Rejected: repairing the colors here, as `refactor-database-screens` did for its files. Those repairs edit the form lines that the next change replaces, and they reduce git's rename detection. Removing the entries in the next change still meets the milestone's condition that no entry of milestone 5 remains.

## Risks / Trade-offs

- [The move changes no behavior, but a wrong relative path can still break a test import] → Lint, typecheck through the build, and the full frontend suite run before handover.
- [Re-pathed style entries could look like hiding findings] → Their count, constructs and milestone are unchanged, and the next change's tasks remove them.
- [The next change depends on the move being finished] → `refactor-database-connection-form` starts with a check of the moved paths.
