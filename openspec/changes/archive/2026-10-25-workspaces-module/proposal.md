## Why

`workspaces` is the second module in the order of `docs/architecture.md`, and every later module checks access through its `Authorize`. Today it is an old feature: global getters, controllers that answer errors as 400 with the text of the internal error, no description of its operations, a `MembershipRepository` that joins the `users` table, and workspace deletion through listeners that each write outside any shared transaction. A failure halfway through a deletion leaves the workspace partly emptied, and a crash during an ownership transfer can leave two owners. The member operations also let a workspace admin give someone the owner role, because they accept every role value, so an admin can grant the powers reserved for the owner. Milestone 5 of the backend rewrite plan moves the feature into `internal/modules/workspaces` with its operations in the contract. This change is the second of three. It needs `workspaces.Role`, `workspaces.Action` and the single permission check from `workspaces-roles-and-authorize`.

## What Changes

- The module `internal/modules/workspaces` appears with `module.go`, `Deps`, `New` and its code under `internal/`. It owns workspaces, members, workspace roles, invitations to a workspace and ownership transfer. Its services take the time from the clock, open transactions through the `database.Transactor` from `Deps` and write audit events through `lib/audit` in the same transaction.
- `module.go` grows from the types of `workspaces-roles-and-authorize` into the module's contract: `Authorize(ctx, actor, workspaceID, action)`, a batch lookup of workspaces by ID, and the event `WorkspaceDeleting` with a way to subscribe to it. `Authorize` answers 404 `not_found` when the actor is not a member of the workspace or the workspace does not exist, and 403 `permission_denied` when the actor's role does not allow the action. An instance admin passes every check.
- 10 operations enter `backend/api/openapi.yaml` under the tag `workspaces`, and the module serves them from a generated strict server: list, create, read, update and delete a workspace, transfer its ownership, and list, add, change and remove its members. Members are a resource of their own, `/workspace-members`, listed with a `workspaceId` filter, as the api-design skill asks for resources of a workspace. Both lists page with a cursor.
- Every error of these operations is a `Problem` with a code. The contract gains `already_member`, `cannot_change_own_role`, `owner_requires_transfer` and `workspace_not_empty`, each with status 409.
- A request may set only the roles `WORKSPACE_ADMIN`, `WORKSPACE_MEMBER` and `WORKSPACE_VIEWER`. The owner role changes hands only through an ownership transfer. Changing the role of an admin, or making a member an admin, needs the permission to manage admins, which only the owner and an instance admin have. Today a workspace admin can demote another admin through the role change, while removing that admin is refused.
- Deleting a workspace runs in one transaction. The module publishes `WorkspaceDeleting` inside it, and `internal/app` subscribes the old features in one wiring file, in this order: `databases` refuses with `workspace_not_empty` while the workspace has databases, `notifiers` deletes the workspace's notifiers, and `storages` drains pending file deletions and deletes the workspace's storages. Any failure rolls back the whole deletion. The listener interface `WorkspaceDeletionListener` and its registration through `di.go` are deleted.
- Adding a member by an unknown address invites the user and adds the member in one transaction, and the invitation email goes out after the commit. Transferring ownership demotes the old owner to admin and promotes the new one in one transaction.
- `users.Module` gains a batch lookup of users by ID. The member list reads the email and name of its members through it and no longer joins the `users` table.
- The table `workspace_memberships` becomes `workspaces_members`. The migration adds a `CHECK` on the role, after setting any role outside the four values to `WORKSPACE_VIEWER`, and a unique index that allows one owner per workspace, after demoting every owner but the earliest of a workspace to admin. The references from `databases`, `storages` and `notifiers` to `workspaces`, and from the members to `users`, change from `ON DELETE CASCADE` to `ON DELETE RESTRICT`. Three indexes that duplicate others are dropped.
- The old code reaches the module through a bridge: `features/workspaces` keeps only the installed `workspaces.Module`, the type `workspaces_bridge.Access` with `IsActorAllowed` and `GetWorkspaceByID` for the old callers, and the test helpers behind the `testhelpers` tag. The old features keep their own errors and statuses; their move to `Authorize` comes with their own modules. The workspace audit log route moves to `features/audit_logs` unchanged until `audit-log-views`.
- The test helpers of the old code create and remove workspaces and members with SQL instead of the old routes, as the helpers of `users` already do, and lose their `router` parameter. Removing a test workspace first deletes its databases, notifiers and storages, standing in for the cascade that `RESTRICT` replaces.
- The frontend calls the 10 operations through the generated client and types. The hand-written types of `entity/workspaces` and the `WorkspaceRole` enum of `entity/users` are deleted, `workspaceMembershipApi` becomes `workspaceMemberApi`, the member screens address a member by its ID, the role selectors offer no owner role, and every new code gets a translation in the six interface dictionaries.
- The tests of `workspaces` become module API tests through `apptest` and the contract-checking client. `apptest` gets helpers that create a workspace and add a member through the API.
- **BREAKING** for users' scripts: the paths of the member operations change from `/workspaces/memberships/{workspaceId}/members...` to `/workspace-members`, a member is addressed by its member ID instead of the user ID, the ownership transfer names the new owner by member ID instead of email, the workspace list becomes a page with `items` and `nextCursor` instead of `workspaces`, and errors are `Problem` bodies. A request with the owner role is refused with 400 `invalid_request`.
- **BREAKING** for the schema: one table is renamed, a `CHECK` and a unique index are added, and four references change from `CASCADE` to `RESTRICT`.

