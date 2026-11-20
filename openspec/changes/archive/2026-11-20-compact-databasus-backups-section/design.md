## Context

See proposal.md for the motivation. The section today is one long column rendered by `frontend/src/pages/instance-settings/ui/instance-backups/InstanceBackupsSection.tsx:36-130`: a status line (`:64`), a separate incident alert (`:68`), mail notices, `MasterKeyPanel` (`:112`), `InstanceBackupSettingsForm`, `InstanceBackupHistory` and `InstanceRecoveryImport` (`:125`, keyed by the session generation).

Facts that shape the approach:

- The status response already carries everything the status block needs: `health`, `incident`, and per destination `health`, `lastSuccessfulSnapshotAt` and `lastError` (`backend/api/openapi.yaml:6253-6289`, `:6340-6380`). The status block needs no API change.
- The settings form keeps an acknowledged baseline and a draft, and shows Save and Reset only for a dirty draft (`InstanceBackupSettingsForm.tsx:185`, `:292`). It is always mounted.
- `MasterKeyExport` reads the key as soon as it mounts when the instance does not require two-factor sign-in (`frontend/src/features/export-master-key/ui/MasterKeyExport.tsx:51-56`). `MasterKeyPanel` mounts it only after a button press, so opening settings exports nothing.
- Recovery keeps the archive, the key and the server-side recovery in component state. Cancel aborts an upload or cancels the recovery on the server (`recovery/InstanceRecoveryImport.tsx:80-104`). Replacement opens `ApplyRecoveryDialog` on top of the form (`:186`, `:266`).
- The server allows one pending recovery per instance and expires an unapplied one after an hour (`openspec/specs/databasus-recovery/spec.md:149-169`).
- History loads pages of 10 and polls every 15 seconds (`InstanceBackupHistory.tsx:23-24`).
- The database settings screen already uses read-only sections with an edit button that swaps in the form in place (`frontend/src/pages/databases/settings/ui/DatabaseSettingsSection.tsx`, `DatabaseConfigComponent.tsx:350-366`).
- The themed Mantine `Alert` already gives `success`, `warning` and `danger` their soft background and border, and other colors the neutral `border` role (`frontend/src/shared/theme/mantineTheme.ts:146-152`, `:228-232`). The section already uses `Alert` for its notices, and recovery uses `color="accent"` for its restoring notice (`recovery/InstanceRecoveryImport.tsx:165`).
- `InstanceBackupDestinationList.tsx` already renders a storage's logo, name, workspace, provider, health, last error and the local-storage warning, with a remove button.
- The status response has no "ever selected" flag. `NOT_CONFIGURED` means no destination was ever selected, and `STOPPED` means the last one was removed (`backend/api/openapi.yaml:6245-6252`).
- The onboarding screen blocks the whole application while the current key is unacknowledged, so inside settings the key is almost always acknowledged.

Constraints: `frontend/AGENTS.md` (Mantine controls, Lucide icons, Tailwind layout, `@/` imports, FSD placement, blank-line steps), `docs/frontend-design.md` (semantic color roles, status words beside status colors, dialogs with titles and focus return, disclosures as buttons with expanded state, 320-pixel and 200-percent-text support, backup configuration forms keep Save visible) and `docs/frontend-localization.md` (every dictionary updated together, safe interpolation).

## Goals / Non-Goals

**Goals:**

- The healthy section fits in roughly one screen at 1280 pixels: statement, the storage lines, five history rows and two actions, with the status block only when something needs attention.
- A root admin who never opens a dialog sees no long explanation, no command line, no file input and no key field.
- Every capability the section has today stays reachable within one click.

**Non-Goals:**

- A visual redesign of the rest of the settings page.
- Moving `EditIconButton` out of the databases page into `shared/ui`. The section uses its own labelled Edit button instead, because a page cannot import from another page.

## Layout

Healthy instance:

```
+--------------------------------------------------------------+
| Databasus backups                                            |
| Saves Databasus itself: users, settings, database ...        |
| If this server breaks, you can recover Databasus ...         |
|                                                              |
| Storages                                          [Edit]     |
| +----------------------------------------------------------+ |
| | [S3] Offsite (workspace: Main)                           | |
| +----------------------------------------------------------+ |
| | [disk] Local (workspace: Main)                           | |
| |        ! A local storage is on the same server ...       | |
| +----------------------------------------------------------+ |
|                                                              |
| Recent backups                             [Back up now]     |
| +----------------------------------------------------------+ |
| | 18 Nov 2026, 03:00 (a minute ago)  (spin) In progress    | |
| | 17 Nov 2026, 03:00 (a day ago)  * Successful  1.2 MB [dl]| |
| | 16 Nov 2026, 03:00 (2 days ago)  * Partial  1.2 MB  [dl] | |
| |   Local (Main): failed                                   | |
| |   disk full                                              | |
| | 15 Nov 2026, 03:00 (3 days ago)  * Failed                | |
| +----------------------------------------------------------+ |
| Show all                                                     |
|                                                              |
| [key] Download master key  [archive-restore] Recover from... |
+--------------------------------------------------------------+
```

New installation:

```
+--------------------------------------------------------------+
| Databasus backups                                            |
| Saves Databasus itself: users, settings, database ...        |
| If this server breaks, you can recover Databasus ...         |
|                                                              |
| +----------------------------------------------------------+ |
| | (shield-off)  Databasus is not backed up                 | |
| |               Choose a storage to start backups.         | |
| |                                                          | |
| |  [Set up backups]  [Recover from backup]  [Master key]   | |
| +----------------------------------------------------------+ |
+--------------------------------------------------------------+
```

Problem:

```
| +----------------------------------------------------------+ |
| | (shield-x)  Databasus backups are failing                | |
| |             Since 17 Nov, 12:00. Last backup 1 day ago   | |
| |             Offsite: connection timed out                | |
| |             The alert email could not be sent: ...       | |
| +----------------------------------------------------------+ |
| (mail) The mail server is not configured.  [Configure SMTP]  |
```

What each element hides compared with today:

| Today | After |
|---|---|
| Two introduction paragraphs | Two short sentences on what is saved and that it recovers Databasus on another server; key and storage advice moves to the master-key dialog and the settings form |
| `Status: Healthy` line and a separate incident alert | One status block |
| Master-key status, description, export button and the `docker cp` command | Master key button; dialog with the key file location behind a disclosure |
| Destination picker, storage cards, shared-storage warning, retention help | Read-only lines; the form appears on Edit |
| Healthy per-storage status and last-copy time | Shown only for a storage that is not healthy |
| History hints about destinations and downloading for recovery | Disabled-button reason; the recovery dialog explains using a downloaded backup |
| Ten cards with trigger, version and per-copy lines, plus Show more | Five one-line rows; only a failed or partial row adds its failure reason and failed copies; Show all expands in place |
| Recovery form with file, key and the replacement panel | Restore button; two-step dialog |

## Decisions

### 1. The status block is a themed alert driven by the existing status response

`getInstanceBackupHealthPresentation` in `frontend/src/pages/instance-settings/model/` maps each `InstanceBackupHealth` to an alert color, a Lucide icon and a title key:

| Health | Alert color | Icon | Section view |
|---|---|---|---|
| `HEALTHY` | no block is rendered | | settings, history, actions |
| `INITIALIZING`, `RESUMING` | `accent` (neutral border) | `ShieldEllipsis` | settings, history, actions |
| `PARTIAL`, `OVERDUE` | `warning` | `ShieldAlert` | settings, history, actions |
| `FAILED` | `danger` | `ShieldX` | settings, history, actions |
| `STOPPED` | `danger` | `ShieldOff` | settings with "No storage is selected", history, actions |
| `NOT_CONFIGURED` | `warning` | `ShieldOff` | empty state only |

`NOT_CONFIGURED` is the client signal for "no destination has ever been selected", and every other value is the signal for "a destination has been selected". `getNewestSuccessfulSnapshotAt` in the same folder returns the newest `lastSuccessfulSnapshotAt` across destinations, or nothing. With a value the block shows "Last backup" through `useLocale().formatRelativeTime` inside a `<time>` element, with `dateTime` holding the instant and a Mantine `Tooltip` showing `formatInstant`. Without one it says that no successful backup exists yet. Failing storages come from destinations with `lastError`.

