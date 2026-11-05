## Context

Why the change is needed is in proposal.md, section Why. This document covers only what shapes the approach. It assumes `workspaces-roles-and-authorize` has landed: `workspaces.Role`, `workspaces.Action` and the role table behind `WorkspaceService.IsActorAllowed`.

**Operations.** `features/workspaces` serves eleven routes from two controllers. `WorkspaceController` (`internal/features/workspaces/controllers/workspace_controller.go:25-30`) creates, lists, reads, updates and deletes workspaces and serves the workspace audit log. `MembershipController` (`membership_controller.go:23-27`) lists, adds, changes and removes members under `/workspaces/memberships/:id` (`membership_controller.go:20`), addressing a member by the user ID, and transfers ownership by the new owner's email. Errors go out as `{"error": err.Error()}`, with 403 for a few permission errors and 400 for everything else.

**Roles in requests.** `AddMember` accepts any valid role, the owner included (`membership_controller.go:77`). `validateCanManageMembership` (`internal/features/workspaces/services/membership_service.go:353`) looks only at the requested role: a request for admin needs the owner, any other request needs an admin. A workspace admin can therefore add a second owner, or turn a member into the owner through `ChangeMemberRole` (`:179`), and can demote another admin to viewer, while `RemoveMember` refuses to remove that admin.

**Writes without transactions.** No service of `workspaces` opens a transaction. `CreateWorkspace` inserts the workspace and the owner's member row separately. `AddMember` invites the user (`membership_service.go:106`), sends the email (`:118`) and only then inserts the membership (`:128`), so a failed insert leaves an invited user who is not a member and an email that points nowhere. `TransferOwnership` promotes the new owner (`:328`) and then demotes the old one (`:336`). A crash in between leaves two owners. The invitation email reads the public address through `config.GetEnv()` (`:385`).

**Deletion.** `DeleteWorkspace` (`internal/features/workspaces/services/workspace_service.go:163`) calls `OnBeforeWorkspaceDeletion` on every registered listener (`:180`) and then deletes the row. The listeners are registered by the `SetupDependencies` of three features (`internal/features/databases/di.go:50`, `internal/features/notifiers/di.go:48`, `internal/features/storages/di.go:71`), which `internal/app/wiring.go:21` calls in the order databases, notifiers, storages. The `databases` listener refuses while the workspace has databases (`internal/features/databases/service.go:600`). The `notifiers` listener deletes each notifier (`internal/features/notifiers/service.go:416`). The `storages` listener drains the pending file deletions of each storage and deletes it (`internal/features/storages/service.go:90`), with `context.Background()` because the listener gets no context (`:98`). Each repository opens its own transaction on the global connection, so a failure leaves the earlier deletions in place. Then `ON DELETE CASCADE` on `databases`, `storages` and `notifiers` (`backend/migrations/20260105170956_add_cascade_delete_to_workspace_fks.sql`, `20260127132617_add_cascade_delete_for_backups_and_databases.sql`) removes whatever a listener left.

**Reads across modules.** The member list joins the `users` table (`internal/features/workspaces/repositories/membership_repository.go:56`). The old code reads workspace names through `GetWorkspaceByID` in `databases`, `storages`, `notifiers` and both backupers (nine call sites), for transfer audit messages and notification texts.

**The schema.** `workspaces` and `workspace_memberships` (`backend/migrations/20251031155004_add_workspaces.sql`). The member table has a unique `(user_id, workspace_id)` and four indexes: `user_id`, `workspace_id`, `(user_id, workspace_id)` and `(workspace_id, user_id)`. Its reference to `users` is `ON DELETE CASCADE`. The role column is `TEXT` without a check, and nothing limits the number of owners.

**Tests.** The workspace tests drive the old controllers: about 2,500 lines in `controllers/membership_controller_test.go`, `workspace_controller_test.go` and `e2e_test.go`. 54 test files and 9 non-test helper files of other packages use `workspaces_testing` (`restores/testing.go`, `restores/restoring/testing.go`, `tests/logical/shared/hang.go`, `router.go` and `requests.go`, `tests/physical/postgresql/shared/helpers.go`, `backups/backups/controllers/logical/testing.go`, `backups/backups/usecases/physical/postgresql/testing.go` and `backups/backups/backuping/logical/testing.go`): `CreateTestWorkspace` 356 times and `RemoveTestWorkspace` 387 times, both with a router whose routes the helpers call. Whether the deletion listeners are registered in a test binary depends on whether the package calls `SetupDependencies`, and the cascade cleans up after the ones that do not.

