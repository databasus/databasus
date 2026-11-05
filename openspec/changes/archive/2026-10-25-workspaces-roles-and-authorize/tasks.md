## 1. The module root and the role type

- [x] 1.1 Add `backend/internal/modules/workspaces/module.go` with the package comment, `Role` with its four constants and `IsValid`, and `Action` with its seven constants, as design.md decision 1 shows. Verify with `mise run backend:lint` and `GOFLAGS= go run ./tools/archlint` in `backend/` exiting with status 0.
- [x] 1.2 Replace `users_enums.WorkspaceRole` and its four constants with `workspaces.Role` and `RoleOwner`, `RoleAdmin`, `RoleMember`, `RoleViewer` in every file by the `gofmt -r` rules of design.md decision 4, run `goimports`, and delete `backend/internal/features/users/enums`. Verify with `rg -n 'users_enums|features/users/enums' backend/` printing nothing and `mise run backend:lint` exiting with status 0.

## 2. One permission check

- [x] 2.1 Add `WorkspaceService.IsActorAllowed` with the role table of design.md decision 2, switch the checks inside `features/workspaces` to it as the table of decision 3 maps them, and delete `CanUserAccessWorkspace`, `CanUserManageWorkspace`, `CanUserManageDBs`, `CanUserManageMembership`, `CanUserManageAdmins` and `GetUserWorkspaceRole` from the service. Add the test of design.md decision 5. Verify with `mise run backend:test-package ./internal/features/workspaces/...`.
- [x] 2.2 Switch the 55 call sites outside `workspaces` to `IsActorAllowed` with the action decision 3 maps their method to, rename each `canView`, `canAccess` and `canManage` result to an `is` name, and change the `workspaceService` interface of `features/databases` and the fake in its controller test. Verify with `rg -n 'CanUser(AccessWorkspace|ManageWorkspace|ManageDBs|ManageMembership|ManageAdmins)|GetUserWorkspaceRole' backend/internal` printing only `MembershipRepository.GetUserWorkspaceRole` and its callers in `features/workspaces/services`, and with `mise run backend:test-package ./internal/features/databases/... ./internal/features/storages/... ./internal/features/notifiers/... ./internal/features/backups/backups/services/... ./internal/features/backups/backups/controllers/... ./internal/features/backups/config/... ./internal/features/restores/... ./internal/features/healthcheck/... ./internal/features/verification/config/... ./internal/features/verification/runs/... ./internal/app/...`.

## 3. Documents and checks

- [x] 3.1 Add to `docs/glossary.md` the rows of design.md decision 1: workspace role with its four values and the note that "member" alone means any user in a workspace, workspace admin with the note that `admin` alone means the instance admin, owner, workspace content and workspace action. Verify with `rg -n 'Workspace role|Workspace admin|Owner|Workspace content|Workspace action' docs/glossary.md` showing a row for each.
- [x] 3.2 Update `temp/refactor/plan.md`: split milestone 5 of the checklist into the sub-items `5a` `workspaces-roles-and-authorize`, `5b` `workspaces-module` and `5c` `audit-log-views` with a line each, and check `5a`. Verify with `rg -n '^\s*- \[x\] \*\*5a\.' temp/refactor/plan.md` and `rg -n '^\s*- \[ \] \*\*5[bc]\.' temp/refactor/plan.md` showing the three lines.
- [x] 3.3 Search the code and the documents for the deleted names and bring every mention in line. Verify with `rg -n 'CanUserAccessWorkspace|CanUserManageDBs|CanUserManageWorkspace|CanUserManageMembership|CanUserManageAdmins|users_enums' backend/internal backend/cmd docs AGENTS.md backend/AGENTS.md` printing nothing.
- [x] 3.4 Run `mise run lint` and fix what it reports. Verify that it exits with status 0.

## Change verification

```sh
mise run backend:test-package ./internal/features/workspaces/... ./internal/features/databases/... ./internal/features/storages/... ./internal/features/notifiers/... ./internal/features/backups/backups/services/... ./internal/features/backups/backups/controllers/... ./internal/features/backups/config/... ./internal/features/restores/... ./internal/features/healthcheck/... ./internal/features/verification/config/... ./internal/features/verification/runs/... ./internal/app/...
```
