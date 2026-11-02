## Context

See `proposal.md` for the motivation and the delta specs for the required behavior. Paths are under `frontend/src`, and line numbers come from commit `864ae947`, before `clean-frontend-residue`, so each task checks them again.

- `pages/databases/ui/DatabaseConfigComponent.tsx:91,104,114,130,161` call `alert` for failed copy, test, delete and rename. `:84,108` call `ToastHelper`. `:399-417` are three icon buttons with no name. `:210` is a clickable `div` around an image with no `alt`.
- `pages/databases/ui/healthcheck/ShowHealthcheckConfigComponent.tsx:29` and `EditHealthcheckConfigComponent.tsx:34,49` use `alert` for failed loads and saves. `HealthcheckAttemptsComponent.tsx:74,98` use it for loads, and line 74 also runs for the refresh every 60 seconds. `pages/databases/ui/show/ShowDatabaseNotifiersComponent.tsx:32` uses it, and the view then spins forever because `:40` waits for a list that never comes.
- `pages/databases/ui/DatabaseComponent.tsx:102-126` draws tabs as clickable `div` elements. `ChoosePostgresBackupTypeComponent.tsx:54` and `EditDatabaseBaseInfoComponent.tsx:69` draw selection cards the same way. `CreateLimitedUserComponent.tsx:226` is a clickable `span`, and `DatabaseSettingsSection.tsx:16` a clickable `div` around an unnamed image.
- `pages/databases/ui/verification/ShowVerificationConfigComponent.tsx:30-60` shows the established load failure: an AntD `Alert` with a retry button. `pages/databases/ui/verification/VerificationsComponent.tsx` guards late answers with `useSessionGuard`.
- `lint/fsd.ts:62-74` and `lint/styles.ts:38-50` compare findings with `fsd-baseline.json`, which is empty, and `style-baseline.json`, which holds four `ToastHelper` entries. `lint/checkProjectDependencies.ts` skips paths outside `LAYERS`.
- `oxlint.config.ts:10-14` disables the React rules. A trial run reports `exhaustive-deps` 70 findings in 36 files, `set-state-in-effect` 43 in 40 files, `refs` 15, `purity` 2, `immutability` 1.
- `oxlint.config.ts:4` lists only `react` and `typescript`. A trial config shows that a `jsx-a11y` rule runs only with the plugin listed there, and that the plugin then also warns by default. The counts are `click-events-have-key-events` 9, `no-static-element-interactions` 9, `alt-text` 3, `control-has-associated-label` 17, `anchor-has-content` 11, `prefer-tag-over-role` 5, `no-noninteractive-tabindex` 3, `no-autofocus` 3 and `anchor-is-valid` 1.
- Of the `exhaustive-deps` findings, 31 flag `.current` in an effect cleanup, about 20 omit a `load*` function, and the rest need reading one by one.
- 21 sites in 15 files write `useRef(new RequestRevision())`. Their cleanup reads `.current`, and the rule warns about that without seeing that the revision object never changes.
- `react/refs` flags `latestX.current = x` during rendering at 8 sites, `ref={parameters.scrollContainerRef}` at `shared/hooks/pagedListScreen.fixture.tsx:52`, the query check at `pages/users/ui/UsersPage.tsx:176-178`, and revision reads at `features/edit-storage/model/useStorageDraft.ts:266` and `features/edit-notifier/model/useNotifierDraft.ts:290`.
- `DatabasesComponent.tsx:122-130` selects a copied database even when another one is selected by then. `pages/storages/ui/StorageComponent.tsx:249` keeps a failed connection test on the screen in state.
- `HealthcheckAttemptsComponent.tsx:60-105` loads the config and the attempts in one effect, keeps the old rows during a period change and keeps polling after a failure.

The approach is bound by the root `AGENTS.md` (comments, naming, tests) and `frontend/AGENTS.md` (AntD only, component structure, design and localization skills). It changes "Keep state near its lifecycle" and the check table in `docs/architecture.md`, in this change.

## Goals / Non-Goals

**Goals:**

- No tolerated dependency or style finding, no React or accessibility rule disabled without a stated reason, and only single-line exceptions in code.
- One feedback mechanism on the frontend.
- Effects fire when they fire today. A firing or request change happens only for a defect reproduced by a test, and `coverage.md` names the site.

**Non-Goals:**

- A new visual language for feedback or forms.
- Changing polling intervals or the set of requests.

## Decisions

### 1. Action outcomes use AntD feedback, and load failures stay on the screen

The copy and connection-test successes use `notification.success` from `App.useApp`, as `StorageComponent.tsx:76` does, so existing dictionary keys and copy stay. Rename and delete have no notice today and get none: the new name and the shorter list show the result.

