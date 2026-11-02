## Context

See `proposal.md` for the motivation and the delta specs for the required behavior. Paths are under `frontend/src`. Line numbers come from the checkout before `move-physical-and-verification-screens`, under `features/backups/physical`. All defects were found by reading the code; each task reproduces its defect.

- `ui/PhysicalBackupsComponent.tsx:233` has no `catch` and sets the loading flag only in `then`, and `:418` replaces the screen with a spinner while it is set. `:230-237` follows `[database]`.
- `:227` passes every `onError` to `alert`, and `firstLoadError` from `shared/hooks/usePagedList.ts:80` is unused, so a failed first page renders `:551`, the empty text. `:215` defaults `totalUsageMb` to 0, and `:460-462` renders it unconditionally. `:190` and `:207` use `alert`. `:165-215` has no guard, so a late start, cancel or delete can refresh the list or show a message after the user left.
- `:465` shows the start to everyone. `backend/api/openapi.yaml:2186` requires `WORKSPACE_MEMBER`.
- `:174-175` waits one second and reloads, and `shared/lib/PagedListController.ts:64` resets the list on a reload, which loses the tail and the scroll. The operation answers 202 with no body and only records the request (`openapi.yaml:2186-2195`), so the backup appears on a later poll.
- `pages/databases/ui/logical-backups/LogicalBackupsComponent.tsx:43-46`, `:127-145` and `:205-217` already hold the required configuration load, its effect guard, its error and retry, and the disabled text. `useLogicalBackupActions.ts` guards actions with `useSessionGuard`. `formatLogicalBackupSize.ts` does the work of `:52-59`.
- `model/physicalBackupStatus.ts` holds the badge colors, and the style check resolves them to the list. `ui/PhysicalBackupTypeCellComponent.tsx` uses indigo for `FULL` and `INCREMENTAL` and slate for `WAL`.
- `ui/PhysicalRestoreComponent.tsx:123-145` has no guard, so any answer sets `bundleUrl` and `isGenerating`. `:157-159` issues a token per target time, and `:422` turns a cleared picker into `undefined`, which requests the latest restorable moment.
- `monitoring/ui/WalStreamMonitoringComponent.tsx:169-180` creates `FullBackupRequestController` on `[database.id, message, t]`. The comment at `monitoring/lib/fullBackupRequestController.ts:48-49` records the workaround. `monitoring/lib/walStreamRefreshController.ts:74` lets `retry()` start the interval again after `stop()`, and the request's late `refreshNow` at `WalStreamMonitoringComponent.tsx:173` is the only caller that can reach it after `stop()`.

Monitoring's behavior is specified in `physical-backups/wal-stream-monitoring`, and its colors follow "Theme infrastructure stays below app" in `docs/architecture.md`. The approach is bound by the root `AGENTS.md` (logical units, names, refactoring within a feature) and `frontend/AGENTS.md` (component structure, design skill). It follows "Keep state near its lifecycle" and "Shared session state owns account transitions" in `docs/architecture.md`. No architecture rule changes.

## Goals / Non-Goals

**Goals:**

- Failures that end in a recoverable state, with no `alert`.
- One owner for each request, whose work ends with its screen or session.
- A list and a restore screen split by concept, with the same flows and commands.

**Non-Goals:**

- Changing monitoring's schedules, calculations or tests beyond the one controller fix.
- Merging polling lifecycles into one hook.

## Decisions

### 1. The list takes the logical list's structure and shares its configuration load

`PhysicalBackupsTable`, `PhysicalBackupCards`, `PhysicalBackupActions`, `PhysicalBackupStatusBadge` and `usePhysicalBackupActions` sit in `ui/physical-backups`, as their logical counterparts do. WAL rows and the gap badge stay physical. `usePhysicalBackupActions` guards its refreshes and feedback with `useSessionGuard`, as `useLogicalBackupActions` does.

`useBackupConfigLoad(databaseId)` and `BackupConfigNotice` move out of `LogicalBackupsComponent` into `ui/backup-config`, and both lists use them. Their effect invalidates a late answer on cleanup. `formatLogicalBackupSize` becomes `formatBackupSize` in `pages/databases/lib`, because it is about backups and not their configuration. The contract requires `sizeMb` on both rows, so the physical `?? 0` goes with its copy. The logical list's private icon action button becomes `ui/BackupActionButton.tsx`, because the physical actions need the same keyboard button with a name.

