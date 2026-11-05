## Context

Why the change is needed is in proposal.md, section Why. This document covers what shapes the approach. Paths are under `backend/` unless they start with another top-level directory. `FV` stands for `internal/features/verification` and `BL` for `internal/modules/backups/internal/logical`.

**The tables.** `migrations/20260520120000_add_restore_verification.sql:4-150` creates all four tables, and no later migration touches them. `backup_verification_configs` is keyed by `database_id` and references `databases` with `CASCADE` (`:69-73`). `restore_verifications` references `databases` and `backups` with `CASCADE` and `verification_agents` with `SET NULL` (`:95-112`). `migrations/20261027140000_move_logical_backups_into_module.sql:1-5` renamed `backups` to `backups_logical`, so `fk_restore_verifications_backup` now points at `backups_logical` and still cascades. `restore_verification_table_stats` cascades with its verification (`:134-138`). The indexes are `idx_verification_agents_token_hash` (unique), `idx_restore_verifications_{database_created,status_created,agent_status}` and `idx_restore_verification_table_stats_verification`. Nothing indexes `restore_verifications.backup_id`.

**The enums.** The code writes:
- verification status: `PENDING`, `RUNNING`, `COMPLETED`, `FAILED`, `CANCELED` (`FV/runs/enums.go:5-11`);
- trigger: `MANUAL`, `SCHEDULED`, `AFTER_BACKUP` (`:15-19`);
- schedule type: `INTERVAL`, `AFTER_BACKUP` (`FV/config/enums.go:12-15`);
- interval type: the five types of `lib/schedule` (`internal/lib/schedule/schedule.go:14-20`), through `internal/features/intervals`;
- notification: `VERIFICATION_SUCCESS`, `VERIFICATION_FAILED` (`FV/config/enums.go:5-8`).

The notification choice is stored as one `TEXT` column joined with commas. `BeforeSave` joins it and `AfterFind` splits it (`FV/config/model.go:35-63`). The interval columns default to `WEEKLY` with no time of day (`20260520120000:56-62`). The schedule is checked only when automatic verification is turned on with the interval type (`FV/config/model.go:70-74`), so a stored `WEEKLY` row without a time is valid while it is off.

**Who names the tables.** `TableName` in `FV/agents/model.go:25`, `FV/runs/model.go:37,49` and `FV/config/model.go:31`, and one raw join in `FV/runs/repository.go:233-235`. The table statistics model names the column `restore_verification_id` (`FV/runs/model.go:43`). Old tests name only `verification_agents`, whose name stays. Two old tests check the cascades by name: `Test_DeleteDatabase_RemovesVerifications_ViaCascade` (`FV/runs/controller_test.go:497-527`) deletes a database with a verified backup, and `Test_DeleteBackup_RemovesVerifications_ViaCascade` (`:528-555`) deletes a backup row with SQL. The comment at `FV/runs/service.go:364-366` names the config cascade as one way a running verification disappears. Many old runs tests delete their database in a deferred cleanup while verifications still exist (`FV/runs/controller_test.go`, through `databases_bridge.RemoveTestDatabase`). The test workspace removal deletes databases with SQL after the physical rows (`internal/features/workspaces/testing/testing.go:161-183`).

**How logical backups are deleted.** Two functions of `BL` delete rows:
- `deleteBackupsWithFiles` (`BL/service.go:449-457`) deletes by IDs. A member's deletion (`BL/service.go:168-206`) and retention (`BL/backup_retention_service.go:72-100`) call it after they lock the row with `getForUpdate` and check that no backup or restore runs.
- `removeDatabaseBackups` (`BL/database_backups_removal_service.go:37-74`) deletes by database. A database deletion (`OnDatabaseDeleting`) and a storage change (`RemoveBackupsForStorageChange`) call it. It first locks every backup row of the database with `lockAndListBackupIDsOfDatabase`, which already collects the locked IDs and throws them away (`BL/backup_repository.go:281-292`), then deletes with `deleteByDatabase`.

Every path runs inside one transaction that the caller opened.

**How events work.** A lower module publishes an event declared in its `module.go`. The handlers run synchronously in the publisher's transaction, in the order `internal/app/wiring.go` registers them (`docs/architecture.md`, "Module boundaries"). `backups` already publishes `LogicalBackupCompleted` after a commit (`BL/service.go:80-115`). `databases` publishes `DatabaseDeleting` inside the deletion (`internal/modules/databases/internal/userdatabase/deletion_service.go:13-37`).

