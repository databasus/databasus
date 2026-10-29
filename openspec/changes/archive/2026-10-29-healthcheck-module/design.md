## Context

Why the change is needed is in proposal.md, section Why. This document covers what shapes the approach. Paths are under `backend/` unless they start with another top-level directory. `FH` stands for `internal/features/healthcheck`.

**The old user routes.** All three sit behind the session middleware (`internal/app/routes.go:68-69`) and answer every error as `400` with the internal text, including a refused access, whose text is "insufficient permissions ..." (`FH/config/service.go`, `Save` and `GetByDatabaseID`).
- `POST /healthcheck-config` takes the database ID in the body, needs `workspaces.ActionEditContent`, saves the whole row, and answers `{"message": ...}` (`FH/config/controller.go`). When the save switches the checks off, it clears the database's health status through `databases.Module.SetHealthStatus(nil)`. The audit event is written after the save, outside any transaction, through the `features/audit_logs` wrapper.
- `GET /healthcheck-config/:databaseId` needs `ActionView`. When the row is missing, it inserts the default and reads it again (`FH/config/service.go`, `GetByDatabaseID`): checks on, notifications on, interval 1, threshold 3, retention 7.
- `GET /healthcheck-attempts/:databaseId?afterDate=` needs `ActionView` and returns every attempt newer than `afterDate`, newest first, with no limit (`FH/attempt/repository.go:13-29`). Without `afterDate` it uses seven days before `time.Now()`.

**The loop.** `FH/attempt/background_service.go:25-75` checks once at start, then ticks every minute. Each tick reads every enabled config and starts one bare goroutine per config (`:66`). The goroutine reads the database through `databases_bridge.Access`, skips it when the last attempt is younger than the interval minus one second (`FH/attempt/check_database_health_uc.go:166-185`), probes it with `databases.Module.ProbeLiveness`, inserts the attempt, changes the status and notifies (`:79-133`), and deletes attempts older than the retention measured from `time.Now()` (`check_database_health_uc.go:67-70`). Any error of `ProbeLiveness` counts as unavailable, including a cancelled context and a database deleted meanwhile. The shutdown cancels the context but does not wait for these goroutines (`internal/app/background_loops.go:17`).

**The transitions.** From available or no status, a failed attempt makes the database unavailable when the threshold is 1, or when the last `threshold` attempts, this one included, all failed (`check_database_health_uc.go:103-131`). From unavailable or no status, a successful attempt makes it available at once (`:88-101`). Each change notifies the database's notifiers through `notifiers_bridge.GetSender()` when `IsSentNotificationWhenUnavailable` is on, with the categories `HEALTHCHECK_FAILED` and `HEALTHCHECK_SUCCESS` (`:188-214`). A new database starts as available, because `databases.health_status` defaults to `AVAILABLE` (`migrations/20250704095930_add_healthckeck.sql:6`) and the insert leaves the column out (`internal/modules/databases/internal/userdatabase/database_repository.go:86`), so its first successful check changes nothing and notifies nobody. A database whose checks were switched off has no status, so its first successful check after they are switched on again sends "back online"; the spec keeps that on purpose.

**Config lifecycle.** `internal/app/wiring.go:51` subscribes `OnDatabaseCreated`, which inserts the default in the creation transaction. `CopyDatabase` publishes `DatabaseCreated` for the copy before `DatabaseCopied` (`internal/modules/databases/internal/userdatabase/service.go:361-371`), so a copy gets the default, not the original's values, while it inherits the original's health status (`database_repository.go:102`). The verification and backups modules copy their configs on `DatabaseCopied`.

**Deletion.** `DeleteDatabase` locks the database row `FOR UPDATE` before it runs the `DatabaseDeleting` handlers and deletes the row (`internal/modules/databases/internal/userdatabase/deletion_service.go:17-38`). Both healthcheck tables reference `databases` with `ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED` (`migrations/20250704095930_add_healthckeck.sql:18-38`), so today the database's delete removes them.

