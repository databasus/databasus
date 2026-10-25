## Context

Why the change is needed is in proposal.md, section Why. This document covers only what shapes the approach.

**The five checks.** `WorkspaceService` answers five permission questions (`internal/features/workspaces/services/workspace_service.go:207`, `:225`, `:247`, `:269`, `:290`). Each one returns `true` for an instance admin, loads the actor's role through `MembershipRepository.GetUserWorkspaceRole` (`internal/features/workspaces/repositories/membership_repository.go:80`) and compares it with a list of roles:

| Method | Allowed workspace roles |
|---|---|
| `CanUserAccessWorkspace` | any member |
| `CanUserManageDBs` | owner, admin, member |
| `CanUserManageWorkspace` | owner, admin |
| `CanUserManageMembership` | owner, admin |
| `CanUserManageAdmins` | owner |

`CanUserAccessWorkspace` also returns the role, and reports a failed lookup as "no access" with a `nil` error (`:217-220`). No caller reads the returned role. Two more checks compare roles by hand: `DeleteWorkspace` lets only the owner or an instance admin delete (`workspace_service.go:163-173`), and `TransferOwnership` does the same through the repository (`membership_service.go:295-302`). `validateCanManageMembership` (`membership_service.go:353`) asks `CanUserManageAdmins` when the requested role is admin and `CanUserManageMembership` otherwise.

**Who calls them.** Outside `workspaces`, the old code asks only two of the questions. `CanUserAccessWorkspace` is called 23 times and `CanUserManageDBs` 32 times, in 14 files of `databases`, `storages`, `notifiers`, `backups/backups/services`, `backups/config/logical`, `backups/config/physical`, `restores`, `healthcheck/attempt`, `healthcheck/config`, `verification/config` and `verification/runs`. Two controllers ask directly (`internal/features/storages/controller.go:229`, `internal/features/notifiers/controller.go`). Each caller holds a `*workspaces_services.WorkspaceService`, except `databases`, which declares a `workspaceService` interface with both methods (`internal/features/databases/interfaces.go:26`); the interface and the fake below are not counted as calls. Its controller test replaces `CanUserManageDBs` with a fake (`internal/features/databases/controller_test.go:2240`). Each caller turns `false` into its own error, such as `ErrInsufficientPermissionsToManageStorage`, and its controller maps that error to 403 or 400.

**The role type.** `users_enums.WorkspaceRole` (`internal/features/users/enums/workspace_role.go:3`) is the last type in `features/users/enums`. It occurs 379 times, most of them in the tests of `workspaces`, `databases`, `storages`, `notifiers`, `backups` and `healthcheck`.

**Rules in force.** `internal/modules` is in the target layout. `archlint` lets only `module.go` live in a module's root package and lets other packages import only that root. The old code is outside the layout, and `archlint` does not check its imports. Names follow the root `AGENTS.md`: a predicate reads as a question, and a name states the intent, not the mechanism. The change answers to the root `AGENTS.md` and `backend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- One function decides which workspace role allows which action, and every workspace permission check in the code goes through it.
- No code names `users_enums.WorkspaceRole`. The workspace role is `workspaces.Role`.
- Every check keeps its rule, and every existing test passes with only the type and call changes.

**Non-Goals:**

- `Authorize` with its 404 and 403 answers. It needs the module's repositories and arrives with them in `workspaces-module`.
- Any other part of the module. `module.go` holds only what the old code needs to stop using the five methods and the old enum.

## Decisions

### 1. `module.go` starts with `Role` and `Action`

```go
type Role string

const (
	RoleOwner  Role = "WORKSPACE_OWNER"
	RoleAdmin  Role = "WORKSPACE_ADMIN"
	RoleMember Role = "WORKSPACE_MEMBER"
	RoleViewer Role = "WORKSPACE_VIEWER"
)

func (r Role) IsValid() bool

type Action string

const (
	ActionView              Action = "VIEW"
	ActionEditContent       Action = "EDIT_CONTENT"
	ActionUpdateWorkspace   Action = "UPDATE_WORKSPACE"
	ActionManageMembers     Action = "MANAGE_MEMBERS"
	ActionManageAdmins      Action = "MANAGE_ADMINS"
	ActionTransferOwnership Action = "TRANSFER_OWNERSHIP"
	ActionDeleteWorkspace   Action = "DELETE_WORKSPACE"
)
```

The role values stay as stored, so the column, the API and the frontend do not change. `IsValid` replaces the method of the deleted enum, which the member controller uses to reject an unknown role. The constant `RoleMember` keeps the stored word. The glossary already uses "member" for any user in a workspace, so it gets a row for the workspace role that names all four values and says that `WORKSPACE_MEMBER` is the role and "member" alone is any user in the workspace, as decision 1 of `users-actor` did for the global role. "Admin" gets the same treatment: the glossary maps the word `admin` to the instance admin, so a row for the workspace admin (`WORKSPACE_ADMIN`) says that `admin` alone in code and text means the instance admin, and that `workspaces.RoleAdmin` and `ActionManageAdmins` name workspace admins.

An action names what the actor is about to do, not a role threshold. "Edit content" covers the databases, storages and notifiers of a workspace and everything attached to them: backup configs, backups, restores, health checks and verification configs. The glossary gets a row for workspace content, so the word means the same thing in every module.

The architecture describes `module.go` as the place of `Module`, `Deps` and `New` as well. They arrive in `workspaces-module` together with the code they build. A module root that holds only types for one change is valid for `archlint`, as `users-actor` showed.