**Frontend.** `entity/workspaces` has two hand-written API files and twelve model files. `entity/users/model/WorkspaceRole.ts` holds the role enum and `WorkspaceRoleLabelKeys.ts` its labels. The workspace list is read by `widgets/main/MainScreenComponent.tsx` and by the transfer dialogs of databases, storages and notifiers.

**Rules in force.** The module lands in the target layout and passes every linter and `archlint`: `module.go` alone in the root package, `Deps` and `New`, repositories with `database.Conn`, errors in `errors.go`, no package-level state. The handlers implement the strict server and hold no logic. `docs/architecture.md` fixes the shape of `Authorize`, the synchronous event in the publisher's transaction with its order set in one wiring file in `internal/app`, the batch lookup between modules, and `RESTRICT` between modules. The api-design skill fixes paths, the filter for resources of a workspace, cursor pages and the Problem codes. The frontend uses only types from `schema.gen.ts` and shows errors through `translateApiError`. The change answers to the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- `workspaces` is a module with its operations in the contract, and `Authorize` is the one place where a role meets an action.
- Every write of the module runs in one transaction with its audit event, and a workspace deletion, its event handlers included, is one transaction.
- The old features keep working with their current answers and reach the module only through the bridge.

**Non-Goals:**

- Moving any old feature to `Authorize`, to `Problem` bodies or to the contract.
- The audit log views. `audit-log-views` moves them.
- A general event library. One event exists, and the module holds its subscribers itself; a shared mechanism is worth designing when a second module publishes one.

## Decisions

### 1. The module's packages and its contract

```
internal/modules/workspaces/
  module.go        Module, Role, Action, Workspace, WorkspaceDeleting, Deps, New
  internal/
    access/        the role table and Authorize
    workspace/     workspaces: create, list, read, rename, delete, publish WorkspaceDeleting
    members/       members, invitations, ownership transfer, the invitation email
    httpapi/       the generated strict server and its handlers
    module/        the Module implementation that ties the services together
```

`module.go` gains:

```go
type Workspace struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
}

// WorkspaceDeleting is published inside the deletion's transaction, before the workspace row goes.
type WorkspaceDeleting struct {
	WorkspaceID uuid.UUID
}

type WorkspaceDeletingHandler func(ctx context.Context, event WorkspaceDeleting) error

type Module interface {
	RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks)

	// Authorize answers not_found when the actor is not a member of the workspace or the workspace
	// does not exist, and permission_denied when the actor's role does not allow the action.
	Authorize(ctx context.Context, actor users.Actor, workspaceID uuid.UUID, action Action) error
	// GetWorkspaces leaves out the IDs that match no workspace.
	GetWorkspaces(ctx context.Context, workspaceIDs []uuid.UUID) ([]Workspace, error)
	// SubscribeToWorkspaceDeleting is called by internal/app while it builds the application.
	SubscribeToWorkspaceDeleting(handler WorkspaceDeletingHandler)
}

type Deps struct {
	DB         *gorm.DB
	Transactor *database.Transactor
	Clock      clock.Source
	Logger     *slog.Logger
	Mailer     mailer.Sender
	AuditLog   *audit.Log
	Users      users.Module
	PublicURL  string
}
```

`Role` and `Action` become aliases of the types in `access`, as `users.Role` is an alias of `identity.Role`. The table of `workspaces-roles-and-authorize` moves into `access` unchanged.

*Rejected: subscribers as a field of `Deps`.* The subscribers belong to modules above `workspaces`, and those modules take `workspaces.Module` in their own `Deps`. `internal/app` cannot build them before `workspaces`, so it cannot pass their handlers into `workspaces.New`. Subscribing after construction breaks that cycle, and `internal/app` still sets the order in one place.