**The schema.** `healthcheck_configs (database_id PK, is_healthcheck_enabled, is_sent_notification_when_unavailable, interval_minutes, attempts_before_considered_as_down, store_attempts_days)` with no `CHECK`; the positive numbers are checked only by a GORM `BeforeSave` hook (`FH/config/model.go`). `healthcheck_attempts (id, database_id, status TEXT, created_at)` with the index `(database_id, created_at)` and no `CHECK` on `status`. `idx_healthcheck_configs_database_id` duplicates the primary key. `migrations/20260512000000_disable_healthcheck_for_agent_dbs.sql` switched the checks off for the old agent-based databases.

**Consumers.** Telemetry reads no healthcheck table; it treats a database without health status as one with checks off (`internal/features/telemetry/service.go:267-268`), so clearing the status on switch-off must stay. `workspaces_bridge.Access` and `databases_bridge.Access` have no consumer outside `FH` and `internal/app/getters_test.go`. The `features/audit_logs` wrapper has none outside `FH`, `internal/app/app.go` and `getters_test.go`.

**The frontend.** `frontend/src/entity/healthcheck/api/*.ts` call the old routes through `apiHelper`, with hand-written models. `HealthckeckAttemptsComponent.tsx` shows the periods today, 7 days, 30 days and all time, computes `afterDate` on the client, polls today every minute, guards against a stale answer with a module-level `lastLoadTime`, and draws one mark per attempt.

**Tests.** 8 top-level tests and 26 subtests in three files. The config and attempt controller tests drive the old Gin router with users and workspaces made through the old SQL helpers. The use-case test calls `Execute` directly with a mocked sender and mocked databases, and its physical subtest probes a replication-ready cluster of the shared test server for each physical version.

## Goals / Non-Goals

**Goals:**
- Healthcheck is a module in the target layout, and its operations are in the contract with `Problem` errors.
- The rules of a check behave as they do today; the spec `database-healthcheck` writes them down.
- Stopping Databasus waits for running checks and records nothing for an interrupted one.

**Non-Goals:**
- A faster attempts chart, or an aggregated history. The list keeps one record per attempt.
- New rules for the transitions or the notifications.

## Decisions

### 1. The packages

```
internal/modules/healthcheck/
  module.go                 the contract: Module, Deps, New
  internal/
    configs/                config model, repository, service (read, save, default on creation,
                            copy, delete)
    attempts/               attempt model, repository, service (list), the probe job in its own
                            file, the transition rule, notifications
    httpapi/                generated server, handlers, conversions
    module/                 wiring of the packages and the Module implementation
```

`attempts` imports `configs`, because a pass reads the due configs; `configs` does not import `attempts`. The deletion handler deletes the attempts first and then the config, so `internal/module` calls both in that order instead of one package reaching into the other.

`module.go` offers `RegisterRoutes`, `GetJobs`, `OnDatabaseCreated`, `OnDatabaseCopied` and `OnDatabaseDeleting`. No other module reads healthcheck data, so the contract has no read methods.

Alternatives rejected:
- *One package.* The attempts and the job change for other reasons than the editable config, and the plan's layout for the module already names the two packages.
- *Keeping the package names `config` and `attempt`.* The singular `config` shadows the common variable name, and the verification module set the plural convention.

### 2. The services follow the module rules

- Repositories run every query through `database.Conn(ctx, r.db)`, return `(T, bool, error)` for a lookup, and never return `gorm.ErrRecordNotFound`.
- The save runs in one transaction from the `Transactor`: it looks the database up through `databases.Module.GetDatabases`, answers `not_found` when it is missing, calls `workspaces.Authorize` with `ActionEditContent`, updates the row, clears the health status when the checks are switched off, and writes the audit event through `lib/audit` with the database's workspace and the text the old code writes. The reads use `ActionView`.
- The validation of the old `BeforeSave` hook moves to the contract (`minimum: 1`) and to the schema (`CHECK`). The generated server decodes requests but does not enforce these limits. The healthcheck HTTP routes therefore validate requests against their generated embedded contract before dispatching them; authentication stays with the existing users middleware. The service keeps no third copy.
- Every time comes from the clock in `Deps`, including the retention cut, which the old code took from `time.Now()`.

