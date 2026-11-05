## Context

See `proposal.md` for the motivation and the delta specs for the required behavior. Paths are under `frontend/src`. Line numbers come from the checkout before `move-physical-and-verification-screens`, under `features/verification`. The defects were found by reading the code; each task reproduces its defect.

- `runs/ui/VerificationsComponent.tsx:227` passes `onError` to `alert`, and `firstLoadError` stays unused, so the empty text renders. `:157` uses `alert` for a failed cancel. `:43-65` formats a duration as hours, minutes and seconds. `:295-311` shows the total and the restore duration, and the mobile cards at `:395-439` show no duration.
- `runs/ui/VerificationDetailDrawer.tsx:124` calls `alert`, the only place that shows the reason, and then sets the boolean `hasLoadError`, which renders the fixed text `verification.runs.details.loadFailed` at `:177-179`. `:25-42` has a second duration format, which never shows hours, and `:237-244` shows the restore and verify rows. `shared/lib/formatDuration.ts` is the third format.
- `agents/ui/VerificationAgentsComponent.tsx:287-299` reports every failed load through `message.error`, and `:310-317` repeats the load every 15 seconds. `hasOpenModal` at `:278` pauses the refresh while a dialog is open, although a refresh changes only `agents`, which the table and the empty text read, and no dialog state (`:73-85`). `:265` removes a deleted row as soon as the server confirms. `:77-79` keeps the token in state, `:99-103` clears it, and `:583` and `:596` use `TOKEN_PLACEHOLDER` for later install commands. The test at `agents/ui/VerificationAgentsComponent.test.tsx:14` pins the clearing only.
- `verificationAgentApi.listAgents` walks `nextCursor` with `limit: 100` and returns the merged array. `shared/hooks/usePagedList.ts` and `shared/lib/PagedListController.ts` provide `firstLoadError`, `reload`, a silent `refreshFirstPage` that ignores late answers, and `replaceOrPrependItem`.
- `:90-320` defines plain functions, then hooks, in an order that reads hook results before their declarations. The component holds four dialogs, a delete `Popconfirm` and the table in one 603-line file.

The approach is bound by the root `AGENTS.md` (logical units, names, refactoring within a feature) and `frontend/AGENTS.md` (component structure, clipboard helper, design skill). It follows "Start in the page" in `docs/architecture.md`. No architecture rule changes.

## Goals / Non-Goals

**Goals:**

- The same failure behavior as the logical and physical lists.
- One duration format, and files that each hold one concept.
- The token and install command guarantees kept, and pinned by tests.

**Non-Goals:**

- New agent features, and any change to intervals or API operations.

## Decisions

### 1. The verification list follows the other lists

`VerificationsTable`, `VerificationCards`, `VerificationActions` and `VerificationStatusLabel` sit in `ui/verification`, and `firstLoadError` renders an `Alert` with a retry. `VerificationsComponent` keeps `cancelVerification` and its pending ID under `useSessionGuard`, so a late answer reports nothing, because the list has only this one action and the pending ID blocks cancelling a second row. The `hasAgents` hint stays as it is, including its silence on a failed count, because a failed count says nothing about agents.

Alternatives rejected: a generic list shared with backups, for the reasons `refactor-physical-backup-screens` gives. An actions hook like `useLogicalBackupActions` would be a file for ten lines.

### 2. Durations use the shared function

The list and the drawer call `formatDuration` from `shared/lib`, and `pages/databases/ui/verification/formatOptionalDuration.ts` wraps it to return `-` for a missing value. Both local copies are deleted. A verification under a minute then reads `0m 5s`, as the physical list does. The comparison in the spec uses the restore duration, the one value both views show, and the mobile cards stay without a duration.

Alternative rejected: keeping a verification-specific format saves that small visible change, but it keeps a third implementation of the same calculation. Teaching the shared function a seconds-only form would change the physical and logical lists, which this milestone does not need.

### 3. The drawer keeps the caught error and stays mounted through its close

`loadError` replaces `hasLoadError`, and the failure renders `translateApiError(loadError, t)` with the retry. `verification.runs.details.loadFailed` is deleted from all six dictionaries.

`VerificationDetailDrawer` takes an `open` prop, and `VerificationsComponent` keeps the last verification ID until `afterOpenChange(false)`. Today `VerificationsComponent.tsx:456-461` unmounts the drawer on close and `VerificationDetailDrawer.tsx:170` fixes `open={true}`, while rc-drawer returns focus only after the close transition of a drawer that stays mounted.

