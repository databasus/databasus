## Context

Why the change is needed is in proposal.md, section Why. This document covers what shapes the approach. Paths are under `backend/` unless they start with another top-level directory. `FV` stands for `internal/features/verification`. The schema is the one `verification-schema` leaves: `verification_agents`, `verification_configs`, `verification_runs` and `verification_run_table_stats`, with `RESTRICT` to `databases` and `backups_logical`, and the handlers of `LogicalBackupsDeleting` and `DatabaseDeleting` in the old code.

**The old user routes.** Every route sits behind the session middleware (`internal/app/routes.go:63-71`). The config and runs controllers answer every error as `400` with the internal text; the agents controller answers `404` for a missing agent and `500` with the internal text for other failures. Only the `401` without a session and the admin check have a stable code.
- Agents (`FV/agents/controller.go:18-29`): `GET /verification/agents/availability` returns `{count, hasAgents}` for any signed-in user, and `count` is every agent without `deleted_at` (`FV/agents/repository.go:35-44`). `POST`, `GET`, `POST /:id/rotate-token` and `DELETE /:id` need an instance admin through `users_bridge.RequireAdmin()`. A token is a UUID without dashes, stored as its SHA-256 hex (`FV/agents/service.go:203-210`), and returned only by create and rotate. The list is newest first by `created_at` and hides deleted agents (`FV/agents/repository.go:24-33`). A delete sets `deleted_at` and does nothing else; the reaper requeues the agent's running verifications on its next pass (`FV/runs/scheduler.go:200-213`).
- Config (`FV/config/controller.go:16-19`): `GET` and `PUT /verification-config/:databaseId`. The read creates the default row when none exists (`FV/config/service.go:148-162`): off, after each backup, weekly on weekday 0 at `04:00`, only `VERIFICATION_FAILED`. The save needs the right to edit content and validates the schedule only when automatic verification is on with the interval type (`FV/config/model.go:65-86`). Saving cancels nothing; the scheduler's third pass cancels the automatic runs of a disabled schedule.
- Runs (`FV/runs/controller.go:16-21`): `POST /verifications/enqueue` needs the right to edit content, a completed backup and no other pending or running manual run of the database, and answers `200` (`FV/runs/service.go:42-117`). It checks neither for an agent nor for an engine that can be verified. `POST /verifications/:id/cancel` takes a pending or running run to `CANCELED` and answers `204` (`:126-170`). `GET /verifications/:id` includes the table statistics (`FV/runs/repository.go:26-42`). `GET /verifications/by-database/:databaseId` pages with `limit` (default 10, no maximum) and `offset`, newest first, without statistics, and returns a total (`FV/runs/service.go:172-214`).

**Background work.** `FV/runs/scheduler.go:41-75` is a ticker loop of its own with a 15-second tick and an `atomic.Bool` guard, started from `internal/app/background_loops.go:18`. Each tick runs three passes in order:
1. create scheduled runs for configs with the interval type, when the schedule is due since the last finished run, for the newest completed backup (`:77-172`);
2. reap: requeue or fail running runs whose agent is gone or silent for 5 minutes, and fail pending runs older than 24 hours (`:174-244`);
3. cancel the automatic runs of disabled schedules (`:247-274`).

The thresholds are package variables that tests shrink: `maxPendingDuration` (`:24`), `StaleAgentThreshold` (`FV/runs/constants.go:20`), which the system health check also reads, and `agentJobReclaimGrace` (`:31`). The old code takes the time from `time.Now()`.

**Retries and outcomes.** `RequeueOrFail` (`FV/runs/service.go:437-464`) puts a run back to pending with one more attempt and a new `created_at` when the reason is on the agent's side and attempts remain (`MaxAgentSideAttempts = 3`). Otherwise it fails the run, sets the backup's verification status through `backups.Module.SetLogicalBackupVerificationStatus` and notifies. A successful report does the same with the completed status (`:404-435`). Notifications go to the database's notifiers through `notifiers.Module.Notify` when the config chooses that outcome (`:672-729`). Cancelled runs neither notify nor change the backup's status.

**Events.** `OnLogicalBackupCompleted` cancels the pending automatic runs of the database and adds an after-backup run when the config is on with the after-backup type (`FV/runs/service.go:499-589`). `OnDatabaseCopied` copies the config (`FV/config/service.go:135-146`). The heartbeat listener requeues a running run that a live agent stopped reporting after 90 seconds, and returns the IDs the agent should abort (`FV/runs/service.go:344-377,473-497`). All three are wired in `internal/app/wiring.go:18,52,61`.

