## Context

Why the change is needed is in proposal.md, section Why. This document covers only what shapes the approach.

**Writing.** `AuditLogService.WriteAuditLog` (`internal/features/audit_logs/service.go:24`) does three things:

- sends the message to the audit logger with `user_id` and `workspace_id` (`service.go:32`);
- inserts the row through `AuditLogRepository.Create` (`repository.go:17`);
- logs an insert error and returns nothing.

`Create` takes no context on purpose: the row of an action that already happened must be written even if the client disconnects. The service has 49 call sites in the old code, counted as the calls of `WriteAuditLog` and of the private `writeAuditLog` helpers of `users` outside tests. They pass an `AuditEntry` with the message, the optional user ID and the optional workspace ID. Ten services get it as `*audit_logs.AuditLogService` from `GetAuditLogService()`, for example `internal/features/databases/di.go`. The three services of `users` get it through `SetAuditLogWriter` in `audit_logs.SetupDependencies` (`audit_logs/di.go:49`), behind the `users_interfaces.AuditLogWriter` interface. The tests of `users` replace it with `users_testing.InstallAuditLogRecorder`.

**Reading.** `GetGlobal`, `GetByUser` and `GetByWorkspace` (`repository.go:25`, `:61`, `:99`) join `users` for `user_email` and `user_name` and `workspaces` for `workspace_name`. They page with `LIMIT`/`OFFSET` and an optional `beforeDate`. The frontend shows the author as "name (email)" in all three views and the workspace name in the global view (`frontend/src/features/settings/ui/AuditLogsComponent.tsx`, `features/workspaces/ui/WorkspaceAuditLogsComponent.tsx`, `features/users/ui/UserAuditLogsSidebarComponent.tsx`). An account can change its name and its email address (`root-admin-account` spec), so today an old entry shows the author's current values.

**Schema.** `audit_logs` (`migrations/20251031145001_add_system_audit_logs.sql`) has `user_id` with `ON DELETE SET NULL`, `workspace_id` without a foreign key, and `created_at TIMESTAMP` written as `time.Now().UTC()`. The product never deletes an account, but `users_testing.DeleteTestUser` does.

**Cleanup.** `AuditLogBackgroundService.Run` (`background_service.go`) is a hand-written ticker loop: every hour (`:35`) it deletes rows older than 365 days, with a `hasRun` guard (`:19`). `internal/app/background_loops.go:32` starts it through `background.Group` together with 12 other loops.

**Logging context.** `lib/logging` carries two hardcoded context values, `request_id` and `user_id` (`internal/lib/logging/request_context.go`). `fanOutHandler.Handle` copies them into every record (`fan_out_handler.go:40`). `internal/middleware/request_id.go:19` and `internal/features/users/middleware/middleware.go:46` set them. The access log (`internal/middleware/access_log.go:46`) imports `features/verification/agents` to add `agent_id`, so a package outside the features depends on a feature.

**Time.** No clock exists. `docs/architecture.md` ("Long-running work") requires time that drives behavior to come from a clock in `Deps`. `docs/testing.md` ("The test application") requires a test application with a clock the test moves and with jobs whose passes the test runs explicitly.

