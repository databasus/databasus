## 1. Contract and handlers

- [x] 1.1 Describe the nine operations of design.md decisions 1 to 3, 6 and 7 in `backend/api/openapi.yaml` under the tag `backups`, following the `api-design` skill:
  - their schemas under the names of design.md decisions 1, 2, 5, 6 and 7: `PhysicalBackupId`, `PhysicalBackupListRowType`, `PhysicalBackupStatus`, `PhysicalBackupPage` with the description of `totalUsageMb`, `PhysicalBackupStart` with `PhysicalBackupStartKind`, the `PhysicalRestoreTokenCreate` `oneOf`, `PhysicalRestoreToken`, `WalStreamStatistics` and `WalStreamLive`;
  - the public marks with their reasons;
  - every status the access checks and the specs give, and the `default` problem.

  Add `no_extendable_chain`, `no_restorable_backup` and `target_time_unreachable` to `ProblemCode` and to `lib/apierror` with `409`, and their keys to `CONTRACT_ERROR_CODE_KEYS` and all six dictionaries in `frontend/src/shared/i18n/locales`. Run `mise run generate-api`. Implement the nine handlers in `internal/modules/backups/internal/httpapi`, one file per area, each mapping the request to one call of `physical.Service` and its result to the generated type:
  - `IssueRestoreToken` with a `RestoreTokenSubject` and the token's expiry, per design.md decision 2;
  - the streaming response type of decision 3, with the recovery script still read from the old controller's embed until task 3.1;
  - the enum conversions of decision 7.

  Pass the physical service to `httpapi.Deps`. Verify with `mise run backend:lint` exiting 0, `pnpm --dir frontend typecheck` exiting 0 and `mise run backend:test-package ./internal/modules/backups ./internal/lib/apierror` exiting 0.

## 2. Backups: list, start, cancel and delete

- [x] 2.1 Turn the errors of the list, the start, the cancel and the delete in `internal/physical` into the problems of design.md decision 4:
  - an outsider's not-found becomes the database's or the backup's;
  - a database that is not physical gets `400` after the access check;
  - `limit` outside 1..1000 gets `400` instead of the clamp;
  - a malformed cursor, `not_running` and `no_extendable_chain`.

  Remove the WAL segment branch of `DeleteBackup` and `chains.DeleteWalSegment` with their tests, per decision 8, including the case "keeps the next anchor and its WAL when an older WAL segment is deleted" in `physicalapitests/catalog/operations_test.go`. The old controller keeps compiling over the same sentinels. Verify with `mise run backend:test-package ./internal/modules/backups/internal/physical/... ./internal/modules/backups/physicalapitests/...` exiting 0 and `rg -n '\bDeleteWalSegment\b' backend` printing nothing.
- [x] 2.2 Move `physicalapitests/catalog` and `physicalapitests/backuping` from `CallOldAPI` to the generated client:
  - the access tests expect `403` and `404` instead of `400`;
  - add the API tests of the `physical-backups/operations` requirements on the list, the start (including the full-only backup type), the cancel and the delete, the `workspace-access`, `backup-list-paging` and `physical-backup-wal-ranges` deltas, and the cases the `testing` skill lists for each of these operations, per design.md decision 10;
  - switch `cmd/seed_dev/seed.go` to `startPhysicalBackup`.

  Delete the old start, cancel and delete routes, with their temporary methods in `module.go` and `internal/module/physical.go` and their aliases. Verify with `mise run backend:test-package ./internal/modules/backups/physicalapitests/catalog/... ./internal/modules/backups/physicalapitests/backuping/... ./cmd/...` exiting 0 and `rg -n 'physical/database/[^"]*/trigger|physical/backups/[^"]*/cancel|DELETE\("/backups/physical' backend/internal backend/cmd` printing nothing.

## 3. Restore tokens, restore sets and the recovery script

- [x] 3.1 Finish the restore side per design.md decisions 2 to 4:
  - `IssueRestoreToken` and `OpenRestoreStream` translate the `chains` sentinels into `no_restorable_backup` and `target_time_unreachable`, with the latest restorable LSN in the detail of a WAL gap;
  - an unknown, used or expired token is `401 unauthenticated`, and a database that is not physical gets `400`;
  - the recovery script and its embed move into `restorestream`;
  - preserve binary response streaming in the contract-checking client while validating statuses, media types and headers, and cover the change in `internal/lib/apiclient`.

  Move `physicalapitests/restoring` and the restore, recovery script and list helpers of `internal/app/apptest/physicaltest` to the generated client. The WAL gap scenario expects `409 target_time_unreachable`. Add the API tests of the `physical-backups/operations` requirements on tokens, restore sets and the recovery script, including the script fetched with and without a session with the same body and neither the instance's address nor the issued token in it, per design.md decision 10. Delete the old token, restore stream and recovery script routes with their temporary methods and aliases. Verify with `mise run backend:test-package ./internal/modules/backups/physicalapitests/restoring/... ./internal/modules/backups/internal/physical/... ./internal/lib/apiclient ./internal/lib/httpserver` exiting 0 and `mise run backend:test-group physical-1-of-1` exiting 0.

## 4. WAL stream statistics and current values