*Rejected: `GetWorkspace(ctx, id)` for one ID.* `docs/architecture.md` asks for a batch method, and `audit-log-views` reads the names of many workspaces at once. The bridge serves the old single lookups from it.

### 2. `Authorize` answers 404 before 403

`Authorize` lets an instance admin through. For anyone else it loads the actor's member in the workspace. When there is none, it answers `not_found`, whether the workspace exists or not, so a user outside a team cannot learn its workspace IDs. When the role does not allow the action, it answers `permission_denied`. A failed query returns the error. Both refusals are `*apierror.Problem` values, so a service of any module returns them unchanged and its handler answers them through the hooks.

The services of the module call `Authorize` like every other module. An operation on a member first loads the member by its ID, answers `not_found` when there is none, and then authorizes against the member's workspace, as `docs/architecture.md` describes for objects inside a workspace.

*Rejected: 403 for a non-member of an existing workspace.* That is today's behavior, and it confirms to any user that a workspace ID exists. The api-design skill gives 404 for a workspace the actor is not a member of.

### 3. Ten operations, with members as their own resource

| Operation | Method and path | Answers |
|---|---|---|
| `listWorkspaces` | `GET /workspaces?limit&after` | 200 `WorkspacePage` |
| `createWorkspace` | `POST /workspaces` | 201 `Workspace`, 403 |
| `getWorkspace` | `GET /workspaces/{workspaceId}` | 200 `Workspace`, 404 |
| `updateWorkspace` | `PUT /workspaces/{workspaceId}` | 200 `Workspace`, 403, 404 |
| `deleteWorkspace` | `DELETE /workspaces/{workspaceId}` | 204, 403, 404, 409 |
| `transferWorkspaceOwnership` | `POST /workspaces/{workspaceId}/transfer-ownership` | 204, 403, 404 |
| `listWorkspaceMembers` | `GET /workspace-members?workspaceId&limit&after` | 200 `WorkspaceMemberPage`, 404 |
| `addWorkspaceMember` | `POST /workspace-members` | 201 `AddedWorkspaceMember`, 403, 404, 409 |
| `changeWorkspaceMemberRole` | `PUT /workspace-members/{memberId}` | 200 `WorkspaceMember`, 403, 404, 409 |
| `removeWorkspaceMember` | `DELETE /workspace-members/{memberId}` | 204, 403, 404, 409 |

Every operation also declares 400, 401 and the `default` Problem. None carries an access mark: each needs a signed-in user, and the services authorize.

- `Workspace` has `id`, `name`, `createdAt` and `currentUserRole`, the signed-in user's role in the workspace. It is `nullable`, because an instance admin sees workspaces they are not a member of. An instance admin who is a member gets their role, where the old list gave an instance admin no role at all (`membership_repository.go:121`), so the screens can show the role the admin holds. `name` is 1 to 255 characters, as the old binding allowed.
- `WorkspaceMember` has `id`, `workspaceId`, `userId`, `email`, `name`, `role`, `isInvited` and `createdAt`. `isInvited` is true while the user has not finished sign-up, so the member screen can mark the row.
- `addWorkspaceMember` takes `workspaceId`, `email` and `role` and answers the member with `outcome`, `ADDED` or `INVITED`, in the pattern of the sign-in outcome of `users-module`.
- `changeWorkspaceMemberRole` takes `role`. `PUT` replaces the member's only editable field.
- `transferWorkspaceOwnership` takes `newOwnerMemberId`.
- The role in a request is the schema `AssignableWorkspaceRole` with `WORKSPACE_ADMIN`, `WORKSPACE_MEMBER` and `WORKSPACE_VIEWER`. The role in a response is `WorkspaceRole` with all four values. The generated server rejects the owner role before the handler runs.
- The workspace list is ordered by name and then ID. The member list is ordered by the time a member joined and then ID. Both take `limit` from 1 to 100, default 20, and an opaque cursor of the last row's sort key, as the user list of `users-module` does.

*Rejected: keep members nested under `/workspaces/{workspaceId}/members/{userId}`.* The api-design skill lists a resource of a workspace with a filter instead of nesting it, and the user ID names the user, not the member. A member ID also lets the change and removal find the workspace from the member itself.

