# databasus-backup-ui Specification

## Purpose
Provides the root administrator with one place to configure Databasus backups, save the master key, import an archive with a master key entered as text and resolve protection or mail-delivery problems without navigating between workspace screens.

## Requirements

### Requirement: Global settings contain a root-only Databasus backups section

The section SHALL show, in this order: a short statement of what the Databasus backup saves and that it lets Databasus be recovered on another server if this server breaks, the Databasus backup health unless it is healthy, the backup settings, the recent backups with a manual backup action, and actions that open the master-key dialog and the recovery dialog. On a wide screen the read-only storages and the recent backups SHALL sit side by side, storages first; on a narrow screen they SHALL stack in that order. Master-key acknowledgment, selected destinations, editable shared schedule and retention, manual backup, encrypted archive download, file-based recovery and paged history with each backup's outcome and the storages whose copy failed SHALL remain available from the section, either directly or through one of its dialogs or disclosures. Only root admin SHALL see or operate this section. Ordinary workspace storage forms SHALL have no Databasus backup selection checkbox.

The section SHALL state the snapshot's contents. The need to retain the key and access to at least one storage outside this server SHALL be explained in the master-key dialog and in the settings form. A local destination SHALL carry a visible warning that it does not protect against loss of the same host, both in the read-only settings and in the settings form.

When no destination has ever been selected, the section SHALL show only the statement, the not-backed-up health with any mail notices that apply, and three actions: setting up backups, restoring from a backup and downloading the master key. Setting up backups SHALL open the settings dialog. Cancelling that form SHALL return to the same empty state, send no change and select no destination.

#### Scenario: Another instance administrator opens global settings

- **WHEN** an administrator other than root admin opens global settings
- **THEN** the existing settings remain available but the Databasus backup section and master-key actions do not appear

#### Scenario: Root admin opens the section on a new installation

- **WHEN** root admin opens the section on an instance where no destination has ever been selected
- **THEN** the section shows that Databasus is not backed up and offers to set up backups, restore from a backup and download the master key
- **AND** it shows no settings summary, no history and no recovery form fields

#### Scenario: Root admin cancels the first setup

- **WHEN** root admin presses Set up backups on an instance where no destination has ever been selected, picks a storage and presses Cancel
- **THEN** the section returns to the not-backed-up empty state
- **AND** no settings change is sent and no destination is selected

#### Scenario: Root admin opens the section on a protected installation

- **WHEN** root admin opens the section while backups are healthy
- **THEN** the read-only storages and the recent backups are visible, with no status block and no schedule or retention
- **AND** no destination picker, recovery file input, master-key text or container copy command is visible until root admin asks for it

### Requirement: Recovery starts with an archive file and a master key entered as text

The section SHALL offer a restore action that opens a recovery dialog. The dialog SHALL offer import through a labeled file picker for an encrypted Databasus backup, which also accepts a dropped file and shows the chosen file's name and size, and a masked text field where the owner pastes or types its source master key, with a control that shows or hides it. Both inputs SHALL be required; the UI SHALL NOT ask for a key file. It SHALL explain that the owner can obtain the archive from S3, another storage, another server or this section's history, and that the target's own key does not replace the source key. Import SHALL be available without destinations or backup history on the target. History SHALL retain archive download; using an archive from that history SHALL follow the same download-then-import flow without a direct restore action.

The dialog SHALL show feedback identifying a missing archive or empty master-key field, upload and preparation progress, cancellation before apply, and actionable validation errors beside the import. A failed validation SHALL keep the current installation usable and allow the owner to correct the selected archive or entered key. Closing the dialog before apply is accepted SHALL dismiss the flow: it SHALL abort a running upload or cancel a preparing or prepared recovery on the server, and it SHALL clear the selected archive handle, entered key and root-only preparation details. If that cancellation fails, the dialog SHALL stay open and show the error. Changing session SHALL clear the same data. The selected archive and entered key SHALL NOT be placed in browser persistent storage. All controls SHALL support keyboard use and narrow layouts.

#### Scenario: Owner imports an independently downloaded archive

- **WHEN** root admin opens the recovery dialog on an initialized target without destinations or history
- **THEN** the dialog accepts an archive file and a source master key pasted or typed as text without requesting storage configuration
- **AND** successful preparation shows the authenticated snapshot date and source version before replacement is possible

#### Scenario: The archive or text key is missing

- **WHEN** root admin selects an archive without entering a master key, or enters a key without selecting an archive
- **THEN** the dialog identifies the missing archive or empty key field and does not start the upload