**Access and calls to other modules.** The old code reaches `workspaces.Authorize` through `workspaces_bridge.Access.IsActorAllowed` and databases through `databases_bridge.Access.GetDatabaseByID`. It calls `backups.Module.GetLogicalBackups`, `GetLatestCompletedLogicalBackup`, `OpenLogicalBackupFile` and `SetLogicalBackupVerificationStatus`, which `backups/module.go:155-174` describes as serving "the old verification runs", and `databases.Module.GetPostgresqlLogicalConnectionSettings` for the engine version of a claim (`FV/runs/assigned_database.go:25-49`). Audit events: agent created, rotated and deleted without a workspace, and config saved, verification started and cancelled with the workspace.

**The agent protocol.** `RequireAgentAuth` (`FV/agents/middleware.go:26-93`) parses the agent ID, applies a rate limit of 10 requests a second per agent, reads the bearer token and compares its hash in constant time. The heartbeat (`POST /agent/verification/:agentId/heartbeat`), the claim, the backup stream and the report (`/agent/verifications/:agentId/...`) call `AgentService.Heartbeat`, `ClaimVerification`, `GetBackupFile` and `SubmitReport`. "Verification gone" is matched by its text (`FV/runs/agent_controller.go:124`). The claim admits a run by the disk budget of `FV/runs/capacity.go` and sends only the engine type and version.

**Consumers.** The system health check reads stale agents with `GetStaleAgents(StaleAgentThreshold)` and names them (`internal/features/system/healthcheck/service.go:64-80`). Telemetry reads the enabled configs and every agent's capacity (`internal/features/telemetry/service.go:127,194`). `internal/features/backups/config` stays for both of them and for `internal/app`; only `SeedTestLogicalBackup` in it serves verification tests. `internal/features/intervals` serves only the old config.

**The frontend.** `entity/verification/{agents,config,runs}/api/*.ts` call the old routes. The UI uses only `hasAgents` of the availability, polled every 5 seconds, and lists runs with `limit=50` polled every second, using `total` only to compute whether more pages exist (`frontend/src/features/verification/runs/ui/VerificationsComponent.tsx:140-155`). The old errors have no `code`, so `translateApiError` shows the backend's text. The trigger enum and its labels have no `AFTER_BACKUP` (`frontend/src/entity/verification/runs/models/VerificationTrigger.ts`, `VerificationTriggerLabelKeys.ts`), so an after-backup run shows no trigger label today.

**Tests.** 122 tests and 27 subtests in 13 files: 47 drive user routes, 45 the agent protocol, 26 the scheduler, 2 pure capacity logic and 2 private middleware checks. Every runs test seeds its backup with `SeedTestLogicalBackup`; 16 change package variables, and 6 sleep. No test drives the backup stream; the agent end-to-end tests of `internal/agent/e2e` do.

## Goals / Non-Goals

**Goals:**
- Verification is a module in the target layout, and its user operations are in the contract with `Problem` errors.
- Every rule of today's scheduling, retries, capacity, notifications and the agent protocol behaves as before.
- The old code that remains serves only the agent protocol, on temporary methods of `module.go`.

**Non-Goals:**
- `agent.yaml` and anything about the agent's side of the protocol.
- New product rules for verification. Where the contract forces a visible change, decision 7 lists it.

## Decisions

### 1. The packages

```
internal/modules/verification/
  module.go                 the contract: Module, Deps, New, types, temporary agent methods
  internal/
    agents/                 agents: model, repository, service (create, list, rotate, delete,
                            count, credentials, heartbeat record, stale list)
    configs/                verification configs: model with a lib/schedule schedule, repository,
                            service (read with default, save, copy, delete)
    verifications/          verifications: model, repository, service (start, cancel, read,
                            list), claim and capacity, report, retries, reconciliation of a
                            heartbeat, notifications, the scheduler job, the event handlers
    httpapi/                generated server, handlers, conversions
    module/                 wiring of the packages and the Module implementation
```

The packages are named in the plural, `agents`, `configs` and `verifications`, because the singular `agent` would read like the agent binary's own `internal/agent`, and `verification` like the module itself. A verification is `verifications.Verification`, and its table is `verification_runs`, as the glossary row states.

