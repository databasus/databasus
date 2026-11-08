# frontend-navigation Specification

## Purpose

Let users share and revisit application screens through their addresses, with browser history and automatic recovery to an accessible destination when a requested screen is unavailable.

## Requirements

### Requirement: Screen addresses identify the visible selection

The application SHALL provide the following addresses. Once navigation has resolved, the address SHALL identify the section, workspace, selected database and database tab actually shown. An explicit address SHALL take priority over remembered selections. The existing sign-in and Google storage callback addresses SHALL retain their separate behavior.

| Destination | Address |
| --- | --- |
| Workspace databases | `/workspaces/:workspaceId/databases` |
| Database tab | `/workspaces/:workspaceId/databases/:databaseId/:tab` |
| Workspace storages | `/workspaces/:workspaceId/storages` |
| Workspace notifiers | `/workspaces/:workspaceId/notifiers` |
| Workspace settings | `/workspaces/:workspaceId/settings` |
| Profile | `/profile` |
| Instance users | `/users` |
| Instance settings | `/databasus-settings` |

Database tabs SHALL be `backups`, `config`, `verifications` and `monitoring`, subject to the selected database's offered tabs. A database address without a tab SHALL resolve to its Backups address. Opening `/` SHALL resolve to the accessible remembered workspace or the first accessible workspace, and then its databases screen. A wholly unknown address SHALL resolve through `/`.

The databases address without a database selection SHALL show the list on mobile. On desktop it SHALL select the available remembered database or the first available database, replacing the address with that database's Backups address. With no databases, it SHALL keep the workspace databases address and show the existing empty-list experience. Explicit database addresses SHALL open details directly on every viewport.

With no accessible workspaces, `/` SHALL show the existing empty-workspace experience, including creation only when permitted. Profile and permitted instance screens SHALL remain reachable.

#### Scenario: Reload restores the selected tab

- **WHEN** a user reloads an accessible database's Configuration address
- **THEN** the same workspace, database and Configuration tab appear and the address remains that tab's address

#### Scenario: An explicit link overrides local selection

- **WHEN** a user with another workspace remembered opens an accessible database link
- **THEN** the workspace and database named by the link are selected instead of the remembered selection

#### Scenario: A mobile user opens a database link

- **WHEN** a user opens an accessible database-tab address on mobile
- **THEN** the details and tab appear directly instead of the database list

#### Scenario: A database address omits its tab

- **WHEN** a user opens an accessible database address without a tab
- **THEN** Backups opens and replaces the address with its canonical tab address

#### Scenario: A desktop user enters through the root

- **WHEN** a desktop user opens `/` with an accessible remembered workspace and database
- **THEN** that database's Backups opens and the address becomes its canonical tab address

#### Scenario: No workspace is available

- **WHEN** root navigation resolves with no accessible workspace
- **THEN** the address is `/`, the empty-workspace experience is shown and permitted account screens remain reachable

### Requirement: User navigation participates in browser history

Choosing a different main section, database or database tab SHALL add one history entry for the resolved destination. Choosing a different workspace while a workspace section is open SHALL retain that section and resolve its selection in the new workspace. Back and Forward SHALL restore the destination recorded in history, subject to current access and availability. Choosing the already displayed destination SHALL NOT add another entry.

Automatic fallback and address completion SHALL replace the current history entry. They SHALL NOT add intermediate unavailable or incomplete destinations. Returning to the mobile database list SHALL use the workspace databases address and SHALL work when the details were entered through a direct link with no prior application history. Changing the workspace preference on a global account screen SHALL retain that global address without adding a history entry for an invisible selection.

#### Scenario: Back restores the previous tab

- **WHEN** a user opens Backups, then Configuration, then presses Back
- **THEN** Backups and its address return, and Forward restores Configuration

#### Scenario: Back restores the previous workspace

- **WHEN** a user changes from one accessible workspace's Storages section to another and presses Back
- **THEN** the first workspace's Storages section and address return

#### Scenario: A direct mobile link returns to the list

- **WHEN** a mobile user opens a database link in a new tab and chooses Back to list
- **THEN** the workspace database list appears at its list address without leaving the application

