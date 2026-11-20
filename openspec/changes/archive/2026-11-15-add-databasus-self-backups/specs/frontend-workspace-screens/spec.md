## MODIFIED Requirements

### Requirement: Workspace changes publish acknowledged state without losing newer edits

Workspace creation, settings, membership and deletion SHALL retain existing role and instance-policy restrictions. Ordinary creation SHALL select the created workspace. Creation inline within Databasus backup configuration SHALL instead add the workspace to available choices while preserving the current settings page, its address, the previous workspace selection or absence of one, and the caller's unfinished backup draft. The inline flow SHALL use the created workspace for its storage-creation step without changing application navigation. An acknowledged rename SHALL update its application-wide label without replacing newer name input. A pending save SHALL prevent duplicate submission, success SHALL acknowledge only submitted values, returning to the acknowledged baseline SHALL make the form clean, and failure SHALL preserve the draft. Successful deletion SHALL remove the workspace from navigation and select another accessible workspace or show the permitted empty-workspace experience without reloading the application.

#### Scenario: A workspace name changes while saving

- **WHEN** a workspace rename is pending and the user enters a newer name
- **THEN** success publishes the acknowledged name to navigation and retains the newer input as an unsaved change

#### Scenario: A workspace is deleted

- **WHEN** an authorized user successfully deletes the selected workspace
- **THEN** it disappears from navigation and its previous content is replaced by an accessible workspace or the empty-workspace experience

#### Scenario: A member manages workspace membership

- **WHEN** a user adds, invites, changes the role of or removes a member
- **THEN** existing role restrictions, invitation behavior and refusal feedback remain effective
- **AND** a failed action does not imply a successful membership change

#### Scenario: Ordinary workspace creation succeeds

- **WHEN** a permitted user creates a workspace outside inline Databasus backup configuration
- **THEN** the created workspace becomes the selected workspace

#### Scenario: Inline workspace creation succeeds in backup configuration

- **WHEN** root admin creates a workspace within Databasus backup configuration
- **THEN** the workspace becomes available to the inline storage-creation step without changing the current page, address or application-wide workspace selection
- **AND** the backup draft remains available, including when there was no selected workspace
