## 1. Baseline and reproductions

- [x] 1.1 Load the testing skill. Run `pnpm --dir frontend test` and create `coverage.md` in this change with the baseline count and the map format of `openspec/changes/archive/2026-11-01-move-logical-backup-screens/coverage.md`, listing the current logical editor, list, restore, show view and controller tests. Verify that the file exists and names `EditLogicalBackupConfigComponent.test.tsx`, `LogicalBackupsComponent.test.tsx`, `RestoresComponent.test.tsx`, `RestoreTargetComposition.test.tsx` and `PagedListController.test.ts` with `rg -c 'EditLogicalBackupConfigComponent.test|LogicalBackupsComponent.test|RestoresComponent.test|RestoreTargetComposition.test|PagedListController.test' openspec/changes/refactor-logical-backup-screens/coverage.md`.
- [x] 1.2 Add screen tests on the current code through surfaces that survive the split: one reproduction per defect, plus the guard scenarios named below, which pin behavior the refactoring must keep. Guards are a refused logical and physical save keeping the form and reporting no save, a config with no GFS periods getting the defaults, advanced settings staying closed for defaults, selecting a logical database again reopening verifications, and a successful answer after a tab switch or a session change showing no feedback. A scenario about the physical editor has `physical` in its name, and a scenario about the configuration view has `configuration view` in its name, so later tasks can select them:
  - `CreateDatabaseComponent.test.tsx`: a parent rerender keeps logical and physical input without a spinner and keeps storage creation open for both, Continue works after Back from notifiers, and logical input survives Back to the connection step and return.
  - `DatabasesComponent.test.tsx`: a `visibilitychange` and a background refresh during the wizard keep the step's input, and a rename answering after another database was selected keeps that database shown while the list shows the new name.
  - `DatabaseConfigComponent.test.tsx`: Cancel is disabled during a pending logical and physical save, a refused logical and physical save keeps the form and input and reports no save, a failed logical config load in the section offers Retry and a way to close, a failed config load on the settings tab shows exactly one error and one Retry and no `alert`, and a successful Retry shows the configuration, a failed storage load in the configuration view shows its reason with Retry and no `alert`, a switch to GFS and back keeps values, a config with no GFS periods gets the defaults, hourly-only GFS can be saved, GFS with no shown period and backups off keeps Save disabled, reverting an edit and a policy round trip on a config with no GFS periods disable Save, advanced settings open for retry off and stay closed for defaults, and the toggle is absent with backups off.
  - `DatabaseComponent.test.tsx`: a logical and a physical save answering after `databaseId` changed keep the new database shown, do not call `onDatabaseChanged` and show no feedback; as guards, a successful logical and physical answer after a tab switch and after a session change shows no feedback; as reproductions that fail today because of the `alert`, a refused logical and physical answer after a tab switch and after a session change shows no feedback; a MySQL database selected on the verifications tab and a database without monitoring selected on the monitoring tab show the backups tab; and selecting a logical database again reopens verifications.
  - `LogicalBackupsComponent.test.tsx`: failing polls show no `alert` and keep rows, a failed first page shows an error with Retry and not the empty text, a failed config load shows an error and not "scheduled backups disabled", the restore dialog survives its backup leaving the list, values entered in its target form stay when a poll drops the backup, and a verification start answering after unmount does not navigate.
  - `RestoresComponent.test.tsx`: failing polls show no `alert` and keep restores, a failed first load shows an error with Retry, and that state offers no target selection.

  Run `pnpm --dir frontend test src/pages/databases/ui`, confirm that each reproduction fails for the reason its defect describes and that every guard and every older test passes, and record the results in this file under "Recorded results". If a reproduction does not fail, keep it as a guard, drop that defect's fix, say so in the same section and correct the artifacts.

## 2. Paged list controller

- [x] 2.1 Implement decision 12 of `design.md` in `shared/lib/PagedListController.ts` and `shared/hooks/usePagedList.ts`. Rewrite the controller test that pinned the refresh report, record the reason in `coverage.md`, and add scenarios for a failed refresh staying silent with its rows, cursor and pending next page intact and still allowing the next refresh, `firstLoadError` set by a failed first page and cleared by `reload`, a stale failure publishing nothing, `onError` receiving `failedPage` as `first` and as `next`, and a current next page rejecting once with unchanged rows and cursor before a repeated `loadNextPage` requests it again. Add screen scenarios to `PhysicalBackupsComponent.test.tsx` and `VerificationsComponent.test.tsx` for a failing poll and for a failing refresh after a successful cancel. Verify with `pnpm --dir frontend test src/shared/lib/PagedListController.test.ts src/shared/hooks src/features/backups/physical/ui/PhysicalBackupsComponent.test.tsx src/features/verification/runs/ui/VerificationsComponent.test.tsx` and `pnpm --dir frontend typecheck`.

