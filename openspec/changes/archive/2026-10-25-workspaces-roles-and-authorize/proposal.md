## Why

Every old feature asks `WorkspaceService` its own permission question. Five methods (`CanUserAccessWorkspace`, `CanUserManageDBs`, `CanUserManageWorkspace`, `CanUserManageMembership`, `CanUserManageAdmins`) are called 63 times, 55 of them from 14 files of other old features, and each method repeats the same role lookup and its own role comparison. The workspace roles live in `features/users/enums`, although they belong to `workspaces`. `docs/architecture.md` says that only `workspaces.Authorize(ctx, actor, workspaceID, action)` checks access to a workspace and that the module's types are declared in its `module.go`. Before `workspaces-module` moves the module, the old code has to ask one question with a named action and use the role type of `workspaces`. Doing it in a change of its own keeps that wide mechanical diff apart from the move of the module. This change is the first of three in milestone 5 of the backend rewrite plan; `workspaces-module` and `audit-log-views` follow it.

## What Changes

- `internal/modules/workspaces/module.go` appears with the first part of the module's contract:
  - the workspace role `workspaces.Role` with `RoleOwner`, `RoleAdmin`, `RoleMember` and `RoleViewer`, keeping the stored values `WORKSPACE_OWNER`, `WORKSPACE_ADMIN`, `WORKSPACE_MEMBER` and `WORKSPACE_VIEWER`;
  - the action `workspaces.Action`, one constant for each thing a member can be allowed to do: view the workspace, edit its content, update the workspace, manage members, manage admins, transfer ownership and delete the workspace.
  The module's `Module`, `Deps`, `New` and `Authorize` arrive in `workspaces-module`.
- One table maps each workspace role to the actions it allows, and the instance admin is allowed every action. `WorkspaceService.IsActorAllowed(ctx, actor, workspaceID, action)` reads the actor's role in the workspace and answers from that table. It replaces the five `CanUser*` methods and `GetUserWorkspaceRole`, which are deleted.
- The 55 call sites in `databases`, `storages`, `notifiers`, `backups`, `restores`, `healthcheck` and `verification`, and the checks inside `workspaces` itself, call `IsActorAllowed` with the action that matches their current method. Each call site keeps its own error and status.
- `users_enums.WorkspaceRole` is deleted, and its 379 occurrences use `workspaces.Role`. The `features/users/enums` package disappears, because the workspace role is the last type in it.
- A failed role lookup is returned as an error instead of being read as "no access". Today `CanUserAccessWorkspace` turns a database failure into a refusal (`internal/features/workspaces/services/workspace_service.go:217-220`), so a database outage looks like a permission problem.
- The behavior of the product, the HTTP API and the schema does not change otherwise.

## Capabilities

### New Capabilities

None. The change sets `skip_specs`, because no observable behavior changes. The workspace access rules enter the specs in `workspaces-module`, where their answers change to 404 and 403.

### Modified Capabilities

None.

## Out of scope

- The `workspaces` module itself: `Module`, `Deps`, `New`, `Authorize` with its 404 and 403 answers, the repositories, the contract operations, the event on workspace deletion and the migration. That is the change `workspaces-module`.
- The audit log views. That is the change `audit-log-views`.
- Changing which role may do what. Every check keeps its current rule. The known hole that lets a workspace admin give someone the owner role through the member operations is closed by `workspaces-module`, where the contract defines which roles a request may set.
- Changing the error texts and statuses of the old features. Each keeps its own until its module moves.
- The workspace lookup `GetWorkspaceByID`, which the old code keeps calling for workspace names until `workspaces-module`.

## Impact

- Code: new `backend/internal/modules/workspaces/module.go`; `backend/internal/features/workspaces` (services, repositories, models, DTOs, test helpers); the services and controllers that check workspace access in `features/databases`, `features/storages`, `features/notifiers`, `features/backups/backups/services`, `features/backups/config/logical`, `features/backups/config/physical`, `features/restores`, `features/healthcheck/attempt`, `features/healthcheck/config`, `features/verification/config` and `features/verification/runs`; deleted `backend/internal/features/users/enums`.
- Tests: about 15 test files change `users_enums.WorkspaceRole...` to `workspaces.Role...`. The fake workspace service of `features/databases/controller_test.go` implements `IsActorAllowed` instead of `CanUserManageDBs`. A new test of the role table checks every role against every action. No other test changes what it checks.
- Linters: `internal/modules/workspaces` is in the target layout, so `module.go` passes every linter and `archlint` at once.
- Documents: `docs/glossary.md` (workspace role, workspace admin, owner, workspace content, workspace action), `temp/refactor/plan.md` (milestone 5 splits into three sub-items, and the first is checked).
- Rules the change answers to: the root `AGENTS.md` and `backend/AGENTS.md`.