The imports run one way: `verifications` imports `agents` and `configs`, and neither imports it back. The heartbeat needs both: `agents` records the capacity and the time, and `verifications` reconciles the runs the agent reports. `internal/module` calls the two in order, so `agents` needs no listener list, and `AddAgentHeartbeatedListener` with its interface goes away.

Alternatives rejected:
- *One package for the whole module.* Agents are instance-wide and change for other reasons than a database's config and runs; one package would mix the admin area with workspace content.
- *Keeping the listener of `FV/agents/listeners.go`.* It exists only because the old packages could not see each other. Inside a module the wiring package already sees both.

### 2. The services follow the module rules

- Repositories run every query through `database.Conn(ctx, r.db)` and change single columns with one `UPDATE`. The claim keeps its `FOR UPDATE SKIP LOCKED` over the 100 oldest pending runs and its conditional update on a pending row without an agent (`FV/runs/repository.go:99-133`). The success report keeps its conditional update on the agent's running row and inserts the statistics in the same transaction (`:139-172`).
- Transactions come from the `Transactor` in `Deps` instead of `storage.GetDb().Transaction` (`FV/runs/service.go:75,265,541`). Audit events are written in the same transaction through `lib/audit`, with no workspace for the agent events and the database's workspace for the others.
- Notifications and the backup's verification status are written after the commit: the status through `backups.Module.SetLogicalBackupVerificationStatus`, the message through `notifiers.Module.Notify`. A failure of either is logged, as today.
- Every time comes from the clock in `Deps`. The three thresholds become constants of `verifications`, and tests move the `apptest` clock instead of shrinking variables. The health check reads the stale agents through the module, so it no longer imports the threshold.
- A service looks the object up first and answers `not_found` when it is missing, then calls `workspaces.Authorize` with the database's workspace. A verification and a config find their workspace through `databases.Module.GetDatabases`.

### 3. "Verification gone" is a sentinel

`errors.go` of `verifications` declares `ErrVerificationGone`. The report returns it in the two cases where the old code returns the text "verification gone": the verification no longer exists (`FV/runs/service.go:412-415`), or a completed report finds no running row of the agent to update (`FV/runs/repository.go:166-168`). A failed report and a completed report below the size guard keep calling the retry rule without checking the owner or the status, as today (`FV/runs/service.go:419-425`): adding that check would change a retry rule, which this change keeps. The backup stream returns the error for a missing, finished or foreign run. `module.go` re-exports it, and the old agent controller answers it with `410` and `{"reason":"gone"}` through `errors.Is`. Other errors of the report keep `400`, and the backup stream keeps answering every error with `410`, as the agent expects (`FV/runs/agent_controller.go:71-75`).

### 4. The scheduler is one job of `background.Runner`

`GetJobs` returns one job, `verification_scheduler`, with a 15-second interval. A pass runs the three steps of Context in the same order, so a run that a pass creates or requeues is seen in the same order as today. A step's error is logged, and the next step still runs, as today. The `atomic.Bool` guard and the ticker loop go away, and `internal/app/background_loops.go` loses its verification entry.

The interval schedule of a config is a `schedule.Schedule` of `lib/schedule`, mapped to the five columns that `features/intervals` mapped. `internal/features/intervals` is deleted, together with the comment of `internal/features/intervals/model.go:9` that names it as the old config's schedule.

Alternatives rejected:
- *Three jobs, one per step.* Their order would no longer be fixed, and a requeue of the reaper could land before the cancellation step of the same tick or after it, which today never varies.

### 5. The module reacts to four events

`module.go` offers:
- `OnLogicalBackupCompleted`, the after-backup run of Context;
- `OnLogicalBackupsDeleting`, which deletes the runs of the backups, moved from the old runs code;
- `OnDatabaseCopied`, which copies the saved config of the original and writes nothing when the original has none, as the backup configs do (`internal/modules/backups/internal/config/event_handler_service.go:42-53`): a missing config reads as the default;
- `OnDatabaseDeleting`, which deletes the database's config, moved from the old config code.

`internal/app/wiring.go` subscribes them where `verification-schema` subscribed the old handlers, and the old handlers, their `testhelpers` subscriptions and the old `SetupDependencies` go away.

### 6. The agent routes stay in the old code on temporary methods

