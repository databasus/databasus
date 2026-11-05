Each task that fixes a defect starts by adding its tests on the unchanged code, with responses declared on the real screen, and by seeing them fail for the reason the task names. A test that already holds on the unchanged code is noted as a guard.

## 1. Map coverage

- [x] 1.1 Create `coverage.md` in this change, mapping `pages/databases/ui/backup-config/EditPhysicalBackupConfigComponent.test.tsx`, `ShowPhysicalBackupConfigComponent.test.tsx`, `EditLogicalBackupConfigSection.test.tsx`, `pages/databases/ui/NotifierComposition.test.tsx` and the wizard and settings tests that touch them to their final owners, with the file and test counts of `pnpm --dir frontend test`. Verification: `coverage.md` lists each of these files and the counts.

## 2. Draft and view

- [x] 2.1 Add `PhysicalBackupConfigDraft`, `toPhysicalBackupConfigDraft`, `toPhysicalBackupConfigInputFromDraft`, `createDefaultPhysicalBackupConfigDraft`, `isPhysicalBackupConfigChanged` and `isPhysicalBackupConfigComplete` under `pages/databases/model/physical-backups`, with node tests for each retention policy, each full-backup policy and each backup type. Cover a `LAST_N` input with zero GFS periods, a GFS input with count 0, and a retained `walLagThresholdBytes`. Verification: `pnpm --dir frontend test src/pages/databases/model/physical-backups`.
- [x] 2.2 Add tests where a failed settings-tab load shows one error and one retry and no `alert`, and where a physical view's failed storage load shows the other values, the reason and a retry and no `alert`. Pass the loaded config to `ShowPhysicalBackupConfigComponent`, with an inline failure and retry for its storage, and replace its raw colors and its `style-baseline.json` entries. Verification: `pnpm --dir frontend test src/pages/databases/ui/backup-config/ShowPhysicalBackupConfigComponent.test.tsx src/pages/databases/ui/DatabaseConfigComponent.test.tsx` and `mise run frontend:lint`.

## 3. Containers

- [x] 3.1 Extract `EditBackupConfigSection` from `EditLogicalBackupConfigSection`, which becomes a wrapper with unchanged behavior. Verification: `pnpm --dir frontend test src/pages/databases/ui/backup-config/EditLogicalBackupConfigSection.test.tsx`, unchanged.
- [x] 3.2 Extract the controlled `PhysicalBackupConfigFields`, `PhysicalRetentionFields` and `PhysicalNotificationTriggerFields` from the editor, using semantic roles and named controls, and update the editor's `style-baseline.json` counts. Move the tests of `EditPhysicalBackupConfigComponent.test.tsx` that `coverage.md` maps to the fields. Verification: `pnpm --dir frontend test src/pages/databases/ui/backup-config` and `mise run frontend:lint`.
- [x] 3.3 Add tests for: a failed editor load offers a retry and Cancel; a refused save shows its reason inline; a reverted edit and a tried and abandoned policy disable Save; all-zero GFS periods with backups off disable Save; switching full-backup policy and back shows the entered values; and saving sends the exact request body for `CHAINS` and for `LAST_N`. Add a guard that a save keeps `walLagThresholdBytes`. Add `EditPhysicalBackupConfigSection`, use it in `DatabaseConfigComponent`, move the mapped section tests, and delete the post-save history request and `acceptedSwitch`. Verification: `pnpm --dir frontend test src/pages/databases/ui/backup-config src/pages/databases/ui/DatabaseConfigComponent.test.tsx`, and `rg -n 'initialOperation|acceptedSwitch' frontend/src/pages/databases/ui/backup-config` finds nothing.
- [x] 3.4 Add tests where Back from the wizard's physical step and returning shows the entered values, and where Back without continuing, another backup type and Continue shows the new type's defaults. Add `CreatePhysicalBackupConfigStep` with `onBack` and `onContinue` receiving the draft, have the wizard hold the draft, record the backup type on both, and build the input only for the request. Delete `EditPhysicalBackupConfigComponent`, its test file and its `style-baseline.json` entries, after moving the remaining mapped tests. Verification: `pnpm --dir frontend test src/pages/databases/ui/backup-config src/pages/databases/ui/CreateDatabaseComponent.test.tsx` and `mise run frontend:lint`.
- [x] 3.5 Split `EditDatabaseNotifiersComponent` into `DatabaseNotifiersFields`, `CreateDatabaseNotifiersStep` and `EditDatabaseNotifiersSection`, and delete `toUnsavedDatabase`. Verification: `pnpm --dir frontend test src/pages/databases/ui` passes, and `rg -n 'toUnsavedDatabase|isSaveToApi|isShowSaveOnlyForUnsaved|isShowBackButton|isShowCancelButton|saveButtonText' frontend/src` finds nothing.

## 4. Browser and handover

- [x] 4.1 Add `frontend/browser/scenarios/physicalBackupConfig.spec.ts`: on a mocked API, edit the retention in a long locale, switch the policy and back, save, assert the request's retention, and read the config again. Verification: `pnpm --dir frontend test:browser --project=ui`.
- [x] 4.2 Add rows to `docs/glossary.md` for the physical backup config draft, stating how it differs from `PhysicalBackupConfigInput`, and for the physical database backup type `PhysicalDatabaseBackupType`, stating how it differs from a list item's `PhysicalBackupType`. Verification: `rg -n 'PhysicalBackupConfigDraft|PhysicalDatabaseBackupType' docs/glossary.md` finds both.
- [x] 4.3 Update `docs/architecture.md` if "Database pages compose reusable editing" names anything removed here. Complete `coverage.md` with the final counts and each test changed beyond its imports. Mark this item done in `temp/refactor/frontend-plan.md`. Verification: `rg -n 'toUnsavedDatabase|EditPhysicalBackupConfigComponent|EditDatabaseNotifiersComponent' docs .agents frontend/README.md` finds nothing, `rg -n 'Final' openspec/changes/refactor-physical-backup-config/coverage.md` finds the counts, and `rg -n '\[x\] \*\*7\.4\.' temp/refactor/frontend-plan.md` finds the line.
- [x] 4.4 Format the changed files and run `mise run frontend:lint`, `pnpm --dir frontend build` and `pnpm --dir frontend test`. Verification: all three pass.

## Change verification

```sh
pnpm --dir frontend test
pnpm --dir frontend test:browser --project=ui
```
