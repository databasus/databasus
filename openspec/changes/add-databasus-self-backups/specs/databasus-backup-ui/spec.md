## Purpose

Provides the root administrator with one place to configure Databasus backups, save the master key and resolve protection or mail-delivery problems without navigating between workspace screens.

## ADDED Requirements

### Requirement: Global settings contain a root-only Databasus backups section

The section SHALL show master-key acknowledgment, selected destinations, editable shared schedule and retention, manual backup, encrypted archive download and paged history with outcomes per destination. Only root admin SHALL see or operate this section. Ordinary workspace storage forms SHALL have no Databasus backup selection checkbox. The section SHALL explain the snapshot's contents and the need to retain the key and access to at least one storage separately. A local destination SHALL carry a visible warning that it does not protect against loss of the same host.

#### Scenario: Another instance administrator opens global settings

- **WHEN** an administrator other than root admin opens global settings
- **THEN** the existing settings remain available but the Databasus backup section and master-key actions do not appear

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