Until `agent-contract`, `internal/features/verification` keeps only what the agent calls: the four routes, `RequireAgentAuth` with its rate limit, and the request and response types of `agents/dto.go` and `runs/dto.go`. They call `module.go` through a bridge, `internal/features/verification/bridge.go`, which `internal/app` installs, as the other old bridges do. One temporary method serves each route:

| Route | Method |
|---|---|
| token check of every route | `AuthenticateAgent(ctx, agentID, token) (Agent, error)` |
| `POST /agent/verification/:agentId/heartbeat` | `RecordAgentHeartbeat(ctx, agent, AgentHeartbeat) ([]uuid.UUID, time.Time, error)`: the IDs to abort and the time recorded |
| `POST /agent/verifications/:agentId/claim` | `ClaimVerification(ctx, agent, AgentCapacity) (VerificationAssignment, bool, error)` |
| `GET .../:id/backup-stream` | `OpenVerificationBackup(ctx, agent, verificationID) (io.ReadCloser, error)` |
| `POST .../:id/report` | `SubmitVerificationReport(ctx, agent, verificationID, VerificationReport) error` |

`module.go` marks these methods as serving the old agent controller until `agent.yaml` replaces it. The types are plain Go structs of the module; the old controller converts its DTOs to them, so the JSON of the protocol stays where it is today and `agent-contract` deletes it with the controller. Paths, bodies and status codes do not change.

The methods the system health check and telemetry call stay in the contract without that mark, because those consumers live until milestone 13:
- `ListStaleAgents(ctx)`, the agents silent for more than 5 minutes;
- `ListAgents(ctx)`;
- `ListEnabledConfigs(ctx)`.

The methods of `backups/module.go` that served "the old verification runs" now serve this module, and their comments change to say so.

The old code reads other modules once per record in two loops: the scheduler's first pass reads the latest backup, the database and the engine version of each enabled config (`FV/runs/scheduler.go:77-172`), and the claim reads the database and the engine version of each candidate (`FV/runs/service.go:292-297`). `docs/architecture.md`, "Modules", asks for batch reads instead, so the module reads each kind once per pass or claim and joins in Go:
- databases through the existing `databases.Module.GetDatabases(ctx, databaseIDs)`;
- the engine versions through a new `databases.Module.GetPostgresqlLogicalVersions(ctx, databaseIDs) (map[uuid.UUID]string, error)`, which leaves out a database of another type or without a version, so such a database is skipped as today;
- the latest completed logical backups through a new `backups.Module.GetLatestCompletedLogicalBackups(ctx, databaseIDs) (map[uuid.UUID]LogicalBackup, error)`. `GetLatestCompletedLogicalBackup` stays for telemetry until milestone 13.

The after-backup handler reads one database and keeps the single-record calls.

Alternatives rejected:
- *Moving the agent routes into the module now, on Gin.* The module's routes come from the generated strict server. Hand-written Gin routes inside the module would be a second way to serve HTTP that `agent-contract` removes one change later.
- *One method for the whole protocol.* The old controller would have to know the module's internals to tell the routes apart.

Manual and scheduled creation take a transaction-scoped advisory lock per database before inspecting active rows. A row lock alone leaves an empty queue unprotected: the concurrent HTTP test reproduced ten manual verifications from sixteen requests. Cancellation updates only pending or running rows, so a report committed meanwhile cannot be overwritten.

### 7. Eleven operations under the tag `verification`

| Operation | Answer |
|---|---|
| `GET /verification-agent-counts` `getVerificationAgentCount` | `{agentCount}` |
| `POST /verification-agents` `createVerificationAgent`, `x-access: admin`, with `{name}` | 201 `{agent, token}` |
| `GET /verification-agents?limit=&after=` `listVerificationAgents`, `x-access: admin` | page of `VerificationAgent`, newest first by `(createdAt, id)` |
| `POST /verification-agents/{agentId}/rotate-token` `rotateVerificationAgentToken`, `x-access: admin` | 200 `{token}` |
| `DELETE /verification-agents/{agentId}` `deleteVerificationAgent`, `x-access: admin` | 204 |
| `GET /verification-configs/{databaseId}` `getVerificationConfig` | `VerificationConfig` |
| `PUT /verification-configs/{databaseId}` `updateVerificationConfig` with `VerificationConfigInput` | 200 `VerificationConfig` |
| `POST /verifications` `startVerification` with `{backupId}` | 202 `Verification` |
| `POST /verifications/{verificationId}/cancel` `cancelVerification` | 200 `Verification` |
| `GET /verifications/{verificationId}` `getVerification` | `VerificationDetails` |
| `GET /verifications?databaseId=&limit=&after=` `listVerifications` | page of `Verification`, newest first by `(createdAt, id)` |

