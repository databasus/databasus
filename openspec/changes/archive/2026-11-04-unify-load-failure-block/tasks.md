## 1. Cursor collection

- [x] 1.1 Add API test scenarios for `restoreApi`, `backupConfigApi`, `workspaceMemberApi` and `verificationAgentApi`, following `entities/workspaces/api/workspaceApi.test.ts`, that assert every page is collected and that `limit`, the query and `after` are sent. Verify they pass on the current loops with `pnpm --dir frontend exec vitest run src/entities/restores src/entities/backup-configs/api src/entities/workspaces src/entities/verification-agents`
- [x] 1.2 Replace the cursor loops in those four facades with `collectCursorPages`, and verify with `pnpm --dir frontend exec vitest run src/shared/lib/collectCursorPages.test.ts src/entities src/pages/databases/ui/restores src/pages/workspace-settings src/pages/instance-settings`

## 2. Backup configuration load

- [x] 2.1 Rename `retryBackupConfigLoad` to `reloadBackupConfig`, make `DatabaseConfigComponent` use `useBackupConfigLoad` for its load and its reload after a save, and verify with `pnpm --dir frontend exec vitest run src/pages/databases/ui`

## 3. Load failure block

- [x] 3.1 Add `shared/ui/LoadFailure.tsx`, export it from `shared/ui/index.ts`, and add `LoadFailure.test.tsx` with the scenario "shows the translated error and retries on click". Verify with `pnpm --dir frontend exec vitest run src/shared/ui/LoadFailure.test.tsx`
- [x] 3.2 Use the block in `app/ui/ApplicationScreen`, `pages/auth/ui/AuthPage`, `pages/users/ui/UsersPage`, `pages/profile/ui/ProfilePage` and `pages/instance-settings/ui` (`InstanceSettingsPage`, `verification-agents/VerificationAgentsComponent`), with the profile and instance settings pages keeping the plain `Alert` when data is present and a save fails. Add the scenario "offers no retry when saving fails with the data loaded" to `ProfilePage.test.tsx` and `InstanceSettingsPage.test.tsx`, and to `UsersPage.test.tsx` a scenario that holds the reload with `mockApi.delayed`, asserts the retry button is gone and that one reload request is sent. Verify with `pnpm --dir frontend exec vitest run src/app src/pages/auth src/pages/users src/pages/profile src/pages/instance-settings`
- [x] 3.3 Change `AuditEventList` to one optional `failure` prop, use the block there and in `WorkspaceAuditLogsComponent`, `WorkspaceSettingsComponent` and `WorkspaceMembersComponent`, and update `UserAuditLogsSidebar` and `InstanceAuditLogs`. Add scenarios to `AuditEventList.test.tsx` for a failure with an empty list and with loaded events, and one through `UserAuditLogsSidebar` that a failed load retries on click. Verify with `pnpm --dir frontend exec vitest run src/entities/audit-logs src/pages/workspace-settings src/pages/users src/pages/instance-settings`
- [x] 3.4 Use the block in `pages/notifiers/ui` (`NotifiersComponent`, `NotifierComponent`, `NotifierTransferDialogComponent`) and `pages/storages/ui` (`StoragesComponent`, `StorageComponent`, `StorageTransferDialogComponent`). `NotifierComponent` and `StorageComponent` show the block only while the entity is absent and keep the plain `Alert` otherwise, each with a screen scenario that a failed operation on a loaded entity offers no retry. Verify with `pnpm --dir frontend exec vitest run src/pages/notifiers src/pages/storages`
- [x] 3.5 Use the block in the top level of `pages/databases/ui` (`DatabasesComponent`, `DatabaseComponent`, `DatabaseConfigComponent`, `DatabaseNotifiersFields`, `BackupStorageSelection`), `show/ShowDatabaseNotifiersComponent` and `restores/RestoresComponent`. Verify with `pnpm --dir frontend exec vitest run src/pages/databases/ui`
- [x] 3.6 Use the block in `pages/databases/ui/backup-config` (`BackupConfigNotice`, `EditBackupConfigSection`, `ShowLogicalBackupConfigComponent`, `ShowPhysicalBackupConfigComponent`), `logical-backups/LogicalBackupsComponent`, `physical-backups/PhysicalBackupsComponent`, `healthcheck` (`ShowHealthcheckConfigComponent`, `EditHealthcheckConfigComponent`, `HealthcheckAttemptsComponent`), `verification` (`VerificationsComponent`, `VerificationDetailDrawer`, `ShowVerificationConfigComponent`, `EditVerificationConfigComponent`) and `wal-stream-monitoring/WalStreamMonitoringComponent`. Verify with `pnpm --dir frontend exec vitest run src/pages/databases/ui`
- [x] 3.7 Confirm with `rg "common\.actions\.retry" frontend/src --glob '*.tsx' --glob '!*.test.*'` that only `LoadFailure` and `DatabaseTransferDialogComponent` still render this button text, confirm that every other state passed to the block is set only by its load, and drop imports the replacements left unused, verified by `mise run frontend:lint`

## 4. Corner of the card under the tabs

- [x] 4.1 Move the tab panels of `DatabaseComponent` into a new wrapper with `[&>:first-child]:rounded-tl-none`, give the logical backups, physical backups, verifications, monitoring, healthcheck and configuration cards plain `rounded-md`, and remove `isDirectlyUnderTab`, `isHealthcheckVisible` and `onVisibilityChange` with their use in `WalStreamMonitoringComponent.test.tsx` and `HealthcheckAttemptsComponent.test.tsx`. Rename the healthcheck scenario "repeats no request when the parent passes a new callback or the language changes" to cover only the language change and drop its rerender, leaving other assertions unchanged. Verify with `pnpm --dir frontend exec vitest run src/pages/databases/ui` and `pnpm --dir frontend typecheck`
- [x] 4.2 Add a scenario to `frontend/browser/scenarios/databaseResources.spec.ts` that reads the top-left radius of the backups card with the healthcheck enabled and disabled, and verify with `pnpm --dir frontend test:browser --project=ui`

## 5. Final checks

- [x] 5.1 Format only the changed files with `pnpm --dir frontend exec oxfmt <changed files>`, then verify with `mise run frontend:lint` and `pnpm --dir frontend build`

## Change verification

```sh
mise run test
pnpm --dir frontend test:browser --project=ui
```
