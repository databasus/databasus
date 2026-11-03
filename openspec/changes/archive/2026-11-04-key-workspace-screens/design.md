## Context

See `proposal.md` for the motivation. The current state that shapes the approach:

- `frontend/src/app/ui/ApplicationScreen.tsx:219` and `:243` key the database and settings screens by workspace. The storage and notifier screens at `:223` and `:230` have no key, so their guards still run.
- `useSessionGuard` returns `false` once its component has unmounted (`frontend/src/shared/auth/useSessionGuard.ts`). After a key change, a late answer for the destroyed screen is therefore dropped by `isCurrent()` alone.
- `latestWorkspaceId` (`DatabasesComponent.tsx:136`, `:217`) compares a value with itself on a keyed screen, because the workspace of one screen instance never changes.
- `WorkspaceMembersComponent.tsx:404-432` resets about twenty state fields in an effect. Its `workspaceRevision` changes only in that effect and its cleanup, so on a keyed screen it adds nothing to `isCurrent()`.
- `selectedWorkspace` falls back to the first workspace when the selected one disappears (`frontend/src/app/model/useApplicationStartup.ts:140`). A key on its ID also covers a switch caused by lost access.
- The workspace selector sits in the application header, outside the block that the key recreates.
- `frontend/src/app/ui/WorkspaceResources.test.tsx:293` is the only check that an old list answer cannot reach a current storage or notifier screen. It changes a prop on a mounted screen, a path the application will no longer take.

The approach follows the frontend section of `docs/architecture.md`, including "Resource pages own collection policy", and the React component structure and verification rules of `frontend/AGENTS.md`. "The best code is unwritten" and "Refactoring within a feature" in the root `AGENTS.md` set the scope of the removal.

## Goals / Non-Goals

**Goals:**

- A workspace switch recreates every workspace screen through one mechanism.
- Application-level tests keep the existing switch guarantees and fail without the key.

**Non-Goals:**

- Removing request-ordering code. It orders requests inside one workspace and has nothing to do with a switch.
- Changing what happens when a workspace becomes unavailable. A screen still clears its collection on `403` and `404`, because it stays mounted until the application has refreshed the accessible workspaces.

## Decisions

### One key on the block of workspace screens

`ApplicationScreen` replaces the fragment at `:213` with `<Fragment key={selectedWorkspace.id}>` and drops the keys at `:219` and `:243`.

Rejected: a key on each screen, as the task first proposed. It keeps four places to maintain, and a fifth screen could omit its key and show another workspace's data. Also rejected: a wrapper component around the screens, because one fragment gives the same result without a new unit.

### Screens stop tracking the workspace prop

The effects drop `workspace.id` and `selectedWorkspace.id` from their dependencies, together with the resets that only applied to a changed workspace. `isCurrentWorkspaceAnswer` becomes `isCurrent`, and `workspaceRevision` in the members component goes away, so its `isActive` closures check `isCurrent` and the search or read revision that remains. The inner detail keys of the storage and notifier screens (`StoragesComponent.tsx:185`, `NotifiersComponent.tsx:200`) lose their workspace prefix.

`StoragesComponent` also loses `searchQuery`, `setSearchQuery` and `filteredStorages`. That screen has no search input: the reset at `:107` is the only caller of the setter, so the filter at `:129` never filters anything, and removing the reset would leave an unused setter.

Rejected: keeping the workspace ID in the dependencies as a reload safety net. A screen would then be half prepared for a prop change: it would reload but keep its other state, and that hides a missing key instead of exposing it. Rejected as well: keeping the guards as defense in depth. They are the unreachable code this change exists to remove, and they would need tests through an impossible path.

### What stays

- `loadRevision`, `memberReadRevision` and `searchRevision` order retries and refreshes inside one workspace.
- `workspaceRevision` in `WorkspaceSettingsComponent` stays because `handleUnavailable` invalidates it to stop late answers after access is lost.
- `latestSelectedDatabaseId` protects against a stale selection inside one workspace (`DatabasesComponent.tsx:168`, `:190`).
- `createEmptyCollection` stays for the `403` and `404` path and loses only its workspace parameter.
- `resource.workspaceId !== workspace.id` stays because it checks what the server answered, not whether the screen changed.

Rejected: removing the listed revisions as the task's "extra revisions" wording could suggest. Without them a slow older read could overwrite a newer one in the same workspace.

### `useScopedSelection` keeps its scope handling

All three of its callers will be keyed, so the scope change inside the hook becomes unreachable in the application. The hook stays unchanged, along with its test at `useScopedSelection.test.tsx:40`.

Rejected: removing the handling. That saves three or four lines, but a future caller that changes `scopeKey` without a key would silently write one scope's selection under another scope's preference. A changed option is part of a shared hook's contract, while a workspace screen has a single render site.

### Switch guarantees move to the application level

New tests in `WorkspaceNavigation.test.tsx` run through the real selector. The first holds a read of the first workspace, switches to the second and releases the old read with a success, a `404` and a `500`. It holds `/databases`, `/storages` and `/notifiers` on their screens, and both `/workspaces/{workspaceId}` and `/workspace-members` on the settings screen, so the late member answer stays covered. It checks that current content stays, and on the three list screens that the selected resource and its stored preference are unchanged. The second test covers selection and focus on the database, storage and notifier screens, and search on the database and notifier screens, the two that have a search input. The settings screen has no resource selection. Tests that rerender a mounted screen with another workspace are then deleted or rewritten.

Rejected: keying the screen inside each component test. Such a test would pass even if `ApplicationScreen` lost its key, and that render site is exactly what must be protected.

### The rule is written in `docs/architecture.md`

The "Resource pages own collection policy" entry gains one sentence: the application recreates the database, storage, notifier and workspace settings screens by a key on the workspace, so they do not track a workspace change themselves. A future screen author needs this rule, and it changes how screens are written.

Rejected: a comment in `ApplicationScreen` alone, because screen authors read the architecture document and the screen code, not the render site.

## Risks / Trade-offs

- [A missing key shows the previous workspace's data under the new workspace's name, and a user could change a resource in the wrong workspace.] → One key covers all screens, and the new tests must be seen failing without it (task 4.1).
- [Remounting cancels no request by itself, so a late answer still arrives.] → `isCurrent()` drops it, and the new tests release the old read after the switch with both a success and a refusal.
- [happy-dom does not model every browser focus movement, so the focus assertion proves only that the switch does not take focus from the selector.] → This change adds no focus code. The screen's content is already replaced by a loading indicator during a switch, so a focused element inside it is lost today as well.
- [The expected saving is about 60 lines of screen code and is an estimate.] → The main gain is that no reset list needs extending by hand, and the task does not depend on a line count.

## Migration Plan

No data or deployment steps. Reverting the commit restores the previous code.