**Rules in force.** `internal/lib` and `internal/app` are in the target layout. `lib` imports no `internal/features` package. A repository holds `*gorm.DB` and queries through `database.Conn`, which the change `lib-crypto-mailer` adds. Log messages are constant, and values travel as attributes. The change answers to the root `AGENTS.md` and `backend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- `lib/audit` is the only code that writes and deletes audit events. The old read endpoints are the only temporary exception: they read the table directly, see decision 3.
- No old caller of `WriteAuditLog` changes.
- `Runner`, `lib/clock` and `logging.With` exist with their first consumers and with the behavior `docs/architecture.md` describes.

**Non-Goals:**

- Listing events through `lib/audit`. That method arrives with its first consumer, the audit log views of the `workspaces` module in milestone 5.
- Taking the author from the request context. Here the author is always explicit. The context arrives when a module that knows only the `Actor` first writes events, which is `workspaces` in milestone 5 at the earliest.

## Decisions

### 1. The event stores the author's ID, email and name

`audit.Event` carries `Author *audit.Author` with `UserID`, `Email` and `Name`, plus the optional `WorkspaceID` and the `Message`. `lib/audit` writes the three author fields to `author_user_id`, `author_email` and `author_name`. A `CHECK` requires all three to be set or all three to be empty. There is no foreign key to `users`.

`docs/architecture.md` says the snapshot holds the ID and the email of the event's "actor". This change adds the name and calls the snapshot the event's author, because the glossary gives "actor" to the request's user ID and global role. `docs/glossary.md` gets the term "audit author", and the architecture's audit paragraph and its logging section use it. All three views show "name (email)", and the global view has to keep doing so for an author whose account no longer exists.

*Rejected: snapshot the email only, and let each view resolve the name by user ID.* The workspace view would resolve names through `users` and the global view through its own table. That is two lookups for a value that should not change after the action anyway. An author who no longer exists would show an email without a name.

*Rejected: keep the foreign key with `SET NULL`.* `lib` would depend on a module's table, which `docs/architecture.md` forbids. Removing an account would also erase the author from history, which the snapshot exists to prevent.

The workspace name stays out of the snapshot. The view resolves it, as decision 3 describes.

### 2. The old service becomes a bridge that resolves the author by ID

`AuditLogService.WriteAuditLog` keeps its signature. When the entry has a user ID, the bridge reads that account's email and name through the old `users` repository. It then calls `auditLog.Record` with `context.WithoutCancel(ctx)`, which keeps today's guarantee that a disconnecting client does not lose the row. An error is logged as today. An entry whose user ID matches no account is recorded without an author and logs a warning. That can only happen for an account deleted by a test between the action and the write.

*Rejected: change the 49 call sites to pass the author.* They only have the user ID, and each module rewrites its calls when it moves. *Rejected: take the author from the request context in this change.* The console commands and the sign-in operations write events with no signed-in user in the context. The bridge would need the lookup for them anyway, so a second mechanism adds nothing here.

The old package keeps the `users` import it has today. The `audit_logs ↔ users` cycle disappears in `users-module`: the users services write through `lib/audit`, so `users` no longer imports the old package, and the bridge reads accounts through the `users` module.

### 3. The old read endpoints read `lib_audit_events` directly until their owners move

The three old repository methods keep their SQL shape: offset paging, `beforeDate` and the join with `workspaces` for the workspace name. They change the table name and read the author from the snapshot columns instead of joining `users`. Their requests and responses do not change, so the frontend does not change.

This breaks the rule that only `lib/audit` touches its table. `temp/refactor/plan.md` records the break in its table of temporary relaxations, and milestone 5 removes it for all three views.

All three views move to the `workspaces` module in milestone 5. Today `docs/architecture.md` gives the global view to `users`, but every view shows the workspace name of each event, and `users` sits below `workspaces` in the module order, so it cannot read workspace names. `workspaces` can apply each view's access rule: the global view needs only the global role from the `Actor`, a user's view needs the actor's own ID or the admin role, and the workspace view needs the membership that `workspaces` owns. This change updates the audit paragraph and the module table of `docs/architecture.md` accordingly.

*Rejected: `users` serves the global view and asks for workspace names through an interface in its `Deps` that `app` wires to `workspaces`.* That is a dependency on a higher module through the back door, and `docs/architecture.md` allows only events for that direction. *Rejected: keep the workspace name in the snapshot.* Most writers know only the workspace ID, so each of them would have to load the name first. *Rejected: drop the workspace name from the global view.* The administrator would lose the one column that says where an action happened.

*Rejected: a `List` with cursor paging in `lib/audit` now, with the three old endpoints and their screens moved to the cursor.* Milestone 5 puts the views into the contract and rewrites the endpoints and the screens again, so the cursor work would be done twice. *Rejected: a temporary offset `List` in `lib/audit`.* That puts code that must be deleted into the target layout, where nothing marks it as temporary.

### 4. `lib/audit` takes a database, a clock and the audit logger

`audit.New(audit.Deps{DB, Clock, Logger, AuditLogger})` returns a `*audit.Log` with two methods. Callers name the value `auditLog`, after the audit log it writes:

- `Record(ctx, Event) error` inserts through `database.Conn`, so the event joins a transaction in the context. It sends a record with the constant message `audit event recorded` and the attributes `message`, `author_user_id` and `workspace_id` to the audit logger. `CreatedAt` comes from the clock.
- `DeleteExpired(ctx, now) (int64, error)` deletes events older than `now` minus one year.

The audit logger keeps its level bypass and its `log_type=audit` mark (`internal/lib/logging/logging.go:60`), so the `log-export` requirement that audit records survive the loss of the database still holds. The record is sent before the insert, as today. If the surrounding transaction rolls back, the file keeps a record of an attempt that left no event. That is acceptable for a forensic copy, and it is how the file behaves now. The author's email reaches the record only through the event text, and the log handler masks addresses there as before.

*Rejected: a package function `audit.Record(ctx, db, event)` as in the cloud project.* The time of the event must come from a clock, and the record must also go to the audit logger. A function would take both as parameters at every call site.

### 5. `background.Runner` runs jobs; the test application runs passes itself

`background.NewRunner(RunnerDeps{Logger, Clock}, jobs...)` takes `Job{Name, Interval, Run func(ctx, now time.Time) error}`. The behavior:

- `Start(ctx)` starts one loop per job. The first pass runs right away, and each next pass starts one interval after the previous one ended, so passes of one job never overlap. A second `Start` panics.
- Each pass gets `job_name` and `job_id` through `logging.With`. An error or a panic is logged with the stack, and the loop goes on. A pass interrupted by the end of the context is logged at info level.
- `RunPass(ctx, name) error` runs one pass synchronously and returns its error. `apptest` exposes it as `RunJobPass(t, name)`, and a test application never calls `Start`.
- `Wait(ctx)` returns when every loop has returned. The shutdown waits for it next to the `background.Group` of the old loops.

`app.Serve` builds the runner with the job `audit_events_cleanup`: every hour it calls `auditLog.DeleteExpired(ctx, now)` and logs the count. The old loop, its `hasRun` guard and its entry in `background_loops.go` are deleted, and so are `AuditLogService.CleanOldAuditLogs` (`internal/features/audit_logs/service.go:142`) and `AuditLogRepository.DeleteOlderThan` (`repository.go:149`). Their tests in `background_service_test.go` become tests of `DeleteExpired`.

The runner takes its own `RunnerDeps`, because `background.Deps` of `Group` has no use for a clock, and `exhaustruct` would make every `Group` caller supply one.

The runner is modeled on the cloud project, minus its Prometheus metrics and its job states. The states feed the health endpoint, which moves into `app` in milestone 13, and the job states arrive with that consumer.

*Rejected: keep the loop and only give it a clock.* `backend/AGENTS.md` forbids hand-written ticker loops and `hasRun` guards, and the runner is the place that milestone 13 extends to every job.

### 6. `lib/clock` is an interface with a system clock and a manual clock

The interface `clock.Source` has `Now() time.Time` and always returns UTC. It is named for its role, because `clock.Clock` would stutter. `clock.NewSystem()` returns a source that reads the real time. `clock.NewManual(start)` returns a `*clock.Manual` with `Advance(d)` and `Set(t)`, safe for concurrent use. Fields that hold a source keep the noun: `Deps.Clock`, `app.Environment.Clock` and `apptest.App.Clock`. `app.Environment` gets a `Clock`: `cmd/databasus` passes the system clock, and `apptest` passes a manual clock that starts at the real time and is exposed as `App.Clock`.

The runner's timers use real time: an interval in a test does not elapse, because a test application never starts the runner. Only the `now` passed to a pass comes from the clock.

*Rejected: `Now func() time.Time` in `Deps`, as in the cloud project.* An interface lets `apptest` hand one manual clock to the whole application and move it, and a named type documents the dependency better than a bare function.

### 7. `logging.With` replaces the two hardcoded context values

`logging.With(ctx, args ...any) context.Context` appends attributes to those already in the context. `fanOutHandler.Handle` adds them to every record. `ContextWithRequestID`, `ContextWithUserID`, `GetRequestID`, `GetUserID` and their wrappers in `util/logger` are deleted. Their two production callers switch to `logging.With`: the request ID middleware (`internal/middleware/request_id.go:19`) and the users middleware (`internal/features/users/middleware/middleware.go:46`). The agent middleware sets nothing today, because the access log reads the agent itself. It starts adding `agent_id` through `logging.With`, the same way the users middleware adds `user_id`.

No getter replaces the deleted ones. Production code has no reason to read an attribute back, and tests that did so read the record instead. Only the fan-out handler adds the context attributes, so a test puts its capturing sink behind the fan-out handler of `lib/logging` and reads the attribute from the captured record. A capturing handler passed straight to `slog.New`, as `backuper_test.go` in `features/backups/backups/backuping/logical` does today, would see none of them, so that test changes to the same setup. That affects the tests of `lib/logging` (`redaction_test.go`), `lib/apierror` (`hooks_test.go`), the moved middleware tests (`request_id_test.go`, `access_log_test.go`), `features/audit_logs` (`service_test.go`) and `features/backups/backups/backuping/logical` (`backuper_test.go`).

The middleware of `internal/middleware` moves into `lib/httpserver`: the request ID, the access log and the cache headers. The access log no longer imports `features/verification/agents`, and the `agent_id` of a request reaches its record through the context. The access log reads the context of `ctx.Request` after `ctx.Next()`, so values that later middleware adds are visible, as the `user_id` is today.

*Rejected: keep `ContextWithUserID` next to `With`.* Two ways to attach an attribute invite the third. The hardcoded keys exist only because `With` did not.

## Risks / Trade-offs

- [The bridge adds a query per audit write] → The old code writes events on user actions only, at most a few per request. The lookup goes away as each module moves and passes its author.
- [Old tests that delete an account and then write an event for it] → The bridge records such an event without an author instead of failing, as decision 2 says.
- [A tool that reads the log file by the message of an audit record] → The event text moves into the `message` attribute, and the proposal marks this as breaking. The record keeps `log_type=audit`, which is how the `log-export` spec says to find audit records.
- [The migration runs on large audit tables] → One `UPDATE ... FROM users` fills the snapshot, and the type change rewrites the table once. Retention keeps the table to one year of events. Migrations run before the server accepts requests.

## Migration Plan

One migration, in its transaction:

1. rename `audit_logs` to `lib_audit_events` and `user_id` to `author_user_id`;
2. drop `fk_audit_logs_user_id`;
3. add `author_email TEXT` and `author_name TEXT`, and fill them from `users` where `author_user_id` is set;
4. set `author_user_id` to null on rows whose account is missing; with the old foreign key there are none, but a database restored from a dump is not guaranteed to have one;
5. change `created_at` to `TIMESTAMPTZ` with `USING created_at AT TIME ZONE 'UTC'`;
6. add the `CHECK` on the three author columns;
7. rename the three indexes to the `lib_audit_events` prefix.

The upgrade test writes events through the API of the previous release and checks after the upgrade that each event shows its author.