**The old verification code and transactions.** The old repositories run on `storage.GetDb()` without the context's transaction. A handler that must join the publisher's transaction therefore runs its query through `database.Conn(ctx, db)`, as the old config's copy handler already does through `Save(ctx, ...)`.

**The test build.** Old packages build modules through bridges. The backups module of the test build subscribes its own handlers (`internal/features/backups/config/load_without_app_testhelpers.go:49-53`). The old config package subscribes its copy handler to the bridge's databases module in a `testhelpers` file (`FV/config/setup_dependencies_testhelpers.go`).

**What an agent sees when a verification disappears.** The report of a verification that no longer exists answers `410` (`FV/runs/service.go:412-415`, `FV/runs/agent_controller.go:124-127`), and so does the backup stream (`FV/runs/agent_controller.go:72-75`). The agent drops such a job without retrying (`internal/agent/runner/runner.go:464-465,620-621`). `Test_SubmitReport_WhenRowWasReapedWhileAgentWasSilent_Returns410` (`FV/runs/agent_controller_test.go:460`) pins the `410` for a verification that no longer exists, and nothing in that path changes here.

**The old test router.** `createTestRouter` in `FV/runs/controller_test.go:44-49` calls `verification_config.SetupDependencies`, a `testhelpers` file that subscribes the config's copy handler to the bridge's databases module, and the production `SetupDependencies` of `FV/runs/di.go:79-81`, which registers the heartbeat listener. `FV/runs` has no `testhelpers` setup file.

**The upgrade test.** `e2e/upgrade` writes data through the newest release, `v3.63.1`, and checks it through the current code. It keeps the requests of each release apart: `previous_release_*_test.go` speaks the old API, `current_release_*_test.go` the current one, and both compare through `upgradecheck` items that name what differs. `v3.63.1` serves the verification routes the current code still serves: `POST /verification/agents` and `GET /verification/agents` for instance admins, `PUT` and `GET /verification-config/:databaseId`, `POST /verifications/enqueue` and `GET /verifications/:id` (`FV/agents/controller.go:18-29`, `FV/config/controller.go:17-18`, `FV/runs/controller.go:17-20`). A pending verification that no agent claims is failed only after 24 hours (`FV/runs/scheduler.go:24`).

## Goals / Non-Goals

**Goals:**
- After the migration, every verification table carries the `verification_` prefix, references other modules with `RESTRICT` and rejects values the code never writes.
- Every deletion of a logical backup removes its verifications in its own transaction, and a refused deletion keeps them.
- The old verification code behaves as before on the renamed tables.
- The upgrade test checks the verification tables against data written by the previous release.

**Non-Goals:**
- Moving verification code, which `verification-module` does.
- Changing what any deletion refuses or the order of the handlers of a database deletion.

## Decisions

### 1. The schema goes first, as its own change

Milestone 12d is split into this change and `verification-module`. This one changes the schema and the cross-module deletion on the old code. The next one moves the code.

The order follows milestone 10c. The upgrade test then checks the renamed tables against a real release before the move begins. The move becomes a diff with no schema change. The new `backups` event and its four paths are tested here on their own, so the move does not change the deletion and the code under it in one diff.

Alternatives rejected:
- *One change for the whole of 12d.* The migration, a change to `backups`, the module, 11 operations, the frontend and the rewritten tests would land in one diff. The plan already allowed this split if the change did not fit.
- *The event in the module change, the rest of the schema here.* `RESTRICT` to `backups_logical` cannot land without the event, because every deletion path would fail on the first backup with a verification.

### 2. Names follow the module convention

The tables become `verification_configs`, `verification_runs` and `verification_run_table_stats`, and `verification_agents` keeps its name. A verification of one backup is a "run" in the table names because "verification" is already the module prefix, and `verification_verifications` would repeat it. The statistics column becomes `run_id` to match its table. The glossary keeps one term per concept, so its "Verification" row names the table `verification_runs` in its code column and says why the table says "run". The next change, `verification-module`, then names its code by that row.

