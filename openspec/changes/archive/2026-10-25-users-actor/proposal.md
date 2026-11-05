## Why

Every old feature receives the GORM model of the signed-in user, `*users_models.User`. The users middleware puts it into the Gin context, 74 handlers take it out with `GetUserFromContext`, and the services of 14 packages accept it, with about 95 references to the type. `docs/architecture.md` says a module never receives a `users` model: the request's user is an `Actor` with the user ID and the global role. The model moves under `internal/modules/users/internal` in the change `users-module`, where Go's `internal/` rule makes it unreachable for the old code. Before that, the signed-in user has to stop being the model. From the signed-in user, the old code reads only `ID` and `Role` outside `users`, apart from two permission checks and one name in `workspaces`, so the switch is mechanical. `workspaces` also loads other accounts through `UserService.GetUserByEmail` and `GetUserByID` and reads their email and name. Those lookups keep returning the model here, and `users-module` replaces them with calls to the module. Doing it in a change of its own keeps that wide diff apart from the move of the module. This change is the third of four in milestone 4 of the backend rewrite plan. It comes after `lib-crypto-mailer` and `audit-in-lib`, and `users-module` follows it.

## What Changes

- `internal/modules/users/module.go` appears with the first part of the module's contract:
  - the global role `users.Role` with `RoleAdmin` and `RoleMember`;
  - `users.Actor` with `UserID` and `Role`;
  - `users.WithActor`, which puts an actor into a context;
  - `users.GetActor`, which reads it back.
  The module's `Module`, `Deps` and `New` arrive in `users-module`.
- The users middleware puts the `Actor` into the request context instead of the model into the Gin context. `RequireRole` takes a `users.Role` and reads the actor.
- The 74 handlers of the old code read the actor with `users.GetActor(ctx.Request.Context())`. Their services take `users.Actor` instead of `*users_models.User`, in `workspaces`, `databases`, `storages`, `notifiers`, `backups`, `restores`, `healthcheck`, `verification` and `audit_logs`. The audit controller stops reading `ctx.MustGet("user")`.
- Two permission checks of the users model become questions to the old settings service: whether an actor may invite users and whether it may create workspaces. The model methods `CanInviteUsers`, `CanCreateWorkspaces`, `CanManageUsers` and `CanUpdateSettings` are deleted. The invitation email of `workspaces` loads the inviting account's name by the actor's user ID.
- Inside `features/users`, the controllers pass the actor, and the services that need more than the ID and the role load the account themselves.
- `users_enums.UserRole` is deleted, and every reference uses `users.Role`. `docs/glossary.md` gets the term "global role" with its values `ADMIN` and `MEMBER`, kept apart from "member" of a workspace. The change covers about 550 references in the tests of other packages, mostly the role passed to `users_testing.CreateTestUser`. `users_enums` keeps `WorkspaceRole`, which moves in milestone 5, and `UserStatus`, which moves with the module.
- The behavior of the product, the HTTP API and the schema do not change.

## Capabilities

### New Capabilities

None. The change sets `skip_specs`, because no observable behavior changes.

### Modified Capabilities

None.

## Out of scope

- The `users` module itself: its `Module`, `Deps`, `New`, repositories, contract operations, the move of the model and the migration. That is the change `users-module`.
- Taking the audit author from the context. `audit-in-lib` resolves the author by user ID, and every writer passes it explicitly.
- Moving `WorkspaceRole` and the workspace permission checks: milestone 5.
- Changing which roles may do what. Each check keeps its current rule and only reads the role from the actor.

## Impact

- Code: new `backend/internal/modules/users/module.go`; `backend/internal/features/users` (middleware, controllers, services, the model's permission methods, `enums`); the handlers and services of `features/workspaces`, `features/databases`, `features/storages`, `features/notifiers`, `features/backups`, `features/restores`, `features/healthcheck`, `features/verification` and `features/audit_logs`; `backend/internal/app` where it names a role.
- Tests: about 70 test files change `users_enums.UserRole...` to `users.Role...`. Four test files outside `users` build a `*users_models.User` for a service call and build an `Actor` instead. No test changes what it checks.
- Linters: `internal/modules/users` is in the target layout, so `module.go` passes every linter and `archlint` at once.
- Documents: `docs/glossary.md` (the term "global role"), `temp/refactor/plan.md` (the milestone 4 checklist).
- Rules the change answers to: the root `AGENTS.md` and `backend/AGENTS.md`.
