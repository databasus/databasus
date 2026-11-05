## Why

The database list, tabs, settings and creation wizard still live in `features/databases`. To reach backups and verification, they import three sibling features, and the page hands them its own editors through render slots. The screen also breaks its user's flow:

- saving any settings section swaps the whole screen for a spinner and returns to the Backups tab;
- copying or transferring a database reloads the browser page;
- going back from the wizard's notifier step loses the logical backup settings;
- Cancel reloads the database with no error handling, so a failed request leaves a permanent spinner.

Milestone 5 moves the screen to its final owner before 5.3 rewrites the connection forms, so their consumers already stand where they will stay.

## What Changes

- Move the database list, card, tabs, settings sections, creation wizard, show components and healthcheck config/attempts from `features/databases` to `pages/databases` without behavior changes. Leave the connection, base-info, PostgreSQL backup type and read-only user forms in `features/databases`, behind its public API. Restore imports its connection form through that API until 5.3.
- Delete the `DatabaseResourceComposition` slots. The page imports storage selection, notifier settings and the transfer dialog directly, and it reaches backups, monitoring and verification through their features' public APIs.
- Remove the eight database `fsd/forbidden-imports` entries and the restore `project/private-slice-import` entry from `fsd-baseline.json`. The restore `fsd/forbidden-imports` entry is replaced by one whose target is `features/databases/index.ts`, with owner `restores` and milestone 5. That entry stays until 5.3.
- Select databases through `useScopedSelection` and lay the screen out with `ResourceListDetail`. Delete the screen's own localStorage selection, the `contentHeight` state in `ApplicationScreen` and the `onContentHeight` measurement in `widgets/app-shell`, whose only consumer is this screen.
- Replace the six `isEdit*` flags with one `editingSection` and give renaming its own name draft.
- Saving a section closes only that section. The selected database takes the server's response and the list refreshes silently, so the open tab stays.
- Copy selects the copy, and transfer moves the selection to a neighboring database. Neither reloads the browser page.
- The creation wizard keeps each step's result with one owner. `EditLogicalBackupConfigComponent` accepts `initialConfig`, as the physical editor already does.
- Cancel only closes its section. The healthcheck and restore verification summaries already read their stored values when they remount after their editor closes, so the database reload on close is deleted.
- The database and backup config loads gain request revisions, and the list keeps its existing one. All three loads distinguish pending from failure, offer a translated retry instead of a browser alert, ignore late responses for another database or workspace and stop when the screen unmounts.
- The last backup error stays visible after a successful connection test. Its hint names the next successful backup as the only way to clear it. Today the hint also promises a connection test, but the server never clears the error on a test (the `databases` spec forbids that), so the screen hides the error only until its next reload.
- **BREAKING (internal imports only)**: the `features/databases` exports of `DatabasesComponent`, `DatabaseComponent`, `DatabaseConfigComponent` and the `DatabaseResourceComposition` types are deleted, along with `pages/databases/ui/databaseResourceComposition.tsx`. No forwarding exports remain. Tabs, addresses and HTTP requests stay the same.

## Capabilities

### New Capabilities

- `frontend-database-screens`: the database screen's settings sections, saving, copy and transfer, creation wizard steps, and recoverable loads.

### Modified Capabilities

- `frontend-workspace-screens`: the workspace-scoped selection requirement extends from storages and notifiers to databases.

## Impact

This is the second change of milestone 5, after `fix-restore-target-connection`. Root `AGENTS.md` and `frontend/AGENTS.md` apply. Placement follows `docs/architecture.md#frontend` and the feature-sliced-design skill, tests follow the testing skill, and visible copy follows the frontend-design and frontend-localization skills.

Affected areas:

- `frontend/src/features/databases`, `frontend/src/pages/databases`, `frontend/src/app/ui/ApplicationScreen.tsx` and `frontend/src/widgets/app-shell`;
- the public API of `features/backups/logical` (`initialConfig`) and `shared/ui/ResourceListDetail` (a ref to its detail scroll region);
- the restore import path in `features/restores`;
- `frontend/fsd-baseline.json`;
- the six dictionaries, for the last backup error hint;
- `docs/architecture.md` and `docs/frontend-design.md`, which describe the render contracts and the `contentHeight` transition this change removes;
- the screen tests, plus a browser UI scenario for the screen's scroll owner.

There is no backend, HTTP contract, generated code or dependency change.

### Out of Scope

- The connection forms themselves, drafts per role, MySQL/MariaDB and PostgreSQL form merging, test-to-save revisions and the restore form slot. These belong to `refactor-database-connection-form` (5.3).
- Logical and physical backup config internals, schedules, retention, backup and restore lists, WAL monitoring and verification. These belong to milestones 6 and 7, and the page uses them as they are.
- Browser alerts for refused mutations (test connection, copy, delete). Only load failures get inline feedback here.
- A full 320-pixel and 200-percent text pass over the database screen. This change covers only the scroll ownership it alters.
- Moving one-off cards or panels into entities, new deep links, a generic resource page or form engine.
- Backend authorization changes. UI permissions keep hiding controls, and the backend still decides.
