## Context

See proposal.md for the motivation. The current state that shapes the approach:

- **The old controller.** `internal/features/backups/backups/controllers/physical/controller.go` registers eight routes behind the session (lines 23-31) and two public ones (lines 35-36). It reaches the module through `backups_bridge.GetModule()` (`di.go`). It maps most errors to `400` with `err.Error()` (for example line 60) and builds a relative restore URL into the token response (`controller.go:191-194`). `internal/app/routes.go:60` and `:68` mount it.
- **The temporary seam.** `module.go:125-155` holds ten temporary methods, and `module.go:379-418` holds about 25 aliases of physical types and errors. `internal/module/physical.go` adapts them to `physical.Service`. Only `GetLatestCompletedPhysicalFullBackup` has a caller outside the controller: `features/telemetry/di.go:127`.
- **The physical service errors are plain Go errors.**
  - `internal/physical/errors.go:5-14`;
  - `chains/errors.go` (`ErrNoChainForRestore`, `ErrTargetBeforeEarliest`, `WalGapBeforeTargetError` with the latest restorable LSN);
  - `restorestream/errors.go` (`ErrInvalidToken`).
  - `authorizeDatabase` (`internal/physical/service.go:62-80`) returns the `workspaces.Authorize` error as it is. An outsider therefore gets the workspace's not-found problem, not the database's or the backup's.
  - `config.ErrNotPhysicalDatabase` is plain too (`internal/config/errors.go`).
- **The logical side already has the target shape.**
  - Handlers map one request to one service call (`internal/httpapi/logical_backup_handler.go`).
  - Problems live in `logical/errors.go`.
  - An outsider's not-found becomes the backup's not-found (`logical/service.go:312-315`).
  - The service checks `limit` itself (`logical/service.go:231-240`), because the generated server does not enforce the contract's bounds.
- **The streaming shapes.**
  - A restore stream is a `Write(io.Writer)` function with a `Close` (`internal/physical/restore_stream_service.go:16-20`), not a reader.
  - The token is spent before the user stream is taken (`restorestream/service.go:111-130`).
- **The list.** `internal/physical/backup_list_service.go:110-125` clamps `limit` to 1..1000 with a default of 50 instead of rejecting it. The frontend asks for 50 (`PhysicalBackupsComponent.tsx:42`).
- **Deletion of a WAL segment.** `DeleteBackup` also deletes by WAL segment ID (`internal/physical/backup_deletion_service.go:55-73`), the only caller of `chains.DeleteWalSegment` (`chains/service.go:235`).
- **A manual start** only sets a request time on the physical config (`internal/config/service.go:299-315`). The update reports whether a row changed, but the caller ignores it, so a start for a database without a saved physical config does nothing and still answers `202`.
- **The frontend** decides restore hints by HTTP status (`PhysicalRestoreComponent.tsx:42-49`) and builds the recovery script URL from the old path (`PhysicalRestoreComponent.tsx:163`).
- **Path rules.** `backend/api/vacuum.yaml:76-85` requires every resource segment to be a plural noun, and `docs/architecture.md`, "HTTP API", says a route without an operation answers 500.

Constraints come from `backend/AGENTS.md`: a handler holds no logic, a service returns `lib/apierror` problems and checks access first, and every error of a package sits in its `errors.go`. They also come from `frontend/AGENTS.md` (types only from `schema.gen.ts`, six dictionaries), from the `api-design` skill and from the `testing` skill.

## Goals / Non-Goals

**Goals:**

- Every physical route is a contract operation served by the module's strict server, with `Problem` errors and access answers like the logical side's.
- `module.go` keeps one temporary physical method, the one telemetry needs.
- Every physical test, helper and seed calls the generated client. Only the steps of the upgrade test that drive the previous release keep raw paths.

**Non-Goals:**

- No change to how a backup runs, how the catalog decides a status or how the scheduler chooses. The change is limited to the errors the service returns, the access mapping, the list's `limit` check, the single restore token entry point and the removed WAL segment deletion.
- No new read operation for one physical backup. The frontend reads backups through the list.

## Decisions

### 1. Paths follow the logical operations

