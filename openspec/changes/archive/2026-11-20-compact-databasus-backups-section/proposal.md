## Why

The Databasus backups section in global settings shows every control and every explanation at once. It contains three paragraphs of introduction, a master-key panel with the container copy command, a destination form that is always in edit mode, ten large history cards and the full recovery form. Root admin opens the section to learn one thing, whether Databasus is backed up, and the answer is a single word lost among controls that are needed once a year.

After a Databasus recovery the recovered installation stays paused: no backups, WAL streaming, restores, verifications, healthchecks, notifications or storage writes run until root admin confirms that the old installation is stopped and presses Activate. The pause guards against two installations writing to the same storages and databases, but it also leaves a freshly recovered installation without backups until someone notices the notice. Root admin recovers because the old server is gone, so the extra confirmation mostly delays protection. The owner decided that recovery should continue work at once, and that stopping the old installation is root admin's responsibility.

## What Changes

- A status card appears under the introduction whenever the Databasus backup health is anything but healthy. It shows the health, when the last successful backup was made and, when there is a problem, the incident, the failing storages and the email delivery error. A healthy instance shows no card, because the recent backups already show that backups are made. The separate status line and the separate incident alert go away.
- The read-only view shows only the selected storages under a Storages heading; the schedule and the retention appear only in the form. An Edit action opens the existing form in a dialog. Save or Cancel closes it. The destination picker, the storage-creation flow, the schedule and the retention appear only in the form, without the shared-storage and retention explanations.
- An instance where no storage has ever been selected shows a short empty state with Set up backups, which opens the same form, Recover from backup and Download master key. An instance whose last storage was removed keeps the read-only settings, which say that no storage is selected, and its history.
- History shows the five newest backups as one-line rows: time as "14 Nov 2025, 13:00 (4 days ago)" in the interface language, outcome with a spinner while a backup runs, size and download. Only a failed or partial row adds its failure reason and failed copies below it; the trigger, the source version and successful copies are no longer shown. Show all expands the list in place and pages through the rest.
- Every time in the section shows the relative time in parentheses after the exact time. Selected storages are compact lines with the provider logo, the storage name and its workspace in parentheses.
- The master key moves into a dialog opened by Download master key, which reads the key at once and looks like the onboarding step. The container copy instructions sit inside the dialog behind a disclosure.
- The onboarding step's "I've saved it in a safe place" action also waits 15 seconds after the step appears and counts them down, so root admin cannot confirm before reading the explanation.
- Recovery moves into one dialog with two steps and a simpler first step, a file card that also accepts a dropped file and a key field with a show button: file and source key, then the prepared snapshot with the replacement confirmation and, with two-factor sign-in, the emailed code. The separate replacement dialog on top of the form goes away. Closing the dialog before apply cancels the upload or the prepared recovery and clears the file and the key.
- The introduction shrinks to two short sentences: what a Databasus backup saves and that it lets Databasus be recovered on another server if this server breaks. The advice to keep the key and a storage outside this server moves into the master-key dialog and the settings form.
- The shared storage picker keeps a selected storage at its natural width instead of one letter per line.
- The interface dictionaries of all seven languages gain the new copy and lose the copy nobody renders any more.

- Recovery confirmation no longer asks for the password. While the instance requires two-factor sign-in it asks for an emailed code, as the master-key export does; otherwise the replacement confirmation is enough.
- A downloaded backup is named after its snapshot time.

User data does not change; a migration renames the recovery pause columns to a recovery marker, and another drops the storage validation status. The API loses `masterKeyPath`, and the recovery apply takes only an optional emailed code, as Impact describes. The browser end-to-end recovery scenario follows the new dialog.

Recovery no longer pauses the recovered installation:

- A published recovery, whether from mounted files at startup or from an upload in the interface, starts every job and accepts every action at once, as an ordinary start does.
- **BREAKING** for API scripts: `POST /instance-recoveries/{instanceRecoveryId}/activate` (`activateInstanceRecovery`) and its `InstanceRecoveryActivation` body are deleted. `InstanceBackupStatus` loses `isRecoveryPaused`. The `recovery_paused` problem code is deleted, so no operation answers it any more. `InstanceRecoveryStatus` loses `PAUSED` and `ACTIVE`; a published recovery reports `COMPLETED`.
- The pause notice with its Activate dialog disappears from the application. When a recovery dropped pending file deletions of the lost installation, the Databasus backups section says that leftover files may need manual cleanup.
- A recovered local storage works at once, like every other storage: backups write to it, downloads and restores read from it and cleanup deletes from it without a connection test. Root admin is responsible for its folder existing on the new host; while it does not, its writes fail with the folder's error, as on any host.
- **BREAKING** for API scripts: `Storage` loses `recoveryValidationStatus` and a connection test result loses its status. No storage operation answers 409 `storage_unavailable` because of a recovery; a Databasus backup download still answers it when no copy can serve the file.
- The persisted pause marker in the internal database becomes a plain marker of the recovery that produced the database, because publication still needs it to recognize its own staged database.
- `docs/architecture.md` and `docs/glossary.md` drop the recovery pause and the local-storage validation.

