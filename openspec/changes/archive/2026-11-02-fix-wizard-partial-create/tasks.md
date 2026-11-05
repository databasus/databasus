Each task starts by adding its tests on the unchanged code, with responses declared on the real screen, and by seeing them fail for the reason the task names. A test that already holds on the unchanged code is noted as a guard.

## 1. Create once and retry the rest

- [x] 1.1 In `frontend/src/pages/databases/ui/CreateDatabaseComponent.test.tsx`, add tests where a refused configuration followed by Complete sends one create request and sends the configuration again, where a failed backup start followed by Complete sends only that request, where a successful first path sends each request once and calls `onClose` after the last one, and where changed notifiers reach the created database through an update with no second create. In `frontend/src/pages/databases/model/startFirstBackup.test.ts`, add tests where disabled backups on the logical and the physical path start no backup, because the wizard has no switch that a screen test could turn off (see Risks in `design.md`). In `CreateDatabaseComponent`, hold `createdDatabase` and `isBackupConfigAcknowledged`, and complete through one function that follows decision 1 of `design.md`. Verification: `pnpm --dir frontend test src/pages/databases/ui/CreateDatabaseComponent.test.tsx src/pages/databases/model/startFirstBackup.test.ts`.
- [x] 1.2 Add tests where the wizard unmounts while its create or save request is pending: no later request follows, no error appears and `onClose` is not called. Another test opens a new wizard while the save request is pending and then releases the response: no later request is sent, no feedback appears, and the new wizard stays on its first step with an empty draft. Guard the chain with `useSessionGuard`. Verification: `pnpm --dir frontend test src/pages/databases/ui/CreateDatabaseComponent.test.tsx`.

## 2. List and selection

- [x] 2.1 In `frontend/src/pages/databases/ui/DatabasesComponent.test.tsx`, add tests where the created database is not selected while the wizard is open, is selected after the wizard finishes with the screen showing the saved configuration and no "scheduled backups disabled" text, and is listed and selected after the user closes the wizard following a failure. The backup configuration list response holds the saved storage only after the configuration save, and the created database's card shows that storage after the wizard finishes. Add a test that closes the wizard during a pending create, lets it answer, opens and cancels a new wizard, and asserts that the selection is unchanged and that the next periodic refresh lists the database. Make `onCreated` only remember the database as decision 3 describes, and route `onClose` and the modal's cancel through one `closeWizardAndAcceptCreatedDatabase`. Verification: `pnpm --dir frontend test src/pages/databases/ui/CreateDatabaseComponent.test.tsx src/pages/databases/ui/DatabasesComponent.test.tsx`.

## 3. Ways back and messages

- [x] 3.1 Add tests for the logical and the physical path: after a refused configuration, the backup configuration step offers no way to the connection step, a changed configuration is saved for the created database, and after a failed backup start the notifier step offers no way to the configuration step. Add a test where the server refuses the creation and every earlier step stays reachable. Make `onBack` of `CreateLogicalBackupConfigStep` optional, pass `isShowBackButton={false}` to the physical editor, and hide the notifier step's Back after acknowledgment. Verification: `pnpm --dir frontend test src/pages/databases/ui/backup-config/CreateLogicalBackupConfigStep.test.tsx src/pages/databases/ui/CreateDatabaseComponent.test.tsx`.
- [x] 3.2 Add tests where a failure after creation shows the message with the reason on the notifier step and the kept values, the message stays after moving to the configuration step and returning, a retry shows the finishing text and not the creating text, and a refused creation shows the server's reason and mentions no created database. Show the stored failure as an `Alert` on both steps, and add `databases.create.createdButNotFinished` and `databases.create.finishing` to all six dictionaries under the frontend-localization skill. Verification: `pnpm --dir frontend test src/pages/databases/ui/CreateDatabaseComponent.test.tsx` and `mise run frontend:lint`.

## 4. Physical backup type

- [x] 4.1 Add a test where the user continues from the physical configuration step, goes back to the connection step, chooses FULL and continues: the step shows the FULL defaults, and completing the wizard sends policy `FULL_BACKUPS`, `chainsCount` 0, no incremental schedule and WAL lag 0. Add a test where choosing another backup type and returning to the original one keeps the entered configuration. Record `acceptedPhysicalBackupType` and clear the configuration on Continue when the type differs. Verification: `pnpm --dir frontend test src/pages/databases/ui/CreateDatabaseComponent.test.tsx`.

## 5. Handover

- [x] 5.1 Mark this item done in `temp/refactor/frontend-plan.md`. Verification: `rg -n '\[x\] \*\*7\.1\.' temp/refactor/frontend-plan.md` finds the line.
- [x] 5.2 Format the changed files and run `mise run frontend:lint`, `pnpm --dir frontend build` and `pnpm --dir frontend test`. Verification: all three commands pass.

## Change verification

```sh
pnpm --dir frontend test src/pages/databases
```