- [x] 4.1 Turn the errors of the statistics and of the current values into the problems of design.md decision 4: not found, not physical and an unknown period. Move `physicalapitests/streamstatistics`, `physicalapitests/walstreaming` and `internal/app/apptest/apptest_test.go` to the generated client, and add the API tests of the `physical-backups/wal-stream-monitoring` delta. Delete the old statistics, current values and list routes with their temporary methods and aliases, and remove the JSON tags of the internal structs per decision 7. Verify with `mise run backend:test-package ./internal/modules/backups/physicalapitests/streamstatistics/... ./internal/modules/backups/physicalapitests/walstreaming/... ./internal/app/...` exiting 0 and `rg -n '/api/v1/backups/physical/|"/backups/physical/' backend/internal backend/cmd --glob '!**/catalog/contract_test.go'` printing nothing.

## 5. Removing the old controller

- [x] 5.1 Delete `internal/features/backups/backups/controllers/physical` and its mounting:
  - its lines in `internal/app/routes.go` and its getter in `internal/app/getters_test.go`;
  - the remaining temporary physical methods and aliases of `module.go`, except `GetLatestCompletedPhysicalFullBackup` and `PhysicalFullBackup` for telemetry, with the comment rewritten to name only telemetry;
  - `CallOldPublicAPI`, `OpenOldPublicAPI` and `DownloadOldPublicAPI` in `internal/app/apptest/old_api.go`, and `internal/app/apptest/old_api_test.go`, which tests only `OpenOldPublicAPI`;
  - the old controller in the test router of `cmd/seed_dev/main_test.go`.

  Switch the current-release checks of `e2e/upgrade/current_release_physical_test.go` to the new list and restore token paths. Leave `previous_release_physical_test.go` on the old paths, per design.md decision 10. Add an API test to `physicalapitests/catalog` that calls the old list path `/api/v1/backups/physical/database/{id}/backups` and gets `404` with `code` `not_found`. Verify with `rg -n 'backups_controllers_physical|IssuePhysical|OpenPhysicalRestoreStream|GetWalStreamLiveState|func \(m \*Module\) ListPhysicalBackups' backend` printing nothing, `mise run backend:lint` exiting 0 and `mise run backend:test-package ./internal/modules/backups ./internal/modules/backups/physicalapitests/catalog/... ./internal/app/... ./internal/features/telemetry ./cmd/...` exiting 0, and `mise run backend:test-upgrade` exiting 0.

## 6. Frontend

- [x] 6.1 Rewrite `src/entity/backups/physical/api` on `getApiClient` with the generated paths: the list with `after` and `createdBefore`, the start with `AUTO`, `FULL` and `INCREMENTAL`, the cancel, the delete, one restore token call taking either variant, the statistics, the current values, and URL builders for the restore set and the recovery script. Replace the model files that mirror the contract with aliases of `ApiComponents`; keep the label key files and `PhysicalBackupsFilters`. Verify with `pnpm --dir frontend typecheck` exiting 0 and `rg -n 'backups/physical' frontend/src/entity` printing nothing.
- [x] 6.2 Move the features to the new client:
  - `PhysicalBackupsComponent.tsx`, `PhysicalBackupsFiltersPanelComponent.tsx` and `CreateDatabaseComponent.tsx` start backups with the new kinds and page with `after`;
  - `PhysicalRestoreComponent.tsx` builds both URLs from the entity's builders and chooses its hints by `code` (`already_running`, `target_time_unreachable`, `no_restorable_backup`) instead of by status;
  - the Monitoring tab reads the generated types.

  Update the Vitest tests and their fixtures, including `restoreCommands.test.ts`, and add a test for each restore token variant. Verify with `pnpm --dir frontend lint`, `pnpm --dir frontend typecheck`, `pnpm --dir frontend test` and `pnpm --dir frontend build` exiting 0, and `rg -n '/api/v1/backups/physical/|error\.status' frontend/src/features/backups/physical` printing nothing.

## 7. Documents

- [x] 7.1 In `docs/glossary.md`, add `createPhysicalRestoreToken` to the code column of "Restore token", as "Download token" names `createDownloadToken`, and add the term "Restore token subject" with `RestoreTokenSubject`: what a restore token opens, a database to a target time or one backup. Update the rewrite plan at `/home/rostislav/development/project/databasus/temp/refactor/plan.md`:
  - mark the 10d line;
  - strike the relaxation row about the old physical routes;
  - narrow the `backups_bridge` row to verification and telemetry;
  - add a relaxation row for the physical start and cancel that answer `202` without a body, per design.md decision 5, and one for the storage total on every page of the physical list, per decision 6, both to be settled in milestone 11.

  Search the repository for every mention of the old paths and the removed names, and bring documents and comments in line. Verify with `rg -n '/api/v1/backups/physical/|"/backups/physical/' docs AGENTS.md backend/AGENTS.md frontend/AGENTS.md .agents/skills backend/internal frontend/src --glob '!**/catalog/contract_test.go'` printing nothing, `rg -n 'createPhysicalRestoreToken|RestoreTokenSubject' docs/glossary.md` showing both rows, and `mise run lint` exiting 0.

## Change verification

```sh
mise run backend:test-group physical-api-1-of-1
mise run backend:test-group physical-1-of-1
mise run backend:test-package ./internal/modules/backups ./internal/modules/backups/internal/... ./internal/lib/apierror ./internal/lib/apiclient ./internal/lib/httpserver
mise run backend:test-package ./internal/app/... ./internal/features/telemetry ./cmd/...
pnpm --dir frontend test
```