The list and the start sit on `/physical-backups` with `databaseId` as a query parameter or a body field, as `/logical-backups` does. The cancel and the delete sit on `/physical-backups/{backupId}`, with a parameter component `PhysicalBackupId` of their own described as the ID of a full or incremental physical backup. The shared `BackupId` stays the logical backup's, as its description says. The WAL stream, which a database has at most one of, is addressed by its database like `/backup-configs/{databaseId}`: `GET /wal-streams/{databaseId}` returns its current values, and `GET /wal-streams/{databaseId}/statistics?period=` returns its statistics.

*Rejected:* `/databases/{databaseId}/physical-backups/...`, which the rewrite plan suggested. A script would then find logical and physical backups by two different schemes, and `api-design` lists workspace resources with a filter rather than under a parent path.

### 2. One restore token operation with a `oneOf` body

`POST /physical-restore-tokens` takes `PhysicalRestoreTokenCreate`, a `oneOf` with the discriminator `type`:

- `POINT_IN_TIME` with `databaseId` and an optional `targetTime`;
- `BACKUP` with `backupId`.

It answers `201` with `PhysicalRestoreToken {token, expiresAt}`. `physical.Service` gets one `IssueRestoreToken(ctx, actor, RestoreTokenSubject)`, where `RestoreTokenSubject` holds either the database with its target time or the backup. The name avoids "restore point", which PostgreSQL uses for `pg_create_restore_point`, and "restore target", which the glossary gives to the database a restore writes into; task 7.1 adds the term to `docs/glossary.md`. The handler only converts the variant, and the service keeps both access paths. The service resolves the restore set at issue as it does now, and `restorestream` reports the expiry it already applies.

*Rejected: two paths.* `POST /physical-restore-tokens` with `/physical-backups/{backupId}/restore-tokens` beside it gives one entity two shapes. Two operations on one path and method are impossible.

*Rejected: a handler that switches between two service methods.* The handler would then hold the choice, which `backend/AGENTS.md` keeps out of handlers.

### 3. The restore set and the recovery script

- **The restore set.** `GET /physical-restore-sets?token=` carries `x-access: public` and answers `application/x-tar` with `Content-Disposition`. The handler returns a response type of its own in `httpapi` that implements the generated response interface. Its visit method sets the headers, calls the stream's `Write` on the response writer and closes the stream when it returns. Every problem the service can answer happens in `OpenRestoreStream`, before the first byte: the token, the user stream and the restore set. An error during the copy is logged by the physical service and aborts the response with `http.ErrAbortHandler`. The HTTP recovery middleware lets that sentinel reach `net/http`, so it closes the connection without appending a problem body to the archive.
- **The recovery script.** `GET /physical-recovery-scripts` carries `x-access: public` and answers `text/x-shellscript`. The script and its `go:embed` move from the controller into `restorestream`. The tar layout that `restorestream/tar.go` writes and the script that unpacks it then change in one package.
- **The token response.** `url` is dropped, and the frontend builds both URLs from the contract's paths, as `getLogicalBackupFileUrl` does.

*Rejected: an `io.Pipe` with a goroutine*, which would fit the generated binary response that wants a reader. It adds a goroutine and an error handover for nothing, because the writer function already exists.

*Rejected: the singular `/physical-recovery-script` with an exception in `vacuum.yaml`.* It changes a rule for one path.

*Rejected: the script in `engines/postgresql/physical`.* The script knows the tar layout of the module's restore set, not an engine fact.

### 4. Errors become problems in the physical service

`internal/physical/errors.go` defines the problems the facade returns:

| Problem | Status and code |
|---|---|
| database or backup not found, outsider included | `404 not_found` |
| not a physical database (wraps the config's sentinel at the facade) | `400 invalid_request`, field `databaseId` |
| `limit` outside 1..1000 | `400 invalid_request`, field `limit` |
| malformed cursor | `400 invalid_request`, field `after` |
| unknown statistics period | `400 invalid_request`, field `period` |
| backup not running | `409 not_running` |
| no extendable chain | `409 no_extendable_chain` |
| no restorable backup | `409 no_restorable_backup` |
| target time unreachable | `409 target_time_unreachable`; the detail names the latest restorable LSN for a WAL gap |
| invalid restore token | `401 unauthenticated` |

The rules behind the table:

- `chains` keeps its sentinels, because retention and the restore stream use them internally. One function of the facade translates them where a restore set is resolved: at token issue and when the stream opens.
- `authorizeDatabase` turns the workspace's not-found into the database's not-found, or into the backup's not-found when the operation names a backup, as `logical/service.go:312-315` does.
- Every operation that names a database checks that its backup method is physical after the access check. That covers the list, the start, the statistics, the current values and the database restore token.
- `download.ErrUserStreamHeld` already answers `409 already_running`.
- The three new codes join `ProblemCode`, `lib/apierror` with `409`, `CONTRACT_ERROR_CODE_KEYS` and the six dictionaries.

*Rejected: `422` for an unreachable target.* The status table of `api-design` has no `422`, and the condition is a state of the catalog, which the table answers with `409`.

*Rejected: `404` for no restorable backup.* A client could not tell it from "no such database or no access".

*Rejected: one code for "no extendable chain" and "no restorable backup".* A chain can be broken for extension while it still restores, so the user needs different advice.

### 5. A start and a cancel answer `202` without a body

A start only records a request, and the scheduler creates the backup at its next pass. A cancel interrupts the running process, and the run records `CANCELED`. Neither has a resource to return in its in-progress state, which "HTTP API" in `docs/architecture.md` and the `api-design` skill ask for. The deviation is recorded in the rewrite plan's table of temporary relaxations, to be settled in milestone 11, when the catalog becomes the only owner of the status. The start takes `PhysicalBackupStart {databaseId, kind}`, where `kind` is `PhysicalBackupStartKind`: `AUTO`, `FULL` or `INCREMENTAL`. The field is `kind` rather than `type`, so it is not read as the config's backup type.

*Rejected: inserting the backup row at the start.* It changes the scheduler's ownership of the start, which is milestone 11's work.

*Rejected: returning the list row of the cancelled backup.* There is no read operation for one backup, and the row comes from a union query of the list.

### 6. The list keeps its semantics with the contract's names

`listPhysicalBackups` takes `databaseId` (required), `type` and `status` as arrays of `PhysicalBackupListRowType` (`FULL`, `INCREMENTAL`, `WAL`) and `PhysicalBackupStatus` (`IN_PROGRESS`, `COMPLETED`, `ERROR`, `CHAIN_BROKEN`, `CANCELED`), `createdBefore`, `limit` (1..1000, default 50) and `after`. It answers `PhysicalBackupPage {items, nextCursor, totalUsageMb}`. Today's clamp of `limit` becomes a `400`, as the logical list does. The upper bound stays at 1000 because the tests and the upgrade test read whole lists with it. `totalUsageMb` stays on the page, although the Pagination rule of the `api-design` skill offers no totals: the screen shows it above the list, and moving it to its own operation costs one more request and one more operation without a gain to the user. The operation's description says that the figure covers every backup of the database, not the page. The exception is recorded in the rewrite plan's table of temporary relaxations, to be settled in milestone 11 together with the one table for full and incremental backups. The row keeps today's fields. A field omitted today when empty becomes optional, and `hasGapBefore` becomes a required boolean. Sizes stay in megabytes, as `LogicalBackup` keeps them.

*Rejected: the logical bounds 1..100 with a default of 20.* The screen pages by 50, and tests would need extra pages for nothing.

### 7. The WAL stream schemas keep today's field names

`getWalStreamStatistics` answers `WalStreamStatistics`, whose `current` is `WalStreamCurrent`, the newest stored sample. `getWalStream` answers `WalStreamLive` with `isStreamerRunning` and a nullable `current` of `WalStreamLiveValues`, the current values the specs name. These schemas, `WalStreamSeries`, `WalStreamKeepingUp`, `WalStreamReason`, `WalStreamNotice` and their enums enter the contract with the JSON names the endpoint sends today. Those names already follow `camelCase` and the unit rule of the `wal-stream-monitoring` spec. Nullable pointers become `nullable` fields, and `omitempty` pointers in reasons and notices become optional fields. The handler converts the service's structs to the generated types, and an enum value with no contract value fails as an internal error, as `errUnknownStoredValue` does for configs. The JSON tags on the internal structs of `physical`, `catalog` and `streamhealth` that served the old controller are removed.

*Rejected: renaming fields while moving.* It would make the frontend migration a rewrite of the Monitoring tab's logic instead of a type swap.

### 8. The WAL segment branch of the deletion is removed

The user agreed to remove it. `DeleteBackup` deletes full and incremental backups only, an ID of a WAL segment falls through to the backup's not-found, and `chains.DeleteWalSegment` with its unit and API tests goes away. `deleteFull` and `deleteIncremental` keep stopping a running backup before they delete it, unlike the logical `409`, because milestone 10 moves the behavior as it is.

### 9. Old routes go away area by area

Each task that moves an area to the contract deletes the old routes, the temporary methods and the aliases of that area, and moves that area's tests to the client. The old controller therefore shrinks while the remaining old tests keep passing, and no task leaves two routes for one operation in use. The last task deletes the package, its getter in `internal/app/getters_test.go`, its mounting in `routes.go`, the comment in `module.go`, the physical old-API helpers of `apptest` and `internal/app/apptest/old_api_test.go`, whose only test exercises `OpenOldPublicAPI`. `CallOldAPI` itself stays, because `notifiers/transfer_test.go` uses it. The same task adds an API test that an old path answers `404 not_found`.

*Rejected: the contract first, then one large switch of every test.* It leaves a task whose verification would be the whole physical suite with every test file changed at once.

### 10. Tests

- **API tests in `physicalapitests`.** Every call moves from `CallOldAPI` with a path to the generated client, and the expected statuses follow the specs. `catalog/access_test.go:37-42` changes from `400` to `403` and `404`. New tests per operation cover what the `testing` skill lists that is missing today:
  - invalid input: a page size, a cursor, a period, and a database that is not physical;
  - a request without a session;
  - an outsider and a viewer;
  - the state conflicts: `not_running`, `no_extendable_chain`, `no_restorable_backup`, `target_time_unreachable` and `already_running` at issue and at download;
  - a deletion by WAL segment ID;
  - a used and an unknown token;
  - the recovery script fetched with and without a session, with the same body and neither the instance's address nor an issued token in it;
  - the full-only backup type for a start of the scheduler's choice;
  - an old path answering `404 not_found`, once the last old route is gone.
- **Covered by existing tests.** Two spec scenarios need no new physical test, because the behavior lives in shared code that every strict handler goes through:
  - an internal failure: the physical handlers return the service's error unchanged to the `lib/apierror` hooks, whose test "answers 500 without the error text and logs it under the request ID when a handler fails" pins the `500 internal` body without the error text;
  - the token in the request log: the access log of `lib/httpserver` records the path without the query, which its test "leaves the query string out of the record when the path has one" pins, and `restorestream` logs the database and user IDs, never the token.
- **Contract client streams.** The contract-checking client validates a binary response’s status, media type and headers without buffering its body. JSON responses, including errors of downloads, retain full body validation. This lets the tests hold a user stream open and download large restore sets without buffering them in the client.
- **End-to-end and helpers.** The end-to-end scenarios change only through `apptest/physicaltest`, whose list, token, download and recovery script helpers call the client. The WAL gap scenario expects `409 target_time_unreachable` instead of `422`.
- **Local test timeout.** The local shard runner uses the same 40-minute timeout as CI. A successful unsharded WAL streaming package run took about 24 minutes, which exceeds its former 15-minute timeout.
- **Shell lint after a move.** The shell lint task filters out deleted tracked paths before invoking ShellCheck, so moving the recovery script does not make verification fail on its old path.
- **Seed and upgrade.** `cmd/seed_dev` starts its backup through `startPhysicalBackup`, and its test mounts the module's routes instead of the old controller. The upgrade test's current-release checks call the list and the restore token through the new paths with its own small client, and its previous-release steps keep the old paths.

## Risks / Trade-offs

- [The upgrade test needs images for both releases] → `mise run backend:test-upgrade` builds the current image locally and tests it against the newest release. Its previous-release half does not change, and its current-release half uses the same operations the API tests cover.
- [A user's script on the old paths breaks on upgrade] → The proposal marks the change as breaking for the release notes, and an old path answers `404 not_found` rather than something misleading.
- [The `oneOf` token body needs care in the generated TypeScript] → The frontend builds the body through the generated union type, and a Vitest test checks both variants.
- [Translating `chains` sentinels in one facade function could miss a call site] → Both call sites, token issue and stream open, are covered by API tests that assert the codes.
- [A start for a database whose physical config was never saved still answers `202` and does nothing] → Kept on purpose as current behavior and named as out of scope in the proposal, so the change does not touch the creation flow of the frontend.

## Migration Plan

Nothing is migrated: the schema and stored data stay as they are. Deploying the release switches the API at once. The frontend in the same image calls the new paths, and users' scripts move to them following the release notes. Rolling back restores the old image and with it the old paths; no data written in between depends on the new ones.
