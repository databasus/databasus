## Context

See proposal.md for motivation. The state this design starts from, with paths relative to `frontend/src`:

- `pages/databases/ui/DatabasesPage.tsx:12` only forwards to `features/databases`. It passes the slots built in `pages/databases/ui/databaseResourceComposition.tsx:7`, which `features/databases/model/DatabaseResourceComposition.ts:29` defines.
- `features/databases/ui/DatabasesComponent.tsx:31` keeps its own localStorage selection. Line 186 sizes the list from `contentHeight`, which `app/ui/ApplicationScreen.tsx:30` stores from `widgets/app-shell/ui/AppShell.tsx:38`. No other screen reads it.
- `DatabasesComponent.tsx:245` answers every section save with a non-silent `loadDatabases()`. Line 150 then renders a spinner instead of the screen, so `DatabaseComponent` remounts on its default tab (`features/databases/ui/DatabaseComponent.tsx:36`). The connection section closes only because of that remount. A failed first load shows an alert and then an empty list.
- `features/databases/ui/DatabaseConfigComponent.tsx` holds six edit flags (lines 60-66) and a JSON clone of the whole database for renaming (line 167). On close, `loadSettings` (line 81) clears the database and reloads it, with no `catch` and no check for a newer selection. The reload is not needed for the healthcheck and verification summaries: each section renders either its editor or its summary, and the summaries fetch their config on mount (`features/databases/ui/healthcheck/ShowHealthcheckConfigComponent.tsx:20`, `features/verification/config/ui/ShowVerificationConfigComponent.tsx:37`), so closing an editor already shows the stored values. The backup config saves at lines 367 and 381 report the stale `database`. Copy and transfer call `window.location.reload()` (lines 100 and 568).
- `DatabaseConfigComponent.tsx:128` hides the last backup error after a successful connection test. The server keeps the error (`openspec/specs/databases/spec.md`, "A connection test reports its outcome and changes nothing"), so it returns on the next reload, which the following `onDatabaseChanged` triggers at once.
- `features/databases/ui/CreateDatabaseComponent.tsx:186` passes `initialConfig` to the physical config editor. The logical editor at line 202 has no such prop, and with an empty database ID it starts from defaults (`features/backups/logical/ui/EditLogicalBackupConfigComponent.tsx:234`).
- `DatabaseComponent.tsx:43` owns the scroll container whose ref the backup and verification lists use for paging. `shared/ui/ResourceListDetail.tsx` creates its own scrolling detail region and exposes no ref to it.
- `fsd-baseline.json` holds eight `fsd/forbidden-imports` entries owned by `databases` (wizard, `DatabaseComponent` and `DatabaseConfigComponent` importing backups and verification). It also holds two restore entries for `features/restores/ui/RestoresComponent.tsx:26`, which imports a private file of `features/databases`.

The approach is bound by these rules: root `AGENTS.md` (logical units, naming, no compatibility shims, refactoring within the feature) and `frontend/AGENTS.md` (component structure, AntD only, design and localization skills, verification). Placement follows `docs/architecture.md#frontend`: start in the page, use each slice's public API and keep state near its lifecycle.

## Goals / Non-Goals

**Goals:**

- `pages/databases` owns the database screen and composes features from above.
- `features/databases` keeps only the forms that restore and the page both use.
- Every behavior fix is pinned by a test that fails on the code before it.

**Non-Goals:**

- Replacing `isSaveToApi`, `isShowBackButton` and the other mode flags of the connection forms. 5.3 replaces them together with the form.
- Rewriting the backup, restore, monitoring or verification components beyond the two narrow props named below.

## Decisions

### 1. Move first, then change behavior, inside one change

The first implementation task moves the files with their tests and changes only imports and the deleted slots. Each later task changes one behavior and adds its failing test first. The user chose one change over two. Separate tasks keep the move reviewable during apply, even though the archive produces one commit.

Rejected: changing files while moving them, because the diff would then show deletions and new files with no visible behavior delta.

### 2. The forms stay in `features/databases`, behind its public API

`features/databases/index.ts` exports the connection, base-info, PostgreSQL backup type and read-only user forms. The page and restore import them from there. The restore `project/private-slice-import` entry disappears. The baseline identifies a finding by rule, source and target (`frontend/lint/fsd-baseline.ts`), so the restore `fsd/forbidden-imports` entry is replaced by one targeting `features/databases/index.ts`, with owner `restores` and milestone 5. It stays until 5.3 passes the form to restore through a slot.

Rejected: moving the forms into `pages/databases` now. Restore is a feature and cannot import a page, so 5.3 would have to happen inside this change.

### 3. The screen's units stay local to the page

`pages/databases/ui` holds the list page, the card, the detail with its tabs, the settings sections, the creation wizard, the show components and `ui/healthcheck`. Each keeps one responsibility, so the page does not grow into one large component. One-off cards and panels do not move to entities, because no second consumer exists (`docs/architecture.md`, "Start in the page").

Rejected: a `features/database-settings` slice. It would have one consumer and would bring back the sibling imports this change removes.

