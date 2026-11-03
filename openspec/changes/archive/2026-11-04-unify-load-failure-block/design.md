## Context

See `proposal.md` for the motivation. This design covers the choices that are not obvious from the code.

Retry blocks currently follow one shape, such as `frontend/src/pages/notifiers/ui/NotifiersComponent.tsx:178-183`. The exceptions decide the design:

- `frontend/src/pages/profile/ui/ProfilePage.tsx:173-190` and `frontend/src/pages/instance-settings/ui/InstanceSettingsPage.tsx:130-147` share one `failure` state between loading and saving, and offer retry only while the data is absent.
- `frontend/src/pages/storages/ui/StorageComponent.tsx:239-250` and `frontend/src/pages/notifiers/ui/NotifierComponent.tsx:235-246` have the same shape: their `error` is also set outside `loadSettings` (`StorageComponent.tsx:86,118,175`, `NotifierComponent.tsx:101,122,174`), and retry appears only while the entity is absent.
- `frontend/src/pages/users/ui/UsersPage.tsx:331-344` disables the button during a reload, while `retryUsers` at `:38-44` already ignores those clicks and `fetchPage` at `:191` clears the failure when the request starts.
- `frontend/src/entities/backup-configs/ui/StorageSwitchHistoryComponent.tsx:161` has its own button text.
- `frontend/src/pages/databases/ui/DatabaseTransferDialogComponent.tsx:360-372` disables the button during a transfer and starts four loads.
- `frontend/src/entities/audit-logs/ui/AuditEventList.tsx:16` has an optional `onRetry`, and `frontend/src/pages/workspace-settings/ui/WorkspaceAuditLogsComponent.tsx:51` omits it because that screen shows its own failure.

`frontend/src/shared/lib/collectCursorPages.ts:3` already serves four API facades, while `frontend/src/entities/restores/api/restoreApi.ts:15`, `frontend/src/entities/backup-configs/api/backupConfigApi.ts:18`, `frontend/src/entities/workspaces/api/workspaceMemberApi.ts:17` and `frontend/src/entities/verification-agents/api/verificationAgentApi.ts:8` copy its loop.

`frontend/src/pages/databases/ui/DatabaseConfigComponent.tsx:201-216` repeats `frontend/src/pages/databases/ui/backup-config/useBackupConfigLoad.ts:13-35`, and it also reloads after a save at `:183`.

In `frontend/src/pages/databases/ui/DatabaseComponent.tsx:104-120`, `<Tabs>` is the first child of the element that holds the tab panels. `isHealthcheckVisible` at `:44` exists only to tell a panel whether it sits directly under the tabs. The monitoring and verification panels always receive `true` at `:164` and `:173`.

The approach follows the frontend layer rules in `docs/architecture.md#frontend`, the component structure and the AntD-only rule in `frontend/AGENTS.md`, and the root `AGENTS.md` rules on naming and logical units.

## Goals / Non-Goals

**Goals:**

- One implementation of the load failure block, with one look.
- One cursor collection function, apart from the healthcheck loop.
- No React state whose only purpose is a corner radius.

**Non-Goals:**

- Moving load state, reload logic or error handling into a shared hook. Screens keep their own load logic and only pass the result to the block.

## Decisions

### `LoadFailure` takes `error`, `onRetry` and an optional `className`

It renders an error `Alert` with an icon, the `translateApiError` text and a button with `common.actions.retry`. The button text wrapping from `ProfilePage` is built in, because it costs nothing on the screens that do not need it.

- Rejected: `size` and `isRetryDisabled` props. They would keep the differences this change removes, and the only screen that needs a disabled button stays outside the block.
- Rejected: a `useLoadWithRetry` hook that also owns the load. Loads differ in staleness guards and resets, so the hook would be a much larger rework for the same visible result.

### The block lives in `shared/ui/LoadFailure.tsx`

`app`, `entities` and `pages` all use it, and only `shared` can be imported by all of them. It reads translations from `shared/i18n`, which is in the same layer.

- Rejected: `widgets` or `features`. `entities` could not import it.
- Rejected: the name `LoadFailureComponent`. `shared/ui` already has both forms, and the shorter name reads better at 39 call sites.