Constraints and indexes follow the convention of `20261027140000_move_logical_backups_into_module.sql`: `<table>_pkey`, `fk_<table>_<target>`, `ck_<table>_<column>`, `uq_<table>_<columns>` and `idx_<table>_<columns>`. So:
- `fk_verification_configs_database`;
- `fk_verification_runs_database`, `fk_verification_runs_backup`, `fk_verification_runs_agent`;
- `fk_verification_run_table_stats_run`;
- `uq_verification_agents_token_hash`;
- `idx_verification_runs_database_created_at`, `idx_verification_runs_status_created_at`, `idx_verification_runs_agent_status`, `idx_verification_run_table_stats_run`.

A constraint whose action does not change is renamed in place. A reference whose action changes is dropped and added again.

A new index `idx_verification_runs_backup` covers `backup_id`. With `RESTRICT`, PostgreSQL checks every deleted backup row against `verification_runs.backup_id`, and the handler of decision 3 deletes by that column. Without the index, both scan the whole table on every backup deletion, retention pass included.

Alternatives rejected:
- *`verification_verifications`.* It repeats the prefix.
- *Keeping the old constraint and index names.* PostgreSQL names the constraint in a violation's message, and a name that disagrees with its table makes the error harder to trace.

### 3. `backups` publishes `LogicalBackupsDeleting` inside every deletion

`backups/module.go` declares:
- `LogicalBackupsDeleting` with the IDs of the backups about to be deleted;
- `LogicalBackupsDeletingHandler`;
- `SubscribeToLogicalBackupsDeleting`, which `internal/app` calls while it builds the application.

The handlers run in the deletion's transaction, in subscription order, and a handler's error rolls the deletion back.

Both deletion functions of `BL` publish the event through one place. `lockAndListBackupIDsOfDatabase` returns the IDs it locked. `removeDatabaseBackups` then deletes those IDs through `deleteBackupsWithFiles`, the function the per-backup paths already use, and `deleteByDatabase` is removed. `deleteBackupsWithFiles` publishes the event before it deletes the rows, and does nothing when the list is empty.

The event is published after the rows are locked. A verification insert takes a key share lock on its backup's row for the foreign key check. If the deletion already holds the row with `FOR UPDATE`, the insert waits, then fails its reference to the deleted row, and the request creates no verification. If the insert commits first, the deletion's handler sees and deletes it. Publishing before the lock would let an insert land between the handler and the `DELETE`, and `RESTRICT` would then refuse the whole deletion.

Deleting by IDs is complete in both callers of `removeDatabaseBackups`. A database deletion holds the database row, so a new backup's reference to it waits and then fails. A storage change holds the config row that every backup start shares (`BL/database_backups_removal_service.go:21-24`). A backup inserted after the lock therefore never exists while the deletion runs.

Alternatives rejected:
- *Verification subscribes to `DatabaseDeleting` and deletes the database's runs there.* That covers one path of four. A member's deletion, retention and a storage change have no event the verification code could hear.
- *Keeping `CASCADE` to `backups_logical` as an exception.* `docs/architecture.md`, "Data", allows `CASCADE` only inside a module, and `archlint` fails a new cross-module cascade.
- *An after-commit event.* The rows must go before the `DELETE` that `RESTRICT` checks, so the handler has to run inside the transaction.
- *An event that carries the database ID for the per-database paths.* The handler would need two queries and the event two shapes. The IDs are already in hand after the lock.

### 4. The old verification code subscribes until the module takes over

Two handlers are added to the old code. `verification-module` moves both into the module.

- **`OnLogicalBackupsDeleting`** in `FV/runs` deletes `verification_runs WHERE backup_id IN (...)` through `database.Conn(ctx, ...)`. The table statistics go with their runs through the cascade inside the module. A running verification is deleted like any other; its agent gets `410` as today.
- **`OnDatabaseDeleting`** in `FV/config` deletes the database's row of `verification_configs` the same way. The runs of the database are already gone by then: every run has a backup, and the `backups` handler of `DatabaseDeleting` publishes `LogicalBackupsDeleting` for all of the database's backups. The two `DatabaseDeleting` handlers do not depend on each other's order.

`internal/app/wiring.go` registers both: the first in `subscribeToLogicalBackupEvents`, the second in `subscribeToDatabaseEvents`. The test build registers them too, on the bridge modules:
- `FV/config/setup_dependencies_testhelpers.go` adds the `DatabaseDeleting` subscription beside the copy subscription it already holds;
- a new `FV/runs/setup_dependencies_testhelpers.go` declares `SetupTestDependencies`, a `sync.OnceFunc` that subscribes `OnLogicalBackupsDeleting` to `backups_bridge.GetModule()`, and `createTestRouter` calls it beside the two calls it makes today.