### 3. The config always exists, and a read writes nothing

- `OnDatabaseCreated` inserts the default in the creation transaction, as today.
- `OnDatabaseCopied` overwrites the copy's row, which `OnDatabaseCreated` inserted moments earlier in the same transaction, with the original's values. The spec `databases` states this.
- `OnDatabaseDeleting` deletes the attempts and the config in the deletion transaction.
- The migration of decision 8 inserts the default for every database without a row. After it, every database has exactly one row, and a missing row is a fault: the read answers `internal`, not a default.

Alternatives rejected:
- *A missing row reads as the default and stores nothing, as `getBackupConfig` and `getVerificationConfig` do.* Their defaults are off, so the scheduler can ignore databases without a row. The healthcheck default is on, so the pass would have to find databases that have no row, which needs either a JOIN with the `databases` table, forbidden between modules, or a read of every database on every pass.
- *Copy only on `DatabaseCopied` and skip `DatabaseCreated` for a copy.* `CopyDatabase` publishes both, and the order is fixed by `userdatabase/service.go:361-371`; overwriting is simpler than teaching the creation handler to tell a copy apart.

### 4. The probe job

`GetJobs` returns one job, `healthcheck_probe`, with an interval of one minute. A pass:

1. Reads, in one query of the module's own tables, every enabled config with the time of its database's last attempt, and keeps those that have no attempt or whose last attempt plus the interval is not later than one second after `now`. The one-second slack is today's and keeps a one-minute check from being skipped by a pass that starts a few milliseconds early.
2. Reads the due databases in one call of `databases.Module.GetDatabases(ctx, ids)`, as `docs/architecture.md`, "Modules", asks for. A database missing from the answer was deleted after step 1 and is skipped.
3. Probes the databases with at most 32 at once, as decision 5 describes, and waits for all of them. A failed probe is a result, not an error. An error of one record step is logged with the database ID and does not stop the others.
4. For each database: calls `ProbeLiveness`. When the pass's context has ended, it returns without recording. Otherwise it takes `now` from the clock, so the attempt carries the time of its own probe rather than the start of the pass, and in one transaction locks the database, reads the current config, discards a result if checks are disabled, then inserts the attempt, reads the last `threshold` attempts when it needs them, sets the status through `databases.Module.SetHealthStatus` when the rule of Context says so, and deletes the attempts older than the retention. The notification is sent through `notifiers.Module.Notify` from an after-commit hook, so a rolled-back transaction notifies nobody.

The transition rule is a pure function of the old status, the new result, the threshold and the last attempts, in its own file of `attempts`, so the table of transitions has unit tests besides the API tests.

`internal/app/background_loops.go` loses its healthcheck entry; telemetry keeps the loop until 13b.

Alternatives rejected:
- *One job per database, or a goroutine per due database as today.* The first does not fit `Runner`, whose jobs are fixed at start. The second leaves goroutines that shutdown cannot wait for, which `docs/architecture.md`, "Long-running work", forbids outside `lib`.
- *A limit of 10.* A probe gives up after about 15 seconds (`internal/modules/databases/internal/providers/*/connected_database.go`), and `Runner` starts the next pass one interval after the previous one ended (`internal/lib/background/runner.go:31-33`). With 10 at once, 40 unreachable databases already stretch a pass to a minute and make every check late. With 32, that point is near 130. The cost is up to 32 connections and SSH tunnels at once.
- *No transaction around the record step.* The old code inserts the attempt and writes the status separately, so a failure between them leaves a history that disagrees with the status. A transaction costs nothing here, and the after-commit hook keeps the notification out of a rolled-back step.

