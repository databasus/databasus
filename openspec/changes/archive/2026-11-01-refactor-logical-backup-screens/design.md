## Context

See `proposal.md` for the motivation and the delta specs for the required behavior. All paths below are under `frontend/src` unless they say otherwise. The defects were found by reading the code; task 1.2 reproduces each one before the fix. The current state:

- `pages/databases/ui/backup-config/EditLogicalBackupConfigComponent.tsx` serves the wizard and the settings section through `isSaveToApi` and `database.id`. Its load effect depends on `[database]` at line 240, while `pages/databases/ui/CreateDatabaseComponent.tsx:50` builds a new object on each render. `DatabasesComponent.tsx:93` sets its collection on every background refresh, which rerenders the wizard, so the effect reseeds the draft. `features/backups/physical/ui/EditPhysicalBackupConfigComponent.tsx:288` has the same dependency.
- `EditLogicalBackupConfigComponent.tsx:643` requires `isUnsaved` in both modes, so Continue is disabled after Back. `CreateDatabaseComponent.tsx:236` drops the draft on Back to the connection step.
- `EditLogicalBackupConfigComponent.tsx:186-192` records the accepted switch and requests `hasBackupHistory` after a save, and then `onSaved` closes the section (`DatabaseConfigComponent.tsx:150`). The accepted switch is therefore visible only until that second request answers. `ShowLogicalBackupConfigComponent.tsx:104` then loads the switch history itself.
- `EditLogicalBackupConfigComponent.tsx:633` leaves Cancel active during a save, so a user can close the section and open another one, which the late `closeSection()` then closes. `features/database-connection/ui/DatabaseConnectionForm.tsx:167` already disables its controls during a submit.
- `DatabaseComponent.tsx:51` accepts any changed database. It has no `key`, so it outlives a selection change. A late answer from `DatabaseConfigComponent.tsx:153` or `:160` puts the old database back.
- `DatabaseComponent.tsx:169` and `:178` render nothing when the kept `currentTab` does not apply to the new database, and `logical-backups/LogicalBackupsComponent.tsx:215` calls `onNavigateToVerifications` without checking that the list is still on the screen.
- `shared/lib/PagedListController.ts:95` passes a refresh failure to `onError`, and `LogicalBackupsComponent.tsx:264`, `features/backups/physical/ui/PhysicalBackupsComponent.tsx:227` and `features/verification/runs/ui/VerificationsComponent.tsx:227` show it with `alert`. `PagedListController.ts:68` ends a failed first load with empty rows, which the list renders as "no backups".
- `restores/RestoresComponent.tsx:77` shows `alert` from its 1-second poll. `LogicalBackupsComponent.tsx:711` casts `backups.find(...)`, which is `undefined` once the row is gone. `LogicalBackupsComponent.tsx:567` treats a failed config load as disabled backups.
- `EditLogicalBackupConfigComponent.tsx:367-381` resets policy values on each switch, against the contract text of `LogicalRetention`. The reset also hides a real case: `backend/internal/modules/backups/internal/config/defaults.go:39` gives a never-saved config `Count: 0` and empty GFS, so a saved `TIME_PERIOD` config can hold zeros.
- `EditLogicalBackupConfigComponent.tsx:266-272` ignores `gfsHours`, while `backend/internal/modules/backups/internal/config/backup_config.go:80` counts it. `:291-292` skips every check with backups off, while `backend/internal/modules/backups/internal/config/logical_config.go:119-125` always validates retention. `:294` requires a storage only in the wizard.
- `EditLogicalBackupConfigComponent.tsx:122` computes `hasAdvancedValues` before the load, so the block never opens by itself. Retry defaults to on, so the same check after the load would open it almost always.
- `ShowLogicalBackupConfigComponent.tsx:84` shows `alert` and stays empty at line 93, while `DatabaseConfigComponent.tsx:172` loads the same config and already shows an error with a retry.

The approach is bound by the root `AGENTS.md` (logical units, names, blank lines, no shims, refactoring within a feature) and `frontend/AGENTS.md` (component structure, AntD, design and localization skills). It follows `docs/architecture.md`: the Frontend section keeps single-screen logic in the page, and "Database pages compose reusable editing" says each container owns its draft, saving and navigation. No architecture rule changes. `docs/frontend-design.md:17` and `:25` set the rules for disclosure and failed loads, `:19` sets a dirty state rule this form deviates from in one point (decision 3), and `docs/frontend-localization.md:50` keeps persistent errors translated during rendering.

## Goals / Non-Goals