Alternative rejected: keeping the fixed text removes the only place that states the reason once the alert goes.

### 4. The agent screen uses the shared paged list, with one file per dialog

`usePolledPagedList` with a constant `resetKey`, the 15-second interval and `fetchPage: async () => ({ items: await verificationAgentApi.listAgents() })` owns the list. It has no `nextCursor`, so `mergeRefreshedFirstPage` returns the fresh list and a vanished agent leaves. One small interval in the screen ticks `currentTimeMs` for the last-seen status, and `hasOpenModal` goes. A created agent goes through `replaceOrPrependItem`. A confirmed deletion calls `reload`, as `useLogicalBackupActions.ts:90` does: it empties the rows at once and invalidates a refresh in flight, so the deleted agent leaves and cannot come back.

Reads rely on `ApplicationScreen` being remounted by `session.generation` (`app/ui/App.tsx`), as the other lists do, and on `usePagedList` invalidating late answers on unmount. The poll cases of `suppresses old agent completion after session replacement` are therefore removed, because nothing can render a late answer once the screen has gone. `sends no request after an unmount and ignores an answer that arrives after it` and the `PagedListController` tests own that behavior, and the delete cases stay. The top component runs create, rotate, delete and the clipboard copy under its existing session guard, and passes only callbacks to the dialogs and the table.

`VerificationAgentsTable`, which keeps the delete confirmation, `CreateAgentDialog`, `RotateAgentTokenDialog`, `RevealedAgentTokenDialog` and `AgentInstallCommandsDialog` take props, and `InstallCommands` holds the shared command blocks. The top component holds which dialog is open, the revealed token and `selectedArch`, so the chosen architecture still carries over between the token and install command dialogs.

Alternatives rejected: a hand-written hook would repeat the revision, the first-load state and their tests. Keeping a separate interval to pause behind a dialog protects nothing, as the context shows. A silent refresh after a deletion keeps a deleted row on screen while refreshes fail. A new row removal in the shared controller avoids the loading state, but it adds shared code to make this list differ from the logical and physical ones. Keeping one file with `render*` helpers leaves four dialogs and their state interleaved, against "Code is split into logical units".

### 5. Only the first load reports a failure, and only a retry recovers from it

`refreshFirstPage` does nothing until a first load has succeeded, so while the first-load failure is shown, the periodic tick sends no request and only the retry reloads. This matches the other lists. Create is disabled during that failure, as the logical list disables its start, because a created row would sit in a list that cannot refresh. It is also disabled while a deletion or any list load runs, the reload after a deletion included, through `isLoading || firstLoadError !== undefined || deletingAgentId !== undefined`, so no create dialog can be open when the failure appears. A mutation keeps its own feedback.

Alternative rejected: letting the tick clear the failure recovers by itself as today's screen does, but it needs its own state beside the controller for an outage the retry already handles.

### 6. The token stays in memory in the top component

The revealed token stays in React state of the top component, and `RevealedAgentTokenDialog` receives it as a prop, so closing clears it in one place. A new spec requirement pins this because it is security-relevant.

Alternative rejected: keeping the token inside the dialog looks tidier, but creation and rotation both produce it, and the dialog would need to run those requests too.

### 7. Semantic roles and the code block

Text, borders and surfaces take semantic roles. `#ff0000` and `#155dfc` in actions become `danger` and `accent`. The dark command block on the agent dialogs uses the `code` role, which `shared/ui/InlineCodeComponent` already uses.

Alternative rejected: an always-dark terminal look needs raw colors, and it is no clearer than the theme's code role.

## Risks / Trade-offs

- A verification shorter than a minute reads `0m 5s` instead of `5s`. It is a small visible change in exchange for one implementation, and the proposal names it.
- During an outage, an admin who keeps the screen open sees agents that may be out of date, without a message. Each agent's "last seen" age keeps ticking, and `currentTimeMs` keeps that calculation live, so its status still moves to stale.
- After a failed first load, the screen no longer recovers by itself when the server returns. The retry does that, as in the other lists.
- A refresh already in flight when an agent is created can replace the list without the new row, behind the token dialog. The row returns with the next refresh that succeeds.
- A confirmed deletion shows a brief loading state, and a failed reload replaces the remaining agents with the failure and its retry, as in the logical list.
- Splitting the dialogs risks losing the focus return to the invoker. AntD modals return focus, the drawer returns it only because decision 3 keeps it mounted through its close, and the browser scenario checks the drawer and one split dialog.
- Rollback: reverting the commit restores the old screens. No stored data changes.
