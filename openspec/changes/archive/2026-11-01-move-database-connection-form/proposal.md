## Why

The connection forms are about to be rewritten by `refactor-database-connection-form`. A change lands as one commit, so moving and rewriting them together would show 2900 deleted lines and new files, and the behavior changes would be lost among them. This change moves the code to its final owners first, so the rewrite's diff shows only real changes. It also ends the imports of private entity files, which the frontend milestone 5 baseline still allows.

## What Changes

- Move the connection forms, `CreateReadOnlyComponent`, `toDatabaseSettingsTarget` and their tests from `features/databases/ui/edit` to a new `features/database-connection/ui` slice. The slice's public API exports `EditDatabaseSpecificDataComponent` and `CreateReadOnlyComponent`, because the connection form renders the read-only user dialog itself and the wizard also uses that form.
- Move `EditDatabaseBaseInfoComponent` and `ChoosePostgresBackupTypeComponent` to `pages/databases/ui`, because only the wizard uses them.
- Export the four connection string parsers from the `entities/databases` public API and rename `ConnectionStringParser` to `PostgresqlConnectionStringParser`, since a public name must state its engine.
- Update `fsd-baseline.json`: remove the five `project/private-slice-import` entries, and point the restore `fsd/forbidden-imports` entry at `features/database-connection/index.ts`. That entry stays until `refactor-database-connection-form`.
- Update the 63 `direct-color` entries of `style-baseline.json` to the new paths and owners without changing their constructs. The rewrite removes them.
- **BREAKING (internal imports only)**: delete `features/databases` with no forwarding exports, and consumers stop using private entity paths. Screens, addresses and HTTP requests stay the same.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This change moves code without changing any behavior, so `.openspec.yaml` sets `skip_specs: true`.

## Impact

This is the third change of milestone 5, after `fix-restore-target-connection` and `refactor-database-screens`. Root `AGENTS.md` and `frontend/AGENTS.md` apply. Placement follows `docs/architecture.md#frontend` and the feature-sliced-design skill, and moved tests follow the testing skill.

Affected areas:

- `frontend/src/features/databases`, which is deleted, and the new `frontend/src/features/database-connection`;
- `frontend/src/pages/databases/ui` and the restore import in `frontend/src/features/restores/ui/RestoresComponent.tsx`;
- `frontend/src/entities/databases`, for the public exports and the parser rename;
- `frontend/fsd-baseline.json` and `frontend/style-baseline.json`;
- `docs/architecture.md` and `frontend/README.md`, which name `features/databases`;
- `temp/refactor/frontend-plan.md`, which records the split of item 5.3.

There is no backend, HTTP contract, generated code, dictionary or dependency change.

### Out of Scope

- Any change to the props, state or behavior of the moved forms, including dead props such as `isShowDbName` and the unused `isSaveToApi` branch of the base-info form.
- Moving `CreateReadOnlyComponent` to the page and `toDatabaseSettingsTarget` to the entity. `refactor-database-connection-form` does that when the connection form stops rendering the dialog.
- Repairing the 63 `direct-color` findings. The moved form files are rewritten by the next change.
- Role drafts, merging the MySQL and MariaDB forms or the PostgreSQL forms, the test-to-save lifecycle, the restore form slot and the connection form defects. These belong to `refactor-database-connection-form`.
- Merging the MySQL and MariaDB parsers.