### 5. Bounded concurrency on `background.Group`

A pass creates its own `background.New` group and a buffered channel of 32 slots, as `backups/internal/physical/backuping/orphan_slots.go:25-48` does. Before each probe it takes a slot, or stops starting probes when the pass's context has ended; the probe gives the slot back when it returns. `Group.Go` runs the probe inside `lib`, turns a panic into an error through `RunRecovering` and logs a returned error. After the loop the pass calls `Wait` with `context.WithoutCancel(ctx)`, because `Group.Wait` returns as soon as its context ends, and the pass must wait for its started probes even during a shutdown. The pass itself returns an error only when step 1 or step 2 of decision 4 fails; the group logs the errors of single record steps.

Alternatives rejected:
- *A new generic helper in `lib/background` that bounds the goroutines and joins their errors.* `Group` with a channel already gives bounded goroutines that recover panics and that the caller can wait for, and the existing caller of that pattern needs nothing more. A helper would be a second way to do the same.
- *`golang.org/x/sync/errgroup` with `SetLimit`.* It is only an indirect dependency today (`go.mod:323`), it cancels the siblings on the first error, which a probe must not do, and it does not recover panics.

### 6. What a stop and a deletion do to a running probe

- **Stop.** The root context ends, `Runner` starts no new pass, and the running pass waits for its probes. The shutdown waits for the runner (`internal/app/server.go:85`), so it waits for the probes too. A probe whose context ended returns before it records anything (decision 4, step 4), so an interrupted check never counts as unavailable and never notifies.
- **Config changes and deletion.** `databases.Module.GetDatabaseForUpdate` requires the caller's transaction and returns the current database, including notifier IDs, while holding its row `FOR UPDATE`. Config saves authorize first, take that lock, and recheck access if the workspace changed. A probe takes the same lock only after the network call, reads the current config, and discards its result when checks are disabled. Threshold, notifications and retention also use the current config. Both paths lock the database before writing any healthcheck rows, matching `DeleteDatabase`, which locks the database before deleting attempts and config. A probe that finds the database gone records nothing. This lock order also prevents a config save and deletion from deadlocking. Notifications registered by an earlier committed probe can still be delivered after a later disable; a discarded probe registers no notification.

### 7. Three operations under the tag `healthcheck`

| Operation | Answer |
|---|---|
| `GET /healthcheck-configs/{databaseId}` `getHealthcheckConfig` | `HealthcheckConfig` |
| `PUT /healthcheck-configs/{databaseId}` `updateHealthcheckConfig` with `HealthcheckConfigInput` | 200 `HealthcheckConfig` |
| `GET /healthcheck-attempts?databaseId=&since=&limit=&after=` `listHealthcheckAttempts` | `HealthcheckAttemptPage` |

- **`HealthcheckConfig`** holds `databaseId`, `isHealthcheckEnabled`, `isNotificationEnabled`, `intervalMinutes`, `failedAttemptsBeforeUnavailable` and `attemptRetentionDays`. **`HealthcheckConfigInput`** holds the same without `databaseId`, all required, the three numbers with `minimum: 1`. The paths follow `/backup-configs/{databaseId}`.
- **Renamed fields.** `isSentNotificationWhenUnavailable` becomes `isNotificationEnabled`, because the switch governs both the unavailable and the online notification. `storeAttemptsDays` becomes `attemptRetentionDays`, the unit stays in the name as the `api-design` skill asks. `attemptsBeforeConcideredAsDown` becomes `failedAttemptsBeforeUnavailable`, which fixes the typo and names the state as the `DatabaseHealthStatus` enum does, `UNAVAILABLE` rather than "down". The columns are renamed to match in decision 8, so the model does not map new names onto old columns.
- **`HealthcheckAttempt`** holds `id`, `status` of the existing schema `DatabaseHealthStatus` and `createdAt`. **`HealthcheckAttemptPage`** holds `items` and an optional `nextCursor`.
- **The list.** `databaseId` and `since` (`date-time`) are required. The list returns attempts with `createdAt` at or after `since`, newest first by `(createdAt, id)`, paged with `lib/database.TimeCursor`. `limit` runs from 1 to 1440 with a default of 1440. The description of the operation says why the maximum differs from the 100 of the other lists: the chart needs every attempt of the chosen period, and 1440 is one day of checks at the default interval, so the default period is one page and a week is seven. A cursor the list did not issue answers 400 `invalid_request`.
- **Errors.** Only existing codes: `invalid_request`, `unauthenticated`, `permission_denied`, `not_found` and `internal`. The six dictionaries need no new text.

