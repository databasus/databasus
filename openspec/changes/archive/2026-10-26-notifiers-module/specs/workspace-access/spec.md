## MODIFIED Requirements

### Requirement: Each workspace role allows a fixed set of actions

A member's workspace role SHALL decide what the member may do in the workspace, and the roles SHALL build on each other:

- a viewer MAY see the workspace, its members and its content, and MAY send test messages through the workspace's notifiers;
- a member MAY also create, change and delete the workspace's databases, storages and notifiers and everything attached to them;
- an admin MAY also rename the workspace and add, change and remove members who are not admins or the owner;
- the owner MAY also add, change and remove admins, transfer ownership and delete the workspace.

The operations on workspaces, their members and their notifiers SHALL apply these rules with the answers of the requirements below. The operations on a workspace's databases and storages SHALL apply the same rules, and SHALL keep their current error answers until those parts of the product move behind the API contract.

#### Scenario: Viewer reads the workspace

- **WHEN** a viewer of a workspace reads it and lists its members
- **THEN** both requests succeed

#### Scenario: Viewer renames the workspace

- **WHEN** a viewer of a workspace renames it
- **THEN** the response is `403` with `code` `permission_denied`, and the name does not change

#### Scenario: Member renames the workspace

- **WHEN** a user with the role `WORKSPACE_MEMBER` renames the workspace
- **THEN** the response is `403` with `code` `permission_denied`, and the name does not change

#### Scenario: Member adds a member

- **WHEN** a user with the role `WORKSPACE_MEMBER` adds another user to the workspace
- **THEN** the response is `403` with `code` `permission_denied`, and the user does not become a member

#### Scenario: Member and viewer on the workspace's content

- **WHEN** the content operations ask whether a user with the role `WORKSPACE_MEMBER` and a viewer of the workspace may edit its content
- **THEN** the member may and the viewer may not

#### Scenario: Viewer reads the workspace's notifiers

- **WHEN** a viewer of a workspace lists its notifiers and reads one of them
- **THEN** both requests succeed

#### Scenario: Viewer tests a notifier

- **WHEN** a viewer of a workspace tests one of its saved notifiers
- **THEN** the request succeeds and reports the outcome of the test send

#### Scenario: Member changes a notifier

- **WHEN** a user with the role `WORKSPACE_MEMBER` creates, changes and deletes a notifier of the workspace
- **THEN** each request succeeds

#### Scenario: Admin renames the workspace

- **WHEN** an admin of a workspace renames it
- **THEN** the workspace has the new name

#### Scenario: Admin deletes the workspace

- **WHEN** an admin of a workspace who is not its owner deletes it
- **THEN** the response is `403` with `code` `permission_denied`, and the workspace remains

### Requirement: A user outside a workspace learns nothing about it

An operation on a workspace, on one of its members or on one of its notifiers, asked by a user who is not a member and not an instance admin, SHALL get `404` with `code` `not_found`, the same answer as for a workspace or a notifier that does not exist. The response SHALL NOT reveal the workspace's name, its members or its notifiers.

#### Scenario: Reading another team's workspace

- **WHEN** a user who is not a member of a workspace reads it by its ID
- **THEN** the response is `404` with `code` `not_found`

#### Scenario: Same answer as a missing workspace

- **WHEN** a user who is not a member reads an existing workspace and then a workspace ID that does not exist
- **THEN** both responses are `404` with `code` `not_found` and the same body apart from the ID

#### Scenario: Changing a member of another team's workspace

- **WHEN** a user who is not a member of a workspace changes the role of one of its members by the member's ID
- **THEN** the response is `404` with `code` `not_found`, and the role does not change

#### Scenario: Reading another team's notifier

- **WHEN** a user who is not a member of a workspace reads one of its notifiers by ID and then a notifier ID that does not exist
- **THEN** both responses are `404` with `code` `not_found` and the same body apart from the ID

#### Scenario: Listing another team's notifiers

- **WHEN** a user who is not a member of a workspace lists the notifiers of that workspace
- **THEN** the response is `404` with `code` `not_found`

### Requirement: A role too low for the action is refused without a change

An operation on a workspace, on one of its members or on one of its notifiers, asked by a member whose role does not allow it, SHALL get `403` with `code` `permission_denied` and SHALL change nothing.

#### Scenario: Viewer adds a member

- **WHEN** a viewer of a workspace adds another user as a member
- **THEN** the response is `403` with `code` `permission_denied`, and the user does not become a member

#### Scenario: Viewer changes a notifier

- **WHEN** a viewer of a workspace renames one of its notifiers
- **THEN** the response is `403` with `code` `permission_denied`, and the notifier keeps its name
