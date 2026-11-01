## Why

The database creation wizard sends its work as separate requests: it creates the database, saves the backup configuration and starts the first backup. If a later request fails, the database already exists, the wizard stays on its last step, and pressing Complete again calls the create operation a second time. The user ends up with a duplicate database and no explanation that the first one exists.

One path makes this failure likely. A physical database's backup type is chosen on the connection step, but the wizard compares only the database type. After going back and choosing another backup type, the wizard keeps a configuration built for the old one. Its hidden chain settings and incremental schedule then reach the server, which refuses them after the database is created.

This fix comes first in frontend milestone 7 because it protects user data and does not depend on the screen moves that follow.

## What Changes

- The wizard creates a database once. A retry repeats only the steps that have not finished: it applies a changed notifier selection, saves the backup configuration if the server has not acknowledged it, and starts the first backup.
- The wizard stays on its last step after a partial failure. A lasting error says the database was created and gives the server's reason, and Complete remains available.
- When the wizard closes, the database it created joins the list and becomes selected. The screen therefore loads what the server holds at that moment, whether the wizard finished or the user gave up.
- After the database exists, the user cannot return to the connection step, because those values would no longer be applied. After the server acknowledges the backup configuration, the user cannot return to that step either, because another storage would start a storage switch the wizard does not manage. The notifier selection stays editable.
- Closing the wizard while its requests run stops the later steps, and a late answer changes nothing in a wizard opened afterwards.
- Choosing another physical backup type on the connection step starts that database's backup configuration over, in the same way a different database type already does.
- `onCreated` no longer closes or selects. The wizard reports the creation with `onCreated`, which the host only remembers, and its completion with `onClose`. A closed wizard calls neither.

Nothing here breaks an API, a schema or a config.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `frontend-database-screens`: limits going back to the connection step to the time before the database exists, and adds that completing the wizard creates one database and that a changed physical backup type starts its backup configuration over.

## Impact

This is the first change of frontend milestone 7. The root `AGENTS.md` and `frontend/AGENTS.md` apply, with the frontend-localization and testing skills.

Affected areas:

- `frontend/src/pages/databases/ui/CreateDatabaseComponent.tsx` and its test;
- `frontend/src/pages/databases/ui/DatabasesComponent.tsx`, which hosts the wizard, and its test;
- `frontend/src/pages/databases/ui/backup-config/CreateLogicalBackupConfigStep.tsx`, whose Back action becomes optional, and its test;
- `frontend/src/pages/databases/model/startFirstBackup.ts`, which holds the rule that disabled backups start no backup, and its test;
- the six dictionaries in `frontend/src/shared/i18n/locales`, for the new error and progress texts;
- `temp/refactor/frontend-plan.md`, the working plan of the milestone.

There is no backend change, migration or new dependency.

### Out of Scope

- Keeping unsaved physical input through Back to the connection step, the physical editor's structure and `toUnsavedDatabase`. `refactor-physical-backup-config` handles them.
- A single server operation that creates a database with its configuration atomically. It would change the contract and is not needed to stop duplicates.
- Removing a database left behind when the user abandons the wizard after a failure. The user can delete it in its settings.
- Storage switches inside the wizard, which the locked step prevents once the server has acknowledged the save. `design.md` records the exception for a lost response.