## 3. Physical editor

- [x] 3.1 In `EditPhysicalBackupConfigComponent`, make its effect depend on `database.id`, disable Cancel during a save and guard answers with `useSessionGuard`, per decision 8. Keep its other behavior. Verify with `pnpm --dir frontend test src/pages/databases/ui/EditPhysicalBackupConfigComponent.test.tsx src/features/backups/physical`, and with `pnpm --dir frontend test src/pages/databases/ui/CreateDatabaseComponent.test.tsx src/pages/databases/ui/DatabaseConfigComponent.test.tsx -t physical`.

## 4. Config editor

- [x] 4.1 Add the pure units of decision 1 in `pages/databases/model`, following decisions 2, 3, 5, 6 and 7, with node tests that cover the policy switch with kept values and with zeros, hourly-only GFS on hourly and daily schedules, retention checked with backups off while schedule and encryption are not, storage not required by `isLogicalBackupConfigComplete`, a schedule change resetting `gfsHours`, triggers compared as a set, the fields of an unchosen policy ignored by `isLogicalBackupConfigChanged`, and non-default advanced values. Verify with `pnpm --dir frontend test src/pages/databases/model`.
- [x] 4.2 Load the frontend-design skill. Add `LogicalBackupConfigFields`, `LogicalRetentionFields`, `LogicalNotificationTriggerFields` and `LogicalAdvancedFields` in `pages/databases/ui/backup-config`, using `AdvancedSettingsToggleComponent` and a button for the GFS hint, with theme roles for colors. Verify with `pnpm --dir frontend typecheck` and `mise run frontend:lint`.
- [x] 4.3 Add `CreateLogicalBackupConfigStep` per decision 2 and use it in `CreateDatabaseComponent`, with `onBack` and `onContinue` storing the draft and without `toUnsavedDatabase` for this step. Add a scenario that Continue stays disabled without a storage. Verify with `pnpm --dir frontend test src/pages/databases/ui/CreateDatabaseComponent.test.tsx`.
- [x] 4.4 Apply decision 15: `DatabaseConfigComponent` keeps the config and passes it to `ShowLogicalBackupConfigComponent`, which loses its config request and `alert`, gains an inline storage error with Retry and uses theme roles. Verify with `pnpm --dir frontend test src/pages/databases/ui/backup-config/ShowLogicalBackupConfigComponent.test.tsx` and `pnpm --dir frontend test src/pages/databases/ui/DatabaseConfigComponent.test.tsx -t 'configuration view'`.
- [x] 4.5 Add `EditLogicalBackupConfigSection` per decisions 3, 4, 8 and 11, use it in `DatabaseConfigComponent`, and delete `EditLogicalBackupConfigComponent.tsx`. Move its test scenarios to `EditLogicalBackupConfigSection.test.tsx`, moving the "without remounting" scenario to the settings screen per decision 4, and record every move in `coverage.md`. Verify with `pnpm --dir frontend test src/pages/databases/ui/backup-config src/pages/databases/ui/DatabaseConfigComponent.test.tsx` and `rg -n 'isSaveToApi|isShowBackButton|isShowCancelButton' frontend/src/pages/databases/ui/backup-config` printing nothing.

## 5. Database screen

- [x] 5.1 Implement decisions 9 and 10 in `DatabaseComponent`. Verify with `pnpm --dir frontend test src/pages/databases/ui/DatabaseComponent.test.tsx src/pages/databases/ui/DatabasesComponent.test.tsx`.

## 6. Backup list

- [x] 6.1 Split `LogicalBackupsComponent` per decision 16 without changing its behavior, moving the download to the click, and keep the scenarios from before this change passing with accessible queries. The scenarios of task 1.2 stay failing until task 6.2. Verify with `pnpm --dir frontend typecheck`, and with `pnpm --dir frontend test src/pages/databases/ui/logical-backups` showing only those 1.2 scenarios failing.
- [x] 6.2 Apply decisions 10 to 13 to the list: inline errors with Retry for the first page and for the config, `message.error` for actions and for a next page, a silent refresh, `restoringBackup`, and `useSessionGuard` before the verification message and navigation. Replace the colors with theme roles, including `LogicalBackupsFiltersPanelComponent`. Add scenarios for a refused start, cancel, delete and download showing their reason once, a failed next page showing its message once with the rows kept and a later scroll requesting it again, a repeated click sending one request, and a single-use token for each download. Verify with `pnpm --dir frontend test src/pages/databases/ui/logical-backups`.

## 7. Restore dialog