A failed action's reason stays in state, as `connectionError` does in `StorageComponent`, and renders as a closable `Alert`: the error itself is kept, or the server's connection text, and `translateApiError` or `t` runs at render. This follows `frontend-localization`, which limits the transient exception to success notices, and a long server reason does not vanish before it is read. The next action of that database clears it, and so does leaving the settings tab or selecting another database, because the state goes with the unmounted component. The reason describes an action on the screen the user just left, so it does not need to outlive that screen.

A failed load renders the established `Alert` with a retry, and the healthcheck editor adds a close action, as the verification editor does. A refused save stays inside the section. A failed background refresh changes nothing on the screen, following the rule for polled lists in `frontend-list-loading`.

In `HealthcheckAttemptsComponent` the config and the attempts each keep their own failure. A period change clears the old rows before its load. Polling starts only after a load succeeds and stops on a failed non-background load, so a tick cannot replace the failure, and the retry starts the load again.

Alternatives rejected:

- Reporting load or action failures through `message.error`. A load would stay empty with no way to recover, the failure would repeat with polling, and a fired message cannot follow a language change.
- Rename and delete success notices. They add copy that says what the screen already shows.
- A shared `Alert` with retry. Many files write it inline, so a helper for the new sites would add a second way of doing it, and replacing all of them goes beyond this change.

### 1a. Late answers follow the database they belong to

`DatabaseConfigComponent` unmounts in two cases while an action is pending: another tab opens (`DatabaseComponent.tsx:135`), or another database is selected, because `DatabaseComponent.tsx:63-65` clears the shown database and `:95-97` renders a spinner. Its `useSessionGuard` therefore answers false in both cases, so it can guard only what belongs to the component: local state and feedback. The callbacks to the list always run and carry the source database's ID, and `DatabasesComponent`, which owns the selection and stays mounted, decides behind its existing `isCurrent` check. `DatabaseComponent` stays mounted across a selection change but can still be mounted when the session ends, so `acceptChangedDatabase` (`DatabaseComponent.tsx:48-53`) gets its own `useSessionGuard` before `setDatabase`, and a late rename changes nothing after the session ended.

| Action | Same database, same tab | Same database, another tab | Another database selected | Screen left or session ended |
| --- | --- | --- | --- | --- |
| Copy | Notice, copy selected | Copy selected | Copy joins the list unselected | Nothing |
| Rename | Editor closes, list updates | List updates | List updates, as the existing late-rename test pins | Nothing |
| Delete | Removed, selection moves | Removed, selection moves | Deleted database removed, selection kept | Nothing |
| Test | Notice or reason | Nothing | Nothing | Nothing |
| Any failure | Reason on the screen | Nothing | Nothing | Nothing |

`onDatabaseCopied` becomes `(copiedDatabase, sourceDatabaseId)` and `onDatabaseDeleted` becomes `(deletedDatabaseId)`. The handler that receives them comes from the render that started the action, so its `selectedDatabaseId` is stale. `DatabasesComponent` therefore reads the current selection through `useLatestRef`: it selects a copy only when the source is still selected, and it moves the selection in a delete only when the deleted database is still selected. `removeSelected` becomes `removeDatabase(id)`, which `onDatabaseTransferred` also uses. Tasks reproduce the copy and delete cases first.

Alternatives rejected:

- Guarding the callbacks with `useSessionGuard`. They would stop after a tab or selection change, leaving a deleted database in the list and breaking the late-rename requirement.
- Tracking the shown database inside `DatabaseConfigComponent`. An unmounted component's ref stops updating, so it would still report the old database as shown.
- Adding `key={database.id}` to remount on selection. The component already unmounts, so this changes nothing for late answers.

### 2. Tabs become AntD `Tabs`, and cards become buttons

The database tabs use `Tabs` with `type="card"`, `activeKey` and items without children. The content stays where it is rendered, so no tab remounts and the scroll owner does not move. AntD supplies roles, arrow keys and focus. The look changes slightly, and no reference image covers these tabs.

Selection cards become native `button` elements with `aria-pressed`. Edit icons become `Button` with `aria-label` and an `img` with empty `alt`. The privileges toggle is a button with `aria-expanded` only when it can expand, and plain text otherwise. Notifier logos and the engine card logo at `EditDatabaseBaseInfoComponent.tsx:76` get an empty `alt`, because the name sits beside them, so a card announces its engine once.