Nothing is kept as a shim: the activation operation, the gate, its checks in every module, the job wrapper, the frontend notice, the storage validation status with its extended probe and its interface explanation are deleted.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `databasus-backup-ui`: after sign-in with the restored accounts the interface shows no pause or activation, and the backups section mentions dropped cleanup when it applies; the section becomes status-first with read-only settings edited in a dialog, storages beside the history on wide screens, compact recent history that expands in place, and recovery inside one two-step dialog whose dismissal cancels the pending recovery.
- `master-key-export`: the master-key panel in the instance settings becomes a dialog without a saved-state line, and the instructions for copying the key from the server name the `databasus-data/secret.key` file instead of an absolute path and a `docker cp` command; the onboarding confirmation waits 15 seconds after the step appears.
- `databasus-recovery`: an emailed recovery code is offered and sent only while the instance requires two-factor sign-in; a recovered installation runs at once, without a durable pause or activation, and recovered local storages are used at once without validation.
- `internal-database-migrations`: recovery scenarios no longer say the application starts or returns paused.
- `storages`: a connection test no longer validates a recovered local storage; it only reports its outcome and changes nothing.

## Impact

- **BREAKING** for API scripts: `InstanceBackupStatus` no longer carries `masterKeyPath`. The frontend of the same release stops reading it, and the absolute path depended on the installation anyway.
- **BREAKING** for API scripts: `InstanceRecoveryApply` replaces `proof` with an optional `code`, and a password is no longer accepted; `InstanceBackupStatus` replaces `identityProofMethods` and `isMasterKeyExportCodeRequired` with `isConfirmationCodeRequired`. Without two-factor sign-in a recovery code request answers 409 `identity_proof_unavailable`.
- Frontend: `frontend/src/pages/instance-settings/ui/instance-backups/` is reorganized. The status card is built from the themed Mantine alert, so the theme does not change.
- Interface dictionaries in `frontend/src/shared/i18n/locales/`.
- Tests: the section, recovery and master-key screen tests, the browser UI scenario `frontend/browser/scenarios/instanceBackups.spec.ts` with its regenerated references, and `e2e/browser/scenarios/test_databasus_recovery.py`.
- Rules the change answers to: the root `AGENTS.md`, `frontend/AGENTS.md`, `docs/frontend-design.md` and `docs/frontend-localization.md`.
- Backend: `lib/recoverypause` and `app/recovery_pause.go` are deleted, and the gate leaves the Deps of `backups`, `databases`, `notifiers`, `storages`, `verification` and `instancebackups`. The `instancebackups` recovery service loses activation, and the status loses the pause. A new migration renames the pause columns of `instancebackups_configs` to a recovery marker. The `storages` module loses the validation status, the extended recovered-storage probe and the recovery step that marks local storages; a new migration drops `storages.recovery_validation_status`.
- Contracts: `backend/api/openapi.yaml` and `backend/api/agent.yaml` lose the pause wording, the operation and the problem code; the generated servers, clients and frontend types are regenerated.
- Frontend: `RecoveryPauseNotice` and its dictionary entries in all seven languages are deleted; the recovery dialog treats `COMPLETED` as the finished restore.
- Tests: the pause tests of every module, the pause helpers in `apptest`, the recovery e2e packages, the upgrade test and the browser recovery scenario.

## Out of scope

- Any change to the status computation, or to the recovery protocol beyond the emailed-code rule.
- A "next scheduled backup" time, because the status does not report one.
- The floating notices, the master-key onboarding screen apart from its 15-second wait, and the post-recovery pause notice.
- Restoring directly from a history row; recovery still means download, then import.
- The other sections of the settings page, including the mail server section.
- The WAL receiver's back-pressure pause, slot-rebuild pause and the physical-backup identity pause. They share the word, not the feature.
- The recovered-runtime normalization: interrupted operations, revoked sessions and tokens, dropped deletion obligations and the new full physical chain stay as they are.
- Any detection that the old installation still runs.
