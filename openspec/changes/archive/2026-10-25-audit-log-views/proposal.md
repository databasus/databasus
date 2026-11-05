## Why

The three audit log views, the global one, a user's and a workspace's, are still served by the old `features/audit_logs`. It reads `lib_audit_events` directly and joins `workspaces` in SQL, past both `lib/audit`, which owns the table, and the `workspaces` module, which owns the names. The views page with `offset`, which skips or repeats events when new ones arrive while a user scrolls, and the global view counts the whole table on every page. `docs/architecture.md` gives every audit log view to `workspaces`, because each shows the name of an event's workspace and only `workspaces` can read it. This change is the last of three in milestone 5 of the backend rewrite plan. It needs the `workspaces` module and `Authorize` from `workspaces-module`.

## What Changes

- `lib/audit` gains a read of events with an optional workspace filter and an optional author filter, newest first, paged with a cursor over the time and the ID of the event.
- One operation `listAuditEvents` (`GET /audit-events`) enters `backend/api/openapi.yaml` under the tag `workspaces`, and the `workspaces` module serves it. It takes the filters `workspaceId` and `authorUserId`, `limit` and `after`, and returns `items` and `nextCursor`. Each event carries its message, time, author snapshot and workspace with the workspace's current name.
- The access rules stay as they are and answer in the contract's way. With `workspaceId`, the actor needs to see the workspace: a member of any role or an instance admin, and a non-member gets 404 `not_found`. With only `authorUserId`, the actor must be that user or an instance admin. Without a filter, the actor must be an instance admin. A refusal is 403 `permission_denied`.
- The module reads the names of the events' workspaces in one batch from its own table. An event of a deleted workspace keeps the workspace ID and shows no name, as today.
- The views no longer show a total. The global view stops counting the table, and the screens show "load more" without "N of M", as the user list does since `users-module`.
- One migration replaces the three single-column indexes of `lib_audit_events` with indexes that serve each filter in the order of the views.
- The old read code is deleted: the controller, the repository and the DTOs of `features/audit_logs` and its three routes, including the workspace route that `workspaces-module` moved there. `features/audit_logs` keeps only the bridge that old features write audit events through.
- The frontend calls `listAuditEvents` through the generated client for the settings screen, the user sidebar and the workspace screen, and loads the next page by `nextCursor`, as the user list of the administration screen does. The hand-written types of `entity/audit-logs` are deleted.
- **BREAKING** for users' scripts: `GET /audit-logs/global`, `GET /audit-logs/users/{userId}` and `GET /workspaces/{workspaceId}/audit-logs` are replaced by `GET /audit-events` with filters. `offset`, `beforeDate` and `total` are gone, the list field is `items`, the author and the workspace are nested objects, and errors are `Problem` bodies.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `audit-log`: who may read which audit log view, the order and paging of the views, and how an event shows its workspace.

## Out of scope

- Writing audit events. The old features keep writing through the `features/audit_logs` bridge, and each module writes through `lib/audit` when it moves.
- Keeping the workspace name in the event. A deleted workspace's events show no name, as today; a snapshot of the name would be a change of `lib/audit` and its table.
- Searching or filtering events by text, time or action.
- Changing who may read which view.
- Moving the user sidebar out of the administration screen or showing a user their own log anywhere new.

## Impact

- Backend: `backend/internal/lib/audit` (the read and its tests); `backend/internal/modules/workspaces` (the operation, its service and handler); `backend/api/openapi.yaml`; `backend/internal/features/audit_logs` reduced to the write bridge; `backend/internal/app` (routes); `backend/e2e/upgrade` (the audit checks of the current release).
- Frontend: `frontend/src/entity/audit-logs`, `features/settings/ui/AuditLogsComponent.tsx`, `features/users/ui/UserAuditLogsSidebarComponent.tsx`, `features/workspaces/ui/WorkspaceAuditLogsComponent.tsx`, the six dictionaries in `frontend/src/shared/i18n/locales`, and tests for the settings view.
- Migrations: one migration on `lib_audit_events` that replaces three indexes.
- Running instances: audit events keep their data. The index swap runs once at startup and takes time proportional to the events of the last year.
- Documents: `docs/glossary.md` does not change, because "audit event" and "audit author" are already there. `temp/refactor/plan.md` (the checklist and two rows of the table of temporary relaxations).
- Rules the change answers to: the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