The production `SetupDependencies` of `FV/runs/di.go` stays as it is. Production wiring belongs to `internal/app/wiring.go`, and a production file that subscribed to the bridge module would subscribe a second time in the application. Old tests that delete a database or a backup then behave as the application does.

Alternatives rejected:
- *Leaving a database's runs to the `DatabaseDeleting` handler as well.* It would delete what `LogicalBackupsDeleting` already deletes, and the order of two handlers would start to matter.
- *A temporary `CASCADE` from the runs to `databases`.* The reference stays redundant as long as every run has a backup. With `RESTRICT` it still fails loudly if a run without a backup ever appears.

### 5. A `CHECK` on every enum column, after the repairs

`CHECK` constraints cover:
- `status` and `trigger` of `verification_runs`;
- `schedule_type` and `interval_type` of `verification_configs`;
- `send_notifications_on` of `verification_configs`, which becomes `TEXT[] NOT NULL DEFAULT '{}'` with `CHECK (send_notifications_on <@ ARRAY['VERIFICATION_SUCCESS', 'VERIFICATION_FAILED']::TEXT[])`, the form of `accepted_categories` (`20261025180000_move_notifiers_into_module.sql:466`).

The repairs run before the constraints, and each one changes only values a constraint would reject:
- **Unknown status.** The verification becomes `FAILED`, and `fail_message` becomes "The upgrade replaced the unknown status <value>". This is the rule of `20261027140000_move_logical_backups_into_module.sql:50-53` and of the physical migration.
- **Unknown trigger.** The trigger becomes `MANUAL`. The trigger only labels the run and decides one thing: turning automatic verification off cancels the runs that are not manual. A manual label keeps the run in every case. The old label text is lost, which costs nothing the interface shows.
- **Unknown schedule type.** It takes the column's default `AFTER_BACKUP`.
- **Unknown interval type.** It takes the column's default `WEEKLY`, with `04:00` and Sunday (`0`) filled in where the time of day or the weekday is empty, the values of the code's default config (`FV/config/service.go:155-156`). A repaired config that has automatic verification on with the interval type then still passes the old validation.
- **The notification list.** The new column holds the values of the old string that the `CHECK` admits, without duplicates. An empty string becomes an empty array.

Alternatives rejected:
- *Keeping the comma-separated string.* The schema could not restrict its values, and the old and the new code would both keep the split and join hooks.
- *Failing a run with an unknown trigger.* It would turn a pending manual check into a failure for a label nobody reads.
- *A `CHECK` that the time of day is set for the daily, weekly and monthly types.* The backup configs do not have one either (`20261027120000_move_backup_configs_into_module.sql:264-265`), and stored rows with automatic verification turned off may lack the time.

### 6. The old code switches to the new schema and keeps its behavior

The three `TableName` methods, the raw join and the statistics column change. The config model stores the notification list as `pq.StringArray`, as the notifier and database settings repositories do, and its `BeforeSave` and `AfterFind` hooks are removed. The JSON field `sendNotificationsOn` stays a list, and a config without notifications still returns an empty list rather than `null`. Requests may omit the interval type when the interval is inactive. After validation, the old service stores the column default `WEEKLY` for an empty type so these requests satisfy the new constraint. Active interval schedules still require a valid type.

The two cascade tests of Context are removed. The backup one deletes a row with SQL, which `RESTRICT` now refuses. The database one would pass only through the handlers of decision 4, so its name and message would be false. The deletion tests of decision 7 cover both subjects through the API. The comment at `FV/runs/service.go:364-366` drops the cascade and names the deletion of a backup or a database instead.

The test workspace removal deletes `verification_runs` and `verification_configs` of the workspace's databases before the databases, in the style of the physical rows it already deletes.

### 7. Tests

- **Migration test** in `migrations`, beside `logical_backups_migration_test.go`. It applies the migrations before this one and inserts:
  - a config with an unknown schedule type;
  - a config with an unknown interval type and no time of day;
  - a config whose notification string holds both known values, a duplicate and an unknown value;
  - a config with an empty notification string;
  - a run with an unknown status and an earlier fail message;
  - a run with an unknown trigger;
  - a completed run with two table statistics and an agent.

  It applies this migration and checks every repair, the renamed tables and column, every `CHECK`, that deleting a database or a logical backup row that a run references is refused, and that deleting a run still removes its statistics.
