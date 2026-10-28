## Why

`physical-backups-module` moved the physical backups into the `backups` module, but their HTTP routes still bypass the contract. A gin controller in `internal/features/backups/backups/controllers/physical` serves `/api/v1/backups/physical/...` over ten temporary methods of `module.go` and about 25 type aliases. This has three consequences:

- **Scripts cannot rely on the routes.** They are not in `openapi.yaml`, so users' scripts see no documented paths, fields or errors for physical backups, and `apptest` checks none of their responses against the contract.
- **The answers break the API rules.**
  - A user outside the workspace, a viewer who tries a change and a database that does not exist all get `400`.
  - The body is `{"error": err.Error()}`, so internal error text reaches the client and the frontend cannot choose a message by `code`.
  - The physical services return plain Go errors, not `lib/apierror` problems.
- **The frontend keeps a second API layer.** It holds 43 hand-written model files and two clients built on `apiHelper`, and it decides restore hints by HTTP status.

This change is the last of milestone 10. The state, the jobs and the schema have already moved, so what remains is the transport, the frontend and the tests.

## What Changes

- **BREAKING: the physical operations move into the contract under new paths.** Nine operations under the tag `backups` replace the ten old routes:
  - `listPhysicalBackups`: `GET /physical-backups`, filtered by `databaseId`, `type`, `status` and `createdBefore`, paged by `limit` and `after`;
  - `startPhysicalBackup`: `POST /physical-backups`, answered with `202`;
  - `cancelPhysicalBackup`: `POST /physical-backups/{backupId}/cancel`, answered with `202`;
  - `deletePhysicalBackup`: `DELETE /physical-backups/{backupId}`;
  - `createPhysicalRestoreToken`: `POST /physical-restore-tokens`, one operation for both kinds of restore token;
  - `downloadPhysicalRestoreSet`: `GET /physical-restore-sets?token=`, public;
  - `getRecoveryScript`: `GET /physical-recovery-scripts`, public;
  - `getWalStream`: `GET /wal-streams/{databaseId}`, the current values of the WAL stream;
  - `getWalStreamStatistics`: `GET /wal-streams/{databaseId}/statistics`.

  The old paths, field names and status codes stop working. The list wraps rows in `items` instead of `backups`, takes `after` instead of `cursor` and `createdBefore` instead of `beforeDate`, and the start takes its kind in a `kind` field as `AUTO`, `FULL` or `INCREMENTAL`. A restore token response has `token` and `expiresAt` and no `url`.
- **BREAKING: errors follow the API rules.**
  - A user outside the workspace and a missing database or backup get `404 not_found`.
  - A viewer who starts, cancels or deletes a physical backup gets `403 permission_denied`.
  - A cancel of a backup that is no longer running gets `409 not_running`.
  - A second restore stream of the same user gets `409 already_running`.
  - A physical operation on a database without physical backups gets `400 invalid_request`.
  - Three new codes replace today's `409` without a code, `422` and `404`:
    - `no_extendable_chain`: an incremental backup is requested, but there is no chain to extend;
    - `no_restorable_backup`: there is no completed full backup to restore from, or the chosen backup cannot be restored;
    - `target_time_unreachable`: the target time is earlier than the first full backup, or a WAL gap lies before it.

  Every error has the `Problem` body. Internal error text no longer reaches the client.
- **BREAKING: a WAL segment can no longer be deleted through the API.** `DELETE` with the ID of a WAL segment, which today removes the segment and the WAL after it up to the next anchor, now gets `404`. The interface never offered it, and the `physical-backup-wal-ranges` spec already says WAL rows offer no actions.
- **The handlers live in the module.** The handlers live in `internal/modules/backups/internal/httpapi`. The recovery script moves next to the restore stream code it unpacks. The physical services return `apierror` problems.
- **Deleted:**
  - the old controller package;
  - nine of the ten temporary methods of `module.go` and their aliases;
  - the old routes in `internal/app/routes.go`;
  - the physical old-API helpers of `apptest` (`CallOldPublicAPI`, `OpenOldPublicAPI`, `DownloadOldPublicAPI`) and `old_api_test.go`, which tests only `OpenOldPublicAPI`;
  - the WAL segment branch of the deletion with `chains.DeleteWalSegment`.

  `GetLatestCompletedPhysicalFullBackup` stays for telemetry until milestone 13.
