## 1. Shared library and the providers

- [x] 1.1 Add `database.AfterCommit` to `internal/lib/database` per design.md decision 5: hooks kept by the outermost transaction, run in order after its commit with a context detached from the transaction, dropped on rollback, and refused outside a transaction. Write its tests: a hook runs after the commit, does not run after a rollback, runs once when registered in a nested transaction, and returns an error outside a transaction. Verify with `mise run backend:test-package ./internal/lib/database/...`.
- [x] 1.2 Add `internal/modules/databases/module.go` with the contract of design.md decision 1, `internal/userdatabase` with the aliased types and events, and `internal/providers` with `Provider`, `ConnectedDatabase`, `SettingsRepository` and the registry of decision 2. Add the five provider packages with their settings, validation moved from the old models, secret merge, encryption, `PrepareForCreate`, `CopyForNewDatabase`, conversion to the engine settings with decryption, the `ConnectedDatabase` over the engine source, and the row type and repository of `databases_<type>`. Write unit tests of each provider's validation and merge, which need no database. Verify with `mise run backend:test-package ./internal/modules/databases/internal/providers/...`.

## 2. The first migration and the services

- [x] 2.1 Add the first migration of design.md decision 10 with its repairs. Switch every reader of the old tables in the same task, so the old code keeps working on the new schema:
  - the `TableName` and the list fields of the old models in `features/databases/databases/*`, which read `text[]` instead of the CSV hooks;
  - the attachment queries of the old repository, onto `databases_notifiers`;
  - `apptest.InsertDatabase` and `apptest.AttachNotifierToDatabase` and `BeginNotifierAttachment`, onto the renamed tables;
  - the raw query of `internal/modules/notifiers/deletion_test.go:99` and of `internal/modules/storages/transfer_test.go:107` where it names a renamed table.

  Add a test in `backend/migrations` that applies the migrations before it and then inserts:
  - a provider row of another type and a logical PostgreSQL row without a database;
  - two MySQL rows of one database, a database without its provider row that nothing references, and one that a backup config references;
  - a database of an unknown type that nothing references;
  - a database without a workspace that has backups and one that has nothing;
  - every list column with commas, newlines, tabs, blanks and duplicates;
  - a physical database with completed full and incremental backups and committed WAL segments;
  - an encrypted secret in every secret column.

  It applies the rest and checks each repair, constraint, list conversion and the physical last backup time, that every secret column holds the same bytes, and, in a separate case that inserts a database of an unknown type that a backup references, that the migration stops with its ID. Verify with `mise run backend:test-package ./migrations/... ./internal/features/... ./internal/app/... ./internal/modules/notifiers/... ./internal/modules/storages/... ./internal/modules/workspaces/...` and `goose -dir backend/migrations up` and `down` on the dev database.
- [x] 2.2 Add `userdatabase.Service` with the parent and attachment repositories, the list with `NameCursor` and the notifier filter, create, update and copy with the connection outside the transaction and the notifier check after the insert, the events published in the transaction, and the status writes of decisions 3, 4 and 6. Add the `connection` and `limiteduser` services and the `module` package that implements `Module`. Build the module in `app.New`, install it into the bridge of decision 9 with `loadWithoutApp` for test builds, keep the old routes mounted, and subscribe `OnWorkspaceDeleting` and `OnNotifierTransferring` in `internal/app/wiring.go` in place of the old service's. Verify with `mise run backend:test-package ./internal/modules/databases/... ./internal/app/... ./internal/modules/workspaces/... ./internal/modules/notifiers/...`.
- [x] 2.3 Add the deletion of design.md decision 5. Rewrite the old removal listeners of logical backups, physical cancellation, physical backups and slot cleanup as `DatabaseDeleting` handlers that work in the transaction and register after-commit hooks, keeping the logical refusal while a backup is in progress as the Problem `in_use`, and subscribe them in `internal/app/wiring.go` in that order. Rewrite the creation, copy and backup type change listeners as handlers of the module's events that return errors. Delete the four listener interfaces, their setters and the unregistered `OnDatabaseCopied` of the physical config. Verify with `mise run backend:test-package ./internal/modules/databases/... ./internal/features/backups/... ./internal/features/healthcheck/... ./internal/features/verification/...`.

## 3. The old code