**Goals:**

- One set of fields for the wizard and the section, with the differences held in two small containers.
- One feedback rule for these screens, so a poll can never produce a dialog.
- Each defect pinned by a screen test that survives the split, because it observes the screen rather than the old component.

**Non-Goals:**

- Generalizing the containers for physical config. Milestone 7 decides what it can share.
- A form library or a shared form engine. `docs/architecture.md` and milestone 5 keep drafts domain-specific.
- A shared restores or status presentation with the physical list.

## Decisions

### 1. Two containers over controlled fields

```
 CreateLogicalBackupConfigStep        EditLogicalBackupConfigSection
 workspaceId, initial draft           load, save, storage warning, history
 Continue / Back hand over the draft  retry, Cancel, acknowledged baseline
                  |                         |
                  +-----------+-------------+
                              v
                  LogicalBackupConfigFields
                              |
          +-------------+-----+------+---------------+
          v             v            v               v
     ScheduleEditor  Retention   Notification     Advanced
                     Fields      TriggerFields    Fields
```

`LogicalBackupConfigFields` receives `config`, `onChange`, `workspaceId` and `canClearStorage`, and sends no requests. The section renders the "backups enabled" switch above the fields, because the wizard does not offer it, so the fields need no second flag. `EditLogicalBackupConfigComponent.tsx` is deleted without a bridge. The storage and encryption rows stay inside the fields, because each is one row with no logic of its own.

Pure rules live in `pages/databases/model`, one unit per file: `createDefaultLogicalBackupConfig`, `isLogicalBackupConfigComplete`, `changeLogicalRetentionPolicy`, `getRetentionForSchedule`, `hasNonDefaultLogicalAdvancedValues` and `isLogicalBackupConfigChanged`.

`isLogicalBackupConfigComplete` does not look at the storage, because a section can save a cleared one. It always checks retention, because the form shows retention and the server validates it with backups off. It checks encryption and the schedule only with backups on, as `recurring-schedules` states.

Rejected: keeping one component with a discriminated `mode` prop, which still branches throughout and keeps wizard props on the section. Rejected: a hook that takes a mode, which only moves the same branches.

### 2. The wizard step owns its draft and hands it over

The step starts from `initialConfig ?? createDefaultLogicalBackupConfig()` in `useState`, with no effect, so a rerender of the wizard has nothing to reset. Both `onContinue(draft)` and `onBack(draft)` return the draft, and `CreateDatabaseComponent` stores it. A type change keeps discarding it through `acceptDatabaseType`. Continue requires `isLogicalBackupConfigComplete(draft)` and a chosen `storageId`, so the storage rule stays with the step that has it and needs no mode argument.

Rejected: holding the draft in `CreateDatabaseComponent` and updating it on each field change. The nested storage dialog and the fields would rerender with the whole wizard for no gain, and other steps keep the same hand-over pattern. Rejected: memoizing `toUnsavedDatabase`, which keeps the fake database and the flags.

### 3. Save follows the acknowledged config

The section keeps the loaded config as its baseline and derives `isChanged` with `isLogicalBackupConfigChanged`. It compares what the form shows: the fields of the chosen retention policy only, `gfsHours` only when shown, and notification triggers as a set, because toggling one off and on reorders the array. Values of an unchosen policy therefore do not count, so a round trip through a policy that decision 5 filled with defaults leaves Save disabled. If the user saves another edit, those filled values go to the server with the config, which is harmless because that policy is not in use.

The Save button stays visible and disabled when nothing changed. This deviates from `docs/frontend-design.md:19`, which says submit actions appear only for a dirty draft. The reason is `openspec/specs/recurring-schedules/spec.md:185-210`, which requires Save to be disabled, including for a stored invalid cron expression on an unedited form, and the physical editor shares that rule until milestone 7. The derived baseline does meet the doc's point that reverting a value returns to clean.

Rejected: keeping the `isUnsaved` flag, which stays set after reverting an edit. Rejected: hiding Save, which would need a change to `recurring-schedules` and would make the logical and physical forms differ until milestone 7.

### 4. Delete the state that follows a save

The section stops recording the accepted switch, stops updating `savedStorageId` and stops requesting `hasBackupHistory` after a save. It closes at once, and the show view loads the history, including a switch the server just accepted. This also ends the wait for a second request while the save has already succeeded. `StorageSwitchHistoryComponent` keeps `initialOperation`, which the physical editor still uses.