A healthy instance renders no block: the recent backups right below already show that backups are made, and a green card on every visit only pushes the settings down. The presentation map still covers `HEALTHY`, so it stays total over the enum.

The block is a Mantine `Alert` with `role="status"`, the health color and the shield icon. The theme already gives these colors their soft background and border, so the theme and `index.css` do not change and no raw color enters a component.

Alternatives: a hand-styled tinted card, rejected because it needs new Tailwind roles and an `info-border` token while `Alert` already covers the tones. A backend summary field, rejected because the client already has every input. Keeping the text-only status line, rejected because it is the main reason the answer is hard to find.

### 2. Settings are edited in a dialog

The section owns an `isEditingSettings` flag. `InstanceBackupSettingsView` renders the saved storages under a Storages heading through `InstanceBackupDestinationList`; its Edit action, and Set up backups in the empty state, open the shared `Dialog` with `InstanceBackupSettingsForm`. `InstanceBackupDestinationList` gains an optional `onRemove`; without it the remove button is not rendered. In both the view and the form, the list shows a storage's health and last-success line only when its health is not `HEALTHY`, keeping its error and the local-storage warning; a storage not saved yet has no health line. Each entry is compact: the provider logo, whose alternative text names the provider, then the storage name and "(workspace: name)" in muted text on one line, with health, error and the local-storage warning as short indented lines below. The workspace name is user input, so the line renders through `Trans` with the name as a self-closing tag. One row therefore serves both places.

The dialog takes the form's title, so the form has no heading of its own. It ignores clicks outside, so a stray click does not discard a draft; the close button, Escape and Cancel close it and drop the draft. Save success closes it, and an edit made while the save was in flight keeps it open. Save stays visible and is disabled for an unchanged or incomplete draft, following the backup-configuration-form convention of `docs/frontend-design.md`. `CreateDestinationDialog` opens on top of the settings dialog; Mantine stacks the two and each returns focus to its invoker. The dialog is 560 pixels wide. A Storages block holds the picker and the selected storages; a block separated by a rule holds the schedule and the retention in one 150-pixel label column; the actions sit at the bottom right. Every control uses the `sm` size, which `ResourcePicker` and `ScheduleEditor` accept as an optional prop, so the fields share one height; other callers keep `xs`. The picker marks every selected option with the selected background and a check, keeps its key-receiving field on the line of the pills instead of wrapping it, and gives its search a placeholder. The form does not explain shared workspace storages or retention deletion.

The missing-mail-server notice stays in the section under the dialog, so it is used outside an edit and has no draft to protect.

Alternatives: editing in place, rejected by root admin because the form pushed the history down and squeezed the two-column layout; a drawer, rejected because the form is short and a centred dialog keeps it next to its title.

### 3. History shows five rows and expands in place

`InstanceBackupHistory` keeps one `usePolledPagedList` with a page size of ten and shows its first five rows. Show all appears when more than five backups are loaded or another page exists; it reveals every loaded row in place, and Show more then loads the next page. `InstanceBackupRow` renders time, outcome, size and download on one line and has no disclosure. Only a backup with a failure reason or failed copies adds lines below: the reason and each failed copy with its storage, workspace, status and sanitized error. The trigger, the source version and successful copies are not shown, because a successful outcome already says every copy was stored. Download is one `ActionIcon` without a storage choice: it requests a download token for each downloadable copy in order and follows the first one issued, reporting the last error only when every copy fails. A storage that loses the file after the token is issued fails in the browser download, which the page cannot observe. The backend names the download after the snapshot time in UTC, `databasus-backup-<yyyy-mm-dd-hh-mm-ss>.dbbackup`, while the stored file keeps its unique attempt name. A backup in progress shows the same spinning refresh icon in the accent color as the database backup list. Times use `formatDateTimeWithRelativeTime` from `useLocale`. It formats the instant with `Intl.DateTimeFormat` in the interface language, a numeric day, a short month name, the year and the time, and appends the dayjs relative time in parentheses through the `common.dateTimeWithRelativeTime` key, such as "14 Nov 2025, 13:00 (4 days ago)". The key lets Chinese use full-width parentheses. The same formatter serves a storage's last successful copy, the incident start and the snapshot in the recovery dialog; accessible labels such as the download action keep the exact time only. English uses the `en-GB` tag so the day comes before the month, and the clock follows the browser's 12- or 24-hour preference like the rest of the application.