The transfer, copy, delete and name-editor close actions (`DatabaseConfigComponent.tsx:227`) use `common.actions.transfer`, `.copy`, `.delete` and `.close`. `common.actions.transfer` is new. The logo links at `widgets/app-shell/ui/AppShell.tsx:38` and `pages/auth/ui/AuthNavbarComponent.tsx:19` hold only an `img` with empty `alt`, so they have no name today, and they get `aria-label` from a new `common.websiteLink` key. Both keys go into all six dictionaries through the frontend-localization skill.

Alternatives rejected:

- `role="tab"` on hand-written elements. It needs roving focus and key handling that AntD already has.
- `Radio.Group` for cards. Restyling it costs more than a button, and the cards hold several lines of content.

### 3. The remaining unlabeled controls are checked by hand

A scan found these controls with no `aria-label`, `id` or field wrapper. Some have a visible neighbor that is not connected to them, and others may have no label at all:

| File | Controls |
| --- | --- |
| `features/edit-storage/ui/edit/storages/EditAzureBlobStorageComponent.tsx` | `Radio.Group:30` |
| `pages/databases/ui/DatabaseConfigComponent.tsx` | `Input:218` |
| `pages/databases/ui/DatabaseTransferDialogComponent.tsx` | `Radio.Group:406,487` |
| `pages/databases/ui/EditDatabaseBaseInfoComponent.tsx` | `Input:52` |
| `pages/databases/ui/backup-config/LogicalRetentionFields.tsx` | `Select:85` |
| `pages/databases/ui/healthcheck/EditHealthcheckConfigComponent.tsx` | `Switch:72,96`, `Input:119,149,179` |
| `pages/databases/ui/healthcheck/HealthcheckAttemptsComponent.tsx` | `Select:128` |
| `pages/databases/ui/logical-backups/LogicalBackupsFiltersPanelComponent.tsx` | `Select:45`, `DatePicker:60` |
| `pages/databases/ui/physical-backups/PhysicalBackupsFiltersPanelComponent.tsx` | `Select:49,64`, `DatePicker:79` |
| `pages/databases/ui/physical-backups/PhysicalRestoreComponent.tsx` | `DatePicker:102`, `Segmented:132` |
| `pages/databases/ui/physical-backups/PhysicalRestoreEnvironmentPanel.tsx` | `Input:77,88,102` |
| `pages/databases/ui/wal-stream-monitoring/WalStreamChartsComponent.tsx` | `Segmented:391` |
| `pages/instance-settings/ui/verification-agents/CreateAgentDialog.tsx` | `Input:53` |

A visible label connects through `useId` and `htmlFor`. A control without one gets `aria-label` from an existing key where possible. Screen tests query the repaired controls by role and name.

Alternative rejected: enabling `jsx-a11y/control-has-associated-label` to find them. Of its 17 findings, 2 are the logo links above and 15 are links and buttons with translated children that it cannot see, and it does not look at AntD inputs.

### 4. The baseline mechanism is deleted, not emptied

`lint/fsd-baseline.ts`, `lint/styleBaseline.ts`, both JSON files and `tooling/styleBaseline.test.ts` go. `lint/fsd.ts` and `lint/styles.ts` print each finding and exit with an error. `ArchitectureFinding` moves to `lint/checkProjectDependencies.ts`, where the check produces it. `removalMilestone` and `owner` disappear with the mechanism.

`lint/frontendDependencies.ts` exports `getSourceFiles`, and a new `project/file-outside-layers` check reads that file list rather than the dependency edges, because a module with no imports and no importers has no edge. It covers the script modules that walker matches, except `main.tsx` and `vite-env.d.ts`, so `index.css` stays accepted.

The lint entry points take an optional source root, so tests start them as processes over a fixture and see their real exit code and diagnostic.

Alternative rejected: keeping an empty baseline. A working mechanism invites the next exception, and the spec says none can be recorded.

### 5. Effect dependencies stay stable through `useEffectEvent` and two small hooks