## Capabilities

### New Capabilities

- `workspace-access`: which workspace role allows which action, what an instance admin may do, and how a refused request answers.
- `workspace-members`: adding, inviting, changing and removing members, and transferring ownership, with the rule of one owner per workspace.
- `workspace-deletion`: what deleting a workspace removes, when it is refused, and that it happens completely or not at all.

### Modified Capabilities

- `workspace-invitations`: the invited user and the member are recorded together, and a failed invitation email does not undo them.

## Out of scope

- The audit log views, including the workspace's own log. The old route keeps serving it until `audit-log-views`.
- Moving the old features to `Authorize` and its 404 and 403 answers. Each moves with its own module, milestones 6 to 13.
- After-commit handlers for events. The only effect outside the database in this deletion, the drain of pending file deletions, deletes files that are deleted whatever the outcome, and `docs/architecture.md` gains that exception; see design.md decision 7.
- Deleting a workspace's databases together with the workspace. The current rule stays: the databases are deleted first.
- Keeping the workspace name in audit events. An event of a deleted workspace shows no name, as today.
- Changing which role may do what, apart from the two holes this change closes: granting the owner role through member operations and changing an admin's role without the permission to manage admins.

## Impact

- Backend: new `backend/internal/modules/workspaces`; `backend/internal/modules/users` (the batch user lookup and the invitation tests that called the old member route); `backend/api/openapi.yaml`; `backend/internal/lib/apierror` (four codes); `backend/internal/app` (wiring, routes, the subscriptions of `WorkspaceDeleting`) and `backend/internal/app/apptest`; `backend/internal/features/workspaces` reduced to the bridge and the test helpers; the services, `di.go` and deletion handlers of `features/databases`, `features/notifiers` and `features/storages`; the callers of `GetWorkspaceByID` in `features/backups/backups/backuping`; `features/audit_logs` (the moved workspace log route); `features/users/testing` (a deleted test user stops being a member first); the 54 test files and 9 non-test helper files that use the workspace test helpers, listed in design.md Context; `backend/cmd/seed_dev`; `backend/e2e/upgrade` (the workspace checks of the current release).
- Frontend: `frontend/src/entity/workspaces`, `frontend/src/entity/users/model` (the role enum and its labels), the screens of `features/workspaces`, `widgets/main` and the transfer dialogs of `features/databases`, `features/storages` and `features/notifiers`, the six dictionaries in `frontend/src/shared/i18n/locales`, and tests for the member screen.
- Migrations: one migration renames the members table, repairs and constrains roles and owners, and changes four references to `RESTRICT`.
- Running instances: workspaces and members keep their data. A workspace with more than one owner keeps the earliest owner, and the others become admins. A member with a role outside the four values becomes a viewer.
- Documents: `docs/glossary.md` (member ID, ownership transfer), `docs/architecture.md` (one sentence in "Modules": an effect that stays correct after a rollback may run inside an event handler), `temp/refactor/plan.md` (the checklist and the table of temporary relaxations).
- Rules the change answers to: the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