The test "warns before reassignment ... without remounting" observes a state the real screen cannot reach, because the section closes. It moves to the settings screen: save a cleared storage, open the section again, choose a storage and see the warning. `coverage.md` records this.

Rejected: keeping the section open after a save to show the progress, which contradicts "Saving a database settings section keeps the user's place".

### 5. Switching policy keeps values and fills only an empty policy

`changeLogicalRetentionPolicy` sets the policy and keeps all fields. For `COUNT` with `count <= 0` it sets 100. For `GFS` with no period set among the shown ones it sets the defaults of `createDefaultLogicalBackupConfig`. It uses the same predicate as validation.

Rejected: keeping the reset, which loses input and contradicts the contract. Rejected: keeping values only, which shows zeros and a disabled Save for every config that never used the policy, because of `defaults.go:39`.

### 6. Hourly GFS counts only when the form shows it

A GFS policy is complete, with backups on or off, when any of days, weeks, months or years is set, or `gfsHours` is set and the schedule is hourly or cron. This lets a user save hourly-only retention as the server allows. The client stays stricter than the server for a daily schedule, where `getRetentionForSchedule` sets a hidden `gfsHours: 24` the user never chose, which must not count as a choice.

Rejected: copying the server's rule exactly, which would allow saving a daily schedule with all shown fields at zero.

### 7. Advanced settings open for non-default values

`hasNonDefaultLogicalAdvancedValues` answers true when retry is off or the number of tries differs from the default 3. The section computes it from the loaded config and the step from its seed. The block starts closed in the usual case and in the wizard, as users see today, and it keeps a meaningful configured value visible as `docs/frontend-design.md:17` asks. The toggle becomes `AdvancedSettingsToggleComponent`, and it is hidden when backups are off.

Rejected: the plan's literal fix of reading `isRetryIfFailed` after the load, which opens the block for almost every config. Rejected: removing the automatic opening, which hides a turned-off retry.

### 8. A pending save locks Cancel, and a late answer changes nothing

Both config editors disable Cancel while `isSaving`, and check `useSessionGuard` after each await, as `EditDatabaseConnectionSection` does. After that guard the section calls `onSaved` or shows the refusal, and an obsolete answer does neither.

Rejected: only ignoring the late answer while Cancel stays active. The show view could then load the old values and keep them after the server saved new ones. Rejected: a revision token per section, which solves a race the disabled Cancel removes.

### 9. `DatabaseComponent` guards by database ID, without `key`

`acceptChangedDatabase` becomes `setDatabase((shown) => (shown?.id === changed.id ? changed : shown))`, and it still reports the change to the list. This fixes rename and every other section in one place, and the card of the old database stays correct.

Rejected: `key={selectedDatabaseId}`. It would fix the race but reset the open tab on each selection, a visible change this work should not make.

### 10. The shown tab is derived

`DatabaseComponent` computes the shown tab: `verifications` needs a logical PostgreSQL database and `monitoring` needs `isMonitoringTabShown`, otherwise it is `backups`. The tab bar and the content both use it. `currentTab` stays as the user's choice, so returning to a suitable database reopens it. `LogicalBackupsComponent` checks `useSessionGuard` after a verification start, before the message and the navigation.

Rejected: resetting `currentTab` in an effect, which paints the empty area first and forgets the choice.

### 11. One feedback rule per kind of failure

| Failure | Feedback |
|---|---|
| A first load: editor config, list, configuration view storage, restores | Inline `Alert` with Retry, translated during rendering |
| A refused config save | Inline `Alert` in the open form, translated during rendering |
| A one-shot action: start, cancel, delete, download, next page | `message.error` from `App.useApp()` |
| A background refresh | Nothing |
| An answer after leaving the screen | Nothing |

Rejected: a toast for a refused save, which disappears while a long server reason is still needed, and which `docs/frontend-localization.md:50` would translate only once.

### 12. The paged list controller owns the refresh rule

`refreshFirstPage` no longer calls `onError`. `PagedListState` gains `firstLoadError`, the name `WalStreamRefreshState` already uses for this concept, which `reload` clears and a failed first page sets, and `usePagedList` returns it. `onError` becomes `(error, failedPage: 'first' | 'next')`, so the logical list reacts only to `next`, while existing callers ignore the argument and keep their behavior for both. The first failed load keeps polling off, as `isReady` already does, until Retry succeeds.

The physical backup and verification lists stop showing a dialog on each failed poll. Their first-load `alert` stays until milestone 7. A refresh after a successful cancel in those lists also stays silent, because the cancel succeeded and the next poll updates the row. One controller test pins the old refresh report and is rewritten, with the reason in `coverage.md`. The audit and users lists do not poll, so they see no change.