- [x] 7.1 Implement decision 14: `RestoresComponent` uses `usePolledPagedList` with the complete listing, `resetKey: backup.id` and `refreshIntervalMs: 1_000`, and the hand-written polling and merge are deleted. Add `RestoreCard` and `RestoreErrorDialog` per decision 16, with one `restoreToCancelId`, inline errors with Retry for the first load, a silent refresh, `message.error` for a failed cancel and theme roles. Add scenarios for a failed first load sending no refresh and offering no target selection, then a Retry showing the restores and starting the refreshes; a failed cancel; every page of the complete listing shown; a started restore shown at once; and a start refused as not found keeping every entered value, in `RestoreTargetComposition.test.tsx`. Verify with `pnpm --dir frontend test src/pages/databases/ui/restores`.

## 8. Baseline, browser check, documents and handover

- [x] 8.1 Load the frontend-localization skill if any control needed a new key, and add that key to all six dictionaries. Remove all entries with `"removalMilestone": "6"` from `frontend/style-baseline.json`. Verify with `mise run frontend:lint`, `pnpm --dir frontend test src/shared/i18n` and `rg -c '"removalMilestone": "6"' frontend/style-baseline.json` printing nothing.
- [x] 8.2 Add `frontend/browser/scenarios/logicalBackupConfig.spec.ts`: open the settings of a logical database in Russian at desktop and 320 CSS pixels, edit the retention and advanced settings, save on a stateful fixture, and read the saved values again after selecting the database anew, with the form in one scroll region. Verify with `pnpm --dir frontend test:browser --project=ui`.
- [x] 8.3 Search the repository for `EditLogicalBackupConfigComponent`, `deleteConfimationId` and `showingRestoresBackupId`, and fix any mention outside `openspec/` and `temp/`. Update `docs/architecture.md` only if its "Database pages compose reusable editing" text stopped being true, and complete `coverage.md` with the final count and the reason for every moved or rewritten test. Verify with `rg -n 'EditLogicalBackupConfigComponent|deleteConfimationId|showingRestoresBackupId' frontend docs AGENTS.md` printing nothing and `openspec validate refactor-logical-backup-screens --strict`.
- [x] 8.4 Update item 6.4 of `temp/refactor/frontend-plan.md`: record the defects added by this change and the decisions that differ from the plan's text (no `key`, retention defaults for an empty policy, advanced values compared with the defaults, the controller rule reaching physical and verification, restores on `usePolledPagedList`), then mark 6.4 and milestone 6 done. Verify with `rg -n '^\s*- \[x\] \*\*6\.4\.|^- \[x\] \*\*6\. ' temp/refactor/frontend-plan.md` printing both lines.
- [x] 8.5 Run `mise run frontend:lint` and `pnpm --dir frontend build`, then the commands of the Change verification section, and record their results under "Recorded results".

## Recorded results

### Task 1.2 on the unchanged code

`pnpm --dir frontend test src/pages/databases/ui`: Test Files  6 failed | 10 passed (16) | Tests  34 failed | 248 passed (282)

The list keeps the names from that run. Five scenarios have other names now: `disables Save again when an edit is reverted` is `disables Save again when the retention period is changed and changed back`, `stays open when its backup leaves the list` is `keeps the same restores when its backup leaves the list and refreshes fail`, `offers a retry and a way to close when the logical section cannot load` is split into `shows the form after a successful retry` and `returns to the configuration view on Cancel`, `shows a retry instead of the empty text when the first page fails` is `shows a retry instead of the empty text and offers no backup start when the first page fails`, and `keeps the new database and reports nothing after another database is selected` is `keeps the new database and reports nothing for a $outcome answer after another database is selected`.

Every failure below is a reproduction and fails for its defect, by a stale `alert`, a reset draft, an enabled Cancel, an old database brought back, a missing Retry, a crash on the removed backup or a blank tab:

- CreateDatabaseComponent.test.tsx > the backup configuration step of the wizard > of a 'logical' database > keeps the input without a spinner when the wizard rerender
- CreateDatabaseComponent.test.tsx > the backup configuration step of the wizard > of a 'physical' database > keeps the input without a spinner when the wizard rerende
- CreateDatabaseComponent.test.tsx > the backup configuration step of the wizard > of a 'logical' database > offers Continue without a change after going back from the
- CreateDatabaseComponent.test.tsx > the backup configuration step of the wizard > keeps logical input after going back to the connection step and returning
- DatabaseComponent.test.tsx > DatabaseComponent backup configuration answers > of the 'logical' editor > keeps the new database and reports nothing after another data
- DatabaseComponent.test.tsx > DatabaseComponent backup configuration answers > of the 'logical' editor > shows no feedback for a 'refused' answer after a tab switch
- DatabaseComponent.test.tsx > DatabaseComponent backup configuration answers > of the 'physical' editor > shows no feedback for a 'refused' answer after a tab switch
- DatabaseComponent.test.tsx > DatabaseComponent backup configuration answers > of the 'logical' editor > shows no feedback for a 'refused' answer after a session chan
- DatabaseComponent.test.tsx > DatabaseComponent backup configuration answers > of the 'physical' editor > shows no feedback for a 'refused' answer after a session cha
- DatabaseComponent.test.tsx > DatabaseComponent backup configuration answers > of the 'physical' editor > keeps the new database and reports nothing after another dat
- DatabaseComponent.test.tsx > DatabaseComponent backup configuration answers > when the selected database lacks the open tab > shows the backups of a MySQL database s
- DatabaseComponent.test.tsx > DatabaseComponent backup configuration answers > when the selected database lacks the open tab > shows the backups of a database without
- DatabaseConfigComponent.test.tsx > DatabaseConfigComponent backup configuration > of the 'logical' editor > disables Cancel during a pending save
- DatabaseConfigComponent.test.tsx > DatabaseConfigComponent backup configuration > of the 'physical' editor > disables Cancel during a pending save
- DatabaseConfigComponent.test.tsx > DatabaseConfigComponent backup configuration > offers a retry and a way to close when the logical section cannot load
- DatabaseConfigComponent.test.tsx > DatabaseConfigComponent backup configuration > shows one error and one retry when the settings tab cannot load a logical config
- DatabaseConfigComponent.test.tsx > DatabaseConfigComponent backup configuration > shows the reason with a retry when the configuration view cannot load its storage
- DatabaseConfigComponent.test.tsx > DatabaseConfigComponent backup configuration > keeps entered GFS values through a switch to another policy and back
- DatabaseConfigComponent.test.tsx > DatabaseConfigComponent backup configuration > allows saving a GFS policy that keeps only hourly backups on an hourly schedule
- DatabaseConfigComponent.test.tsx > DatabaseConfigComponent backup configuration > keeps Save disabled for a GFS policy with no shown period and backups off
- DatabaseConfigComponent.test.tsx > DatabaseConfigComponent backup configuration > disables Save again when an edit is reverted
- DatabaseConfigComponent.test.tsx > DatabaseConfigComponent backup configuration > keeps Save disabled after a policy round trip on a config with no GFS periods
- DatabaseConfigComponent.test.tsx > DatabaseConfigComponent backup configuration > opens the advanced settings for a config with retry turned off
- DatabaseConfigComponent.test.tsx > DatabaseConfigComponent backup configuration > removes the advanced settings toggle when backups are turned off
- DatabasesComponent.test.tsx > DatabasesComponent during the backup step of the wizard > keeps the entered values when the page becomes visible and the list refreshes
- DatabasesComponent.test.tsx > DatabasesComponent with a late rename > keeps the other selected database shown while the list shows the new name
- restores/RestoresComponent.test.tsx > RestoresComponent failures > shows no alert and keeps the restores when polls fail
- restores/RestoresComponent.test.tsx > RestoresComponent failures > shows a retry and no target selection when the first load fails
- logical-backups/LogicalBackupsComponent.test.tsx > LogicalBackupsComponent failures > shows no alert and keeps the rows when polls fail
- logical-backups/LogicalBackupsComponent.test.tsx > LogicalBackupsComponent failures > shows a retry instead of the empty text when the first page fails
- logical-backups/LogicalBackupsComponent.test.tsx > LogicalBackupsComponent failures > shows the reason and not disabled backups when the config cannot load
- logical-backups/LogicalBackupsComponent.test.tsx > LogicalBackupsComponent failures > with the restore dialog open > stays open when its backup leaves the list
- logical-backups/LogicalBackupsComponent.test.tsx > LogicalBackupsComponent failures > with the restore dialog open > keeps the values entered in the target form when
- logical-backups/LogicalBackupsComponent.test.tsx > LogicalBackupsComponent failures > does not navigate when a verification start answers after the list left

Every guard and every older test passes. The two "keeps storage creation open when the wizard rerenders" scenarios did not fail. The load effect of each editor sets its loading state and its draft in one batch in the wizard, so the wizard shows no spinner and closes no dialog on a rerender; it only resets the draft. They stay as guards, no separate fix exists for them, and `proposal.md` and `design.md` now say this.

### Task 8.5

- `mise run frontend:lint`: passed, including the architecture, style, type and API contract checks.
- `pnpm --dir frontend build`: passed.
- `pnpm --dir frontend test`: 154 files, 1756 tests passed.
- `pnpm --dir frontend test:browser --project=ui`: 90 passed, including both widths of `logicalBackupConfig.spec.ts`.

## Change verification

```sh
pnpm --dir frontend test
pnpm --dir frontend test:browser --project=ui
```
