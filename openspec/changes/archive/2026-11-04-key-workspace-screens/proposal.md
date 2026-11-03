## Why

The application can recreate a workspace screen by giving it a React key, and the database and settings screens already have one. Their manual protection against another workspace's data can no longer run, while the storage and notifier screens still depend on the same kind of protection. Every screen state added later must also be added to a hand-kept reset list, and tests exercise a path the application never takes.

## What Changes

- One key on the shared block of workspace screens recreates the database, storage, notifier and settings screens when the selected workspace changes. It replaces the two per-screen keys.
- Removed from the screens: the state resets on a workspace change, the workspace field of the collection state, `latestWorkspaceId`, `isCurrentWorkspaceAnswer`, the `workspaceRevision` of the members component, the workspace ID in effect dependencies, the workspace prefix in the inner detail keys of the storage and notifier screens, and the search state of the storage screen, which has no search input.
- Application-level tests pin the workspace switch on all four screens and fail if the key disappears.
- Tests that change the workspace prop of a mounted screen are deleted, moved to the application level or rewritten, because that path stops being supported.

No user-visible behavior changes. The requirement "Workspace-scoped responses cannot populate another workspace" stays as it is, so this change has no delta spec and sets `skip_specs: true`.

**BREAKING (internal only):** a workspace screen no longer reacts to a changed workspace prop. A screen rendered without the key would keep showing the previous workspace. No shim remains, and the application-level tests guard the only render site.

### Out of scope

- How screens load, refresh or retry their data.
- Request ordering inside one workspace: `loadRevision`, `memberReadRevision`, `searchRevision` and the `workspaceRevision` of the settings component.
- The stale-selection protection `latestSelectedDatabaseId` and the checks of a resource's own `workspaceId` in an API answer.
- The shared `useScopedSelection` hook and its `scopeKey` handling.
- `WorkspaceAuditLogsComponent`, `StorageSwitchHistoryComponent` and other units that take a workspace ID and reload through their own hooks.
- Backend production code and HTTP operations.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. The behavior in `frontend-workspace-screens` is preserved.

## Impact

- **Code:** `frontend/src/app/ui/ApplicationScreen.tsx`, `DatabasesComponent.tsx`, `StoragesComponent.tsx`, `NotifiersComponent.tsx`, `WorkspaceSettingsComponent.tsx` and `WorkspaceMembersComponent.tsx`.
- **Tests:** `WorkspaceNavigation.test.tsx`, `WorkspaceResources.test.tsx`, `DatabasesComponent.test.tsx`, `WorkspaceMembersComponent.test.tsx` and `WorkspaceSettingsComponent.test.tsx`.
- **Test infrastructure:** `backend/internal/lib/testcontainers/minio.go` now waits until MinIO accepts S3 requests. This repairs a flaky startup that the change's verification exposed.
- **Documentation:** one rule in the frontend section of `docs/architecture.md`.
- **Rules:** the change answers to the root `AGENTS.md` and `frontend/AGENTS.md`.