Rows are flex items in a bordered list rather than a table, so they wrap at 320 pixels without horizontal scrolling.

Alternatives: a full-history dialog, used first and replaced at the user's request, because a second list in a dialog hid the older rows behind one more window for no gain. The existing `getUserShortTimeFormat`, rejected because it takes the date order from the browser, so an American browser writes "Nov 14, 2025" even with another interface language.

### 4. Download master key opens a dialog that reads the key

The section action is named Download master key, so pressing it is the request for the key. `MasterKeyDialog` mounts `MasterKeyExport` at once and follows the onboarding layout: a key icon, the centred description, the partial preview with Copy and Download, then the advice to keep the key and a storage outside the server, and a disclosure "Copy the key from the server instead" holding `MasterKeyHostInstructions`. The dialog shows no saved-state line: onboarding already blocks the application while the key is unsaved. `MasterKeyHostInstructions` names the file `databasus-data/secret.key` in the folder where Databasus is installed instead of the absolute `masterKeyPath` and a `docker cp` command. The data folder is a volume, so its host path depends on the installation, and in development the reported path is the developer's host path. The frontend therefore no longer reads `masterKeyPath`. With two-factor sign-in the export asks for the emailed code first, as onboarding does. The dialog body mounts only while the dialog is open, so closing it discards the obtained key. The section shows no key status line, because onboarding already blocks the application while the key is unsaved.

Alternatives: a dialog that reads the key only after a second button, used first and dropped at the user's request, because the extra click asked the same question twice. Keeping the panel inline, rejected by the request.

### 5. Recovery is one dialog with steps derived from the recovery status

`InstanceRecoveryDialog` replaces both `InstanceRecoveryImport` as an inline form and `ApplyRecoveryDialog` as a second dialog. Its step follows the state it already tracks: inputs with no recovery or a failed one; progress while uploading or preparing; replacement when ready, holding the snapshot details, the warning, the confirmation checkbox and, with two-factor sign-in, the emailed code; restoring after apply is accepted.

Closing the dialog calls the existing cancel: it aborts an upload or cancels the preparing or prepared recovery on the server. If cancellation fails, the dialog stays open with the error, because a pending recovery would otherwise block the next import for up to an hour. The dialog disables outside clicks so a stray click cannot cancel an upload. While restoring it hides its close button and ignores Escape. The dialog is mounted only while open and keyed by the session generation, so a session change still clears the archive and the key.

The inputs step shows one sentence of description, a file card and the key field. The file card is a dashed button built on Mantine `FileButton` that also accepts a dropped file; it shows the file name and size once chosen, with a separate button that removes the file. Its hint says where to get the file, which replaces the two extra paragraphs. The key field explains under it that the key of this installation opens only its own backups, and an eye button inside the field shows or hides the key. After preparation, the step asks for an emailed code only while `isConfirmationCodeRequired` is true, the same flag the master-key export reads; otherwise the replacement checkbox alone enables Replace. A rejected code (`identity_proof_invalid`) is shown under the code field and clears when the code changes; other apply errors stay in the alert. The password is not a confirmation any more. Without two-factor sign-in the session already proves root admin with the same password, so asking for it again added a step without adding a factor; with two-factor sign-in the emailed code is the second factor, as it is for the master-key export. The backend sends a code only while two-factor sign-in is required and a mail server can reach root admin, and a request otherwise answers 409 `identity_proof_unavailable`. Apply reads the setting when it runs, so turning two-factor sign-in off between sending and applying makes the code unnecessary rather than rejected. The field is a single-line `PasswordInput`. A browser strips line breaks from a single-line value, so a paste is stored from the clipboard text itself and the upload keeps the key's exact bytes; typing goes through the field's own value.

