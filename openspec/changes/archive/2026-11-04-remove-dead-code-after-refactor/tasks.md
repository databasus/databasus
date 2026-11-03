## 1. Backend contracts

- [x] 1.1 Confirm through LSP references that no other module calls `GetLogicalBackupConfig`, `LockAndGetLogicalBackupConfig`, `GetPhysicalBackupConfig`, `ListEnabledLogicalBackupConfigs`, `ListEnabledPhysicalBackupConfigs` and `SaveBackupConfig` on the `backups` contract, then remove them from `backend/internal/modules/backups/module.go` and their forwarding methods from `internal/module/module.go`, keeping the `config` service methods the module uses internally. Remove aliases only these methods needed. Verify with `go build ./...` and `go vet ./internal/modules/backups/...` from `backend/`.
- [x] 1.2 Remove `CanInviteUsers` from the `users` contract and its forwarding method, and reword the comment above it so it describes only `CanCreateWorkspaces`. Verify with `go build ./...` and `go vet ./internal/modules/users/...` from `backend/`.
- [x] 1.3 Remove `DeleteDatabase` from the `databases` contract and its forwarding method, keeping the service method the handler calls. Verify with `go build ./...` and `go vet ./internal/modules/databases/...` from `backend/`.

## 2. Backend dead code

- [x] 2.1 Confirm through LSP references, then delete `GetVersionEnum`, `gfs.FormatTiers`, `chains.GetChainSpan`, both `HasPrivilege` methods in `connection_settings.go` and `DefaultForcedRotationInterval`, along with imports and private helpers this leaves unused. Verify with `go vet ./...` from `backend/`.
- [x] 2.2 Delete `apptest.WithFileTimings`, the `fileTimings` setting and the branch in `apptest.go` that reads it. Verify with `go vet ./internal/app/...` from `backend/`.
- [x] 2.3 Delete `mailpittest.Client.Clear` and the `FirstLSN` method of `fixtureSegmentNo` in `streamstatistics/catalog_seed_test.go`. Keep the `fixtureSegmentNo` type and the identically named `catalog` package helper, which are in use. Verify with `go vet ./internal/lib/mailpittest/... ./internal/modules/backups/physicalapitests/streamstatistics/...` from `backend/`.
- [x] 2.4 Delete `ftpChunkSize` in `storages/internal/providers/ftp/ftp_server.go` and `noRowLock` in `backups/internal/config/config_repository.go`. Leave the `noRowLock` constants of `databases` and `storages`. Verify with `go vet ./internal/modules/storages/... ./internal/modules/backups/internal/config/...` from `backend/`.
- [x] 2.5 Delete `BackupErrorCanceledByDbRemoval`. Leave the applied migration untouched. Verify with `go test ./internal/modules/backups/internal/physical/catalog/...` from `backend/`.

## 3. Code only tests call

- [x] 3.1 Delete `ParseUniqueNames` and `FormatUniqueNames`. Rewrite the splitting, trimming and empty-input cases in `namelist_test.go` to call `NormalizeUniqueNames`, and delete the case that tests only the formatting. Verify with `go test ./internal/lib/namelist/...` from `backend/`.
- [x] 3.2 Delete `CancellationRegistry.WaitUntilEmpty`, the `isEmpty` channel and its tests. Verify with `go test ./internal/lib/background/...` from `backend/`.
- [x] 3.3 Delete `Clear` from `cache.Store` and `MemoryStore`, from the three test `failingStore` types in `lib/cache/json_store_test.go`, `lib/ratelimit/memory_counter_test.go` and `modules/backups/internal/download/user_stream_test.go`, and remove the clearing step from `cachetest/contract.go`. Verify with `go test ./internal/lib/cache/... ./internal/lib/ratelimit/... ./internal/modules/backups/internal/download/...` from `backend/`, and with `rg -n 'Clear\(context\.Context\)' backend/internal` returning nothing.
- [x] 3.4 Delete `CountByReferences` and `FindByReference` from `pending_deletion_repository.go`, along with private helpers this leaves unused. Change the assertion at `store_test.go:343` to list through `ListByStorage` and expect one row, and inline the query into `findPending` in `fixture_test.go`. Verify with `go test ./internal/modules/storages/internal/files/...` from `backend/`.
- [x] 3.5 Delete `mysqlfamily.HasPrivilege` and its own test, and change the two absence checks in `privileges_test.go` to `assert.NotContains` on `EffectivePrivileges`. Verify with `go test ./internal/engines/mysqlfamily/...` from `backend/`.

## 4. Literals and stale references

