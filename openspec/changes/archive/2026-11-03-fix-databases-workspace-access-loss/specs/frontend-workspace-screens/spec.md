## MODIFIED Requirements

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