### 4. Selection and layout reuse the 4.3 primitives

The list page follows `pages/storages/ui/StoragesComponent.tsx`: a collection tagged with its workspace, `useScopedSelection` with the key `selectedDatabaseId_<workspaceId>`, an explicit load error with retry, and `ResourceListDetail`. The stored key keeps its current format, so existing users keep their remembered database. The 30-second visibility-aware refresh and its silent failure stay as they are.

`ResourceListDetail` gets an optional ref to its detail scroll region. `DatabaseComponent` passes that ref to the backup and verification lists instead of creating a scroller of its own.

Rejected:

- A nested `h-full overflow-y-auto` container inside the detail. It creates a second scroll candidate and breaks the one-scroll-owner rule in `docs/frontend-design.md`.
- Keeping `contentHeight`. That is the transition the shell documentation says to remove.

### 5. One `editingSection` and a name draft

The settings component holds `editingSection: 'name' | 'connection' | 'backup-config' | 'healthcheck' | 'verification' | 'notifiers' | undefined` and a separate `nameDraft` string. Edit controls of other sections are hidden while a section is open. The JSON clone and the `editDatabase` state lifted into `DatabaseComponent` are deleted.

Rejected: keeping the flags and closing the others on open. That silently discards the open section's input, while the 4.3 rule keeps the other modes unavailable.

### 6. Saves are accepted in place

The page keeps the selected database's state and the list.

- A section that saves the database passes the server's response up. The detail replaces its database with it, and the page replaces the list item and refreshes the list silently (the `acceptStorage` pattern in `StoragesComponent`).
- A backup config save changes only the storage shown on the card, so it triggers just the silent refresh.
- Closing any section, by save or Cancel, only closes it. The healthcheck and verification editors keep their single `onClose`; their summaries remount and fetch the stored config, which is a config request, not a database request.

Rejected:

- Reloading the database after every close. It flashes the screen and repeats a request the user did not ask for.
- Splitting `onClose` into `onSaved` and `onCancel` with a revision key for the summaries. The remount already re-reads the stored config, so the split would widen two feature APIs for nothing.

### 7. Copy and transfer update the collection

Copy accepts the returned database into the list, selects it and refreshes silently. Transfer calls `selection.remove` for the moved database, drops it from the list and refreshes silently, the same path as delete.

Rejected: keeping the reload. It drops unrelated drafts and the user's place in the application.

### 8. The wizard has one state owner

The wizard keeps one state with the database draft, the logical config draft, the physical config draft and the step. Both config editors receive `initialConfig`. When the base-info or PostgreSQL type step changes the database type, the wizard resets the connection settings (through `changeDatabaseType`, as today) and both config drafts, because a config from another type does not apply.

Rejected: keeping the drafts per step component. The steps unmount as the wizard moves on, which is how the logical config gets lost today.

### 9. Loads accept only the current context

The list load keeps its existing `RequestRevision` and session guard (`features/databases/ui/DatabasesComponent.tsx:128-129`). The database load and the backup config load stay in their effects, and a retry bumps an attempt counter in the effect's dependencies. The effect's cleanup then discards a response from an older database, an older attempt or an unmounted screen. A failure shows an inline `Alert` with `translateApiError` and a retry button (`common.actions.retry`) in place of the browser alert. The summaries of notifiers, restore verification and the healthcheck editor get the same cleanup, because they still showed alerts after the screen unmounted. Refused mutations keep their current alerts (see proposal, Out of Scope).

Rejected: a `RequestRevision` for the two effect loads. Each retry already re-runs the effect, so the cleanup flag covers every overlap a revision would, with less code.

### 10. The last backup error follows the server

The local clearing after a connection test is removed. The hint keys `clearHint`, `clearByTestingConnection` and `clearByNextBackup` are replaced by one sentence saying that the next successful backup clears the error, in all six dictionaries.

Rejected: clearing the error on the server after a successful test. The `databases` spec deliberately keeps a test free of side effects, and a successful test does not prove that the next backup will succeed.

### 11. Documentation follows the new owners

`docs/architecture.md` loses the "retained database/config views through narrow render contracts" wording, and `pages/databases` joins the resource pages that own collection policy. `docs/frontend-design.md` loses the transitional `contentHeight` sentence. The architecture itself does not change; these documents describe the state this change produces.

## Risks / Trade-offs

- [The move breaks a test's import path and the test silently stops running] → `coverage.md` maps every moved test to its new path, and the suite's file and test counts are compared before and after the move.
- [The paging hooks read the scroll ref before the region exists] → `shared/hooks/usePagedList.ts:53` reads the ref in an effect, and React attaches a parent's ref in the same commit, before child effects run. A happy-dom test checks that the next page loads after the region scrolls, and the browser UI scenario checks the real layout.
- [Silent refreshes race an accepted save] → The list load uses its revision, and an accepted item replaces the list entry immediately, as on the storage page.
- [Removing `onContentHeight` from `AppShell` changes a widget API] → `AppShell.test.tsx` and `ApplicationScreen` are its only callers, and both are updated in the same task.