Alternatives rejected:
- *A time window without a cursor.* It keeps the chart to one request, but the `api-design` skill makes every list page with a cursor, and a user's retention of a year at one check a minute would answer half a million rows in one response.
- *A cursor with the usual maximum of 100.* A week at the default interval would take a hundred requests.
- *Keeping `since` optional with a default of seven days.* A default hidden in the server is what the old `afterDate` did; with the cursor, the client states its period, and a missing one is a mistake to refuse rather than guess.
- *Keeping the old field names.* The contract breaks anyway with the new paths, so the names can state their meaning now at no extra cost to users' scripts.

### 8. One migration for the schema

Created with `goose -dir backend/migrations create healthcheck_module sql`. In its `Up` section, in order:

1. Rename `is_sent_notification_when_unavailable` to `is_notification_enabled`, `attempts_before_considered_as_down` to `failed_attempts_before_unavailable` and `store_attempts_days` to `attempt_retention_days`.
2. Repair rows: set `interval_minutes` to 1, `failed_attempts_before_unavailable` to 3 and `attempt_retention_days` to 7 where they are below 1, the default values; delete attempts whose `status` is neither `AVAILABLE` nor `UNAVAILABLE`; delete configs and attempts whose `database_id` matches no database, which the deferred cascade should have prevented but a restored dump may contain.
3. Insert the default config for every database without one.
4. Flush deferred foreign-key checks with `SET CONSTRAINTS ALL IMMEDIATE`, so the preceding repairs and inserts leave no pending trigger events when the tables are altered. Add `CHECK` constraints for the three positive numbers and the attempt status.
5. Drop the two cascading foreign keys and add them again as `ON DELETE RESTRICT`, not deferrable.
6. Drop `idx_healthcheck_configs_database_id`, which duplicates the primary key.

The `Down` section reverses the renames, the constraints, the keys and the index, and leaves the repaired and inserted rows, which the old code reads as valid. `backend/migrations/healthcheck_migration_test.go` seeds each broken row of step 2 and a database without a config, runs the migration, and checks each repair, the inserted default, each `CHECK`, and that deleting a database with a config or an attempt is refused.

Alternatives rejected:
- *A schema change before the move, as 10c1 and 12d1 did.* There are two tables and three column renames, and the upgrade test of decision 10 checks the migration and the move together.

### 9. Frontend

- `entity/healthcheck/api/*.ts` call the generated client. The models become aliases of `ApiComponents` types, and `HealthckeckAttempts.ts` goes away.
- The three healthcheck screens move from the `features/healthcheck` slice into `features/databases/ui/healthcheck/`. Today `features/databases` imports them from `features/healthcheck` in `DatabaseComponent.tsx` and `DatabaseConfigComponent.tsx`, an import between two slices of one layer, which `frontend/AGENTS.md`, "FSD", forbids. The database screens are their only consumer, so by the placement rules of the same section they belong to that slice, and `features/healthcheck` is deleted. The imports of `features/backups` and `features/verification` in `DatabaseComponent.tsx` break the same rule; the change leaves them, because it does not touch those screens.
  Alternative rejected: *rendering the healthcheck screens from `widgets/main`.* `DatabaseComponent` places the attempts between the backups and the verifications and passes their visibility to the blocks below, so moving one block out would split that layout across two layers.
