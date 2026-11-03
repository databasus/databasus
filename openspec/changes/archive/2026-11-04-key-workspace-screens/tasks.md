## 1. Pin the switch at the application level

- [x] 1.1 Add `shows no data of the previous workspace after switching` to `frontend/src/app/ui/WorkspaceNavigation.test.tsx` for each held read of the first workspace: `/databases` on the database screen, `/storages` on the storage screen, `/notifiers` on the notifier screen, and both `/workspaces/{workspaceId}` and `/workspace-members` on the settings screen. Each case switches through the selector, selects a non-default resource on the database, storage and notifier screens, and releases the old read with a success, a `404` and a `500`. It then checks that current content stays, no error appears, no workspace reconciliation starts and, on those three screens, the selected resource and its stored preference are unchanged. Verify with `pnpm test -t "shows no data of the previous workspace after switching" src/app/ui/WorkspaceNavigation.test.tsx` in `frontend/`; it passes on the current code.
- [x] 1.2 Add `resets the selection and keeps focus reachable after switching` to the same file for the database, storage and notifier screens. The settings screen is excluded because it has no resource selection. Each case chooses a non-default resource in the first workspace, switches to the second, checks the second workspace's own selection and focus on the selector, then switches back and checks that the first workspace's selection is restored. The database and notifier cases list at least five resources, enter a search before the switch and check that it is empty afterwards; the storage screen has no search input. Verify with `pnpm test -t "resets the selection and keeps focus reachable after switching" src/app/ui/WorkspaceNavigation.test.tsx` in `frontend/`; it passes on the current code.

## 2. Move to one key

- [x] 2.1 In `frontend/src/app/ui/ApplicationScreen.tsx`, key the fragment around the workspace screens by `selectedWorkspace.id` and drop the two per-screen keys. Verify with `pnpm test src/app/ui/WorkspaceNavigation.test.tsx src/app/ui/ApplicationScreen.test.tsx` in `frontend/`.

## 3. Remove the guards

- [x] 3.1 In `StoragesComponent.tsx` and `NotifiersComponent.tsx`, remove the workspace field of the collection state, the derived list guard, the reset in the effect, the workspace ID in the effect dependencies and the workspace prefix in the inner detail key (`StoragesComponent.tsx:185`, `NotifiersComponent.tsx:200`). In `StoragesComponent.tsx`, also remove `searchQuery`, `setSearchQuery` and `filteredStorages`, because that screen has no search input and the reset is the only caller of the setter. In `WorkspaceResources.test.tsx`, delete the test at `:293`, which task 1.1 replaces, and rewrite the test at `:417` so the workspace is refused without changing a prop. Verify with `pnpm test src/app/ui/WorkspaceResources.test.tsx src/pages/storages src/pages/notifiers` in `frontend/`.
- [x] 3.2 In `DatabasesComponent.tsx`, remove the workspace field of `DatabaseCollection`, the parameter of `createEmptyCollection`, `latestWorkspaceId`, `isCurrentWorkspaceAnswer`, the derived list guard and the reset in the effect, and keep `latestSelectedDatabaseId`. Delete the test at `DatabasesComponent.test.tsx:1520`. A remount forgets the created database because `databaseCreatedInWizard` is a per-instance ref, and task 1.2 covers the selection surviving a return to the first workspace. Verify with `pnpm test src/pages/databases/ui/DatabasesComponent.test.tsx` in `frontend/` and with `rg -n "databaseCreatedInWizard = useRef" frontend/src/pages/databases/ui/DatabasesComponent.tsx`.
- [x] 3.3 In `WorkspaceMembersComponent.tsx`, remove `workspaceRevision`, the state resets in the load effect and the workspace ID in both effects' dependencies, and keep `memberReadRevision` and `searchRevision`. Delete the test at `WorkspaceMembersComponent.test.tsx:178`, whose late `/workspace-members` answer task 1.1 now releases. Verify with `pnpm test src/pages/workspace-settings/ui/WorkspaceMembersComponent.test.tsx` in `frontend/`.
- [x] 3.4 In `WorkspaceSettingsComponent.tsx`, remove the resets from the load effect and the workspace ID from its dependencies, and keep `workspaceRevision` and the dialog cleanup. Remove the `switch` cases from the test at `WorkspaceSettingsComponent.test.tsx:178`. Verify with `pnpm test src/pages/workspace-settings/ui/WorkspaceSettingsComponent.test.tsx` in `frontend/`.

## 4. Prove the key is guarded

- [x] 4.1 Temporarily remove the key from the fragment, run `pnpm test src/app/ui/WorkspaceNavigation.test.tsx` in `frontend/`, confirm that every parametrized case of both new tests fails, then restore the key and confirm that they all pass.

## 5. Document and check

- [x] 5.1 Add the key rule to the "Resource pages own collection policy" entry in `docs/architecture.md`, and verify with `rg -n "by a key on the workspace" docs/architecture.md`.
- [x] 5.2 Format only the changed frontend files, then verify with `mise run frontend:lint` and `pnpm --dir frontend build`.
- [x] 5.3 Repair the flaky MinIO startup that the full test run exposed: the helper waits for `/minio/health/cluster`, because the live and ready endpoints answer before S3 requests work. Verify with `go test -count=1 ./internal/modules/storages/internal/providers/ ./internal/modules/storages/internal/providers/s3/ ./internal/lib/testcontainers/` in `backend/` and `mise run backend:lint`.

## Change verification

```sh
mise run test
```