#### Scenario: Fallback does not create a history trap

- **WHEN** an unavailable database destination falls back to an accessible database and the user presses Back
- **THEN** history returns to the earlier destination rather than reopening the rejected destination

### Requirement: Unavailable destinations resolve automatically to available content

After successful availability checks, an unavailable main section, workspace, database or database tab SHALL resolve automatically to an available destination and replace the address accordingly. Missing resources and resources unavailable to the account SHALL follow the same fallback policy.

- An unavailable or unknown database tab SHALL become Backups for the same accessible database.
- A missing database or one absent from the addressed workspace SHALL become the first available database of that workspace, using the requested tab when offered and Backups otherwise. With none available, the workspace database list SHALL appear at its list address.
- An unavailable workspace SHALL become the first accessible workspace and its databases destination. With no accessible workspace, the empty-workspace experience SHALL appear at `/`.
- An unavailable administrative section SHALL resolve through the default databases destination or the empty-workspace experience.

An explicit database ID SHALL NOT fall back merely because a list has not finished loading or a search filter hides it. A confirmed loss of access after entry SHALL reconcile the current available resources before resolving a replacement. The application SHALL NOT repeatedly retry a rejected destination or loop through unavailable fallbacks.

Network errors, server failures and incomplete startup SHALL NOT be treated as proof of unavailability. They SHALL retain the requested address and show the existing recoverable loading or failure state. Session refusal SHALL follow the session-transition requirements.

#### Scenario: A link names a database of another workspace

- **WHEN** a link names an accessible workspace but its database does not belong to that workspace
- **THEN** an available database of the addressed workspace opens, or its empty list appears
- **AND** the address changes to that displayed destination

#### Scenario: A foreign workspace is unavailable

- **WHEN** a user opens a link into a workspace they cannot access
- **THEN** the first accessible workspace's databases destination opens, or the empty-workspace experience appears if there is none
- **AND** the address changes to the displayed destination

#### Scenario: A database tab is unavailable

- **WHEN** a user opens Verifications for a database that does not offer it
- **THEN** Backups appears and its address replaces the requested address

#### Scenario: The selected database disappears

- **WHEN** a successful refresh establishes that the selected database is no longer available in the workspace
- **THEN** an available replacement database or the empty list appears with its matching address

#### Scenario: Database loading fails temporarily

- **WHEN** a database read fails because the server is unreachable
- **THEN** the requested address remains and a translated error with retry appears without selecting another database

### Requirement: Navigation preserves resource access isolation

Navigation SHALL validate workspace access and database membership before displaying database details or offering actions for that destination. It SHALL NOT request resource content for a workspace already known to be outside the account's accessible workspaces. A database belonging to another workspace SHALL NOT be displayed under the addressed workspace's permissions, even when the account can access both workspaces. Unavailable-resource fallback SHALL NOT disclose whether an inaccessible resource exists or reveal its name or contents.

Read responses from a previous navigation or session SHALL NOT display obsolete details, select a resource, trigger fallback or overwrite the current address. Mutation responses SHALL follow the database action completion requirements: a tab change within a continuously selected database preserves accepted copy selection and removal fallback, while abandoning that database, workspace, main section or session prevents the earlier action from selecting another resource. An accepted list update that removes the current resource SHALL still trigger ordinary availability reconciliation. Navigating to a read-only destination SHALL NOT replay a mutation.

#### Scenario: Both workspaces are accessible but the IDs do not match

- **WHEN** a user opens a link pairing workspace A with a database of workspace B
- **THEN** no database B details or actions appear under workspace A
- **AND** fallback resolves to an available destination in workspace A

#### Scenario: Another navigation completes first

- **WHEN** a user changes the destination while an earlier database or workspace read is pending and that read later succeeds or fails
- **THEN** the current destination and address remain unchanged by the earlier response

#### Scenario: Inaccessible and missing resources are indistinguishable

- **WHEN** a user opens an inaccessible resource link and then an equivalent missing resource link
- **THEN** both follow the same fallback policy without revealing unavailable resource details
