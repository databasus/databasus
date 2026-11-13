## Purpose

Provides the root administrator with one place to configure Databasus backups, save the master key, import an archive with a master key entered as text and resolve protection or mail-delivery problems without navigating between workspace screens.

## ADDED Requirements

### Requirement: Global settings contain a root-only Databasus backups section

The section SHALL show master-key acknowledgment, selected destinations, editable shared schedule and retention, manual backup, encrypted archive download, file-based recovery and paged history with outcomes per destination. Only root admin SHALL see or operate this section. Ordinary workspace storage forms SHALL have no Databasus backup selection checkbox. The section SHALL explain the snapshot's contents and the need to retain the key and access to at least one storage separately. A local destination SHALL carry a visible warning that it does not protect against loss of the same host.

#### Scenario: Another instance administrator opens global settings

- **WHEN** an administrator other than root admin opens global settings
- **THEN** the existing settings remain available but the Databasus backup section and master-key actions do not appear

### Requirement: Recovery starts with an archive file and a master key entered as text

The section SHALL offer import through a labeled file input for an encrypted Databasus backup and a masked text field where the owner pastes or types its source master key. Both inputs SHALL be required; the UI SHALL NOT ask for a key file. It SHALL explain that the owner can obtain the archive from S3, another storage or another server independently, and that the target's own key does not replace the source key. Import SHALL be available without destinations or backup history on the target. History SHALL retain archive download; using an archive from that history SHALL follow the same download-then-import flow without a direct restore action.

The form SHALL show feedback identifying a missing archive or empty master-key field, upload and preparation progress, cancellation before apply, and actionable validation errors beside the import. A failed validation SHALL keep the current installation usable and allow the owner to correct the selected archive or entered key. Dismissing the flow or changing session SHALL clear the selected archive handle, entered key and root-only preparation details. The selected archive and entered key SHALL NOT be placed in browser persistent storage. All controls SHALL support keyboard use and narrow layouts.

#### Scenario: Owner imports an independently downloaded archive

- **WHEN** root admin opens recovery on an initialized target without destinations or history
- **THEN** the form accepts an archive file and a source master key pasted or typed as text without requesting storage configuration
- **AND** successful preparation shows the authenticated snapshot date and source version before replacement is possible

#### Scenario: The archive or text key is missing

- **WHEN** root admin selects an archive without entering a master key, or enters a key without selecting an archive
- **THEN** the form identifies the missing archive or empty key field and does not start the upload

#### Scenario: Preparation rejects the selected archive or entered key

- **WHEN** the server reports a wrong key, damaged archive, version mismatch or upload limit failure
- **THEN** the form shows a safe actionable error and permits correction without claiming that recovery succeeded
- **AND** a version mismatch identifies the required source release

### Requirement: Replacement is confirmed after preparation and reconnects to restored accounts

After successful preparation the interface SHALL show the snapshot date and source version and explain that application replaces current accounts, settings and backup history. Applying SHALL require explicit replacement confirmation and recent root identity proof. Cancelling this confirmation SHALL NOT submit apply. Repeated clicks or request retries SHALL continue observing the same accepted recovery rather than create another one.

Once apply is accepted, the interface SHALL show that Databasus is restoring and may temporarily disconnect. It SHALL NOT report success merely because connectivity changes. When the previous session is invalidated it SHALL guide the owner to sign in using the snapshot's accounts and credentials. After sign-in it SHALL show the durable recovery pause and existing root activation flow. Failed apply SHALL show a safe error without claiming that the imported installation is active.

#### Scenario: Owner cancels replacement

- **WHEN** root admin cancels after viewing the prepared snapshot and replacement warning
- **THEN** no apply request is sent and the current installation remains available

#### Scenario: Restored account differs from the temporary target account

- **WHEN** import replaces a temporary target root account with the snapshot's accounts
- **THEN** the interface requires sign-in with restored credentials and does not retain the temporary account's session
- **AND** a successful sign-in shows the recovery pause until root confirms activation

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
- **THEN** the settings page scrolls to its mail-server section and keyboard focus moves to its accessible heading without losing the backup draft

### Requirement: Initial backup reminders wait until the instance is in use

The initial floating setup reminder SHALL appear only to root admin, only when no destinations are selected, and only after at least one user database or storage exists anywhere in the instance. An empty workspace alone SHALL NOT trigger it. The reminder SHALL link to the backup section and support deferral until the next day. It SHALL be suppressed while that section is open and SHALL not compete with initial key onboarding. Deferral and eligibility SHALL survive reload; a new stopped-protection incident SHALL bypass an older setup deferral.

#### Scenario: Instance has only an empty workspace

- **WHEN** root admin has created a workspace but no user database or storage exists
- **THEN** no initial storage-setup floating reminder appears

#### Scenario: Data exists in another workspace

- **WHEN** a user database or storage exists outside the currently selected workspace and no backup destination is selected
- **THEN** root admin receives the setup reminder regardless of the selected workspace

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