- **Deletion tests** in `internal/modules/backups/verification_removal_test.go`, one `t.Run` per scenario of the `backup-verification` delta. Each creates a real backup with `completeRestorableBackup` and a verification through `apptest.EnqueueVerification`. The retention scenario adds the verification of an older backup with SQL, because the old API allows one pending manual verification per database. The checks list the database's verifications with the old `GET /verifications/by-database/:databaseId` through `CallOldAPI`, and use SQL where the old API has no answer, such as the verifications and the config of a deleted database. The member's deletion has a completed, a running and a pending verification. The pending one comes from the API, and the other two are inserted with SQL, because a running one would otherwise need an agent's claim, which this change does not touch. What an agent sees afterwards is the existing `410` of Context.
- **Old tests.** The old verification package tests run against the renamed tables. They need no other change, apart from the two cascade tests and the comment of decision 6. Their deferred database deletions pass once the handlers of decision 4 exist, so tasks.md runs them after those handlers.
- **End-to-end.** The agent scenarios of `internal/agent/e2e` run unchanged.

### 8. The upgrade test writes verification data through the previous release

Before the upgrade, through the previous release's API:
1. the test's account, which is the instance admin because it signed up first, creates an agent with `POST /verification/agents`;
2. the same account, the workspace's owner, saves the logical PostgreSQL database's config with `PUT /verification-config/:databaseId`. Automatic verification is on, the interval type is used, the schedule is weekly on Wednesday at `03:30`, and only `VERIFICATION_SUCCESS` is chosen. Every value apart from the weekly interval type differs from the default config, which is off, runs after each backup, is weekly on Sunday at `04:00` and chooses only `VERIFICATION_FAILED`;
3. it starts a manual verification of the completed logical backup with `POST /verifications/enqueue`.

No agent runs, so the verification stays `PENDING`. The 24-hour limit for unclaimed verifications is far beyond the test's duration.

After the upgrade, through the current code's routes, which are still the old ones in this change, the test checks:
- the agent in `GET /verification/agents` by ID and name;
- the config from `GET /verification-config/:databaseId` with every written value;
- the verification from `GET /verifications/:id` with its backup, the `MANUAL` trigger and the `PENDING` status.

The requests of the previous release go into a new `previous_release_verification_test.go`, and the checks into `current_release_verification_test.go`. `verification-module` later changes only the second file to the new operations.

Alternatives rejected:
- *Running an agent in the upgrade test so the verification completes.* The agent needs Docker inside the test's flow and a restore container. The statistics table is covered by the migration test, and the agent end-to-end tests cover a completed verification on every pull request.

## Risks / Trade-offs

- **A deletion path is missed and a deletion fails.** With `RESTRICT`, a path that skips the event makes every deletion of a verified backup fail with a foreign key error. → Decision 3 sends every path through `deleteBackupsWithFiles`, and decision 7 tests each of the four paths through the API.
- **A failing handler blocks backup deletion.** An error of the verification handler rolls back the deletion, retention included. → This is the same contract as `DatabaseDeleting`. The handler runs one indexed `DELETE`.
- **Old tests delete databases or backups without the handlers.** A test package that builds its own modules without the subscriptions of decision 4 fails during cleanup. → The subscriptions sit in the `testhelpers` setup of the old packages, and task 3.2 runs every old package that creates verifications in its tests.
- **A repair changes data the user cares about.** Only values the code never wrote are repaired. The status repair keeps the old value in `fail_message`. A dropped unknown notification value or a replaced trigger label cannot be read back. → The migration test checks each repair.

## Migration Plan

The migration runs at start-up like every migration, in one transaction:
1. rename the tables and the statistics column, and rename their constraints and indexes;
2. add `idx_verification_runs_backup`;
3. repair the rows of decision 5 and convert the notification column;
4. add the `CHECK` constraints;
5. drop the references to `databases` and `backups_logical` and add them again with `RESTRICT`.

Agents of the previous release keep working: the agent protocol does not change. A rollback means restoring the internal database from a backup taken before the upgrade, because migrations only go forward and a binary refuses a newer schema (`docs/architecture.md`, "Migrations"). The migration's `Down` section restores the old names, the comma-separated column and the cascades for development use, as the earlier module migrations do.