- **Frontend.**
  - `entity/backups/physical` calls the generated client.
  - Its model files become aliases of `ApiComponents`, except the label keys and the list filter.
  - The restore dialog builds the restore set URL and the recovery script URL from the contract's paths.
  - Restore hints and other errors are chosen by `code`.
  - The six dictionaries get the three new codes.
- **Tests.**
  - The API tests in `physicalapitests`, the end-to-end scenarios in `e2e/postgresqlphysical` and the `apptest` helpers in `physicaltest` call the operations through the generated client.
  - The access tests expect `403` and `404` instead of `400`.
  - New API tests cover every operation's invalid input, missing session, outsider, low role and state conflicts, and the refused WAL segment deletion.
  - `cmd/seed_dev` starts its physical backup through `startPhysicalBackup`.
  - The upgrade test reads the current release through the new operations. Its steps against the previous release keep the old paths, because that binary has no others.

## Capabilities

### New Capabilities

- `physical-backups/operations`: what the physical backup operations do and answer. Covers listing, starting, cancelling and deleting physical backups, issuing restore tokens, downloading a restore set, serving the recovery script, and the error codes of each.

### Modified Capabilities

- `workspace-access`: physical backups, restore tokens and WAL stream reads follow the workspace roles. Viewers read and issue restore tokens, members start, cancel and delete, and outsiders get `404`.
- `backup-list-paging`: the physical list rejects a malformed cursor with `400 invalid_request`, and its filters apply to every page, as the logical list's do.
- `physical-backup-wal-ranges`: the API refuses to delete a WAL segment, as the list already offers no action on a WAL row.
- `physical-backups/wal-stream-monitoring`: a refused request for WAL stream statistics or its current values gets the same answers as the other physical operations: `404` for an outsider or a missing database, and `401` without a session.

## Out of scope

- Every simplification of milestone 11: one owner of the backup status, one table for full and incremental backups, the scheduler as a pure decision, one health verdict, one deletion handler and the remembered codec.
- Changing what a delete of a running physical backup does. It still stops the backup and deletes it, unlike a logical delete, which answers `409`.
- Refusing a manual start for a physical database whose backup config was never saved. It is accepted and does nothing, as today.
- Moving `physicalapitests` into the module's root directory.
- Telemetry, verification and the system health check, which move in milestones 12 and 13.
- Any schema change.
- The recovery script's content and the layout of the restore set.

## Impact

- **Contract:** `backend/api/openapi.yaml` gains nine operations, their schemas and three codes in `ProblemCode`. The generated server, the test client and `frontend/src/shared/api/schema.gen.ts` are regenerated.
- **Backend, changed:**
  - `backend/internal/modules/backups/internal/httpapi`: the physical handlers;
  - `internal/physical` and its `chains` and `restorestream` subpackages: `apierror` problems, one entry point for both restore tokens, no WAL segment deletion;
  - `internal/module`, `module.go`, `internal/app/routes.go`;
  - `lib/apierror`: the three codes;
  - `cmd/seed_dev`, `e2e/upgrade`.
- **Backend, deleted:** `backend/internal/features/backups/backups/controllers/physical`, the physical old-API helpers in `internal/app/apptest/old_api.go` and their test `internal/app/apptest/old_api_test.go`.
- **Tests:** `internal/modules/backups/physicalapitests/...`, `internal/modules/backups/e2e/postgresqlphysical/...`, `internal/app/apptest/physicaltest`, `internal/app/apptest/apptest_test.go`, `cmd/seed_dev`; `internal/app/apptest/old_api_test.go` is deleted with the helper it tests.
- **Frontend:**
  - `src/entity/backups/physical`;
  - `src/features/backups/physical`;
  - `src/features/databases/ui/CreateDatabaseComponent.tsx`;
  - `src/shared/i18n` with its six dictionaries.
- **Users' scripts:** every script that calls `/api/v1/backups/physical/...` breaks and must move to the new paths; the release notes name the change. A recovery script downloaded before the upgrade still works, because it receives the full restore set URL as an argument. A copied restore command stops working anyway when its 15-minute token expires.
- **Running instances:** no schema or data change.
- **Documents:**
  - `docs/glossary.md`: the restore token entry names its operation, as the download token entry does, and a new entry names what a token opens, `RestoreTokenSubject`;
  - the rewrite plan `temp/refactor/plan.md`, which lives only in the main checkout: the 10d line, and the rows of the relaxation table for the old physical controller and for `backups_bridge`, and two new rows: the `202` without a body of the physical start and cancel, and the storage total on every page of the physical list.
- **Rules the change answers to:** the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
