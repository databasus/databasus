# Physical backup config refactoring coverage

## Baseline

Baseline `pnpm --dir frontend test` on 2026-11-02, before this change: 159 files, 1855 tests passed.

## Ownership map

Paths are relative to `frontend/src`.

| Existing test | Final owner | Retained guarantee |
| --- | --- | --- |
| `pages/databases/ui/backup-config/EditPhysicalBackupConfigComponent.test.tsx`, `labels the fields of each schedule separately` | `pages/databases/ui/backup-config/PhysicalBackupConfigFields.test.tsx` | Each schedule has its own named fields |
| Same file, `offers only the type for an empty incremental schedule and keeps Save disabled`, `lifts the schedule rule when backups are turned off`, `leaves the other schedule unchanged in the submitted config when one is edited` | `pages/databases/ui/backup-config/EditPhysicalBackupConfigSection.test.tsx` | Save follows a complete schedule, the rule lifts with backups off, and an edit touches one schedule |
| Same file, `retains the storage selection and configuration draft when nested storage creation is canceled` | `pages/databases/ui/backup-config/EditPhysicalBackupConfigSection.test.tsx` | A canceled nested storage creation keeps the draft and sends nothing |
| Same file, `requires confirmation before reassigning cleared storage with retained history and handles an accepted response` | `pages/databases/ui/backup-config/EditPhysicalBackupConfigSection.test.tsx` and `pages/databases/ui/DatabaseConfigComponent.test.tsx` | A reassignment asks for confirmation, and the accepted switch appears in the configuration view |
| Same file, `warns before reassignment to %s after clearing storage without remounting` | `pages/databases/ui/DatabaseConfigComponent.test.tsx` | A reassignment after a cleared storage asks for confirmation |
| `pages/databases/ui/backup-config/ShowPhysicalBackupConfigComponent.test.tsx` | Same file | A saved weekday 0 shows as Sunday |
| `pages/databases/ui/backup-config/EditLogicalBackupConfigSection.test.tsx` | Same file, unchanged | The logical flow keeps its behavior over the shared section |
| `pages/databases/ui/NotifierComposition.test.tsx` | Same file | Pending notifier creation is ignored after a context change, a failed refresh keeps the creation's result, and a created notifier is selected |
| `pages/databases/ui/CreateDatabaseComponent.test.tsx` | Same file | The physical step keeps its input through rerenders and steps, and a changed backup type starts it over |
| `pages/databases/ui/DatabaseConfigComponent.test.tsx` | Same file | The physical section's pending save, refused save and late failures |
| `pages/databases/ui/DatabasesComponent.test.tsx`, `keeps the settings tab and the list after the notifier settings are saved` | Same file | A saved notifier selection closes the section and keeps the screen |

## Tests changed beyond their imports

- `pages/databases/ui/backup-config/EditPhysicalBackupConfigComponent.test.tsx`: deleted with the component. Its tests moved as mapped above, and every render uses the section or the fields.
- `requires confirmation before reassigning cleared storage with retained history and handles an accepted response` moved to `EditPhysicalBackupConfigSection.test.tsx` without its accepted-switch part, because the section closes at once. `DatabaseConfigComponent.test.tsx` now pins that text in the configuration view, and `entities/backup-configs/ui/StorageSwitchHistoryComponent.test.tsx` keeps the status retry.
- `warns before reassignment to %s after clearing storage without remounting` moved to `DatabaseConfigComponent.test.tsx`, where the logical `warns before reassignment to %s when the section opens again` now runs for both kinds, because the real section closes after a save.
- `pages/databases/ui/backup-config/ShowPhysicalBackupConfigComponent.test.tsx`: the view receives the config, so the test passes it and declares no config request.
- `pages/databases/ui/DatabaseConfigComponent.test.tsx`: `keeps the form, shows the reason and reports no save when the server refuses` expects an inline reason and no `alert` for physical too. `when the logical section cannot load`, `shows one error and one retry when the settings tab cannot load a logical config`, `shows the reason with a retry when the configuration view cannot load its storage` and `shows an accepted storage switch in the configuration view after a save` now run for both kinds.
- `pages/databases/ui/NotifierComposition.test.tsx`: the three renders of the old editor use `CreateDatabaseNotifiersStep` or `EditDatabaseNotifiersSection`. Their guarantees did not change.

New scenarios: the node tests under `pages/databases/model/physical-backups`, `PhysicalBackupConfigFields.test.tsx`, `CreatePhysicalBackupConfigStep.test.tsx`, `CreateDatabaseNotifiersStep.test.tsx`, `EditDatabaseNotifiersSection.test.tsx`, the `of the physical section` group of `DatabaseConfigComponent.test.tsx`, two scenarios of `CreateDatabaseComponent.test.tsx` and `frontend/browser/scenarios/physicalBackupConfig.spec.ts`.

## Guards

These new tests already held on the unchanged code, so they pin behavior this change must keep rather than a defect:

- `pages/databases/ui/DatabaseConfigComponent.test.tsx`: `shows the entered values after a switch of the full-backup policy and back`, `sends no full-backup retention for chains only`, `sends the stored WAL lag threshold of a WAL streaming database unchanged`, the physical runs of `disables Cancel during a pending save`, `warns before reassignment to %s when the section opens again` and `shows an accepted storage switch in the $kind configuration view after a save`.
- `pages/databases/ui/CreateDatabaseComponent.test.tsx`: `shows the defaults of the new type after Back without continuing and another type`, because the old step dropped every value on Back.
- `pages/databases/ui/DatabasesComponent.test.tsx`: the physical run of `keeps the entered $kind values when the page becomes visible and the list refreshes`.

The tests that failed on the unchanged code are the physical runs of `keeps the form, shows the reason and reports no save when the server refuses`, `when the $kind section cannot load` (two tests), `shows one error and one retry when the settings tab cannot load a $kind config` and `shows the reason with a retry when the $kind configuration view cannot load its storage`, plus `disables Save again when an edit is reverted`, `keeps Save disabled when a policy is tried and abandoned on a chains-only config`, `disables Save for all-zero GFS periods with backups off`, `sends the count and zero GFS periods for a last-N retention over stored GFS periods` and `shows the entered values after Back from the step and returning`.

## Result

Final `pnpm --dir frontend test`: 168 files, 1929 tests passed. No guarantee of the baseline tests was dropped; each one kept its scenario or moved as recorded above.
