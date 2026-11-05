## 1. The module root

- [x] 1.1 Add `backend/internal/modules/users/module.go` with `Role`, `RoleAdmin`, `RoleMember`, `Actor`, `WithActor` and `GetActor`, as design.md decision 1 describes, and a test file in the same package that checks that `GetActor` returns what `WithActor` stored and reports `false` on a context without an actor. Verify with `mise run backend:test-package ./internal/modules/users/...` and `mise run backend:lint`.

## 2. The users code

- [x] 2.1 Type the users model's `Role` field as `users.Role`, delete `users_enums.UserRole`, and switch every reference inside `features/users` and in `internal/app` to `users.Role`. Verify with `mise run backend:test-package ./internal/features/users/... ./internal/app/... ./cmd/...`.
- [x] 2.2 Make the middleware put the actor into the request context and `RequireRole` read it, per design.md decision 2, and delete `GetUserFromContext`. Switch the users controllers to `users.GetActor` and the users services to `users.Actor`, loading the account where design.md decision 4 says so. Add `CanInviteUsers` and `CanCreateWorkspaces` to the settings service and delete the four permission methods of the model, per design.md decision 3, with tests in `features/users` for an admin, a member whose setting allows the action and a member whose setting forbids it. Keep the `IsRootAdmin` checks of the management operations on the loaded account, per design.md decision 4. Verify with `mise run backend:test-package ./internal/features/users/...`.

## 3. The other old features

- [x] 3.1 Switch `features/workspaces` to the actor: handlers read it with `users.GetActor`, services take `users.Actor`, the two permission checks call the settings service, and the invitation email loads the inviter's name, per design.md decision 3. Verify with `mise run backend:test-package ./internal/features/workspaces/...`.
- [x] 3.2 Switch the handlers and services of `features/databases`, `features/storages` and `features/notifiers` to the actor. Verify with `mise run backend:test-package ./internal/features/databases/... ./internal/features/storages/... ./internal/features/notifiers/...`.
- [x] 3.3 Switch the handlers and services of `features/backups` and `features/restores` to the actor. Verify with `mise run backend:test-package ./internal/features/backups/... ./internal/features/restores/...`.
- [x] 3.4 Switch the handlers and services of `features/healthcheck`, `features/verification` and `features/audit_logs` to the actor, including the two `ctx.MustGet("user")` reads of the audit controller and the admin group of verification agents. Verify with `mise run backend:test-package ./internal/features/healthcheck/... ./internal/features/verification/... ./internal/features/audit_logs/...`.

## 4. Tests and checks

- [x] 4.1 Replace `users_enums.UserRoleAdmin` and `users_enums.UserRoleMember` with `users.RoleAdmin` and `users.RoleMember` in every test file, as design.md decision 5 describes, and make `users_testing.CreateTestUser` take a `users.Role`. Build an `Actor` in the four test files outside `users` that build a `*users_models.User` for a service call. Verify with `rg -n 'users?_enums\.UserRole' backend/` printing nothing and `mise run backend:test-package ./internal/features/... ./internal/app/... ./cmd/...`.
- [x] 4.2 Check the goal of design.md: `rg -n 'users?_models\.User\b' backend/internal --glob '!backend/internal/features/users/**'` prints nothing.
- [x] 4.3 Add the row "Global role | global role, `ADMIN`, `MEMBER` | A user's role on the whole instance: `ADMIN` makes the user an instance admin, and `MEMBER` grants nothing beyond the user's workspace memberships. The word "member" alone means a member of a workspace." to `docs/glossary.md`, as design.md decision 1 describes. Verify with `rg -n 'Global role' docs/glossary.md`.
- [x] 4.4 Update the milestone 4 checklist in `temp/refactor/plan.md` to mark this change. Verify with `rg -n 'users-actor' temp/refactor/plan.md`.
- [x] 4.5 Run `mise run lint` and fix what it reports. Verify that it exits with status 0.

## Change verification

```sh
mise run backend:test-package ./internal/modules/users/... ./internal/features/... ./internal/lib/httpserver/... ./internal/app/... ./cmd/...
```
