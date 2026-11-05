## Context

Why the change is needed is in proposal.md, section Why. This document covers only what shapes the approach.

**How the user reaches the old code.** `AuthMiddleware` (`internal/features/users/middleware/middleware.go:14`) parses the token, loads the account through `UserService.GetUserFromToken` and calls `ctx.Set("user", user)`. `GetUserFromContext` (`middleware.go:85`) casts it back to `*users_models.User`. `RequireRole` (`middleware.go:53`) compares `user.Role` with a `users_enums.UserRole`. It guards six routes of `users` and the admin group of verification agents (`internal/features/verification/agents/controller.go:23`). The audit controller reads `ctx.MustGet("user")` directly (`internal/features/audit_logs/controller.go:26`, `:54`).

**What the old code reads from the model.** Outside `features/users`, production code reads `user.ID` 48 times and `user.Role` 12 times. Beyond that it uses three things, all in `workspaces`:

- `creator.CanCreateWorkspaces(settings)` (`internal/features/workspaces/services/workspace_service.go:47`);
- `addedBy.CanInviteUsers(settings)` (`membership_service.go:89`);
- `addedBy.Name` in the invitation email (`membership_service.go:112`).

The four permission methods live on the model (`internal/features/users/models/user.go:34-55`).

**Roles in tests.** `users_testing.CreateTestUser(ctx, role)` is called 656 times, and the role comes from `users_enums`. There are about 700 references to `users_enums.UserRole...` in about 70 test files.

**Rules in force.** `internal/modules` is in the target layout. `archlint` lets only `module.go` live in a module's root package and lets other packages import only that root. The old code is outside the layout, and `archlint` does not check its imports. The change answers to the root `AGENTS.md` and `backend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- No code outside `features/users` names `users_models.User` or `users_enums.UserRole`, and every handler and service of the old code gets the signed-in user as an `Actor`.
- `workspaces` still receives loaded models from the lookups `UserService.GetUserByEmail` and `GetUserByID` and reads their email and name. That dependency goes away in `users-module`, which replaces the lookups with `Module` calls.
- Every check keeps its rule, and every existing test passes with only the type changes.

**Non-Goals:**

- Moving anything else of `users` into the module. `module.go` holds only what the old code needs to stop using the model.

## Decisions

### 1. `module.go` starts with `Role`, `Actor` and the two context functions

```go
type Role string

const (
	RoleAdmin  Role = "ADMIN"
	RoleMember Role = "MEMBER"
)

func (r Role) IsValid() bool

type Actor struct {
	UserID uuid.UUID
	Role   Role
}

func WithActor(ctx context.Context, actor Actor) context.Context
func GetActor(ctx context.Context) (Actor, bool)
```

`IsValid` replaces the method of the deleted `users_enums.UserRole`, which the role change and the console account command use to reject an unknown role. The context key is an unexported type in `module.go`. The role values stay `ADMIN` and `MEMBER`, so the column, the token claim and the API do not change. The constant keeps the API's word, `RoleMember`, because client code keeps the domain words of the API. The glossary already uses "member" for a user in a workspace, so it gets a row for the global role that names both values and says that "member" alone means the workspace term. The users model's `Role` field takes the type `users.Role`.

The architecture describes `module.go` as the place of `Module`, `Deps` and `New` as well. They arrive in `users-module` together with the code they build. A module root that holds only types for one change is valid for `archlint`, and nothing builds the module yet.

*Rejected: name the constant `RoleUser` while the stored value stays `MEMBER`.* The code would then use one word and the API, the column and the frontend another for the same value.

*Rejected: put `Actor` into a `lib` package now and move it to the module later.* `docs/architecture.md` names `Actor` as a type of `users`, and a `lib` home would be one more move with every import edited twice. *Rejected: keep `Actor` in the old `features/users` package until `users-module`.* That package disappears there, so the 74 handlers and the services would change their import a second time.

### 2. The actor travels in the request context, not in the Gin context

The middleware calls `users.WithActor` on `ctx.Request.Context()`. Handlers already pass that context into services, and the generated strict handlers of `users-module` see only that context, not the Gin one. Reading the actor from the same place in old and new handlers means the middleware of `users-module` can serve both without a second key.

*Rejected: keep `ctx.Set("user", ...)` with an `Actor` value.* The strict handlers could not read it, so `users-module` would change all 74 call sites again.

### 3. Permission checks become questions to the settings service

`SettingsService` of the old users code gets `CanInviteUsers(ctx, actor) (bool, error)` and `CanCreateWorkspaces(ctx, actor) (bool, error)`. Each loads the settings and applies today's rule: an admin always may, and a member may when the matching setting allows it. `workspaces` and `UserService.InviteUser` call them. `CanManageUsers` and `CanUpdateSettings` have no caller outside `users` and are replaced there by `actor.Role == users.RoleAdmin`. The four model methods are deleted.

The invitation email of `workspaces` needs the inviter's name. `MembershipService.AddMember` loads the inviting account through `UserService.GetUserByID(ctx, actor.UserID)`, which already exists.

*Rejected: put the two checks on `Actor`.* The answer depends on instance settings that only `users` reads, so a method on a plain value would have to take the settings, and every caller would load them itself as today.

### 4. The services of `features/users` take the actor and load the account when they need it

`GetCurrentUserProfile` and `SendTestEmail` need the email, the name or the status of the account. The management operations and the admin listing need `IsRootAdmin` of the acting account (`management_service.go:74`, `:108`, `:154`, `admin_listing.go:55`), because only the bootstrap administrator may change other administrators. Each of them loads the account by the actor's user ID through the repository, and a check on `IsRootAdmin` stays a check on the loaded account, not a role comparison. The other services take only the ID or the role. The profile operation now reads the account twice per request, once in the middleware and once in the service. That cost goes away when `users-module` builds the middleware and the profile handler on one service.

*Rejected: let the middleware keep the model in the Gin context for the users controllers only.* Two ways to read the signed-in user would live side by side, and the users controllers are rewritten in `users-module` anyway.

### 5. The test role constants change by a mechanical replacement

`users_enums.UserRoleAdmin` and `users_enums.UserRoleMember` become `users.RoleAdmin` and `users.RoleMember` in every test file, with the import added. `users_testing.CreateTestUser` takes a `users.Role`. The replacement is one `gofmt -r` rule per constant, followed by `goimports`. The diff of the 67 test files then holds nothing else, and it can be reviewed as a replacement.

## Risks / Trade-offs

- [A handler reads the actor where the middleware did not run] → `GetActor` returns `false`, and the handler answers 401 as `GetUserFromContext` does today. The routes keep their groups, so the same routes run the middleware as before.
- [The checks of the change run `cmd/databasus` under the race detector, and GORM v1.26 races when two goroutines parse related model schemas for the first time] → GORM puts a schema into its cache before it parses the schema's relations, and that parsing writes to the fields of the related schemas without a lock. The background loops preload related models at startup, so `App.New` parses the schemas of every model the old code preloads before any request or loop runs (`internal/app/model_schemas.go`). Later queries only read the cache.
- [Wide diff over old code that later milestones rewrite] → The edit is mechanical in every file outside `users` and `workspaces`, and the tests of each touched package run before handing over.
