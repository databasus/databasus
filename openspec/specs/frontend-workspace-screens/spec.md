# frontend-workspace-screens Specification

## Purpose

Keep workspace, storage and notifier workflows consistent across navigation, overlapping requests and composed editors while preserving existing capabilities and permissions.

## Requirements

### Requirement: Workspace changes publish acknowledged state without losing newer edits

Workspace creation, settings, membership and deletion SHALL retain existing role and instance-policy restrictions. Creation SHALL select the created workspace; an acknowledged rename SHALL update its application-wide label without replacing newer name input. A pending save SHALL prevent duplicate submission, success SHALL acknowledge only submitted values, returning to the acknowledged baseline SHALL make the form clean, and failure SHALL preserve the draft. Successful deletion SHALL remove the workspace from navigation and select another accessible workspace or show the permitted empty-workspace experience without reloading the application.

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

### Requirement: Workspace-scoped responses cannot populate another workspace

Changing workspace SHALL invalidate previous lists, details, editor completions and feedback before displaying the new workspace's content. An unavailable remembered workspace SHALL never be used as an authority to load its content. If a current workspace becomes unavailable, the interface SHALL clear its content, refresh accessible workspaces and offer an authorized selection or recoverable failure. Session replacement and unmount SHALL invalidate obsolete work. Language and theme changes SHALL preserve the active workspace, selection, drafts and pending operations.

#### Scenario: An old workspace request finishes last

- **WHEN** a list, detail load or mutation response from the previous workspace finishes after a workspace switch
- **THEN** it cannot replace current content, update selection, close a current editor or publish old feedback

#### Scenario: Access to the selected workspace disappears

- **WHEN** the server establishes that the selected workspace no longer exists or is inaccessible
- **THEN** its data is cleared and accessible workspace selection is refreshed
- **AND** a failed refresh ends in translated recovery rather than stale content or permanent loading

#### Scenario: A newer selection supersedes an access refresh

- **WHEN** the user selects another workspace and edits its settings while an earlier access refresh remains pending
- **THEN** that refresh cannot replace the selected workspace or discard its draft, whether it returns an older workspace list or fails

#### Scenario: The database list establishes that workspace access disappeared

- **WHEN** an initial or background database list load receives a current HTTP 403 or 404 for the databases or for the workspace backup configurations or storages required to display them
- **THEN** the workspace database list and details are cleared before accessible-workspace refresh completes
- **AND** the unavailable workspace is removed from navigation without a page reload
- **AND** another accessible workspace is selected or the permitted empty-workspace experience is shown when none remains

#### Scenario: Workspace refresh fails after database access is lost

- **WHEN** the database list establishes workspace unavailability and refreshing accessible workspaces then fails
- **THEN** the obsolete database content remains absent and translated recovery with an explicit Retry action is offered
- **AND** a successful retry restores an accessible workspace selection without restoring the unavailable content

#### Scenario: An access refusal follows a temporary collection failure

- **WHEN** an initial or background database list load first receives a network failure or HTTP 500 from one required workspace read and later receives a current HTTP 403 or 404 from another
- **THEN** the later refusal clears the workspace database content and refreshes accessible workspace selection even while another required read remains pending
- **AND** multiple access refusals from that load start at most one workspace reconciliation
- **AND** no local Retry action for the unavailable workspace remains

#### Scenario: An obsolete database list refusal arrives after switching workspaces

- **WHEN** the user has selected another workspace and an earlier database list load returns HTTP 403 or 404
- **THEN** the refusal does not remove a workspace, refresh workspace navigation, change the selection, or publish old feedback

### Requirement: Resource selection remains valid within its workspace

Database, storage and notifier selection SHALL be remembered separately for each workspace and revalidated against successfully loaded items. Changing the workspace or preference key SHALL read that scope's preference without writing the previous selection into it. Desktop initial load SHALL select the remembered available item or the first item; mobile initial load SHALL show the list. Explicit selection, back-to-list, create and delete SHALL preserve the existing list/detail behavior. Creating an item or copying a database SHALL select the new item. After removing the selected item, by deletion or by transfer to another workspace, selection SHALL move to the first remaining item in list order, or become empty. A background refresh that establishes the selected item is gone SHALL apply the same validity rule.

#### Scenario: A preference key changes without remounting

- **WHEN** the selection scope changes to another workspace while the screen remains mounted
- **THEN** the new selection uses only that workspace's available items and preference, and its stored preference is not overwritten by the old ID

#### Scenario: A remembered storage was removed

- **WHEN** a desktop list loads successfully and its remembered storage is absent
- **THEN** the first available storage is selected or an empty state is shown

#### Scenario: A user opens the mobile list

- **WHEN** a database, storage or notifier page first loads on mobile
- **THEN** the list is shown until the user selects an item, and back-to-list remains available from details

#### Scenario: The selected database is transferred

- **WHEN** the selected database is successfully transferred to another workspace
- **THEN** it leaves the list and the first remaining database is selected, or the list shows no selection when none remains

#### Scenario: A database is copied

- **WHEN** a database copy succeeds
- **THEN** the copy appears in the list and becomes the selected database

### Requirement: Connection and delivery tests validate only the current settings revision

A successful storage connection test or notifier delivery test SHALL permit saving only the settings revision it tested. Any settings edit SHALL invalidate that approval, including a change away from and back to the same values. An old test result or error SHALL not change the current approval, feedback or pending state. Changing only the name, language or theme SHALL preserve valid test approval. Notifier delivery tests SHALL run only on explicit activation and SHALL not repeat automatically after failure or presentation changes.

