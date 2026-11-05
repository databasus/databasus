## Why

The audit log belongs to no domain, yet it lives in the feature `features/audit_logs`, which imports `users` while `users` imports it back. To show who did something, each read joins the `users` table, so a log entry depends on the account still existing in its current form. `docs/architecture.md` moves audit events into `lib/audit`, where each event keeps a snapshot of its author. Milestone 4 of the backend rewrite plan does that before `users` moves. The same milestone brings the shared pieces the first module needs: a clock in `Deps`, operation attributes in the logging context and `background.Runner` for periodic jobs. This change is the second of four. It comes after `lib-crypto-mailer`, and `users-actor` and `users-module` follow it.

## What Changes

- `lib/audit` appears with `New(Deps)`. It records an event through `database.Conn`, so an event written inside a transaction belongs to that transaction. It sends every event to the audit log sink as before, and it deletes events older than one year. An event stores its author's user ID, email and name as they were when the action happened. It also stores the workspace ID and the message.
- The table `audit_logs` becomes `lib_audit_events`:
  - `user_id` is renamed to `author_user_id`;
  - `author_email` and `author_name` are added and filled from `users` for every existing event that has an author;
  - `created_at` becomes `TIMESTAMPTZ`;
  - the foreign key to `users` is dropped, and a `CHECK` requires the three author columns to be all set or all empty.
- The old audit service stays as a bridge for its about 50 callers. `WriteAuditLog` keeps its signature, looks up the author's email and name by the user ID it receives and records the event through `lib/audit`. The callers do not change.
- The old endpoints `/audit-logs/global`, `/audit-logs/users/{userId}` and `/workspaces/{id}/audit-logs` keep their requests and responses. They read the author's email and name from the event instead of joining `users`. The frontend does not change.
- All three audit log views move to the `workspaces` module in milestone 5, the global one included. Every view shows workspace names, which only `workspaces` can read, while `users` sits below it. `docs/architecture.md` changes the owner of the global view from `users` to `workspaces`.
- `lib/clock` appears with the system clock and a manual clock for tests. `lib/audit` takes the time of an event and the retention cutoff from it.
- `background.Runner` appears. A job is a function of one pass that receives the current time. The runner puts `job_name` and `job_id` into the context of each pass and never overlaps two passes of one job. A pass that returns an error or panics is logged, and the next pass runs on schedule. A second `Start` panics. Its first job is the audit event cleanup, which replaces the hand-written loop `AuditLogBackgroundService`. The other 12 loops stay on `background.Group` until their modules move.
- `apptest` gets the manual clock and an explicit run of a job's pass, as `docs/testing.md` describes the test application. Jobs never run on their own in a test application.
- `logging.With(ctx, ...)` puts attributes into the context, and the log handler adds them to every record. It replaces `ContextWithRequestID`, `ContextWithUserID` and their getters. Tests that read a context attribute capture the log record instead.
- `internal/middleware` moves into `lib/httpserver`. The access log no longer imports `features/verification/agents`: the agent middleware adds `agent_id` through `logging.With`, the same way the users middleware adds `user_id`.
- **BREAKING** for the schema: `audit_logs` is renamed and loses its foreign key, as described above.
- **BREAKING** for readers of the log file: an audit record carries the constant message `audit event recorded`, and the event text moves into the `message` attribute. The log linter requires a constant message.

## Capabilities

### New Capabilities

- `audit-log`: what an audit event keeps about its author, how long events are kept, and what an event written before the upgrade shows.

### Modified Capabilities

None.

## Out of scope

- `Actor`: the change `users-actor`. The `users` module: the change `users-module`.
- Taking the author of an event from the request context. Every writer passes its author explicitly, and the context arrives when a module that knows only the `Actor` first writes events.
- Moving the three audit log views into the `workspaces` module, describing them in the contract and paging them with a cursor: milestone 5.
- Rewriting the audit messages or giving events a machine-readable action. The messages stay free text.
- Moving the other 12 background loops to `Runner`. Each moves with its module, as the table of temporary relaxations in `temp/refactor/plan.md` says.
- Reporting job results to the health endpoint: milestone 13, when the health endpoint moves into `app`. `docs/architecture.md` already describes that report as part of the runner, so `temp/refactor/plan.md` records the gap in its table of temporary relaxations.
- Writing the old audit events inside the transactions of the old code. The old code opens no transactions in the context, so its events stay separate writes until each module moves.

## Impact

- Code: new `backend/internal/lib/audit`, `backend/internal/lib/clock` and the runner in `backend/internal/lib/background`; `backend/internal/lib/logging` (context attributes) and `backend/internal/lib/httpserver` (the moved middleware); `backend/internal/app` (wiring, the runner, the job, the server middleware); `backend/internal/app/apptest`; `backend/internal/features/audit_logs` (bridge and reads); `backend/internal/features/users/middleware` and `backend/internal/features/verification/agents` (`logging.With`); `backend/internal/util/logger`; deleted `backend/internal/middleware`.
- Migrations: one migration renames and repairs the audit table.
- Running instances: existing events keep their messages, and events with an author get that author's current email and name as their snapshot. Events whose author was removed already have no author, because the old foreign key set it to null. They stay without one.
- Tests: `internal/app/console_test.go`, `internal/features/verification/agents/controller_test.go` and the audit tests read the renamed table. The case of `test_once_protection_test.go` for the old cleanup loop is deleted with the loop.
- Documents: `docs/architecture.md` (the audit paragraph says the snapshot holds the author's ID, email and name and that `workspaces` serves every audit log view; the module table moves the global audit log from `users` to `workspaces`; the logging section says audit records identify the author), `docs/glossary.md` (the term "audit author"), `temp/refactor/plan.md` (the milestone 4 checklist and the table of temporary relaxations).
- Rules the change answers to: the root `AGENTS.md` and `backend/AGENTS.md`.