- **`VerificationAgent`** holds `id`, `name`, `maxCpu`, `maxRamGb`, `maxDiskGb`, `maxConcurrentJobs`, an optional `lastSeenAt` and `createdAt`. `maxRamGb` keeps its unit until `agent-contract` changes the column. The token is in the responses of create and rotate only, which the `api-design` skill allows for a credential the operation itself issues.
- **`VerificationConfig`** holds `databaseId`, `isAutomaticVerificationEnabled`, `scheduleType` of the schema `VerificationScheduleType` (`INTERVAL` or `AFTER_BACKUP`), `schedule` of the shared `Schedule` schema, and `notificationTriggers` of the schema `VerificationNotificationTrigger` (`VERIFICATION_SUCCESS`, `VERIFICATION_FAILED`), declared next to `LogicalNotificationTrigger`. The field name follows `BackupConfig`. The switch is called automatic, not scheduled, because it turns on both the interval runs and the after-backup runs, while the trigger `SCHEDULED` names only the interval runs; the column keeps its old name. A read of a config that was never saved returns the default and stores nothing, as `getBackupConfig` does; the scheduler treats a missing row as the default too.
- **`Verification`** holds `id`, `databaseId`, `backupId`, an optional `agentId`, `trigger` of the schema `VerificationTrigger` (`MANUAL`, `SCHEDULED`, `AFTER_BACKUP`), `status` of the schema `VerificationStatus` (`PENDING`, `RUNNING`, `COMPLETED`, `FAILED`, `CANCELED`), `attemptCount`, `createdAt`, and the optional `startedAt`, `finishedAt`, `restoreDurationMs`, `verifyDurationMs`, `pgRestoreExitCode`, `restoredDatabaseSizeBytes`, `tableCount`, `schemaCount` and `failMessage`. **`VerificationDetails`** adds `tableStats`, the schema, name and row count of each table.
- **Errors.** `startVerification` answers 409 `backup_not_completed` for a backup that is not completed, and 409 `already_running` for a second manual verification of the database. The request itself is valid; the backup's state forbids the action, which the `api-design` skill answers with 409, and no existing code names that state. `backup_not_completed` joins `ProblemCode`, `lib/apierror` and the six dictionaries. `cancelVerification` answers 409 `not_running` for a completed, failed or cancelled one. A cursor the list did not issue answers 400 `invalid_request`. The admin mark answers 401 and 403 through the `users` middleware.
- **Lists.** Both lists use `lib/database.TimeCursor`, with `limit` from 1 to 100 and a default of 20, as `listLogicalBackups`.

Visible changes the contract forces, beyond the paths:
- the manual start answers `202`, a cancel answers `200` with the verification, and the list has no total;
- the cancel answers 409 instead of 400 for a finished verification, a start for a backup that is not completed 409 `backup_not_completed` instead of 400, and a missing verification 404 instead of 400;
- the agent count returns only the number, and the interface computes "has agents" from it.

Alternatives rejected:
- *Keeping `/verifications/enqueue` and `/verifications/by-database/{databaseId}`.* The skill makes a collection's `POST` the start and lists by a filter, not by a nested path.
- *Answering `204` to a cancel.* The interface shows the cancelled verification at once; `200` with the verification saves a read. Unlike a backup's cancel, the verification is already `CANCELED` when the answer leaves, so `202` would promise work that is not happening.
- *400 `invalid_request` for a backup that is not completed.* The skill keeps 400 for a request that breaks the contract or a validation rule, and this request breaks neither.
- *`GET /verification-availability`.* The skill makes paths of plural nouns. The count of agents is the resource the operation reads, so it is `/verification-agent-counts`, beside `/verification-agents`.
- *Paging the agents without a cursor.* The skill makes every list page with a cursor, and an instance rarely has more than a page of agents, so the frontend follows the cursor to the end, as `listAllNotifiers` does.

### 8. Access

The four agent operations carry `x-access: admin`. The agent count carries no mark, so any signed-in user reads it. The config and verification operations call `workspaces.Authorize` with `ActionView` for reads and `ActionEditContent` for the save, the start and the cancel, as today. A verification of a backup that the actor cannot see answers 404 before any other check.