#### Scenario: Settings change during a connection test

- **WHEN** a test of settings A completes after the user changes to B, or to B and back to A
- **THEN** it cannot approve saving the current draft or publish obsolete test feedback

#### Scenario: The storage name changes after testing

- **WHEN** tested settings stay unchanged and the user edits only the name
- **THEN** the current test approval remains valid

#### Scenario: A connection test completes during saving

- **WHEN** a test uses the previous saved credentials and completes before a pending credential replacement is acknowledged
- **THEN** the save acknowledgment invalidates that approval and another explicit test is required for the current saved credentials

#### Scenario: A notifier test fails

- **WHEN** a test notification fails
- **THEN** its current translated failure is shown and another delivery requires a new explicit activation

### Requirement: Saving resources acknowledges submitted values and preserves newer input

Storage and notifier editors SHALL compare the current draft with the last acknowledged baseline. Saving SHALL require a valid current test and prevent duplicate submission. Successful creation SHALL retain the assigned ID and update the containing list even when newer edits keep the editor open. Success SHALL merge normalized returned values and secret acknowledgments only where the corresponding input has not changed. Newer input SHALL remain available; only an unchanged acknowledged draft SHALL complete the editor. Failure SHALL preserve the draft and leave saving available after correction without implying success. A subsequent save after successful creation SHALL update the assigned resource rather than create another one. Unchanged secrets SHALL remain omitted from test and update requests under the provider's existing policy.

#### Scenario: Resource editing modes remain coordinated

- **WHEN** a detail screen is editing the resource name or settings
- **THEN** the other editing mode remains unavailable while the current editor is open, including during pending saves, and subsequent settings saving retains the acknowledged name

#### Scenario: A user edits during creation

- **WHEN** creation succeeds after the user enters newer values
- **THEN** the new item and its assigned ID are retained, the editor stays open with newer values, and the next save updates that item

#### Scenario: A secret changes during saving

- **WHEN** a save succeeds after the user enters a newer secret
- **THEN** the acknowledgment does not erase the newer secret or mark it as already persisted

#### Scenario: A save fails

- **WHEN** the current save is refused or its response fails
- **THEN** pending state ends, the draft remains editable and no completion or automatic retry occurs

### Requirement: Creating a resource inside another workflow preserves its caller

Storage and notifier creation from database settings and transfers SHALL retain the calling draft, target workspace, permissions and resource policy. Canceling creation SHALL preserve the previous selection. Successfully completing creation SHALL refresh the relevant workspace's choices and select the created resource for that caller. A persisted creation with newer unsaved editor input SHALL update available choices without prematurely closing the editor. Selection values SHALL identify real resources only. Canceling a parent workflow or changing its target workspace SHALL prevent a late nested completion from changing the new caller. Existing confirmations before replacing a backup storage SHALL remain effective.

#### Scenario: Storage creation is canceled inside backup configuration

- **WHEN** a user opens and cancels storage creation from logical or physical backup settings
- **THEN** the config draft and previous storage selection remain unchanged

#### Scenario: Storage creation completes inside transfer

- **WHEN** a user creates and completes a storage for a database transfer's target workspace
- **THEN** that storage becomes the selected real target resource without losing transfer options or changing the source workspace

#### Scenario: A target workspace changes during nested creation

- **WHEN** an old nested creation completes after the caller has changed target workspace or closed
- **THEN** it does not select an item for or complete the new caller

### Requirement: Resource operations retain provider and access policy

All existing storage and notifier provider variants, validation, secret omission, saved-type restrictions, connection or delivery tests, deletion conflicts, transfers and storage migration progress SHALL remain available under their existing permissions. Rclone management SHALL retain its instance-admin restriction. A refusal SHALL retain the last acknowledged resource and provide translated correction or recovery. An accepted transfer or migration SHALL continue to show its actual progress rather than imply immediate completion.

#### Scenario: A resource in use cannot be deleted

- **WHEN** deleting a storage or notifier is refused because another resource uses it
- **THEN** the item and selection remain available and the relevant translated conflict is shown

#### Scenario: A viewer opens a resource

- **WHEN** a workspace viewer opens storage or notifier details
- **THEN** existing read and saved-test capabilities remain available and prohibited mutation controls are unavailable

### Requirement: Migrated screens provide bounded loading and usable presentation

Failed list, detail, workspace-settings or audit loads SHALL end pending indicators and show translated feedback with explicit recovery rather than an indefinite spinner or a successful empty state. Retry SHALL reload the current read context without replaying mutations. Workspace, storage and notifier screens and composed editors SHALL remain usable in both themes, six interface languages, at 320 CSS pixels and with text enlarged to 200 percent. Labels, validation, dialogs and actions SHALL be accessible by keyboard and shall not require page-wide horizontal scrolling. Each bounded content region SHALL have one vertical scroll owner. Persistent feedback SHALL follow the active language; changing presentation SHALL not reset input or repeat requests.

#### Scenario: The first workspace-settings request fails

- **WHEN** the initial settings load fails before any workspace detail is available
- **THEN** loading ends and an explicit translated retry can recover the same context

#### Scenario: A long translation appears in a narrow editor

- **WHEN** a user edits a resource at 320 CSS pixels with enlarged text and long translated labels or errors
- **THEN** fields and actions remain reachable without clipping, overlap or page-wide horizontal overflow