*Rejected: name the new owner by email.* Every other member operation takes the member ID, and the screen that offers the transfer already lists the members with their IDs.

*Rejected: an unpaged workspace list.* The api-design skill pages every list. The frontend reads every page where it needs all workspaces, see decision 13.

### 4. Codes and statuses

| Code | Status | When |
|---|---|---|
| `already_member` | 409 | the added user is already a member of the workspace |
| `cannot_change_own_role` | 409 | the member to change is the actor's own member |
| `owner_requires_transfer` | 409 | the request would change the owner's role or remove the owner |
| `workspace_not_empty` | 409 | the workspace still has databases, or another module still references it |

`not_found`, `permission_denied` and `invalid_request` cover the rest. A refusal of `users.Module.InviteUser` arrives as its own Problem and passes through unchanged: `permission_denied` when the instance settings forbid the invitation, and 409 `email_taken` in the race where the address signs up between the lookup and the invitation. A retry then finds the user and adds them, so a mapping to another code would only hide what happened. Each code enters `ProblemCode` in the contract and `apierror.Code` with its status in the same task, and the test that compares the two lists keeps them equal.

*Rejected: `in_use` for a workspace with databases.* The api-design skill names `in_use` for an object that is in use. Here the workspace's content is the obstacle, and the frontend tells the user to delete the databases first, which `in_use` would not say.

### 5. Member rules

`addWorkspaceMember` and `changeWorkspaceMemberRole` authorize with `ActionManageAdmins` when the requested role is admin or, for a change, when the member's current role is admin. Otherwise they authorize with `ActionManageMembers`. `removeWorkspaceMember` does the same with the member's current role. The checks run in a fixed order, which the `workspace-members` spec states:

1. the member is loaded by ID, and a missing member answers `not_found`;
2. `Authorize` runs for the member's workspace, so a non-member gets `not_found` and an actor without the permission gets `permission_denied`;
3. a change of the actor's own member answers `cannot_change_own_role`;
4. a change or a removal of the owner answers `owner_requires_transfer`.

Permission comes first, so an actor without it never learns from a 409 who the owner is. Step 3 comes before step 4, so the owner who changes their own role gets `cannot_change_own_role`. A workspace admin who changes their own role is stopped at step 2, because their current role is admin.

The transfer authorizes with `ActionTransferOwnership`, loads the named member in the same workspace and answers `not_found` when it is missing or belongs to another workspace. A transfer to the current owner returns without a write. Otherwise, in one transaction, it demotes the current owner to admin and then promotes the new owner, so the unique index of decision 10 never sees two owners.

*Rejected: let an admin keep demoting other admins.* That is today's behavior, and it contradicts the refusal to remove an admin. An admin who may not remove another admin could demote them to viewer and then remove them.

### 6. Deleting a workspace is one transaction with its event

`deleteWorkspace` authorizes with `ActionDeleteWorkspace` and then, in one transaction:

1. locks the workspace row with `SELECT ... FOR NO KEY UPDATE`, so two deletions of one workspace run one after the other. That lock does not conflict with the key-share lock a new row of another table takes on the workspace through its foreign key, so a database created meanwhile is not blocked and is caught by the foreign key at step 3;
2. calls every subscriber of `WorkspaceDeleting` in the order it subscribed, with the transaction's context;
3. deletes the workspace row, which removes its members through the module's own `CASCADE`;
4. records the audit event "Workspace deleted" with the workspace's ID and name.

A subscriber error rolls back the transaction and is returned as it is. A foreign key violation on the workspace row (PostgreSQL code `23503`) answers `workspace_not_empty`: some table of another module still points at the workspace, for example a database added after the `databases` subscriber looked.

`internal/app/wiring.go` subscribes three handlers of the old code, in this order:

1. `databases`: answers `workspace_not_empty` when the workspace has databases;
2. `notifiers`: deletes the workspace's notifiers;
3. `storages`: drains the pending deletions of each of the workspace's storages and deletes the storages.

Their methods take the context, and their repositories reach the database through `database.Conn(ctx, storage.GetDb())`, so their writes join the transaction. The repository methods that open a transaction of their own on the global connection open it on that connection instead, which becomes a savepoint of the deletion. The listener interface, its registration in the three `SetupDependencies` and `AddWorkspaceDeletionListener` are deleted.