Alternatives: a dialog on top of a dialog, rejected for focus and Escape handling. Keeping the prepared recovery when the dialog closes, rejected because the reopened dialog would need to restore a server-side state it cannot list.

### 6. The read-only view shows only the storages

The read-only block lists the selected storages under a Storages heading and shows neither the schedule nor the retention. Both rarely change after setup, and Edit shows them in the form, so the healthy section stays short. Without a schedule line nothing needs a one-line schedule summary, so the change adds no such component to `entities/schedules`.

Alternatives: a one-line summary such as "Every hour (keep 30 days)" from a new `ScheduleSummary` entity component, dropped at review because root admin found the line unneeded and the component would have had no consumer.

### 6a. Storages and recent backups share a row on wide screens

The section widens to `max-w-5xl`, while the statement keeps a narrower `max-w-xl` reading width and the status block the `max-w-2xl` width of the other settings sections. From the `lg` breakpoint the storages and the recent backups form a two-column grid, two fifths for the storages and three fifths for the history, because a history row with its time, relative time, outcome, size and download needs about 450 pixels. Editing happens in a dialog, so the grid never has to make room for the form. The master-key and recovery actions follow without a divider.

Alternatives: keeping one column, rejected by root admin as taking too much height; two columns inside the old `max-w-2xl` width, rejected because each history row would wrap to two lines.

### 7. Copy moves rather than grows

New keys cover the status titles and sublines, the read-only labels, the empty state, the dialog titles and the disclosure labels. Keys that no longer render, such as the status label, the history hint about importing below, the backup trigger labels and the history row details, are deleted from all seven dictionaries. Existing explanations move with their elements instead of being rewritten, so translators see as little churn as possible.

## Recovery without a pause

The recovery pause is built from three parts.

- **The gate.** `backend/internal/lib/recoverypause/gate.go` holds an in-memory flag. `backend/internal/app/app.go:114-121` reads the persisted pause at startup, builds the gate and passes it into the Deps of `notifiers`, `storages`, `databases`, `backups`, `verification` and `instancebackups` (`app.go:212-305`). Each module checks it before an effect: starting or deleting backups (`backups/internal/logical/service.go:143,191`, `physical/backup_request_service.go:32`, `physical/backup_deletion_service.go:28`), restores (`restore/service.go:131`), database deletion and limited users (`databases/internal/userdatabase/deletion_service.go:24`, `limiteduser/service.go:95`), notifications (`notifiers/internal/delivery/delivery.go:65`, `notifier/test_send_service.go:67,100`), storage writes and tests (`storages/internal/files/store.go:77`, `storage/connection_test_service.go:61,108`), verifications (`verification/internal/verifications/service.go:98`, `claim_service.go:26`) and Databasus backups (`instancebackups/internal/configs/service.go:112`, `catalog/service.go:90`, `catalog/download_service.go:45,92`). `app/recovery_pause.go` turns every job outside four local maintenance jobs into a no-op while paused (`app.go:354`). `physical/facade_service.go:184-187` skips dropping orphan replication slots on a paused start.
- **The persisted marker.** `instancebackups_configs.recovery_paused_operation_id` and `recovery_paused_at` (`backend/migrations/20261114100000_instance_backups.sql:16-17,42-43`) are written into the staging database before publication (`instancebackups/internal/recovery/preparer.go:115-141`). Publication refuses a database that is not marked by its own operation (`recovery/publisher.go:84,214-231`), so the marker also identifies the staged database after a crash. `Activate` clears it and opens the gate (`recovery/service.go:118-163`).
- **What the interface sees.** `newCompletedRecovery` reports `PAUSED` while the marker names the recovery and `ACTIVE` afterwards (`recovery/recovery.go:127-131`). The status carries `isRecoveryPaused` (`health/status_service.go:54`). The frontend shows `RecoveryPauseNotice` with the activation dialog (`frontend/src/widgets/instance-backup-status/ui/InstanceBackupStatusWidget.tsx:41-43`). The recovery dialog treats `APPLYING` and `PAUSED` as restoring (`InstanceRecoveryDialog.tsx:55-57`).

