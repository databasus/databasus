# Logical backup screens refactoring coverage

## Baseline

Baseline `pnpm --dir frontend test` on 2026-11-01, before this change: 148 files, 1647 tests passed.

## Ownership map

Paths are relative to `frontend/src`.

| Existing test | Final owner | Retained guarantee |
| --- | --- | --- |
| `pages/databases/ui/backup-config/EditLogicalBackupConfigComponent.test.tsx` | `pages/databases/ui/backup-config/EditLogicalBackupConfigSection.test.tsx`, `CreateLogicalBackupConfigStep.test.tsx` and `pages/databases/ui/DatabaseConfigComponent.test.tsx` | The wizard starts from an entered config, a storage reassignment asks for confirmation, a schedule keeps its day and time, GFS hours follow the schedule, and Save follows a complete schedule |
| `pages/databases/ui/backup-config/ShowLogicalBackupConfigComponent.test.tsx` | Same file | A weekly schedule shows its fields and its weekday |
| `pages/databases/ui/logical-backups/LogicalBackupsComponent.test.tsx` | Same file | Cursor paging, the poll keeps the scrolled tail, a started backup shows at once, a download follows the token link, and a refused verification shows its reason |
| `pages/databases/ui/restores/RestoresComponent.test.tsx` | Same file | A physical database offers no target button |
| `pages/databases/ui/restores/RestoreTargetComposition.test.tsx` | Same file | Every target engine sends its exact body, refusals keep input, one start per click, and late answers show nothing |
| `shared/lib/PagedListController.test.ts` | Same file | Paging, refresh merging, stale answers and locks |

## Tests changed beyond their imports

- `shared/lib/PagedListController.test.ts`: `reports an error from the current list and ignores one from a replaced list` is rewritten as `stays silent when a refresh fails for the current list or a replaced list`, because decision 12 makes a failed refresh silent. Its stale refresh still reports nothing, and its current failed refresh now reports nothing too.
- `pages/databases/ui/backup-config/EditLogicalBackupConfigSection.test.tsx`: moved from `EditLogicalBackupConfigComponent.test.tsx`. Every render uses the section, and the accepted storage switch no longer appears in the form, because the section closes at once. `DatabaseConfigComponent.test.tsx` now pins that text in the configuration view with `shows an accepted storage switch in the configuration view after a save`.
- `warns before reassignment to %s after clearing storage without remounting` moved to `DatabaseConfigComponent.test.tsx` as `warns before reassignment to %s when the section opens again`, per decision 4, because the real section closes after a save.
- `starts an unsaved database from the given configuration instead of the defaults` moved to `CreateLogicalBackupConfigStep.test.tsx` as `starts from the given configuration instead of the defaults`.
- `pages/databases/ui/backup-config/ShowLogicalBackupConfigComponent.test.tsx`: the view receives the config, so the test passes it and declares no config request.
- `pages/databases/ui/DatabaseConfigComponent.test.tsx`: `ignores pending configuration errors after the screen unmounts`, `offers a retry when the backup configuration fails to load` and `ignores a late backup configuration failure from an earlier load of the same database` count one config request per load instead of two, per decision 15.
- `features/backups/physical/ui/PhysicalBackupsComponent.test.tsx` and `features/verification/runs/ui/VerificationsComponent.test.tsx`: new scenarios pin a silent failed poll and a silent failed refresh after a successful cancel on these two lists, because decision 12 reaches them.
- `pages/databases/ui/logical-backups/LogicalBackupsComponent.test.tsx` and `restores/RestoreTargetComposition.test.tsx`: icon queries became button names, because the actions are now named buttons. The guarantees did not change.
- `pages/databases/ui/logical-backups/LogicalBackupsComponent.test.tsx`, scenario `refreshes the first page and keeps the scrolled tail when the poll answers`: it now advances a fake clock instead of waiting for the real interval, because `docs/testing.md` says polling tests advance a local fake clock. Its assertions and guarantee did not change.

## Result

Final `pnpm --dir frontend test`: 154 files, 1756 tests passed. No guarantee of the baseline tests was dropped; each one kept its scenario or moved as recorded above.