#### Scenario: Preparation rejects the selected archive or entered key

- **WHEN** the server reports a wrong key, damaged archive, version mismatch or upload limit failure
- **THEN** the dialog shows a safe actionable error and permits correction without claiming that recovery succeeded
- **AND** a version mismatch identifies the required source release

#### Scenario: Owner closes the dialog during preparation

- **WHEN** root admin closes the recovery dialog while the archive is uploading, being prepared or prepared
- **THEN** the upload is aborted or the recovery is cancelled on the server and no apply request is sent
- **AND** reopening the dialog shows an empty file picker and an empty key field

### Requirement: Replacement is confirmed after preparation and reconnects to restored accounts

After successful preparation the recovery dialog SHALL replace the file and key inputs with the snapshot date and source version, explain that application replaces current accounts, settings and backup history, and ask for explicit replacement confirmation in the same dialog. While the instance requires two-factor sign-in, the dialog SHALL also offer to email a confirmation code and ask for it, and applying SHALL require both; otherwise the confirmation alone SHALL be enough and the dialog SHALL NOT ask for a password or a code. A rejected code SHALL be shown under the code field and clear when the code changes. Cancelling at this step SHALL NOT submit apply and SHALL cancel the prepared recovery. Repeated clicks or request retries SHALL continue observing the same accepted recovery rather than create another one.

Once apply is accepted, the dialog SHALL show that Databasus is restoring and may temporarily disconnect, and it SHALL NOT offer a way to close itself. It SHALL NOT report success merely because connectivity changes. When the previous session is invalidated it SHALL guide the owner to sign in using the snapshot's accounts and credentials. After sign-in it SHALL show the ordinary application without a pause notice or an activation step. Failed apply SHALL show a safe error without claiming that the imported installation is active.

#### Scenario: Owner cancels replacement

- **WHEN** root admin cancels after viewing the prepared snapshot and replacement warning
- **THEN** no apply request is sent, the prepared recovery is cancelled and the current installation remains available

#### Scenario: Restored account differs from the temporary target account

- **WHEN** import replaces a temporary target root account with the snapshot's accounts
- **THEN** the interface requires sign-in with restored credentials and does not retain the temporary account's session
- **AND** a successful sign-in opens the ordinary application, whose backups and other work already run

#### Scenario: Session changes while preparation is pending

- **WHEN** a preparation response arrives after sign-out or a switch to another account
- **THEN** it cannot expose the selected archive, entered key, snapshot details or apply controls in the new session

### Requirement: Selection and creation stay in one configuration flow

The destination picker SHALL search existing storages across all workspaces, distinguish their workspace and provider, support multiple selections and offer creation even when choices exist. Creating a storage SHALL allow choosing an existing workspace or creating one before entering storage settings. The flow SHALL reuse ordinary creation and connection validation and select the successfully created storage automatically. It SHALL preserve the current page, current workspace selection, unfinished configuration draft and successful workspace creation when a later storage step fails. Dismissing the flow SHALL restore focus and SHALL NOT activate an unsubmitted destination.

#### Scenario: Owner has no workspace

- **WHEN** root admin creates a destination from global settings on an instance without workspaces
- **THEN** the same flow creates a workspace and then a storage, selects it in the backup draft and returns to the backup section
- **AND** saving the configuration requests the first backup without requiring a visit to a workspace page

#### Scenario: Storage creation fails after workspace creation

- **WHEN** the workspace step succeeds and the storage step fails
- **THEN** the error leaves the storage draft available and retry uses the created workspace without creating another one

#### Scenario: Owner cancels a newly selected destination

- **WHEN** a new storage was persisted but root admin cancels the unsaved Databasus backup configuration
- **THEN** the ordinary workspace storage remains and is not made an active Databasus backup destination

### Requirement: Missing system SMTP has a direct route to its existing section

When system SMTP is not configured, the backup section SHALL say so, explain that emails about failed and overdue backups require it, and offer an action that scrolls to and focuses the existing mail-server section in the same settings page. That section SHALL retain operator-configuration guidance and its test-mail action; this feature SHALL add no SMTP credential form. With SMTP configured but no usable root-admin address, the backup section SHALL instead direct the owner to update their profile. SMTP delivery failure SHALL remain visible without hiding the backup problem. Fixing the condition SHALL remove the corresponding notice.

#### Scenario: Owner chooses Configure SMTP

- **WHEN** root admin activates the missing-SMTP notice's action
- **THEN** the settings page scrolls to its mail-server section and keyboard focus moves to its accessible heading