- [x] 4.1 Bind `StatusFailed`, `StatusCanceled` and `StatusInProgress` as query parameters in `restore_repository.go` instead of the literals in `fail`, `cancel`, `failAllInProgress` and `updateInProgress`. Verify with `go test ./internal/modules/backups` from `backend/`, whose `restore_outcome_test.go`, `restore_cancel_test.go` and `restore_startup_test.go` exercise these queries.
- [x] 4.2 Switch on `backup.Type` with `catalog.BackupTypeFull`, `BackupTypeIncremental` and `BackupTypeWal` in `physical_backup_list_row.go`. Verify with `go test ./internal/modules/backups/physicalapitests/catalog/...` from `backend/`.
- [x] 4.3 Remove the `util/encryption` reference in `chunked_manifest.go:21` and `issue #721` in `archiveownerroles.go:22`, keeping the constraints those comments explain. Verify with `rg -n 'util/encryption|issue #721' backend/internal` returning nothing.
- [x] 4.4 Remove `cmd/cleanup_test_db` from `backend/.air.toml`, `/cleanup_test_db` from `backend/.gitignore`, and `backend/cleanup_test_db.exe` and `victoria-logs-data/` from the root `.gitignore`. Verify with `rg -n 'cleanup_test_db|victoria-logs-data' backend/.air.toml backend/.gitignore .gitignore` returning nothing.

## 5. Repository services and files

- [x] 5.1 Delete `backend-victoria-logs`, `test-physical-postgres-17` and `test-physical-postgres-18` from `docker-compose.yml`, move the replication comment to `wal-backup-postgres-18`, reword its mention of `test-physical-postgres-18`, drop both services from `test_fixtures` in `scripts/dev.sh`, and drop `TEST_PHYSICAL_POSTGRES_17_PORT` and `TEST_PHYSICAL_POSTGRES_18_PORT` from `.env.example`. Verify with `docker compose config --quiet`, `shellcheck scripts/dev.sh`, and `rg -n 'test-physical-postgres|TEST_PHYSICAL|victoria-logs' docker-compose.yml scripts .env.example` returning nothing. The interrupt behavior of `mise run dev` does not change and stays without a script test.
- [x] 5.2 Delete `backend/ui/readme.md`. Verify that `rg -n 'ui/readme' . -g '!openspec/changes/**'` returns nothing.
- [x] 5.3 Change `scripts/test.sh` to `docker compose up -d backend-db-test`. In `.github/workflows/ci-release.yml` make the same change in `e2e-verification-agent-podman` and `test-backend`, and remove their waits for `backend-db`. Add assertions to `scripts/test_sh_test.py` that pin these commands, and drop ports `5007` and `5008` from its port check. Verify with `python3 -m unittest discover -s scripts -p '*_test.py'` and `mise run lint`.

## 6. Frontend and website

- [x] 6.1 Confirm through the TypeScript server, then remove from all six dictionaries `common.actions.add`, `common.actions.confirm`, `common.actions.download`, `common.actions.refresh`, `common.states.disabled`, `app.workspaceSelection.createShort`, `app.workspaceSelection.placeholder`, `app.workspaceSelection.search`, `app.workspaceSelection.notFound`, `app.auth.accountsCheckFailed`, `notifiers.create.title`, `notifiers.teams.showFullUrl` and `notifiers.teams.hideFullUrl`. Keep any key the check shows in use and report it. Verify with `pnpm --dir frontend typecheck` and `pnpm --dir frontend test`.
- [x] 6.2 Delete `frontend/public/favicon.svg` and the icons `icons/warn-red.svg`, `icons/menu/arrow-down-gray.svg`, `icons/menu/users-gray.svg` and `icons/menu/users-white.svg`, keeping `favicon.ico`, which browsers request by default. Verify with `pnpm --dir frontend test` and `pnpm --dir frontend build`.
- [x] 6.3 Remove `@vitest/coverage-v8` from `frontend/package.json` and update `frontend/pnpm-lock.yaml` through `pnpm --dir frontend remove @vitest/coverage-v8`. Verify with `pnpm --dir frontend test`.
- [x] 6.4 Delete the 13 fonts in `website/public/font/` that `website/app/globals.css` does not name, keeping `Jost-Regular`, `Jost-Medium`, `Jost-SemiBold`, `Jost-Bold` and `Jost-ExtraBold`. Verify with `pnpm --dir website build`.

## 7. Final checks

- [x] 7.1 Search the repository outside `openspec/changes` for every removed symbol, service and file name and bring remaining mentions in line. Run `mise run format` and verify with `mise run lint` and `go run golang.org/x/tools/cmd/deadcode@latest -test ./...` from `backend/`, whose report must name none of the removed functions.

## Change verification

```sh
mise run test
```