- [x] 3.1 Switch the old code per design.md decision 9:
  - backups, restores, health checks, verification and telemetry onto `module.go` and the typed getters;
  - the physical config without its association and `Preload`;
  - the restore request onto its own types with today's JSON names;
  - the logical and physical publication onto `RecordBackupSuccess` in the publishing transaction, per decision 6;
  - `fillPhysicalLastBackupTimes` and `GetLastBackupTimesByDatabaseIDs` deleted;
  - the old transfer onto `TransferDatabaseToWorkspace` and `ReplaceDatabaseNotifiers`;
  - `internal/app` without the old routes, the GORM schema entry of `Database` and the old getters.

  Rewrite the test helpers per decision 11 and update their callers. Delete the old controller, service, repository and models with their tests, every other file of `features/databases` except the bridge, and what is left of `features/sshtunnel`. Verify with `mise run backend:test-package ./internal/features/... ./internal/app/... ./internal/modules/...` and `rg -n 'features/databases/databases|features/sshtunnel|GetDatabaseService|GetLastBackupTimesByDatabaseIDs|AddDb[A-Za-z]*Listener|Preload\("PostgresqlPhysical' backend/internal backend/cmd backend/tools` printing nothing.
- [x] 3.2 Add the second migration of design.md decision 10, which drops each provider table's own `id` and makes `database_id` the primary key, and extend the migration test of task 2.1 to it. Verify with `mise run backend:test-package ./migrations/... ./internal/modules/databases/... ./internal/app/...` and `goose -dir backend/migrations up` and `down` on the dev database.

## 4. The operations

- [x] 4.1 Add `listDatabases`, `createDatabase`, `getDatabase`, `updateDatabase` and `copyDatabase` with the schemas of design.md decision 7 and the code `database_connection_failed` in `lib/apierror`, run `mise run generate-api`, and serve them from `httpapi` with the per-type conversion files and the access checks of decision 8. Add `apptest.CreateDatabase` and `apptest.App.HoldNotifierTransfer` of decision 11. Switch `cmd/seed_dev` to the new operations.

  Write API tests for these `databases` requirements:
  - "A saved database is returned without its secrets";
  - "Changing a database keeps the secrets the request leaves out", with the database inserted in the stored shape through `ExecuteSQL` and `EncryptSecret`, so the same test covers the secrets of "An upgrade keeps every database";
  - "A saved database keeps its type", "Invalid settings are refused without a change" and "Creating a database never changes another database";
  - "Databases are listed per workspace in name order", "A copy starts with no backup history", and "A database uses only notifiers of its own workspace" with the concurrent case through `HoldNotifierTransfer`.

  Also write tests for the `workspace-access` database scenarios except tests, limited users and deletion. Port the SSH tunnel, secret lifecycle, exclusion list and WAL summarization tests of `internal/features/databases/controller_test.go` as it stands before this change. Verify with `mise run backend:test-package ./internal/modules/databases/... ./internal/modules/workspaces/... ./internal/modules/notifiers/... ./cmd/seed_dev/...`.
- [x] 4.2 Add `deleteDatabase` per design.md decision 5. Write API tests for every scenario of the `databases` requirement "A database is deleted completely or not at all", including the refusal while a logical backup is in progress, a failing handler that leaves everything in place, a running WAL streamer cancelled with its slots dropped, and a WAL supervisor that starts after the deletion leaving no slot. Add tests for the `workspace-access` deletion scenarios and for `workspace-deletion` "A workspace with databases cannot be deleted" through the new operations. Verify with `mise run backend:test-package ./internal/modules/databases/... ./internal/modules/workspaces/... ./internal/features/backups/...`.
- [x] 4.3 Add `testDatabase`, `testDatabaseSettings`, `checkLimitedUser` and `createLimitedUser` per design.md decisions 7 and 8. Write API tests for every scenario of these `databases` requirements:
  - "A connection test reports its outcome and changes nothing";
  - "Unsaved settings can be tested before saving", the settings of another type than the named database included;
  - "Only a user who may change databases sends stored secrets elsewhere";
  - "A limited user can be created for a database".

  Also cover the modified `internal-postgresql-protection` requirements and the `workspace-access` scenarios of testing a database and of limited users. Verify with `mise run backend:test-package ./internal/modules/databases/...`.
- [x] 4.4 Write tests for the `databases` requirement "The last backup time shows the newest backup of any kind" next to the publishing code: a physical database whose full backup and WAL segment are published through the old physical backuper and the WAL uploader, and a logical backup published through the logical backuper that clears the error of a failed one. Each reads the last backup time and error back through the module. Add a test that `database-status-updates` still holds for `RecordBackupSuccess`, `RecordBackupFailure` and `SetHealthStatus`. Verify with `mise run backend:test-package ./internal/modules/databases/... ./internal/features/backups/backups/...`.

