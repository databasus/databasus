# Database screen coverage

## Baseline

`fix-restore-target-connection` is implemented and archived. Baseline `pnpm --dir frontend test` on 2026-11-01: 132 files, 1270 tests passed.

The first baseline run also exposed a test-infrastructure defect. `src/shared/lib/test/runIsolatedScenario.ts` returned the child Vitest output with terminal colors whenever the parent environment enabled them, so the three harness tests that look for the plain `1 failed | 1 passed` summary failed. The runner now strips terminal control characters, and `mockApi.test.ts` and `screenTestLifecycle.test.tsx` pass.

## Ownership map

Paths are relative to `frontend/src`. No scenario is removed.

| Existing test | Final owner | Retained guarantee |
| --- | --- | --- |
| `features/databases/ui/CreateDatabaseComponent.test.tsx` | `pages/databases/ui/CreateDatabaseComponent.test.tsx` | The wizard advances to the read-only user step with the submitted connection settings |
| `features/databases/ui/healthcheck/EditHealthcheckConfigComponent.test.tsx` | `pages/databases/ui/healthcheck/` | Translated `permission_denied` after a refused save |
| `features/databases/ui/healthcheck/HealthcheckAttemptsComponent.test.tsx` | `pages/databases/ui/healthcheck/` | Every cursor is followed; a period change discards old pages |
| `features/databases/ui/show/ShowPostgreSqlLogicalSpecificDataComponent.test.tsx` | `pages/databases/ui/show/` | Each exclusion list under its own label, hidden when unset |
| `features/databases/ui/edit/*.test.tsx` | Unchanged, behind the `features/databases` public API | Password omission, failed test reasons, MySQL and PostgreSQL form labels |
| `features/restores/ui/RestoresComponent.test.tsx` | Unchanged; the form now comes from the `features/databases` public API | Restore target entered without source settings, refusals keep input, one start |
| `pages/databases/ui/DatabasesComponent.test.tsx` | Unchanged path, imports the moved page component | Unavailable databases first, one backup config list for card storages |
| `pages/databases/ui/DatabaseComponent.test.tsx` | Unchanged path, imports the moved component | Late responses of an older selection, no late error after unmount, settings tab and edits survive a language change |
| `pages/databases/ui/DatabaseConfigComponent.test.tsx` | Unchanged path, imports the moved component | No late config errors after unmount, limited-user hint for editors only |
| `pages/databases/ui/EditLogicalBackupConfigComponent.test.tsx`, `EditPhysicalBackupConfigComponent.test.tsx` | Unchanged paths; storage selection rendered directly instead of through the deleted slots | Replacement confirmation, accepted switch progress, retained draft on canceled nested creation |
| `pages/databases/ui/BackupStorageSelection.test.tsx`, `DatabaseTransferDialogComponent.test.tsx`, `NotifierComposition.test.tsx` | Unchanged | Nested creation, transfer policy and progress, notifier composition |

## Added tests

Each test was run against the code before its repair. "Failed first" marks the tests that reproduced a defect there; the others pin behavior the refactoring must keep.

| Test | Scenario | Before the repair |
| --- | --- | --- |
| `pages/databases/ui/DatabasesComponent.test.tsx` | selects the remembered database on desktop | Failed first: the card had no pressed state to select |
| same | shows the list without a selected database on mobile | Failed first, for the same reason |
| same | selects the first remaining database after the selected one is deleted | Failed first, for the same reason |
| same | offers a retry instead of an empty list when the first list load fails | Failed first: browser alert, then an empty list |
| same | keeps the list without feedback when a background refresh fails | Failed first: the card had no pressed state |
| same | keeps showing the newly selected database when the previous one responds last | Passed: the detail effect already discarded the late response |
| same | renames a database in its settings and its card while keeping the selection | Passed once the page accepted saved databases (task 2.1) |
| same | keeps the settings tab and the list after the connection settings are saved | Failed first: the section stayed open until a remount |
| same | keeps the connection section open with the entered values when saving is refused | Passed |
| same | shows the newly chosen storage on the card after the backup configuration is saved | Passed |
| same | adds and selects the copy after a successful copy, without a page reload | Failed first: the page reloaded instead |
| same | keeps the list and the selection when copying is refused | Passed |
| same | removes the transferred database and selects the first remaining one, without a page reload | Failed first: the page reloaded instead |
| `pages/databases/ui/DatabaseComponent.test.tsx` | offers a retry that reloads the same database when its request fails | Failed first: browser alert and a permanent spinner |
| same | offers no edit control in other sections while one section is open | Failed first |
| same | closes a section on cancel without requesting the database again | Failed first: Cancel reloaded the database |
| same | shows the stored healthcheck values after saving and requests no database | Failed first: closing reloaded the database |
| same | keeps the last backup error after a successful connection test | Failed first: the error was hidden locally |
| same | offers a viewer no edit, copy, transfer or delete control but keeps the connection test | Passed |
| `pages/databases/ui/DatabaseConfigComponent.test.tsx` | shows no notifier or verification error after the screen unmounts | Failed first: two alerts after unmount |
| same | offers a retry when the backup configuration fails to load | Failed first: browser alert |
| `pages/databases/ui/DatabaseTransferDialogComponent.test.tsx` | keeps the dialog and its choices open when the transfer is refused | Passed |
| `pages/databases/ui/CreateDatabaseComponent.test.tsx` | keeps the entered backup configuration after going back from the notifier step | Failed first: the logical editor started from defaults |
| same | creates the database with the configuration changed after going back | Passed |
| same | starts the connection and backup configuration over when the engine changes | Fails when the type-change reset is removed |
| same | sends one create request when Complete is clicked twice while pending | Passed |
| same | keeps the settings tab and the list after the notifier settings are saved | Passed after the section refactor; pins the notifier save path |
| same | leaves no database selected after the only database is transferred | Passed |
| same | keeps the list and the selection when the transfer is refused | Passed |
| `pages/databases/ui/DatabaseComponent.test.tsx` | shows the stored restore verification values after saving and requests no database | Passed after the section refactor |
| same | shows the latest load of a database when an earlier load of it responds last | Fails when the effect cleanup is removed |
| `pages/databases/ui/DatabaseConfigComponent.test.tsx` | ignores a late backup configuration failure from an earlier load of the same database | Fails when the effect cleanup is removed |
| `pages/databases/ui/CreateDatabaseComponent.test.tsx` | keeps the entered physical backup configuration after going back from the notifier step | Passed: the physical editor already took `initialConfig` |
| same | starts the connection and both backup configurations over when the PostgreSQL backup type changes | Fails when the type-change reset is removed |
| `features/backups/logical/ui/EditLogicalBackupConfigComponent.test.tsx` | starts an unsaved database from the given configuration instead of the defaults | New prop |
| `browser/scenarios/databaseResources.spec.ts` | scrolls the database details in one region and loads further backups at 320 and 1280 pixels | New scenario |

The defect behind "shows no notifier or verification error after the screen unmounts" surfaced while writing the deletion test. `ShowDatabaseNotifiersComponent`, `ShowVerificationConfigComponent` and `EditHealthcheckConfigComponent` now ignore responses after unmount.

No existing test was removed.