### Screens with one failure state choose the block by data presence

`ProfilePage`, `InstanceSettingsPage`, `StorageComponent` and `NotifierComponent` show `LoadFailure` while their data is absent and the plain `Alert` otherwise, which preserves today's rule that a failed save or other operation offers no retry. A screen test on each pins this rule.

- Rejected: an optional `onRetry` on `LoadFailure`. A block that can lack a retry would no longer guarantee what its name promises.
- Rejected: splitting `failure` into separate load and save states. It changes more state than this cleanup needs and gives the user nothing.

### `AuditEventList` takes one optional `failure` prop

`error` and `onRetry` become one optional `{ error, onRetry }` value, so a list cannot show a failure with a button that does nothing.

- Rejected: keeping both optional and guarding the block with a condition on each. The types would still allow the broken pair.
- Rejected: a required `onRetry`. `WorkspaceAuditLogsComponent` would have to pass a callback that is never used.

### `UsersPage` drops `disabled`

The failure block disappears when the reload requests its page, and `retryUsers` ignores clicks during a load, so `disabled` adds nothing the user can rely on. A screen test holds the reload and pins that one request is sent.

- Rejected: keeping that screen outside the block. A cosmetic difference does not justify another hand-written copy.

### The existing `collectCursorPages` replaces the copies

No new function is added, and the existing tests in `shared/lib/collectCursorPages.test.ts` already cover following the cursor. Each facade keeps its page limit and query, and new API tests pin that for all four.

- Rejected: a new `listAllPages` function. It would be a second function with the same purpose.
- Rejected: adding an early exit to `collectCursorPages` so the healthcheck can use it. That loop would be the only caller of the option.

### `DatabaseConfigComponent` uses `useBackupConfigLoad`, and the retry function becomes `reloadBackupConfig`

The new name is also correct for the reload after a save. The hook's extra loading flag stays unused on this screen.

- Rejected: keeping `retryBackupConfigLoad`. It would read wrongly at the save call site.
- Rejected: a second hook without the loading flag. It would keep the duplication this removes.

### A new wrapper owns the corner under the tabs

The tab panels move into a new `<div>` below `<Tabs>` with the class `[&>:first-child]:rounded-tl-none`. All six cards use plain `rounded-md` and know nothing about the tabs: logical backups, physical backups, verifications, monitoring, healthcheck (`frontend/src/pages/databases/ui/healthcheck/HealthcheckAttemptsComponent.tsx:184`) and configuration (`frontend/src/pages/databases/ui/DatabaseConfigComponent.tsx:244`). The rule therefore sits next to the `<Tabs>` that causes it. A failed healthcheck configuration load renders as the first child, so its error also gets the square corner, which suits an element directly under the tabs.

- Rejected: `first:rounded-tl-none` on each card. Four cards would carry the class and two would keep a fixed square corner, so six components would still encode their position.
- Rejected: no wrapper. `<Tabs>` is the first child, so a first-child rule would never match.
- Rejected: a sibling selector on the AntD class, such as `[.ant-tabs+&]:rounded-tl-none`. It ties the styles to an internal AntD class name.
- Rejected: keeping the state. The healthcheck component would keep reporting its visibility upward for one corner.

### The corner is pinned by a browser scenario

The scenario goes into `frontend/browser/scenarios/databaseResources.spec.ts` and reads the computed top-left radius of the backups card with the healthcheck enabled and disabled. Screen tests run in happy-dom, which applies no Tailwind styles.

- Rejected: a visual reference image. It would fail on unrelated changes to the screen and needs a deliberate update each time.

## Risks / Trade-offs

- [The wrapper changes the DOM around the tab panels, and the scroll region of the database screen depends on layout] → The wrapper has no sizing of its own, and the existing scroll scenario in `databaseResources.spec.ts` runs at each viewport width.
- [A screen whose failure state also holds non-load errors could gain a retry on a failed save, which would reload the screen] → Those four screens are listed above and pinned by tests. Task 3.7 checks every replaced state for setters outside its load.
- [The backups card changes its corner after the healthcheck configuration arrives, on a database with the healthcheck disabled] → Accepted. Today the same flicker happens on a database with the healthcheck enabled.