- `HealthckeckAttemptsComponent` is renamed to `HealthcheckAttemptsComponent`. It computes `since` from the period as today, sends the epoch for "all time", and follows `nextCursor` until the last page. The guard against a stale answer moves from the module-level `lastLoadTime` into the effect, so a period switch during a multi-page load drops every page of the old period.
- The config screens read and save through the new operations with the renamed fields. Errors reach the user through `translateApiError`.

### 10. Tests

- **API tests on `apptest`**, one file per group in the module's root test package:
  - config: the default after creation, a save with new values and its audit event, values below 1, no session;
  - access: every scenario of the `workspace-access` delta;
  - probe: the threshold, the row broken by a success, the return to available, no notification on the first success of a new database and one after switching the checks on again, notifications off, the interval respected, checks off, the retention cut, the switch-off clearing the status;
  - lifecycle: the copy gets the original's config and no attempts, the deletion removes config and attempts;
  - list: the `since` filter, the pages, an empty list and a foreign cursor;
  - interruption: a stop during a probe and a deletion during a probe.
- **Probed databases.** A reachable database is a PostgreSQL container from `containers.StartPostgres`; a physical one from `containers.StartPhysicalPostgres` for one version, because `internal/engines/postgresql/physical/source_test.go:13` already runs the probe on every supported version. An unreachable database points at a closed local port. A database that does not answer is a local TCP listener that accepts and never replies, which the test closes when it needs the probe to end.
- **Time.** Tests move the `apptest` clock and run a pass with `RunJobPass("healthcheck_probe")`. For the interruption tests, `apptest` gains a call that starts a pass with a context the test cancels and returns its error, because `RunJobPass` uses the test's own context.
- **Notifications** are recorded through a webhook notifier pointing at an `httptest` server, as `internal/modules/verification/notification_test.go:20` does.
- **Unit tests** for the transition rule of decision 4.
- **Coverage.** `coverage.md` in the change lists each old test and subtest at the base commit with its replacement.
- **Migration** and **upgrade** as decisions 8 and 11 describe.

### 11. The upgrade test

`backend/e2e/upgrade/previous_release_healthcheck_test.go` saves, through `POST /healthcheck-config` of the old release, a config of the logical database with checks off, notifications off, an interval of 7, a threshold of 4 and a retention of 11 days. `current_release_healthcheck_test.go` reads it through `getHealthcheckConfig` and returns the items `upgradecheck.Compare` checks. Attempts are not compared: whether the old release recorded one before it stopped depends on timing.

### 12. What `internal/app` does

`internal/app` builds the module after `databases` and `notifiers`, registers its routes and its job, subscribes `OnDatabaseCreated`, `OnDatabaseCopied` and `OnDatabaseDeleting` in `wiring.go` after the subscriptions of `backups` and `verification`, and drops the old routes, the old loop entry, the old getters in `getters_test.go` and the construction of the `features/audit_logs` wrapper. The shutdown order needs no new step: the runner already waits for its passes.

## Risks / Trade-offs

- **The rewritten tests miss a rule the old ones held.** Tests and code change in one diff. → `coverage.md` maps all 34 old tests and subtests, and the spec states the rules before the move.
- **Checks run later than their interval on a large instance with many unreachable databases.** → The limit of 32 keeps a pass under a minute up to about 130 unreachable databases; the spec promises "never earlier", not "exactly on time".
- **The attempts chart stays slow on long periods.** → Out of scope by the user's choice; the list now pages, so the server side is bounded per request.
- **The migration rewrites invalid rows.** → Only rows that the old code itself refused to save, or rows of deleted databases; the migration test seeds each case.
- **Users' scripts break.** → proposal.md marks it BREAKING, so the release notes say it.

## Migration Plan

The migration of decision 8 runs at startup. The release changes the user routes and the frontend together. A rollback to the previous release is not supported, as for every migration (`docs/architecture.md`, "Migrations").
