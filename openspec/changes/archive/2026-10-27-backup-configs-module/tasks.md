## 1. Schedules

- [x] 1.1 Add `internal/lib/schedule` per design.md decision 10: `Schedule` without tags, `Validate`, `IsDue` and `GetApproxPeriod`, with the tests of `features/intervals/model_test.go` moved over. Reduce `features/intervals` to the tagged struct of the old verification config, with `ToSchedule` and delegating methods. Verify with `mise run backend:test-package ./internal/lib/schedule/... ./internal/features/verification/...` and `rg -n 'cron\.' backend/internal/features/intervals --glob '!*_test.go'` printing nothing.

## 2. The module and the migration

- [x] 2.1 Add `internal/modules/backups/module.go` with the config types, enums, methods and the two temporary events of design.md decisions 5 and 6, `Deps` and `New`. Add `internal/config` with the models per method, the defaults, the validation moved from `features/backups/config/logical/model.go` and `features/backups/config/physical/model.go`, and the type coercion moved from `features/backups/config/physical/model.go:131-210`. Add `internal/module` to wire it. Verify with `mise run backend:lint` exiting 0 for `internal/modules/backups`.
- [x] 2.2 Add the migration of design.md decision 3 with its repairs and constraints, and the module's repository over the new tables. Switch the direct SQL on the old tables in `internal/app/apptest/storages.go`, `internal/modules/databases/{copy,deletion}_test.go`, `internal/modules/storages/{transfer,deletion}_test.go` and `internal/modules/notifiers/transfer_test.go` to the new tables. The old config packages cannot read the new tables, so the old backup tests fail from this task until task 4.1 switches their callers, and the module tests that call the old config routes fail until task 5.4 switches them; this task's check runs only the migration test and the module. Add a test in `backend/migrations` that applies the migrations before it and inserts:
  - a database with both a logical and a physical row, once of each method;
  - rows with an unknown retention policy, retention period, schedule type, physical retention, full-backup policy and encryption;
  - notification strings with empty, duplicate and unknown entries;
  - a valid row of each method with every field set.

  It applies the rest and checks each repair, that the valid rows keep every value, and each new constraint. Verify with `mise run backend:test-package ./migrations/...`.

## 3. Services, events and transfer

- [x] 3.1 Add `GetPhysicalBackupTypes` to `databases/module.go` per design.md decision 4, reading the type column without decrypting any setting, with a test in `internal/modules/databases`. Add the config service per design.md decisions 4, 5 and 6: reads and list pages that return the saved config or the default and the coerced physical config without writing, the backup types read once per page through `GetPhysicalBackupTypes`, the save with the storage checks under `GetStorageBlockingTransfer`, `LogicalBackupConfigStorageChanging` in the transaction, `PhysicalBackupConfigSaved` after the commit with its error returned, the list by workspace or storage with a cursor, and the force-request writes. Add the handlers of design.md decision 8 and register them in `internal/app/wiring.go`, with the old logical backup service and the old physical code subscribed to the two events and the old physical code subscribed to `DatabaseBackupTypeChanged` for the WAL stop. Verify with `mise run backend:test-package ./internal/modules/backups/... ./internal/app/...` and, in `backend/`, `go test -race -tags testhelpers -count=1 -run 'Test_GetPhysicalBackupTypes' ./internal/modules/databases/...`.
- [x] 3.2 Add the transfer service of design.md decision 7, with the default config inserted for an unsaved config that switches to a target storage, and give `databases.TransferDatabaseToWorkspace` the actor for its audit event in `module.go` and `internal/userdatabase/transfer_service.go`. Verify with `mise run backend:test-package ./internal/modules/backups/...` and, in `backend/`, `go build ./...`.

## 4. The old code

- [x] 4.1 Turn `features/backups/config` into the bridge of design.md decision 11, with its test helpers behind `testhelpers` on top of `SaveBackupConfig`. Switch every old caller to the types and methods of `module.go`: the logical and physical schedulers, backupers and cleaners, the services, the use cases, the restorer, the WAL supervisor and statistics, the end-to-end test helpers and `cmd/seed_dev`. Delete both config packages, `storage_usage`, their controllers and their tests, the two config models in `internal/app/model_schemas.go`, and the getter cases of the deleted controllers and services in `internal/app/getters_test.go`. Verify with `mise run backend:test-package ./internal/features/... ./internal/app/... ./cmd/seed_dev/...` and `rg -n 'backups_config_logical|backups_config_physical|storage_usage|GetBackupConfigByDbId' backend --glob '*.go'` printing nothing.

## 5. The operations

- [x] 5.1 Add `listBackupConfigs`, `getBackupConfig` and `updateBackupConfig` with the schemas of design.md decision 9, run `mise run generate-api`, and serve them from `httpapi`. Write API tests through `apptest` for these `backup-configs` requirements:
  - "A database has one backup config of its method";
  - "A config that was never saved reads as the default";
  - "An invalid config is refused without a change";
  - "A config follows its database's physical backup type";
  - "Configs are listed per workspace or per storage";
  - "A copy of a database gets the copy of its logical config";
  - "Deleting a database deletes its config".

  Also cover every scenario of the `backup-storage-assignment` delta apart from the logical removal, which task 5.3 covers: the race with a storage transfer as design.md decision 12 describes it, and the storage object dropped from the body. Cover the backup config scenarios of the `workspace-access` delta, and the unfit physical config of an earlier release written with `ExecuteSQL`. Verify with `mise run backend:test-package ./internal/modules/backups/...`.