Rejected: a flag per screen that silences refresh errors, which leaves the same defect one option away in two other lists. Rejected: a query library, which the plan excludes without a proven simplification.

### 13. The restore dialog holds the backup

`LogicalBackupsComponent` stores `restoringBackup: LogicalBackup` instead of an ID, so the dialog has its backup however the list changes. Its form input survives because the dialog stays mounted.

Rejected: closing the dialog when the row disappears, which loses an entered target and can hide a running restore.

### 14. Restores use the polled list over their complete listing

`RestoresComponent` calls `usePolledPagedList` with `fetchPage: async () => ({ items: await restoreApi.listAllRestores(backup.id) })`, `resetKey: backup.id` and `refreshIntervalMs: 1_000`. With no `nextCursor`, `mergeRefreshedFirstPage` (`shared/lib/pagedList.ts:35`) adopts the whole listing, so every restore stays shown and no scroll container is needed. `reload` is the Retry, `refreshFirstPage` replaces the guarded `loadRestores`, and `replaceOrPrependItem` replaces `showRestore`. Polling waits for the first success through the controller's `isReady`, and the first-load error comes from decision 12.

The target selection appears only after a successful load. Otherwise a member could start a restore while nothing refreshes it, and it would stay `IN_PROGRESS` on the screen and lock the button.

Rejected: a hand-written hook, which repeats the load, poll and stale-answer logic the controller already tests. Rejected: starting refreshes after an accepted start in the failed state, which adds a second way to begin polling for a rare case.

### 15. The show view receives the config

`DatabaseConfigComponent` keeps the loaded `BackupConfig` instead of only its storage ID and passes the logical config to `ShowLogicalBackupConfigComponent`, which still loads its storage and shows its own inline error for that request. Reloading after a save clears the config, so the section shows a spinner and then the acknowledged values, and the rest of the screen stays. The physical show view keeps its own load until milestone 7.

Rejected: a second retry inside the show view, which reports one failed config twice on one screen and needs two buttons.

### 16. List and restore units, controls and colors

- `logical-backups/`: `LogicalBackupsComponent` composes the header, filters, Make backup, list and dialogs. `useLogicalBackupActions` owns the mutations and their pending IDs. `LogicalBackupStatusLabel`, `LogicalBackupActions`, `LogicalBackupsTable` and `LogicalBackupCards` hold the rendering, and `formatLogicalBackupSize` serves both views. The download starts directly from the click, without the effect at line 288.
- `restores/`: `RestoresComponent` composes the list and the target form, with `RestoreCard` and `RestoreErrorDialog`. One `restoreToCancelId` replaces two states.
- Clickable icons and spans become AntD buttons named by their existing tooltip text, and the GFS hint becomes a button, as `docs/frontend-design.md:17` asks. A control with no existing text gets a key through the localization skill.
- Raw colors become the theme roles in `frontend/src/index.css`: `text-danger`, `text-success`, `text-accent`, `text-muted`, `border-border`, `bg-surface` and `bg-code`. The extension hint uses a warning `Alert`.

Rejected: leaving the icons clickable for a smaller diff. Their test queries would stay tied to CSS classes, and the actions stay out of keyboard reach.

### 17. Tests observe the screens

Reproductions go through `CreateDatabaseComponent`, `DatabaseConfigComponent`, `DatabaseComponent`, `LogicalBackupsComponent` and `RestoresComponent`, which all survive the split, so the same tests run before and after it. Pure rules get node tests. A new browser scenario edits a config in Russian on a stateful fixture and reads it again.

## Risks / Trade-offs

- [The change is large and touches a controller shared with other lists] → The tasks reproduce defects first, change the controller alone, and split the screens after their tests are in place. `coverage.md` maps each moved test.
- [Physical and logical editors differ until milestone 7] → Only the point fixes reach physical, and its unsaved input on Back to the connection step stays lost, as the delta spec states for logical only.
- [Icon actions become buttons, which may shift the row by a few pixels] → The browser UI project runs on the scenarios that show these screens, and any reference change gets looked at rather than regenerated.
- [A failed first list load stops polling until Retry] → This is deliberate and matches the current controller. The error with Retry is on the screen.
- [A saved cron or GFS value could look different after the policy rule] → The rule only fills fields that hold no usable value, and only when the user switches policy. Stored configs change only on an explicit Save.

## Migration Plan

No data or API migration. Reverting the commit restores the previous behavior.
