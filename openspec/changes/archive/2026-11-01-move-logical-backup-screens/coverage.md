# Logical backup screens move coverage

## Baseline

Baseline `pnpm --dir frontend test` on 2026-11-01, before the move: 141 files, 1544 tests passed.

## Ownership map

Paths are relative to `frontend/src`.

| Existing test | Final owner | Retained guarantee |
| --- | --- | --- |
| `entities/backups/config/api/backupConfigApi.test.ts` | `entities/backup-configs/api/backupConfigApi.test.ts` | Accepted database transfers return their operation, missing operation data is rejected, and 204 completes without data |
| `features/backups/logical/ui/StorageSwitchHistoryComponent.test.tsx` | `entities/backup-configs/ui/StorageSwitchHistoryComponent.test.tsx` | A pending switch is discovered and polled, a failed read keeps the accepted state, history pages load by cursor, and polling stops on unmount |
| `features/backups/logical/ui/LogicalBackupsComponent.test.tsx` | `pages/databases/ui/logical-backups/LogicalBackupsComponent.test.tsx` | Cursor paging, the poll keeps the scrolled tail, a started backup shows at once, downloads follow the token link, and download failures are shown |
| `features/backups/logical/ui/EditLogicalBackupConfigComponent.test.tsx` | Merged into `pages/databases/ui/backup-config/EditLogicalBackupConfigComponent.test.tsx` | An unsaved database starts from the given configuration, including its entered storage |
| `pages/databases/ui/EditLogicalBackupConfigComponent.test.tsx` | `pages/databases/ui/backup-config/EditLogicalBackupConfigComponent.test.tsx` | Storage reassignment confirmation and acceptance, and the day of a monthly schedule shown and submitted in November |
| `features/restores/ui/RestoresComponent.test.tsx` | `pages/databases/ui/restores/RestoresComponent.test.tsx` | A physical database offers no target button; the draft test is removed as described below |
| `pages/databases/ui/RestoreTargetComposition.test.tsx` | `pages/databases/ui/restores/RestoreTargetComposition.test.tsx` | Every target engine sends its exact restore body, refusals keep input, one start per click, no request names the source database, and late answers show nothing |

## Tests changed beyond their imports

- `pages/databases/ui/restores/RestoresComponent.test.tsx`: the test `gives the form a restore draft with no database identity and lists a started restore` is removed, because it observed a render prop that no longer exists. `RestoreTargetComposition.test.tsx` pins its guarantees on the real form: `starts the restore into the entered $target.type target without a connection test` sends the exact MySQL body and waits for the listed restore, `opens the target form as a new PostgreSQL database without the source settings` covers the empty target, and `starts the restore with only entered values and without the member-only connection test` asserts that no request names the source database. The physical database test drops only its check that the render prop was not called.
- `pages/databases/ui/backup-config/EditLogicalBackupConfigComponent.test.tsx`: merged from the feature's and the page's files of the same name. `starts an unsaved database from the given configuration instead of the defaults` answers the storage list and asserts that the selection shows the entered storage, instead of recording the ID passed to the render prop. The November scenarios declare an empty storage list, because the real selection requests it.
- `pages/databases/ui/restores/RestoreTargetComposition.test.tsx` and `pages/databases/ui/logical-backups/LogicalBackupsComponent.test.tsx`: only the removed render prop goes.

## Result

Final `pnpm --dir frontend test`: 140 files, 1543 tests passed. That is one file fewer because of the merged editor tests, and one test fewer because of the removed restore draft test. No other test was dropped.