*Rejected: let `RESTRICT` alone refuse a workspace with databases.* The refusal would come at the last step, after the notifiers and storages were deleted inside the transaction and their drain had run, and the answer could not tell databases apart from other references. The explicit first handler answers before any work. The foreign key mapping stays for the race it cannot see.

*Rejected: subscribe the old handlers inside their own `SetupDependencies`, as today.* The order would follow the call order of `SetupDependencies`. `docs/architecture.md` wants one place that reads as the order, and a subscription line per handler in `wiring.go` is that place.

### 7. The drain stays inside the transaction, and `docs/architecture.md` names the exception

The `storages` handler drains a storage's pending deletions before it deletes the storage, as today. A pending deletion is a file whose backup row is already gone, so the file is deleted whatever happens to the workspace. If the transaction rolls back after the drain, the files the drain removed were due for deletion anyway, and the rows of the files it did not reach are still there for the deletion worker. The drain keeps its budget of ten seconds (`internal/features/storages/pending_deletion_drain.go:17`), which bounds how long the deletion holds its lock.

`docs/architecture.md` says that effects outside the database are registered as after-commit handlers. The drain is such an effect, so the change adds one sentence to that rule in "Modules": an effect that stays correct when the transaction rolls back, such as deleting files that already have a pending deletion, may run inside the handler. No after-commit mechanism is built in this change.

*Rejected: an after-commit hook on the transactor, with the drain registered there.* After the commit the storage row is gone, so the hook would have to carry the storage's settings in memory and could not record a pending deletion for a file it failed to delete. The first effect that must not run before the commit, such as cancelling a running process, arrives with `databases` and `backups`, and the hook is designed then with a real consumer.

### 8. An invitation is written in one transaction, and its email goes out after it

`addWorkspaceMember` runs one transaction that looks the address up through `users.Module.GetUserByEmail`, invites the user through `users.Module.InviteUser` when there is none, inserts the member and records the audit event. `InviteUser` opens its own transaction, which joins this one as a savepoint, because both modules take the one `database.Transactor` and the transaction travels in the context. A unique violation on `(user_id, workspace_id)` answers `already_member`, which also covers two concurrent additions of one user.

After the commit, an invited member gets the invitation email through `Deps.Mailer`. A refused send is logged with the address and does not change the answer, as the `workspace-invitations` delta requires. The email keeps its current text and its link rule from the `workspace-invitations` spec, with the public address from `Deps.PublicURL` instead of `config.GetEnv()`. The inviter's name comes from `users.Module.GetUser`.

*Rejected: send the email inside the transaction.* A failure after the send would roll back the member and the invited user, while the email would already point the person at a workspace they are not in.

### 9. `users.Module` gains a batch lookup

```go
// GetUsers leaves out the IDs that match no user.
GetUsers(ctx context.Context, userIDs []uuid.UUID) ([]User, error)
```

The member list reads one page of members and then their users in one call, and joins the two in Go. The repository of the member list no longer touches `users`. `User` already carries `Status`, from which `isInvited` comes. The audit events of the module take their author from `users.Module.GetUser(ctx, actor.UserID)`.

*Rejected: call `GetUser` once per member.* `docs/architecture.md` forbids a lookup per record in a loop.

### 10. One migration renames, repairs and constrains

In one goose file:

1. `workspace_memberships` becomes `workspaces_members`, with its primary key, its unique constraint, its two references and the remaining index renamed to the new prefix.
2. The indexes on `user_id`, on `workspace_id` and on `(user_id, workspace_id)` are dropped. The unique constraint on `(user_id, workspace_id)` and the index on `(workspace_id, user_id)` already serve every query that used them.
3. A role outside the four values becomes `WORKSPACE_VIEWER`, and the check `ck_workspaces_members_role` allows only the four values.
4. In each workspace with more than one owner, every owner except the one with the earliest `created_at`, then the smallest `id`, becomes `WORKSPACE_ADMIN`. Then `uq_workspaces_members_owner` is a unique index on `workspace_id` where the role is `WORKSPACE_OWNER`.
5. `fk_workspaces_members_user_id`, `fk_databases_workspace_id`, `fk_storages_workspace_id` and `fk_notifiers_workspace_id` are dropped and recreated with `ON DELETE RESTRICT`. The reference of a member to its workspace stays `CASCADE`, because both tables belong to the module.

