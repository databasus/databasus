Each task that fixes a defect starts by adding its tests on the unchanged code, with responses declared on the real screen, and by seeing them fail for the reason the task names.

## 1. Map coverage

- [x] 1.1 Create `coverage.md` in this change, mapping every test under `pages/databases/ui/physical-backups`, `ui/wal-stream-monitoring`, `lib/wal-stream-monitoring` and `lib/physical-backups` to its owner, with the counts of `pnpm --dir frontend test`. Verification: `coverage.md` lists each test file that `rg --files frontend/src/pages/databases/ui/physical-backups frontend/src/pages/databases/ui/wal-stream-monitoring frontend/src/pages/databases/lib/wal-stream-monitoring frontend/src/pages/databases/lib/physical-backups | rg 'test\.tsx?$'` prints.

## 2. Monitoring lifecycle

- [x] 2.1 In `pages/databases/ui/wal-stream-monitoring`, add tests for: a language switch and a scheduled refresh during a full backup request keep the button busy until the refresh after the answer has landed, with one request sent; a failure and a success after a language switch appear in the new language; leaving during a request shows no message and sends no statistics request, and no timer remains; a failure after leaving shows no error, and remounting the tab offers the button and it can be pressed; and hiding the browser tab during a request keeps the message and the busy state. Add `stopReporting` to `FullBackupRequestController` with node tests, create it once per database with callbacks on `i18n.t`, and delete the comment about a recreated controller. Verification: `pnpm --dir frontend test src/pages/databases/lib/wal-stream-monitoring src/pages/databases/ui/wal-stream-monitoring`.
- [x] 2.2 Replace raw colors in `ui/wal-stream-monitoring` with semantic roles and remove their entries from `style-baseline.json`. Reduced motion, numbers and time formats stay as they are. Verification: `pnpm --dir frontend test src/pages/databases/ui/wal-stream-monitoring`, `mise run frontend:lint`, and `rg -n 'wal-stream-monitoring' frontend/style-baseline.json` finds nothing.

## 3. Restore screen

- [x] 3.1 Add `pages/databases/ui/physical-backups/PhysicalRestoreComponent.test.tsx` with tests for the backup and point-in-time flows and for each scenario of the requirement: an earlier token answering last, an earlier request failing last, waiting for the newest target time, a refused newest request with its hint and no commands, clearing a chosen time, and closing during a request. Add `usePhysicalRestoreToken` on `RequestRevision`, taking the restore token subject, and show commands only with its token. Verification: `pnpm --dir frontend test src/pages/databases/ui/physical-backups/PhysicalRestoreComponent.test.tsx`.
- [x] 3.2 Split the screen into its command and environment panel units, clear the copy timer on unmount, and replace raw colors, removing its `style-baseline.json` entries. Verification: `pnpm --dir frontend test src/pages/databases/ui/physical-backups src/pages/databases/lib/physical-backups` and `mise run frontend:lint`.

## 4. List

- [x] 4.1 Extract `useBackupConfigLoad` and `BackupConfigNotice` from the logical list into `ui/backup-config`, and `formatBackupSize` into `lib`, with unchanged logical behavior. Verification: `pnpm --dir frontend test src/pages/databases/ui/logical-backups`, unchanged.
- [x] 4.2 In `PhysicalBackupsComponent.test.tsx`, add tests for: a failed config load offers a retry, ends the spinner and does not say backups are disabled; a late config answer after unmount shows nothing; a failed first page offers a retry, shows no empty text and no total usage, and keeps the start disabled; and a failed next page keeps the rows, reports once and is requested again on the next scroll. Use the shared units and render `firstLoadError`, and show nothing of the usage until the first page has loaded. Verification: `pnpm --dir frontend test src/pages/databases/ui/physical-backups/PhysicalBackupsComponent.test.tsx`.
- [x] 4.3 Add tests where a viewer sees no start, cancel or delete and sees restore, a member sees the start, and a manual backup keeps the tail and shows no loading state, with the new backup appearing through a later poll. Gate the start with `isCanManageDBs`, and delete the wait and the reload after a manual backup. Verification: `pnpm --dir frontend test src/pages/databases/ui/physical-backups/PhysicalBackupsComponent.test.tsx`.
- [x] 4.4 Add tests where a refused start, cancel or delete reports once, keeps the rows and leaves the control usable with no `alert`, and where an answer after unmount or a session change shows nothing. Split the list into `PhysicalBackupsTable`, `PhysicalBackupCards`, `PhysicalBackupActions`, `PhysicalBackupStatusBadge` and `usePhysicalBackupActions` with `useSessionGuard`, keyboard buttons and accessible names on icon actions. Verification: `pnpm --dir frontend test src/pages/databases/ui/physical-backups` passes and `rg -n '\balert\(' frontend/src/pages/databases/ui/physical-backups` finds nothing.
- [x] 4.5 Replace raw colors in the list, filters, type cell and `physicalBackupStatusBadgeStyles` with the roles of decision 6 and remove their entries from `style-baseline.json`. Verification: `mise run frontend:lint` passes and `rg -n 'physical-backups' frontend/style-baseline.json` finds nothing.

## 5. Browser and handover

- [x] 5.1 Extend `frontend/browser/scenarios/wal.spec.ts` to check plot geometry in the dark theme. Add `frontend/browser/scenarios/physicalBackups.spec.ts` on a mocked API for the restore dialog's focus and keyboard use, the list at a narrow viewport, and an accessibility scan of the list with `FULL`, `INCREMENTAL` and `WAL` badges in both themes. Verification: `pnpm --dir frontend test:browser --project=ui`.
- [x] 5.2 Complete `coverage.md`, and mark this item done in `temp/refactor/frontend-plan.md`. Verification: `rg -n 'Final' openspec/changes/refactor-physical-backup-screens/coverage.md` finds the final counts, and `rg -n '\[x\] \*\*7\.5\.' temp/refactor/frontend-plan.md` finds the line.
- [x] 5.3 Format the changed files and run `mise run frontend:lint`, `pnpm --dir frontend build` and `pnpm --dir frontend test`. Verification: all three pass.

## Change verification

```sh
pnpm --dir frontend test
pnpm --dir frontend test:browser --project=ui
```