### 9. Frontend

- The three API files of `entity/verification` call the generated client. Their models become aliases of `ApiComponents` types, with the value objects that the screens compare against, as `entity/notifiers/models/NotificationCategory.ts` does. The label dictionaries stay.
- The agents page follows the agent list to its last page, and shows the token from the create and rotate answers as today.
- The verification list follows `nextCursor` with its infinite scroll and keeps its one-second poll of the first page. The "all loaded" line counts the loaded verifications, as it already does.
- The check for agents calls `getVerificationAgentCount` and reads `agentCount > 0`.
- The six dictionaries gain the text of `backup_not_completed`.
- The trigger enum gains `AFTER_BACKUP`, and the six dictionaries gain its label, because the generated type includes it and `VERIFICATION_TRIGGER_LABEL_KEYS` is a `Record` over that type, so the typecheck needs every value.
- Errors reach the user through `translateApiError`, which now finds a `code` on every answer.

### 10. Tests

- **User operations:** API tests of the module through `apptest` and its contract-checking client, one file per operation group: agents, agent count, config, start, cancel, read and list, and access. They cover every scenario of both deltas and replace the 47 user tests of Context.
- **Agent protocol:** API tests in the module's test package through `CallOldAPI` with the agent's token against the old controller: heartbeat, rate limit, claim with its budget and sanitization, report with its retries and size guard, reconciliation of a heartbeat, notifications through the recording notifier endpoints of `apptest`. They replace the 45 agent tests and move to `agentclienttest` in `agent-contract`.
- **Scheduler:** API tests that set the config through the API, advance the `apptest` clock and run one pass with `RunJobPass`. They replace the 26 scheduler tests without shrinking variables or sleeping.
- **Pure logic:** the capacity tests move with `capacity.go` as unit tests of `verifications`.
- **Backups.** Tests make real backups through the API, as `internal/modules/backups` does. `SeedTestLogicalBackup` and its file are deleted. A state the API cannot reach quickly, such as a running verification without an agent process, is set with `ExecuteSQL`, as the backups tests do.
- **Coverage.** `coverage.md` in the change lists every old test and subtest at the base commit with its replacement or the reason it is removed, as `physical-tests-on-apptest` did. It is written before the old tests are deleted.
- **End-to-end.** `apptest.WithAgentBinariesDir` uses the system clock, so heartbeat timestamps and rate limits advance with the external agent process. The user calls of `internal/app/apptest/verification.go` move to the generated client, and the scenarios of `internal/agent/e2e` run unchanged otherwise.
- **Asynchronous state exposed by verification.** The broad backend run exposed a physical-backup cancellation test checking file deletion before the file store's commit window expired. Advance its manual clock past the configured test window before running cleanup, retaining the file-absence assertion. The WAL retention test waits for the asynchronous PostgreSQL configuration reload to apply before generating the WAL intended to exceed that limit. The WAL sample test waits for both the first persisted sample and the subsequent clearing of the streamer's failing time, because those writes can be observed separately through statistics.
- **Upgrade.** `current_release_verification_test.go` reads the agent, the config and the verification through the new operations.

### 11. What `internal/app` does

`internal/app` builds the module after `backups` and `notifiers`, installs it into the verification bridge, registers its routes and job, subscribes its four handlers, and drops the old getters from `getters_test.go`, the old routes from `routes.go` apart from the agent routes, and `RestoreVerification` from `model_schemas.go`. The shutdown order needs no new step: the module starts no work of its own beyond the job and the notifications that `notifiers` already drains.

## Risks / Trade-offs

- **The rewritten tests miss a rule the old ones held.** Tests and code change in one diff. → `coverage.md` maps all 149 old tests and subtests, and the review checks the mapping. The agent end-to-end tests check the protocol from outside without change.
- **A rule changes during the move.** The scheduler, the retries and the claim are subtle. → Decision 2 keeps their queries and constants, and the protocol tests cover each retry reason.
- **A requeued verification moves in the list.** A requeue gives the verification a new `created_at`, so a client paging older pages can miss one that moved to the top. → The interface polls the first page every second and shows it there. The old offset paging had the same gap.
- **Users' scripts break.** → proposal.md marks it BREAKING, so the release notes say it.

## Migration Plan

No migration. The release changes the user routes and the frontend together. Agents of the previous release keep calling the same agent routes.