- [x] 5.2 Add `transferDatabase` at `POST /databases/{databaseId}/transfer` per design.md decision 9 and serve it. Write API tests for the `backup-configs` requirements "A database moves to another workspace in one step" and "Only a member of both workspaces moves a database": both storage options, the unsaved config switched to a target storage, notifiers moved with one shared notifier staying, replacing notifiers, the rollback after a refused replacing notifier, the shared storage refused with `in_use`, and the 404 and 403 refusals. Also cover the transfer scenarios of the `workspace-access` delta. Verify with `mise run backend:test-package ./internal/modules/backups/...`.
- [x] 5.3 Pin the reactions of the old backups to a config change through the API: a logical storage change deletes the logical backups in the save, is refused with `in_use` while a logical backup runs, and answers 500 with the old storage and the backups kept when a failing handler subscribed by the test refuses the removal, and the physical scenarios of `backup-storage-assignment` still hold through `PhysicalBackupConfigSaved`. Verify with `mise run backend:test-package ./internal/modules/backups/... ./internal/features/backups/...`.
- [x] 5.4 Switch the module tests that call the old config routes to `getBackupConfig`, `updateBackupConfig` and `transferDatabase` through the `apptest` client: `internal/modules/databases/copy_test.go`, `internal/modules/storages/transfer_test.go` and `internal/modules/notifiers/transfer_test.go`. Verify with `mise run backend:test-package ./internal/modules/databases/... ./internal/modules/storages/... ./internal/modules/notifiers/...` and `rg -n 'backup-configs/' backend/internal/modules --glob '*_test.go' --glob '!backend/internal/modules/backups/**'` printing nothing.

## 6. The upgrade test

- [x] 6.1 Make the previous release write the PostgreSQL database's backup config with every value different from the logical default, per design.md decision 12: backups on, encryption on, the local storage, a weekly schedule on Wednesday at 02:30, GFS retention with every period set, five retries and only `BACKUP_FAILED`. Make the current checks read it through `getBackupConfig` and compare every value, per the `upgrade-test` delta. Verify with `mise run backend:test-upgrade`.

## 7. The frontend

- [x] 7.1 Delete the config clients and config types of `entity/backups`, `BackupConfigStorage` and `toBackupConfigStorage`, and switch the config screens of `features/backups` to the generated client and types, per design.md decision 13. Verify with `pnpm --dir frontend typecheck` and `pnpm --dir frontend lint`.
- [x] 7.2 Switch the database list, card and creation of `features/databases`, the database transfer dialog and the storage transfer dialog of `features/storages` to `listBackupConfigs`, `updateBackupConfig` and `transferDatabase`, with one list request per workspace for the cards. Write screen tests on `mockApi` for the card's storage name from one list call, the storage transfer dialog's use check, and the database transfer request of each storage option. Verify with `pnpm --dir frontend test src/features/databases src/features/storages src/features/backups`.

## 8. Documents and checks

- [x] 8.1 In `docs/glossary.md`, change "Backup config" to one config per database of its method, and add "Backup method" and "Notification trigger" (`backups.NotificationTrigger`; `NotificationTrigger` in the API, the backup outcome a config sends a notification for). In `docs/architecture.md`, "Modules", add `PhysicalBackupConfigSaved` per design.md decision 6 as the one event published after a commit whose handler's error becomes the request's answer, with its reason and its removal when physical backups move into `backups`. Add to "Engines and providers" only what design.md decision 2 does not find there. Verify with `rg -n '^\| (Backup config|Backup method|Notification trigger) \|' docs/glossary.md` showing three rows and `rg -n 'PhysicalBackupConfigSaved' docs/architecture.md` showing the sentence.
- [x] 8.2 Update the rewrite plan, which is git-ignored and lives only in the main checkout at `/home/rostislav/development/project/databasus/temp/refactor/plan.md`:
  - check `9b`;
  - cross out the `databases_bridge` row's part about the transfer and the row about `BackupConfigStorage`'s config half, rewriting what stays;
  - add rows for the two temporary events of design.md decision 6, for the bridge `backups_bridge`, and for `features/intervals` serving the old verification config until milestone 12.

  Verify with `rg -n '9b|LogicalBackupConfigStorageChanging|PhysicalBackupConfigSaved|backups_bridge|features/intervals' /home/rostislav/development/project/databasus/temp/refactor/plan.md` showing the lines.
- [x] 8.3 Search the code and the documents for the old routes, tables and names and bring every mention in line. Verify with `rg -n 'backup-configs/save|backup-configs/database|backup-configs/physical|is-using|databases-count|logical_backup_configs|physical_backup_configs|\bBackupConfigStorage\b' backend frontend/src docs AGENTS.md backend/AGENTS.md frontend/AGENTS.md .agents/skills --glob '!backend/migrations/**' --glob '!backend/e2e/upgrade/previous_release_client_test.go'` printing nothing. The excluded file speaks the previous release's API on purpose.
- [x] 8.4 Run `mise run lint` and fix what it reports. Verify that it exits with status 0.

## Change verification

```sh
mise run backend:test-package ./internal/lib/schedule/... ./migrations/... ./internal/app/... ./internal/modules/backups/... ./internal/modules/databases/... ./internal/modules/storages/... ./internal/modules/notifiers/... ./internal/modules/workspaces/... ./internal/features/... ./cmd/seed_dev/...
pnpm --dir frontend test src/features/databases src/features/storages src/features/backups
```