The recovered-runtime normalization (interrupted operations, revoked sessions and tokens, dropped deletions, the request for a new full chain) runs before publication through the normalizers in `app/lifecycle.go:87-100`. It does not depend on activation.

Local-storage validation is a separate restriction. The storages step of that normalization (`storages/internal/instancerecovery/state_repository.go`) marks every local storage `REQUIRES_VALIDATION` in `storages.recovery_validation_status` (`backend/migrations/20261114100200_storages_recovery_validation.sql`). `SettingsReader.GetSettings` then answers `ErrStorageUnavailable` for it (`storage/settings_reader.go`), so every write, read and deletion fails, and a Databasus backup copy to it retries forever while its backup shows as in progress. Only an editor's saved connection test, with an extra probe of the backup directory, clears the mark (`storage/service.go` `markRecoveryValidated`, `connection_test_service.go`). The API exposes the status on `Storage` and in the test result, and the storages page explains it.

The approach answers to `backend/AGENTS.md` (module Deps, `module.go` as the public contract, errors in `errors.go`), `frontend/AGENTS.md` (FSD layers, dictionaries) and the recovery section of `docs/architecture.md`, which this change edits.

### Goals and non-goals

**Goals:**

- A published recovery runs every job and accepts every action from its first start.
- No code, contract, dictionary or document keeps a trace of the pause.
- Publication keeps recognizing its own staged database after a crash.

**Non-Goals:**

- Detecting a still-running old installation.
- Changing the rest of the normalization of recovered runtime state: interrupted operations, revoked credentials, dropped deletions and the new full chain stay.

#### Delete the gate rather than leave it permanently open

The gate, its checks, the job wrapper and the `recovery_paused` problem code are deleted. Keeping a gate that is never closed would leave dead branches in every module and a problem code that no operation returns. An open-only gate would also invite someone to close it again without the activation flow that once made it safe.

#### Keep the marker, renamed, through a new migration

Publication still needs to know that the database it swapped in was produced by this operation (`publisher.go:214-231`). The marker therefore stays, renamed to `recovered_by_operation_id` and `recovered_at`, with the constraint renamed to `chk_instancebackups_configs_recovery_marker`. In Go the fields become `RecoveredByOperationID` and `RecoveredAt`, the getter `GetRecoveredByOperationID` and the repository write `updateRecoveryMarker`; `EndRecoveryPause` and `Config.IsRecoveryPaused` are deleted, because nothing clears the marker or reads it as a pause. `IsDefault` still treats a marked config as non-default, so a recovered target is not mistaken for a fresh one.

A new migration renames the columns and the constraint. Alternatives considered:
- Editing `20261114100000_instance_backups.sql` in place. The feature is unreleased, but development databases and other worktrees have already applied that version, and goose would not rerun it, so their schema would keep the old names.
- Dropping the check. Then a crash between swap and journal update could publish a database the operation never prepared.

#### Report a published recovery as `COMPLETED`

`PAUSED` and `ACTIVE` collapse into `COMPLETED`. `ACTIVE` without an activation step names nothing the user did, and keeping two values for one state would make clients guess. The recovery dialog treats `APPLYING` and `COMPLETED` as restoring until the session ends, as it did with `PAUSED`.

#### Move the dropped-cleanup message into the backups section

The pause notice was the only place that told root admin how many pending deletions the recovery dropped. The Databasus backups section now shows one muted line when the latest recovery dropped any, built from `status.recovery.discardedCleanupCount`. Alternatives considered:
- An audit event only. Root admin would have to know to look for it.
- Keeping a dismissible notice. That brings back a widget whose main purpose is gone.

The line stays while that recovery is the latest one. The leftover files it describes stay in the storages until someone removes them, and the instance cannot tell when that happened, so any expiry would hide the note while the files may still be there. The snapshot time and source release from the old notice are not repeated; the recovery dialog already showed them before apply.