The down migration reverses each step except the repairs, which have nothing to restore.

The reference of a member to `users` costs nothing in `RESTRICT`: the product never deletes a user, and a deactivated user stays a member of their workspaces. `features/users/testing.DeleteTestUser` removes the user's members before the user, see decision 12.

*Rejected: keep the name `workspace_memberships` with the prefix added, `workspaces_memberships`.* The glossary calls a user in a workspace a member, and the API resource is `/workspace-members`. One word in the table, the API and the code is the rule of the glossary.

*Rejected: delete the extra owners or make them viewers.* An extra owner today was given the role by someone allowed to manage members. Admin keeps the most of what they had without breaking the rule of one owner.

### 11. The old code reaches the module through a bridge

`features/workspaces` keeps three things:

- `Install(module)` and `GetModule()`, as the users bridge does, with a module built from the old access points in a `testhelpers` build;
- the type `workspaces_bridge.Access` with the two methods the old code calls: `IsActorAllowed(ctx, actor, workspaceID, action) (bool, error)`, which calls `Authorize` and turns `not_found` and `permission_denied` into `false`, and `GetWorkspaceByID(ctx, workspaceID) (workspaces.Workspace, error)`, which calls `GetWorkspaces` and returns an error for a missing ID;
- `features/workspaces/testing`, behind the `testhelpers` tag.

The old callers keep their call expressions. Their field type changes from `*workspaces_services.WorkspaceService` to `*workspaces_bridge.Access`, their `di.go` takes the bridge's value, and the nine `GetWorkspaceByID` calls pass a context. The `workspaceService` interface of `databases` keeps its two methods with the new signatures. The type is named `Access` because what the old callers need from the module is the access check and the workspace names their access-checked operations show. `Workspaces` would repeat the package's noun, which the naming rules of the root `AGENTS.md` forbid.

The workspace audit log route moves into `features/audit_logs` with the same response, checked through `IsActorAllowed` with `ActionView`. Its path parameter becomes `:workspaceId`, because Gin refuses two wildcard names in one segment and the generated server already registers `/workspaces/:workspaceId`. `audit-log-views` replaces the route.

`temp/refactor/plan.md` gets a row in the table of temporary relaxations: the old features check workspace access through `IsActorAllowed` instead of `Authorize`, until each moves into its module, and by milestone 13 at the latest. The row about `MembershipRepository` joining `users` is crossed out.

*Rejected: switch the old callers to `Authorize` now.* Their controllers map their own errors to 400 or 403, and their tests check those answers. Each later milestone rewrites those controllers for the contract anyway, so this change would rewrite them twice.

### 12. The old test helpers write SQL

`workspaces_testing` creates a workspace by inserting the workspace and its owner's member, adds members by inserting rows, and removes a test workspace by deleting, in this order, its databases, notifiers and storages and then the workspace. The deletes of the content are what the `CASCADE` did before. Their own cascades inside the old features remove what hangs off them, as before. The helpers return a `*workspaces_testing.TestWorkspace` with `ID` and `Name`, and lose their `router` parameter. `CreateTestRouter` stays for the old controllers of other features. `MakeAPIRequest` stays for the old tests that call their own routes. `features/users/testing.DeleteTestUser` deletes the user's members before the user.

The call sites change by `gofmt -r` rules that drop the last argument, such as `workspaces_testing.CreateTestWorkspace(a, b, c, d) -> workspaces_testing.CreateTestWorkspace(a, b, c)`, and the type `*workspaces_models.Workspace` in test files becomes `*workspaces_testing.TestWorkspace`.

The row of the table of temporary relaxations about old helpers that write the tables of `users` with SQL widens to the tables of `workspaces`.

*Rejected: let the helpers call the module's routes.* A deletion through the module needs the subscriptions of decision 6, and only `internal/app` makes them. The helpers cannot subscribe the old features themselves, because `databases`, `notifiers` and `storages` import `workspaces_testing` in their tests, and Go refuses that cycle. Without the subscriptions, `RESTRICT` would fail the cleanup of every test that leaves content behind.

