## Context

Why the change is needed is in proposal.md, section Why. This document covers only what shapes the approach. It assumes `workspaces-module` has landed: the module, `Authorize`, `GetWorkspaces`, and the workspace audit log route moved into `features/audit_logs`.

**The old views.** `features/audit_logs` serves `GET /audit-logs/global` and `GET /audit-logs/users/:userId` (`backend/internal/features/audit_logs/controller.go:15-19`) and, after `workspaces-module`, `GET /workspaces/:workspaceId/audit-logs`. The repository reads `lib_audit_events` with a `LEFT JOIN workspaces` for the name (`repository.go:13-26`). The comment on the repository calls it the one exception to `lib/audit` owning its table. The service pages with `limit` from 1 to 1000, default 100, `offset` and an optional `beforeDate` (`service.go:48-53`). The global view also runs `COUNT(*)` over the whole table (`repository.go:100`), and the other two report the page length as their total. The rules are: the global view only for an instance admin, a user's view for that user or an instance admin, and a workspace's view for anyone who may view the workspace.

**`lib/audit`.** `Log` records events and deletes expired ones (`backend/internal/lib/audit/audit.go:41`, `:58`), and its comment says it is the only code that writes and deletes them. It has no read. The table has three single-column indexes, `idx_lib_audit_events_created_at`, `_workspace_id` and `_author_user_id` (`backend/migrations/20261025095000_move_audit_logs_to_lib_audit_events.sql:57-68`). The rename used `IF EXISTS`, so an instance may lack any of them.

**Frontend.** `features/settings/ui/AuditLogsComponent.tsx` (the global view), `features/users/ui/UserAuditLogsSidebarComponent.tsx` and `features/workspaces/ui/WorkspaceAuditLogsComponent.tsx` load more events by `offset` (`AuditLogsComponent.tsx:74`, `UserAuditLogsSidebarComponent.tsx:72`) and show "Loaded: N of M" and "All logs loaded (N total)" (`AuditLogsComponent.tsx:224`, `UserAuditLogsSidebarComponent.tsx:153`, `:184`). `features/users/ui/UsersComponent.tsx:54-138` already loads more by `nextCursor`.

**Rules in force.** `docs/architecture.md` gives every audit log view to `workspaces` and lets modules use a `lib` table only through its package. The api-design skill pages with `limit`, `after`, `items` and `nextCursor` and offers no totals. The operation lives in the module's strict server, and its tests go through `apptest`. The change answers to the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- `lib_audit_events` is read only through `lib/audit`, and every view is served by `workspaces` through one contract operation.
- Paging is stable while new events arrive, and a page costs one indexed range read, without a count.

**Non-Goals:**

- New filters, search or export. The filters are the two the views need.

## Decisions

### 1. `lib/audit` reads a page of events

```go
// EventListQuery leaves a filter nil to not filter by it. After is the NextCursor of the previous
// page, empty for the first page.
type EventListQuery struct {
	WorkspaceID  *uuid.UUID
	AuthorUserID *uuid.UUID
	Limit        int
	After        string
}

type RecordedEvent struct {
	ID          uuid.UUID
	Author      *Author
	WorkspaceID *uuid.UUID
	Message     string
	CreatedAt   time.Time
}

type EventPage struct {
	Events []RecordedEvent
	// NextCursor is empty on the last page.
	NextCursor string
}

func (l *Log) ListEvents(ctx context.Context, query EventListQuery) (EventPage, error)
```

The query struct follows `UserListQuery` of the users module, because the read takes two filters and two page parameters. The query orders by `created_at DESC, id DESC` and continues after the cursor with a row comparison, `(created_at, id) < (?, ?)`. Because the time and the ID of an event never change, an event written while a client pages sorts before the cursor and never shifts the following pages. The cursor encodes the last event's time and ID. A cursor that does not decode answers a field error on `after`, as the user list of `users-module` does. The query reads one row more than `limit` to know whether a next page exists. The comment on `Log` changes to say it is the only code that touches the table.

*Rejected: keep `offset` and `beforeDate`.* An offset shifts when an event is written, so a user who scrolls sees an event twice or misses one, and the api-design skill rules it out. `beforeDate` alone cannot break ties between events of the same microsecond.

*Rejected: let the `workspaces` repository read `lib_audit_events`.* That is the exception this change removes. `docs/architecture.md` lets a module use a `lib` table only through its package.

### 2. One operation with filters, and the service applies each view's rule

`listAuditEvents` is `GET /audit-events` with the optional query parameters `workspaceId` and `authorUserId`, `limit` from 1 to 100 with default 50, and `after`. It answers 200 `AuditEventPage` with `items` and `nextCursor`, and documents 400, 401, 403, 404 and the `default` Problem. The operation's description documents the fixed order, newest first by `createdAt` with ties broken by `id`, as the api-design skill asks of a list, and the three access rules of the table below. It has no access mark. The service picks the rule from the filters:

| Filters | Rule |
|---|---|
| `workspaceId`, with or without `authorUserId` | `Authorize(ctx, actor, workspaceID, ActionView)`: 404 for a non-member, nothing else is refused |
| only `authorUserId` | the actor is that user or an instance admin, otherwise 403 `permission_denied` |
| none | the actor is an instance admin, otherwise 403 `permission_denied` |

`AuditEvent` has `id`, `message`, `createdAt`, `author` and `workspace`. `author` is `nullable` and holds `userId`, `email` and `name` from the event's snapshot. `workspace` is `nullable` and holds `id` and a `nullable` `name`.

*Rejected: three operations, one per view, with `x-access: admin` on the global one.* The workspace view would have to nest under `/workspaces/{workspaceId}`, which the api-design skill rules out for a workspace's resources, or take the same filter as the others under a second path. The user view would need its own path for a rule that is a single comparison. One list with filters is how the skill lists resources, and the contract describes the three rules in the operation.

*Rejected: let an instance admin's filter be required for the global view.* A missing filter is the global view, and only instance admins read it, so a required filter would add a parameter without adding a rule.

### 3. Workspace names come from one batch of the module's own table

The service reads a page through `lib/audit`, collects the distinct workspace IDs of its events and reads them in one query of the workspace repository, the one behind `GetWorkspaces`. An event whose workspace no longer exists gets `workspace` with its ID and a `null` name. The page size bounds the batch at 100 IDs.

*Rejected: `LEFT JOIN workspaces` inside `lib/audit`.* A `lib` package would read a module's table, which the ownership rule of `docs/architecture.md` forbids, and the event would again depend on the module above it.

### 4. The indexes follow the three views

The migration drops `idx_lib_audit_events_created_at`, `idx_lib_audit_events_workspace_id` and `idx_lib_audit_events_author_user_id` with `IF EXISTS`, and creates:

- `idx_lib_audit_events_created_at_id` on `(created_at DESC, id DESC)`, for the global view and for the hourly cleanup, which deletes by `created_at`;
- `idx_lib_audit_events_workspace_id_created_at_id` on `(workspace_id, created_at DESC, id DESC)`;
- `idx_lib_audit_events_author_user_id_created_at_id` on `(author_user_id, created_at DESC, id DESC)`.

Each view then reads one range of one index and stops after `limit + 1` rows. The down migration restores the three single-column indexes.

*Rejected: keep the single-column indexes.* For a filter, PostgreSQL would read every event of the workspace or the user and sort them, on every page, for up to a year of events.

*Rejected: `CREATE INDEX CONCURRENTLY`.* goose runs each migration in a transaction, where `CONCURRENTLY` is not allowed, and the instance applies migrations before it accepts requests, so no write waits on the build.

### 5. The old read code goes away

The controller, the repository, the DTOs and the read methods of the `features/audit_logs` service are deleted, with the three routes and their tests. `features/audit_logs` keeps `WriteAuditLog`, `InstallAuditLog` and the author lookup, which the old features write through until their modules move. The rows of the table of temporary relaxations in `temp/refactor/plan.md` about old endpoints reading `lib_audit_events` directly and about the workspace audit log route in `features/audit_logs` are crossed out.

### 6. The frontend loads more by cursor and shows no total

`entity/audit-logs/api` calls `listAuditEvents` through the generated client, and its hand-written models are deleted. The three screens load the next page by `nextCursor` and stop when it is absent, in the way of `UsersComponent`. They show the author and the workspace from the nested objects, with the workspace's ID when its name is `null`. The dictionary key `auditLogs.loadedOfTotal` is deleted, and `auditLogs.allLoaded` loses its count in all six dictionaries. A screen test on `mockApi` covers the global view loading a second page and stopping at the last.

### 7. Tests

The tests in `internal/modules/workspaces` call `listAuditEvents` through `apptest` and the contract-checking client, for every scenario of the `audit-log` delta. The scenario of a new event while paging writes the event through an operation that records one, such as renaming a workspace, between the two page reads. A test of `lib/audit` covers the cursor across events of the same time, ordered by ID, and the field error for a cursor that does not decode. The upgrade test reads the audit events of the current release through `listAuditEvents` as the administrator and follows `nextCursor` to the last page, so it checks every event and not only the first 100.

## Risks / Trade-offs

- [The index build takes a lock on `lib_audit_events` at startup] → Events older than one year are deleted every hour, so the table holds one year at most, and the build happens before the server accepts requests.
- [A script that read `total` breaks] → The proposal marks the operation change as breaking for the release notes.
- [A workspace view filtered by an author shows only that author's events in the workspace] → That is what the two filters mean together, and the screens do not combine them.

## Migration Plan

1. The migration runs at startup before the server accepts requests.
2. The upgrade is one-way, as `docs/architecture.md` says for every migration: the previous binary refuses to start on the new schema. Recovery from a failed upgrade means restoring the internal database from a backup taken before it. The down section of the migration, which restores the three single-column indexes, serves development only.