#### Drop orphan replication slots on every start

`facade_service.go:184-187` stops special-casing a paused start. A recovered installation now behaves like any start and drops slots it does not own. Keeping the skip on the first start after a recovery would need a new "first start" state for one corner case, while the accepted risk already says the old installation must be stopped.

#### Use recovered local storages at once

The validation status, the extended probe, the step that marks local storages during recovery, `ErrStorageUnavailable` in the storages module and the interface explanation are deleted. A recovered local storage is an ordinary storage from the first start: its folder either exists on the new host or writes fail with the folder's own error and retry, as they would on any host whose folder disappeared. The owner wants the recovered installation to keep working exactly as before, with no checks.

A new migration drops `storages.recovery_validation_status`, for the same reason the marker rename is a new migration: development databases already applied `20261114100200`. The `storage_unavailable` problem code stays, because a Databasus backup download still answers it when no copy can serve the file.

Alternatives considered:
- Ending a copy to an unvalidated storage at once instead of retrying. That fixes the endless in-progress rows but keeps a manual step after every recovery that the owner does not want.
- Keeping the probe as an optional diagnostic. The ordinary connection test already reports whether the folders are writable.

### Risks

- [Two installations run at once if root admin recovers while the old one still works] → Both write to the same storages and databases; retention of one can delete files the other expects, and WAL streaming competes for slots. The owner accepted this risk. The recovery dialog's existing warning that current state is replaced stays; the documentation states that the old installation must be stopped first.
- [A development database is paused when this change lands] → After the rename migration its marker stays set, but nothing reads it as a pause any more, so the installation simply runs.
- [A recovered local storage points at a folder that is empty on the new host] → Backup history that lived only on that folder lists copies whose files are gone; a download tries the next copy or answers `storage_unavailable`, and a restore fails with the missing file. Writes create new files there as usual. The owner accepted this; Databasus backups kept only on the lost server's disk were never protected against losing it, which the section already warns about.

## Risks / Trade-offs

- [Closing the recovery dialog by accident cancels a long upload] → Outside clicks do not close it; only the close button, Escape and Cancel do, and all three mean cancel.
- [A session change unmounts the dialog without a server-side cancel] → Same as today; the server expires the pending recovery within an hour. No new risk.
- [Users no longer see the trigger, the source version or successful copies of a backup] → A successful outcome means every copy was stored, and failed or partial rows list the failed copies with their errors. The recovery dialog shows the source version of the archive being restored.
- [Browser visual references and the end-to-end recovery scenario drive the old layout] → They are updated in the same change; references are regenerated explicitly.
- [The storage picker squeezed a selected storage to one letter per line] → The shared `ResourcePicker` pill had Mantine's zero flex basis with a wrapping label, and its remove icon had a percentage size, which made the intrinsic width that of a default SVG. The pill now keeps its content width and the icon has a fixed size, which fixes every picker that uses it.

## Migration Plan

No user data changes; the schema changes are the recovery-marker rename and the drop of the storage validation status below. The frontend and the backend ship together, because the API changes with them: `InstanceBackupStatus` drops `masterKeyPath` and replaces `identityProofMethods` and `isMasterKeyExportCodeRequired` with `isConfirmationCodeRequired`, `InstanceRecoveryApply` replaces `proof` with an optional `code`, a recovery code request answers 409 `identity_proof_unavailable` without two-factor sign-in, and a downloaded backup is named after its snapshot time. API scripts that read the removed fields or send a password proof must change with the release. Rolling back means deploying the previous backend and frontend together; nothing persisted needs reverting, and codes issued by either version expire within five minutes.

The rename migration runs at the next start. Rolling back means its down migration renames the columns back; the older code would then read any marker set by a recovery as a pause again. No user data moves.

The drop of `storages.recovery_validation_status` runs at the next start too; a storage still marked `REQUIRES_VALIDATION` becomes usable. Its down migration adds the column back with `AVAILABLE` for every storage, so a rollback does not lock storages that were in use. `Storage.recoveryValidationStatus` and the test result's status leave the API; scripts that read them must change with the release.