### 13. The frontend calls the generated client and reads whole lists where it needs them

`entity/workspaces/api` is rewritten on the generated client. `workspaceMembershipApi.ts` becomes `workspaceMemberApi.ts` with the object `workspaceMemberApi`, and `WorkspaceMembershipComponent.tsx` becomes `WorkspaceMembersComponent.tsx`, because client code keeps the API's word. The hand-written models of `entity/workspaces` and `entity/users/model/WorkspaceRole.ts` are deleted. The role labels key off the generated `WorkspaceRole`. `workspaceApi.listAllWorkspaces()` follows `nextCursor` with `limit` 100 until the last page, for the main screen and the three transfer dialogs, and `workspaceMemberApi.listAllMembers(workspaceId)` does the same for the member screen. The member screen sends the member ID, offers only the assignable roles, marks invited members and names the new owner by member ID. Each new code gets a `translateApiError` key and a translation in every dictionary.

Screen tests on `mockApi` cover the member screen: adding an existing user, inviting an address, a refusal with `already_member`, and the role selector without the owner role. A unit test covers `listAllWorkspaces` over two pages.

*Rejected: a "load more" button on the workspace selector.* The selector and the transfer dialogs need every workspace to be useful, and a team has few. Reading all pages keeps the screens as they are.

### 14. The module's tests go through the API

The tests in `internal/modules/workspaces` use `apptest` and the contract-checking client. `apptest` gets `CreateWorkspace(tb, session, name)` and `AddWorkspaceMember(tb, session, workspaceID, email, role)`, both through the API. The scenario of a deletion that fails after the notifiers and storages were removed and the scenario of a database added during a deletion need a subscriber of their own: the first returns a plain error and checks the generic `internal` answer, and the second inserts a database row through `apptest.App.ExecuteSQL`. For them so `apptest.App` gets `SubscribeToWorkspaceDeleting(handler)`, which adds a handler after the ones `internal/app` subscribed, the way a higher module would. The old tests of `features/workspaces` and the tests of the old workspace routes are deleted with the routes. A test of the old bridge checks that `IsActorAllowed` answers `true` for a `WORKSPACE_MEMBER` asked to edit content, `false` for a viewer asked the same and for a non-member, and returns an error for a failed query. It covers the `workspace-access` scenario "Member and viewer on the workspace's content".

The upgrade test reads the workspaces of the current release through `listWorkspaces` and its members through `listWorkspaceMembers`, and the previous release creates a workspace with a second member, so the renamed table is checked with data.

### 15. New terms enter the glossary

`docs/glossary.md` gets "member ID", the ID of one member of one workspace, which the API uses to change or remove that member, and "ownership transfer". "Member" keeps its meaning.

## Risks / Trade-offs

- [A test of an old feature relied on a listener refusing the deletion] → No old test outside `workspaces` is about deletion, and the helpers now remove content as the cascade did. The tests of each touched package run before handing over.
- [The drain holds the deletion's lock for up to ten seconds] → Only the deletion of the same workspace waits for it. The budget already exists and the request waited for it before.
- [The migration demotes owners on instances that used the hole] → The earliest owner stays, the others keep admin rights, and the release notes say so.
- [Wide diff over test files] → The test edits are `gofmt -r` rules, so the review checks the rules and the helper bodies, not 750 call sites.
- [Two modules share one transaction through the context] → That is how `database.Transactor` is meant to work, and its savepoints keep `InviteUser`'s own transaction intact. The test of the failed addition shows the invited user rolled back.

## Migration Plan

1. The migration runs at startup before the server accepts requests. Every step is a rename, a small `UPDATE` or a constraint on a table of a size proportional to the members, so it finishes in milliseconds.
2. The upgrade is one-way, as `docs/architecture.md` says for every migration: the previous binary refuses to start on the new schema. Recovery from a failed upgrade means restoring the internal database from a backup taken before it. The down section of the migration serves development only.
3. The upgrade test runs the previous release with a workspace of two members and checks after the upgrade that the workspace, both members and their roles are listed.