## 5. The upgrade test

- [x] 5.1 Make the previous release write a MySQL, a MariaDB and a MongoDB database with passwords and an exclusion list each, and give the PostgreSQL database a schema list with the schema of the seeded rows and table and table data exclusion lists that name an empty table the test creates, per design.md decision 14 and the `upgrade-test` delta. Make the current checks read them through `listDatabases`. Verify with `mise run backend:test-upgrade`.

## 6. The frontend

- [x] 6.1 Rewrite `entity/databases/api` on the generated client with `listAllDatabases`. Delete the hand-written models, move the label keys, the logo lookup, `isPostgresType` and the connection string parsers onto the generated types, and move the restore target types to `entity/restores`, per design.md decision 13. Add a unit test of `listAllDatabases` over two pages. Verify with `pnpm --dir frontend test src/entity/databases src/entity/restores` and `pnpm --dir frontend typecheck`.
- [x] 6.2 Update the screens of `features/databases` to the settings union, omitted secrets, the test result with its reason, `checkLimitedUser` and `createLimitedUser` shown only to editors, notifiers from `notifierIds`, and the list sorted with unavailable databases first. Update the database parts of `features/backups`, `features/healthcheck`, `features/restores`, `features/verification`, `features/notifiers` and `features/workspaces`. Verify with `pnpm --dir frontend typecheck` and `pnpm --dir frontend lint`.
- [x] 6.3 Add the text of `database_connection_failed` to the six dictionaries and delete the keys the removed screens used. Verify with `pnpm --dir frontend typecheck`, which fails on a missing translation.
- [x] 6.4 Write the screen tests on `mockApi` that design.md decision 13 lists. Verify with `pnpm --dir frontend test src/features/databases`.

## 7. Documents and checks

- [x] 7.1 Update the documents:
  - in `docs/architecture.md` "Engines and providers", state that an engine's sources offer small sets of methods, and that a consumer that serves several engines through one shape declares the interface it needs;
  - in `docs/architecture.md` "Data", state that an effect outside the database registers an after-commit hook with `database.AfterCommit`;
  - in the module table of `docs/architecture.md`, name the notifiers attached to a database under `databases`;
  - in `docs/glossary.md`, add "Limited user", "Connection settings" and, in the Code section, "After-commit hook", and extend "Provider" to database types, per design.md decision 12.

  Verify with `rg -n 'AfterCommit' docs/architecture.md` showing the line, `rg -n '^\| (Limited user|Connection settings|After-commit hook) \|' docs/glossary.md` showing three rows and `rg -n '^\| Provider \|.*database' docs/glossary.md` showing the row.
- [x] 7.2 Update the rewrite plan, which is git-ignored and lives only in the main checkout at `/home/rostislav/development/project/databasus/temp/refactor/plan.md`:
  - check `8b` and milestone 8;
  - add the relaxation rows of design.md decision 9;
  - cross out the row about `database_notifiers` and `DatabaseNotifier`, and the rows `engines-layer` added for the old engine models and `features/sshtunnel`.

  Verify with `rg -n '^- \[x\] \*\*8\.' /home/rostislav/development/project/databasus/temp/refactor/plan.md` showing the line and `rg -n 'databases_bridge' /home/rostislav/development/project/databasus/temp/refactor/plan.md` showing the row.
- [x] 7.3 Search the code and the documents for the old routes, tables and names and bring every mention in line. Verify with `rg -n 'databases/create|databases/update|test-connection-direct|should-suggest-readonly-user|create-readonly-user|create-replication-only-user|databases/notifier/|postgresql_logical_databases|postgresql_physical_databases|mysql_databases|mariadb_databases|mongodb_databases|database_notifiers|DatabaseNotifier\b' backend/internal backend/cmd backend/e2e backend/tools frontend/src docs AGENTS.md backend/AGENTS.md frontend/AGENTS.md --glob '!**/previous_release_*_test.go' --glob '!**/migrations/**'` printing nothing; the previous-release client of the upgrade test keeps the old shapes on purpose.
- [x] 7.4 Run `mise run lint` and fix what it reports. Verify that it exits with status 0.

## Change verification

```sh
mise run backend:test-package ./internal/lib/database/... ./migrations/... ./internal/app/... ./internal/modules/databases/... ./internal/modules/workspaces/... ./internal/modules/notifiers/... ./internal/modules/storages/... ./internal/features/... ./cmd/seed_dev/...
pnpm --dir frontend test src/entity/databases src/entity/restores src/features/databases src/features/backups
```