### Requirement: Initial backup reminders wait until the instance is in use

The initial floating setup reminder SHALL appear only to root admin, only when no destinations are selected, and only after at least one user database or storage exists anywhere in the instance. An empty workspace alone SHALL NOT trigger it. The reminder SHALL link to the backup section and SHALL offer no way to defer or dismiss it: it stays until a destination is selected. It SHALL be suppressed while that section is open and SHALL NOT appear while key onboarding is shown. Eligibility SHALL survive reload.

#### Scenario: Instance has only an empty workspace

- **WHEN** root admin has created a workspace but no user database or storage exists
- **THEN** no initial storage-setup floating reminder appears

#### Scenario: Data exists in another workspace

- **WHEN** a user database or storage exists outside the currently selected workspace and no backup destination is selected
- **THEN** root admin receives the setup reminder regardless of the selected workspace

#### Scenario: The reminder cannot be put off

- **WHEN** root admin sees the setup reminder
- **THEN** it offers only the way to the backup section, with no "remind me later" action and no close button, and it stays on later visits until a destination is selected

### Requirement: Floating incidents remain truthful and usable

Failed, partially saved, overdue and stopped backup incidents SHALL show root admin an ongoing floating notice with an action to the backup section, including when email delivery succeeds. The notice SHALL not steal focus or obstruct controls and SHALL use an in-flow presentation on narrow screens. It SHALL identify the actual condition rather than repeatedly asking for initial setup. Removing the final selected storage SHALL immediately show that backups are no longer running. Re-adding a destination SHALL show resumption until a new copy succeeds; selection alone SHALL NOT claim successful protection. The detailed incident SHALL remain visible in the backup section while its floating duplicate is suppressed there. Failed status reads SHALL preserve the last known state with retry feedback rather than claim recovery.

#### Scenario: Last destination is removed

- **WHEN** root admin saves a change from one selected storage to none
- **THEN** the response and interface immediately report stopped backups even if no user database remains
- **AND** leaving the backup section shows the stopped-protection floating notice

#### Scenario: A destination is added again

- **WHEN** root admin adds a destination after stopped protection
- **THEN** the interface reports resumption until the first new successful copy, then clears the stopped incident

#### Scenario: Session changes during a status request

- **WHEN** a root-admin status read completes after signing out or switching to another account
- **THEN** it cannot display root-only notices or instance-backup details in the new session

### Requirement: The section shows the Databasus backup health when it needs attention

The section SHALL open with a short introduction that says what a Databasus backup saves, the users, settings, database connections and backup list, and that these backups let root admin recover Databasus on another server if this server breaks. Unless the health is healthy, the section SHALL then show a status block that states the Databasus backup health in words, marks it with an icon and a color that match it, and shows when the newest successful Databasus backup snapshot was taken across the selected storages, as a relative time with the exact time available. When no selected storage holds a successful copy, the block SHALL say that no successful backup exists yet instead of a time. While a Databasus backup incident is open, the same block SHALL show the incident, when it started, the storages whose latest delivery failed with their sanitized errors, and the email delivery error when sending the alert failed. The section SHALL NOT repeat the incident in a separate notice.

The missing-mail-server and missing-address notices SHALL appear under the status block only while their condition holds. When a status refresh fails, the section SHALL keep the last known status and say that it may be out of date.

#### Scenario: Backups are healthy

- **WHEN** root admin opens the section while every selected storage holds a fresh copy
- **THEN** no status block is shown, and the recent backups show the newest copies
- **AND** no incident, mail-server or address notice is shown when those conditions do not hold

#### Scenario: Backups wait for the first copy

- **WHEN** root admin has saved a destination and no copy has succeeded yet
- **THEN** the status block says that backups wait for the first copy and that no successful backup exists yet
- **AND** it shows no last-backup time

#### Scenario: A storage keeps failing

- **WHEN** one selected storage has an open failure incident with a delivery error
- **THEN** the status block shows the failure, its start time, that storage's name and its sanitized error
- **AND** no second notice repeats the same incident in the section

### Requirement: Backup settings are read-only until root admin edits them