- **Functions an effect only calls** become `useEffectEvent`. The effect keeps its dependency list and fires when it does today, while the function reads current values. A trial lint confirms that oxlint 1.87 accepts this, and React 19.2.6 exports it.
- **`useRequestRevision`** in `shared/hooks` returns one stable `RequestRevision` for the component's lifetime. It replaces the 21 `useRef(new RequestRevision())` sites and resolves their cleanup findings. A trial lint confirms the rule accepts a stable hook result listed in the dependencies.
- **`useLatestRef`** in `shared/hooks` writes a value to a ref in `useLayoutEffect`. Async handlers read that ref for acceptance checks, and `useEffectEvent` cannot serve them because it may be called only from effects. Layout effects run before any passive effect, so effects still read current values. The hook replaces the 8 render writes and lets `DatabasesComponent` read the current selection in late answers.
- **The fixture** at `pagedListScreen.fixture.tsx:52` destructures its ref before rendering.
- **The query check in `UsersPage`** moves to `useLayoutEffect` on `searchQuery`, with the map of acknowledged users cleared there. A layout effect runs before any late answer can arrive, so the old answer cannot pass.
- **DOM refs in cleanup**, such as the focus return in `StoragesComponent.tsx:116`, are reviewed one by one, because copying the node at setup can capture nothing when the node appears later. The node does appear later at each site, and a handler cannot focus it either, because the button is not rendered until the form closes. So the cleanup calls a `useEffectEvent` that reads the ref when the form closes.
- **The rest of the missing dependencies** are read individually, including `WalStreamMonitoringComponent.tsx:140,173`, `useEasedTransition.ts:31`, `StorageSwitchHistoryComponent.tsx:109,130`, `EditBackupConfigSection.tsx:95`, `CreateLimitedUserComponent.tsx:154` and `useScopedSelection.ts:49`. By default a value stays out of the firing through `useEffectEvent`. A dependency that retriggers is added only for a defect a test reproduces first, and `coverage.md` names it. Language must not restart polling or free the full backup button.

Alternatives rejected:

- Adding every dependency mechanically. `load*` functions change on each render, so loads and polling would loop.
- Suppressing the 70 findings. The rule would catch nothing in existing code, and each comment would state a reason already known to be repetitive.

### 6. A few sites keep a stated exception

- `useStorageDraft.ts:266` and `useNotifierDraft.ts:290` read the settings revision during rendering. Every invalidation comes with a state update, so the read cannot go stale, and mirroring it in state would add a second source of the same value to tested code.
- `RestoreCard.tsx:43` reads the clock during rendering, because the elapsed time updates with each poll and a slightly old value does no harm.
- `useGoogleDriveAuthorization.ts:136` runs only from the authorization button, so the clock read moves into a function outside the hook.
- `DatabasesComponent.tsx:305` moves into a named handler.

Each exception uses a single-line disable comment with its reason, like the existing `i18next` ones. Lint cannot check that a reason is present, so code review applies that rule; `docs/architecture.md` states it.

### 7. `react/set-state-in-effect` stays off, with its reason in the config

Its 43 findings in 40 files are mostly a pending flag or a reset at the start of a load. Removing them means rewriting how nearly every screen loads, which puts its tested late-answer guarantees at risk with no user-visible gain. The plan allows a disabled rule with a stated reason. The reason goes beside the rule in `oxlint.config.ts`, and `docs/architecture.md` names the decision.

Alternative rejected: enabling it with 43 suppressions, which states the same reason 43 times and enforces nothing.

### 8. The `jsx-a11y` plugin runs four rules, and the others it would warn about are decided one by one

The config lists the plugin, because the rules do not run without it. `click-events-have-key-events`, `no-static-element-interactions` and `alt-text` are errors. `anchor-is-valid` joins them, because its one finding was a real one and is repaired. `control-has-associated-label` and `anchor-has-content` are off, because they cannot see translated children. `prefer-tag-over-role`, `no-noninteractive-tabindex`, `no-autofocus` and `anchor-is-valid` are read during the task: a real finding is repaired, and a rule with only deliberate findings goes off with its reason beside it. Lint output ends with no warning.

Alternative rejected: leaving them as warnings. `pnpm lint` passes with warnings, so nobody would act on them.

### 9. Rule tests run oxlint with the repository config

A clean tree passes lint whether or not a rule is active, so a new `tooling/frontendLintRules.test.ts` runs oxlint with `oxlint.config.ts` over temporary fixtures. Each rule group must fail, and a single-line disable must pass while the same finding on the next line fails. A trial config confirms that oxlint honors `eslint-disable-next-line` for these rules.

## Risks / Trade-offs

- [A dependency change makes polling fire more often] → Existing tests pin intervals and request counts, and `mockApi` fails on any undeclared request. Hand-reviewed sites get a test when their firing changes.
- [`useLatestRef` updates after commit rather than during rendering] → The frontend uses no transitions or deferred values, so rendering and its layout effects run in one synchronous pass and no late answer can arrive between them. The existing late-answer tests run after the change.
- [`react/refs` flags the result of `useLatestRef`] → Task 1.2 checks this with lint when it adds the hook. If it does, the hook returns a getter function and is renamed `useLatestValueGetter`, in the code and in this change's artifacts.
- [The `Tabs` look differs slightly] → This is accepted for the keyboard support, and the design tokens still style it. Browser UI runs in both themes.
- [Screen tests rely on `window.alert`] → Tests that stubbed it move to role queries as part of each task, and they keep asserting the same failure.
