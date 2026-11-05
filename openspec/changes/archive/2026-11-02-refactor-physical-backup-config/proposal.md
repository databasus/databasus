## Why

The physical config editor serves two jobs through flags and through a fake `Database` with an empty ID: the creation wizard's step and the settings section. It is the last consumer of that adapter apart from the notifier step, and the same pattern remains in the notifier step. The two copies of the retention values are synchronized by hand, and failed loads leave an empty section with no way out. The spec even carries an explicit exception for this editor, added until this change.

Two defects follow from this structure. Back from the wizard's physical step discards everything the user has not yet continued with, which logical no longer does. A full-backup retention also sends the values of the unchosen policy, and the server refuses a `LAST_N` retention that keeps GFS periods (`physical_config.go:37-60`), so switching the default GFS retention to `LAST_N` and saving looks refused by the server.

## What Changes

- The wizard step and the settings section become two containers over one controlled set of fields. `isSaveToApi`, `isShowBackButton`, `isShowCancelButton`, `saveButtonText` and the branches on `database.id` are deleted.
- The settings section's load, save and storage-switch flow is shared with the logical section through one unit, so physical does not get a second copy of it.
- The wizard's notifier step and the settings section of notifiers split the same way, so `toUnsavedDatabase` is deleted and both wizard steps take only what they use.
- Retention has one draft that keeps entered values when a policy switches, and one function builds the contract's input at submit: it drops the chain or full-backup part the chosen retention policy forbids, and the count or GFS periods the chosen full-backup policy forbids. `walLagThresholdBytes` stays in the draft unchanged, because the form does not edit it.
- Back from the wizard's physical step keeps the entered values, and the backup type is recorded with them for the rule of `fix-wizard-partial-create`.
- Save follows the values the form shows against the acknowledged config. The retention rules apply with backups off, because the retention stays on the form and the server validates it then too.
- A failed load in the editor offers a retry and Cancel. A refused save shows its reason inline. The show view receives the config the settings screen already loads, so one failed request gives one error and one retry, and a failed storage load offers its own retry. No `alert` remains here.
- The storage-switch state and the history request after a save are deleted, because the section closes at once and the show view loads the history itself. `StorageSwitchHistoryComponent` keeps `initialOperation` for the transfer dialog.
- Raw colors become semantic roles, which removes the style baseline entries of the editor and the view, 27 occurrences.

Nothing here breaks an API, a schema or a config.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `frontend-database-screens`: removes the physical exception from the load requirement, extends the wizard's unsaved-input guarantee to physical databases, and adds the physical retention and save rules.

## Impact

This is the fourth change of frontend milestone 7, after `refactor-verification-config`. The root `AGENTS.md` and `frontend/AGENTS.md` apply, with the frontend-design and testing skills, and the frontend-localization skill if a control needs a new name.

Affected areas:

- `frontend/src/pages/databases/ui/backup-config`, including `EditLogicalBackupConfigSection.tsx`, new units in `pages/databases/model/physical-backups`, and `pages/databases/ui/EditDatabaseNotifiersComponent.tsx`, which is replaced;
- `pages/databases/ui/CreateDatabaseComponent.tsx` and `DatabaseConfigComponent.tsx`;
- `frontend/style-baseline.json` and a new browser scenario;
- `docs/glossary.md`, which gains the physical backup config draft and the physical database backup type, `docs/architecture.md` if the page's description stops being true, and `temp/refactor/frontend-plan.md`.

There is no backend change, migration or new dependency.

### Out of Scope

- A shared retention engine for logical and physical configs, whose rules differ.
- An editor for `walLagThresholdBytes`.
- The schedule rule with backups off, which `recurring-schedules` keeps as it is.
- The physical backup list, restore and monitoring, which `refactor-physical-backup-screens` handles.
- `alert` for copy, test and delete in `DatabaseConfigComponent`, which milestone 8 replaces.