After a destination has been selected, the section SHALL show each storage as one compact line with its provider logo, its name and its workspace in parentheses, such as "Offsite S3 (workspace: Production)", under a Storages heading with an edit action. The read-only view SHALL NOT show the schedule or the retention; the settings form shows them. A storage whose own Databasus backup health is not healthy SHALL show that health and its latest delivery error in this view. When the last storage has been removed, the storages line SHALL state that no storage is selected, and the history SHALL stay visible. An edit action SHALL open the settings form in a dialog over the page, which keeps its scroll position. A click outside the dialog SHALL NOT close it, so a draft is not lost by accident; its close button, Escape and Cancel discard the draft. The form SHALL keep the destination picker, storage creation, the schedule editor and the retention field. It SHALL NOT repeat explanations of shared workspace storages or of how retention deletes copies. Saving successfully or cancelling SHALL close the dialog and return to the read-only view, except that an edit made while the save was in flight SHALL keep the form open with that edit. A failed save SHALL keep the form and the draft.

#### Scenario: One storage is failing

- **WHEN** one of two selected storages has a failed latest delivery with an error
- **THEN** the read-only view shows that storage's health and sanitized error
- **AND** the healthy storage shows only its provider logo, name and workspace

#### Scenario: A local storage is selected

- **WHEN** a local storage is one of the selected storages
- **THEN** the read-only view shows the warning that it does not protect against losing this server

#### Scenario: The last storage was removed

- **WHEN** root admin has removed the last selected storage and earlier backups exist
- **THEN** the status block shows that backups are stopped, the storages line says that no storage is selected and the recent backups stay visible

#### Scenario: Root admin edits the schedule

- **WHEN** root admin presses the edit action, changes the schedule and saves successfully
- **THEN** the new schedule is saved, the form closes and the read-only storages view returns

#### Scenario: Root admin edits while a save is in flight

- **WHEN** root admin saves a retention change and types another retention before the save completes
- **THEN** the save is acknowledged and the form stays open with the newer retention and an enabled Save

#### Scenario: Root admin cancels an edit

- **WHEN** root admin changes the selected storages and presses Cancel
- **THEN** the read-only view shows the saved storages and no change is sent

### Requirement: Recent backups are compact and the full history is one action away

The section SHALL show the five newest Databasus backups, one row each, with the snapshot time, the outcome in words, the size when known, and a download action when at least one copy can be downloaded. The download action SHALL NOT ask root admin to choose a storage: it SHALL try the downloadable copies in order and download the first one whose storage issues a download, and it SHALL report an error only when none does. The downloaded file SHALL be named after the backup's snapshot time in UTC, such as `databasus-backup-2026-11-18-09-53-12.dbbackup`, not after the stored file. A row SHALL NOT expand. A row whose backup failed or reached only some storages SHALL show below it the failure reason and each failed copy with its storage, workspace, status and sanitized error; rows SHALL NOT show the trigger, the source version or successful copies. The manual backup action SHALL sit with the recent backups and SHALL be disabled with a stated reason while no destination is saved. When more backups exist, a Show all action SHALL reveal the older rows in place and let root admin page through the rest. A backup in progress SHALL be marked with a spinning icon. Times in the section SHALL show the day, the short month name in the interface language, the year and the time on the browser's 12- or 24-hour clock, followed by the relative time in parentheses, such as "14 Nov 2025, 13:00 (4 days ago)". The status block's last backup SHALL stay a relative time with the exact time available.

#### Scenario: History has more than five backups

- **WHEN** the instance has twelve Databasus backups
- **THEN** the section shows the five newest rows and a Show all action
- **AND** Show all reveals the older rows in the same list, and Show more loads the rest until all twelve are shown

#### Scenario: The first storage cannot serve a download

- **WHEN** a backup has copies in two storages and root admin downloads it while the first storage refuses the download
- **THEN** the section downloads the copy from the second storage without asking which storage to use

#### Scenario: A backup reached only some storages

- **WHEN** a backup was stored in one selected storage and failed in another
- **THEN** its row shows the partial outcome and, below it, the failed storage with its status and sanitized error
- **AND** the row does not list the storage that stored its copy

### Requirement: The backups section mentions cleanup a recovery dropped

When the latest Databasus recovery dropped pending file deletions of the lost installation, the Databasus backups section SHALL say how many it dropped and that leftover files in the storages may need manual cleanup. It SHALL show nothing about it when the latest recovery dropped none or there was no recovery. The note SHALL NOT block or delay any action.

#### Scenario: A recovery dropped pending deletions

- **WHEN** root admin opens the Databasus backups section after a recovery that dropped three pending file deletions
- **THEN** the section says that three pending deletions were dropped and that leftover files may need manual cleanup

#### Scenario: A recovery dropped nothing

- **WHEN** root admin opens the Databasus backups section after a recovery that dropped no pending deletion
- **THEN** the section shows no cleanup note
