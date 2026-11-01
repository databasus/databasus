## Why

The logical backup config editor, the backup list and the backup's restore dialog each sit in one large component. The editor also serves two different jobs through flags and through a fake database with an empty ID, which is where most of its defects come from:

- Backup settings in the creation wizard reset every 30 seconds and whenever the tab becomes visible again.
- While the server is unreachable, the backup list and the backup's restore dialog each show a blocking alert every second.
- The restore dialog crashes when its backup leaves the list, for example when retention removes it.
- A late save answer can bring back a database the user has already left.

This is the last item of frontend milestone 6. It closes the milestone's style baseline and leaves one editor, list and restore screen that later work can change safely.

## What Changes

**Editor**

- The wizard step and the settings section become two containers over one controlled set of fields. `isSaveToApi`, `isShowBackButton`, `isShowCancelButton` and the branches on `database.id` are deleted from the logical editor, and the wizard stops passing it `toUnsavedDatabase`.
- Defaults, draft changes and validation move out of the display. Retention, notifications and advanced settings become their own units. Fields and default values stay as they are.
- The wizard's logical step keeps its draft through a background refresh and through Back in either direction, and Continue stays available after Back from the notifier step.
- Save follows the values the form shows against the acknowledged config, so reverting an edit disables it again.
- The request for backup history and the accepted switch state after a save are deleted, because the section closes as soon as that request answers, and the show view loads the history itself.
- Switching retention policy keeps the entered values. Defaults fill in only when the chosen policy has nothing set, because the server returns zeros for a policy that was never used.
- A GFS retention that keeps only hourly backups can now be saved while the hourly field is shown, as the server already allows. Retention rules also apply with backups off, because the server refuses such values.
- Advanced settings open by themselves only when they differ from the defaults, and the toggle disappears when backups are off.

**Settings and tabs**

- Cancel is disabled during a pending logical or physical config save, as it already is in the connection form. A refused save keeps the section open with its input.
- An answer for a database that is no longer shown never replaces the shown one. This covers rename and every other section save.
- A selected database that lacks the open tab shows its backups tab instead of an empty area, and a late verification start no longer switches the tab.
- The logical show view receives the config the settings screen already loaded, so one failed request gives one error and one retry.
- In the physical editor only the same defects change: its effect follows the database ID, Cancel is disabled during a save, and a late answer is ignored.

**List and restore dialog**

- The list and the restore dialog split into logical units without a change of flow. Start, cancel, delete, download, verify and single-use download tokens stay, as do the page sizes, the polling intervals, the loaded tail and the absence of automatic mutation retries.
- A failed refresh shows nothing and keeps the rows. A failed first load shows its reason with a retry and no longer looks like an empty list. A failed backup config load no longer reads as "scheduled backups disabled".
- The refresh rule lives in the shared paged list controller, so the physical backup and verification lists also stop showing a blocking alert on every failed poll.
- The restore dialog holds the chosen backup itself and stays open when that backup leaves the list. It uses the shared polled list, and it offers no target selection until its restores have loaded.
- `alert` is replaced by inline errors and context-bound feedback in these four screens. Icon actions become keyboard buttons with names, and raw colors become semantic theme roles, which removes all 40 entries of milestone 6 from `frontend/style-baseline.json`.

Nothing here breaks an API, a schema or a config. The paged list's `onError` callback changes for its internal callers: it no longer fires for a refresh and states which page failed.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `frontend-database-screens`: adds the wizard's backup step guarantees, the pending save rules for a backup config, stale answers for another database, the missing tab fallback, the retention and advanced settings form rules, and extends the load requirement to the logical backup list.
- `frontend-list-loading`: a failed refresh gives no feedback and keeps the shown rows.
- `restores`: the backup's restore dialog survives its backup leaving the list, its loading failures are recoverable and never repeat as blocking alerts, and a failed cancel is reported once.

## Impact

This is item 6.4 of `temp/refactor/frontend-plan.md`, and it depends on `extract-schedule-editor`. The root `AGENTS.md` and `frontend/AGENTS.md` apply, with the feature-sliced-design, frontend-design, frontend-localization and testing skills.

Affected areas:

- `frontend/src/pages/databases/ui/backup-config`, `ui/logical-backups` and `ui/restores`, and new units in `pages/databases/model`;
- `pages/databases/ui/CreateDatabaseComponent.tsx`, `DatabaseConfigComponent.tsx` and `DatabaseComponent.tsx`;
- `frontend/src/features/backups/physical/ui/EditPhysicalBackupConfigComponent.tsx`, for the three point fixes only;
- `frontend/src/shared/lib/PagedListController.ts` and `shared/hooks/usePagedList.ts`, used also by the physical, verification, audit and users lists;
- `frontend/style-baseline.json`, and the six dictionaries in `shared/i18n/locales` only if a new control has no existing text to serve as its name;
- `temp/refactor/frontend-plan.md`, and `docs/architecture.md` only if the page's description stops being true.

There is no backend change, migration or new dependency. Reverting the commit restores the old behavior, and stored configs stay untouched either way.

### Out of Scope

- The physical config editor's structure, `toUnsavedDatabase` for the physical editor and the notifier step, and keeping unsaved physical input through Back to the connection step. Milestone 7 handles these.
- `alert` in `DatabaseConfigComponent`, the healthcheck and notifier views, the physical editor and the physical and verification lists. Milestones 7 and 8 replace them.
- Cancel during a pending save in the notifier, healthcheck and verification sections.
- A shared policy engine for logical and physical retention.
- Keeping the scrolled tail after deleting a backup, which still reloads the list.
- New fields, new default values, changed polling intervals and real API browser tests for a config, because the browser harness has no source database to connect to.
- Raw colors outside milestone 6's baseline entries, such as `verificationStatusBadgeStyles`.