Alternatives rejected: one generic backup list over both types would need options for WAL ranges, chains, verification and download, and those differ by nature. A second copy of the configuration load would duplicate about 40 lines and their tests.

### 2. A manual backup relies on the poll

After 202, the hook only ends its busy state. `usePolledPagedList` already refreshes the first page every second and keeps the tail, and that shows the backup whenever the scheduler creates it and the current filters include it.

Alternative rejected: keeping the wait and the reload preserves the spinner and loses the scroll position, against `frontend-list-loading`. Delete keeps its reload, as the logical list does, because a removed row cannot be merged out of a first page.

### 3. Viewers lose the start and keep restore

`isCanManageDBs` decides whether "Back up now" and its menu exist, as it already does for cancel and delete. Restore needs only `WORKSPACE_VIEWER` (`openapi.yaml:2261`). After a failed first page, the start is disabled rather than absent, as in the logical list.

Alternative rejected: leaving the button and showing the refusal keeps a control that can only fail. Changing the contract so viewers can start physical backups, as they can start logical ones, is a product decision outside this migration, so the proposal names it as out of scope.

### 4. The restore token uses one `RequestRevision`

`usePhysicalRestoreToken` takes the restore token subject: one backup, or a database with an optional target time. It invalidates the revision on each request and on unmount, and it applies the answer, the error and the end of the pending state only for the current revision. The screen shows commands only when the hook holds a token, which also replaces the earlier flash of old commands. A refusal of the newest request clears the commands and shows the reason with its hint.

Alternatives rejected: an `isActive` flag per effect also works, but `RequestRevision` is the project's named unit for this. Debouncing the date picker reduces requests but does not stop a late answer.

### 5. One controller per database and an explicit end of reporting

`FullBackupRequestController` is created in an effect on `[database.id]`. Its callbacks use `i18n.t`, so they read the current language without recreating it, as the agent screen does, and `message` comes from the stable app context. The effect's cleanup calls a new `stopReporting()`, after which a late answer reports nothing and calls no `refreshNow`. That removes the only path to `retry()` after `stop()`, so `WalStreamRefreshController` does not change. The comment about a recreated controller is deleted, because the case no longer exists.

Leaving means opening another tab of the database page or selecting another database, because those unmount the component. Hiding the browser tab keeps the component mounted, so its request still reports.

Alternatives rejected: keeping callbacks in a ref updated on each render works but adds a second lifecycle for what `i18n` already provides. A stopped state in `WalStreamRefreshController` would be a second guard for a path that no longer exists.

### 6. Semantic roles, without a new palette

Text and borders map to `text-muted`, `text-content`, `border-border` and the surface roles. Status badges map `COMPLETED` to `success`, `IN_PROGRESS` to `accent`, `ERROR` to `danger`, `CHAIN_BROKEN` to `warning` and `CANCELED` to `muted`, with their words kept. The type cell maps `FULL` to `accent` with semibold text, `INCREMENTAL` to `accent` with ordinary weight and a lighter background, and `WAL` to `muted`, which keeps today's three visibly different badges. The WAL series colors in `chartColors.ts` keep their exception, and the tones in `toneColors.ts` already use tokens.

Alternative rejected: new indigo and slate roles for the type cell would add theme entries for one badge. The type word already carries the meaning.

## Risks / Trade-offs

- `FULL` and `INCREMENTAL` move from indigo to the accent color, and `WAL` moves from slate to the muted role. The browser scenario's accessibility scan checks their contrast in both themes.
- After a manual backup, the new row appears on a later poll, or not at all if the current filters exclude it. The poll runs every second, so the delay depends on the scheduler, as it already did.
- A dropped button for viewers is a visible change. It replaces a request that always failed.
- `stopReporting()` drops the message if the user leaves before the answer arrives, including a refusal. The request still runs on the server. On return, the tab shows the pending state after a success, and it offers the button again after a refusal, so a silent refusal costs one more click.
- Returning to the Monitoring tab while the first request is still unanswered creates an idle controller, so the button can send a second request until the statistics report the pending backup. The window is the length of one request, and a database can have only one running backup.
- Rollback: reverting the commit restores the old screens. No stored data changes.