*Rejected: name actions by the lowest role that may do them, such as `ActionForAdmins`.* A later rule for a single action, for example letting admins transfer ownership, would then need a new constant and an edit at every call site of the old one. An action named by intent changes only in the table.

*Rejected: keep the five questions as five methods on the future module.* Every new kind of permission would add a method to `Module`, and the role comparison would stay spread over five bodies. `docs/architecture.md` already names the one function with an action.

*Rejected: fewer actions, with `UpdateWorkspace` and `ManageMembers` merged and the three owner actions merged.* Their rules happen to coincide today, but they are different decisions, and a merged action would hide that from the reader of a call site.

### 2. One table maps roles to actions, and `IsActorAllowed` reads it

The old service gets one method in place of the five and `GetUserWorkspaceRole`:

```go
func (s *WorkspaceService) IsActorAllowed(
	ctx context.Context,
	actor users.Actor,
	workspaceID uuid.UUID,
	action workspaces.Action,
) (bool, error)
```

An instance admin is allowed every action. For anyone else the method loads the actor's role in the workspace, answers `false` when there is none and otherwise looks up the action in a table of the old service package:

| Action | Owner | Admin | Member | Viewer |
|---|---|---|---|---|
| `ActionView` | yes | yes | yes | yes |
| `ActionEditContent` | yes | yes | yes | no |
| `ActionUpdateWorkspace` | yes | yes | no | no |
| `ActionManageMembers` | yes | yes | no | no |
| `ActionManageAdmins` | yes | no | no | no |
| `ActionTransferOwnership` | yes | no | no | no |
| `ActionDeleteWorkspace` | yes | no | no | no |

The table reproduces the rules of the Context table, of `DeleteWorkspace` and of `TransferOwnership` exactly. A failed role lookup returns the error. The call sites already return an error from the check, so a database failure reaches their controllers as an internal error instead of a refusal.

The table moves into the module's internal code in `workspaces-module`, where `Authorize` reads it. It lives in the old package for this change because the old code cannot import a module's `internal/` packages, and exporting a rule function from `module.go` would put a second way of checking access into the module's contract.

*Rejected: return the role and let the caller compare.* That is the current state, and it is how the comparisons spread.

*Rejected: return a typed error instead of `bool`.* `Authorize` will return the error, with its 404 and 403. The old call sites map refusals to their own errors and statuses, and they would have to tell a refusal from a failure with `errors.Is` at each of 55 places. A `bool` keeps every call site's shape and its behavior.

### 3. Each call site maps its method to one action

| Old call | New call |
|---|---|
| `CanUserAccessWorkspace(ctx, id, actor)` | `IsActorAllowed(ctx, actor, id, workspaces.ActionView)` |
| `CanUserManageDBs(ctx, id, actor)` | `IsActorAllowed(ctx, actor, id, workspaces.ActionEditContent)` |
| `CanUserManageWorkspace` in `UpdateWorkspace` | `ActionUpdateWorkspace` |
| `CanUserManageMembership` in `RemoveMember` and `validateCanManageMembership` | `ActionManageMembers` |
| `CanUserManageAdmins` in `RemoveMember` and `validateCanManageMembership` | `ActionManageAdmins` |
| the role comparison in `DeleteWorkspace` | `ActionDeleteWorkspace` |
| the role comparison in `TransferOwnership` | `ActionTransferOwnership` |

The parameter order puts the actor first, as `Authorize(ctx, actor, workspaceID, action)` will. A call that read the unused role, `canView, _, err :=`, becomes `isAllowed, err :=`, and the local variable takes the `is` prefix the naming rules ask for. The `workspaceService` interface of `databases` declares `IsActorAllowed` in place of the two methods, and the fake in its controller test implements it. The DTOs of `workspaces` and the membership model take `workspaces.Role`.

*Rejected: switch the old callers to an interface now, so that `workspaces-module` swaps the implementation without touching them.* The callers also use `GetWorkspaceByID`, whose replacement is decided in `workspaces-module`. The field types change there in one place per caller, which is cheaper than designing that interface twice.

### 4. The type constants change by a mechanical replacement

`users_enums.WorkspaceRoleOwner`, `...Admin`, `...Member` and `...Viewer` become `workspaces.RoleOwner`, `RoleAdmin`, `RoleMember` and `RoleViewer` in every file, and `users_enums.WorkspaceRole` becomes `workspaces.Role`. The replacement is one `gofmt -r` rule per name, followed by `goimports`. The diff of the test files then holds nothing else.

### 5. The role table gets its own test

`IsActorAllowed` is the one place the rules now live, so a test in `features/workspaces/services` creates a workspace with one user of each role through the test helpers. It checks every action for every role against the table of decision 2, and checks that an instance admin who is not a member is allowed every action and that a user who is not a member is allowed none. A last case calls `IsActorAllowed` with a cancelled context and checks that it returns an error, not `false`, since that path is the one behavior this change alters. The existing tests of each feature keep checking the refusals of their own operations.

## Risks / Trade-offs

- [A call site gets the wrong action] → The replacement maps a method name to an action, so the review checks one table instead of 55 bodies, and the tests of each feature that check a viewer's refusal still run.
- [A database failure in the role lookup now answers 400 or 500 where it answered 403] → Only an outage causes it, and a refusal hid the outage from the logs before.
- [Wide diff over old code that later milestones rewrite] → The edit is mechanical in every file outside `workspaces`, and the tests of each touched package run before handing over.
