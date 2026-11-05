## Why

One connection form serves the creation wizard, the settings section and the restore target, and it infers which of them it is serving from `id: ''` and boolean props. That inference caused the source password leak fixed in `fix-restore-target-connection`, and nothing in the form's types prevents it from coming back. Five near-copies of the same form also disagree in small ways, and they let a late test answer approve settings that were never tested.

This change depends on `move-database-connection-form`, which moves the code to its final owners first.

## What Changes

- Add `DatabaseCreateDraft`, `DatabaseEditDraft` and `RestoreTargetDraft` to `entities/databases`, with a `scenario` field. Create and edit drafts go straight to the existing `databaseApi` operations. A new `toDatabaseSettingsTarget` in the entity accepts only those two drafts and replaces the feature's private function, which is deleted with the old forms, so a restore target draft has no database identity and cannot produce a test request.
- Make the form controlled. The wizard, the settings section and restore own their draft, saving, navigation and the limited user step. Delete `isSaveToApi`, `isRestoreMode`, `isShowBackButton`, `isShowCancelButton`, `isShowDbName` and `EditDatabaseSpecificDataComponent`.
- Replace `CreateReadOnlyComponent` with `CreateLimitedUserComponent` in `pages/databases/ui`, since both of its consumers now live on the page. The old component is deleted with the old forms.
- Merge the MySQL and MariaDB forms into one editor and their parsers into `MySqlCompatibleConnectionStringParser`, adding MariaDB parser tests before the merge.
- Share one PostgreSQL connection and TLS unit between the logical and physical forms. MongoDB keeps its own form. All engines share the SSH section, the connection string import and the test lifecycle.
- Add a domain test hook whose answer applies only to the settings revision it tested.
- Pass the form to restore through a page render prop, and remove the last milestone 5 entry from `fsd-baseline.json` and all milestone 5 entries from `style-baseline.json`.
- Fix these defects, each with a test that first fails on the current code:
  - a late successful test approves settings changed after the test began;
  - a saved connection stays unknown to the page if the limited user dialog is closed or the check before it fails;
  - going back through the wizard forgets the user's SSL choice, so a later host change can turn SSL off and erase certificates;
  - going back from the connection step loses the input of that step;
  - choosing another engine and returning to the original one before continuing discards the connection input, and so does clicking the highlighted PostgreSQL tile for a physical database;
  - a certificate typed in the wizard or a restore target shows as masked and stored when the form opens again, and a saved certificate shows as stored after SSL was turned off and on even though the save removes it;
  - changing the SSH settings of a physical database keeps showing the previous failed test;
  - the restore target offers MySQL, MariaDB and MongoDB exclusion settings that the restore request ignores.
- Replace browser alerts and the manual toast in these forms with inline and context-bound AntD feedback, and link visible labels to their controls.
- **BREAKING (internal props only)**: the form's props and the prop types of the base-info and limited user forms change, and no adapter for the old form remains. Addresses and HTTP requests stay the same.

## Capabilities

### New Capabilities

- `frontend-database-connection`: what the connection form asks for in each scenario, how it treats stored secrets and certificates, how a test approves settings, how SSL follows the host, how connection strings are imported, and how failures and pending work appear.

### Modified Capabilities

- `frontend-database-screens`: a saved connection reaches the page before the limited user suggestion resolves, the section states when it closes, and the wizard keeps the connection input when going back and compares the type on Continue.
- `restores`: the restore target form shows only the settings a restore uses, and it keeps the previous target's values while the backup's dialog stays open.

## Impact

This is the last change of milestone 5. Root `AGENTS.md` and `frontend/AGENTS.md` apply. Placement follows `docs/architecture.md#frontend` and the feature-sliced-design skill, tests follow the testing skill, and visible copy follows the frontend-design and frontend-localization skills.

Affected areas:

- `frontend/src/entities/databases` and `frontend/src/features/database-connection`;
- `frontend/src/pages/databases/ui`, for the wizard, the settings section and the render prop;
- `frontend/src/features/restores` and the `LogicalBackupsComponent` prop in `frontend/src/features/backups/logical`;
- `frontend/fsd-baseline.json` and `frontend/style-baseline.json`;
- the six dictionaries, only if the inline test failure needs new copy;
- `docs/architecture.md`, `docs/glossary.md` and `temp/refactor/frontend-plan.md`;
- screen tests and a new browser UI scenario.

There is no backend, HTTP contract, generated code or dependency change.

### Out of Scope

- Rewriting the logical and physical backup config editors. They keep their `database` prop, so the wizard adapts its draft to that shape in one place until milestones 6 and 7 replace them.
- A generic form engine, a new form library or a change to `useStorageDraft`.
- Merging MongoDB with the other engines, or merging restore policy and replication diagnostics into shared code.
- Restore list, polling, cancel and download behavior, which milestone 6 covers.
- A cancel button in the restore target form. The dialog's close control stays the way out.
- Skipping the second limited user check the dialog makes after the section's check.
- How the server treats a stored client key when its certificate is removed. That rule stays unchanged.
- Real API browser tests for databases, because saving needs a reachable source database that the harness does not provide.
- Backend authorization and the rule that adding or changing a database requires a successful test.
