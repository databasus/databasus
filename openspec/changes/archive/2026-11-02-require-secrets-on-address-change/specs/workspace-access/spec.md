## MODIFIED Requirements

### Requirement: Each workspace role allows a fixed set of actions

A member's workspace role SHALL decide what the member may do in the workspace, and the roles SHALL build on each other:

- a viewer MAY see the workspace, its members and its content, MAY send test messages through the workspace's saved notifiers, MAY test the connection of the workspace's saved storages and saved databases, and MAY start, download and restore the logical backups of the workspace's databases;
- a member MAY also create, change and delete the workspace's databases, storages and notifiers and everything attached to them, MAY cancel and delete logical backups and cancel restores, MAY test notifier, storage and database settings that are not saved yet, except where the `storages` spec reserves rclone settings for instance admins, and MAY check and create limited users of the workspace's databases;
- an admin MAY also rename the workspace and add, change and remove members who are not admins or the owner;
- the owner MAY also add, change and remove admins, transfer ownership and delete the workspace.

The operations on workspaces, their members, their notifiers, their storages, their databases, their databases' backup configs, logical backups and restores SHALL apply these rules with the answers of the requirements below. Moving a database to another workspace SHALL need the right to edit content in both workspaces.

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

#### Scenario: Viewer tests unsaved notifier settings

- **WHEN** a viewer of a workspace tests notifier settings that are not saved
- **THEN** the response is `403` with `code` `permission_denied`, and no message leaves the backend

#### Scenario: Member changes a notifier

- **WHEN** a user with the role `WORKSPACE_MEMBER` creates, changes, tests unsaved settings of and deletes a notifier of the workspace
- **THEN** each request succeeds

#### Scenario: Viewer reads and tests the workspace's storages

- **WHEN** a viewer of a workspace lists its storages, reads one of them and tests its connection
- **THEN** the list and the read succeed, and the test answers `200` and reports the outcome of the connection test

#### Scenario: Member changes a storage

- **WHEN** a user with the role `WORKSPACE_MEMBER` creates, changes, tests unsaved settings of and deletes a storage of the workspace
- **THEN** each request succeeds

#### Scenario: Viewer reads and tests the workspace's databases

- **WHEN** a viewer of a workspace lists its databases, reads one of them and tests its connection
- **THEN** the list and the read succeed, and the test answers `200` and reports the outcome of the connection test

#### Scenario: Member changes a database

- **WHEN** a user with the role `WORKSPACE_MEMBER` creates, changes, copies, tests unsaved settings of and deletes a database of the workspace
- **THEN** each request succeeds

#### Scenario: Member checks and creates a limited user

- **WHEN** a user with the role `WORKSPACE_MEMBER` checks whether a limited user is advisable for a database of the workspace and then creates one
- **THEN** both requests succeed

#### Scenario: Viewer reads the backup configs

- **WHEN** a viewer of a workspace lists its backup configs and reads the config of one of its databases
- **THEN** both requests succeed

#### Scenario: Member saves a backup config and moves a database

- **WHEN** a user with the role `WORKSPACE_MEMBER` in two workspaces saves the backup config of a database of the first and then moves the database to the second
- **THEN** both requests succeed

#### Scenario: Viewer backs up, downloads and restores

- **WHEN** a viewer of a workspace starts a backup of one of its databases, gets a download token for a completed backup and starts a restore of it
- **THEN** each request succeeds

#### Scenario: Member cancels and deletes a backup

- **WHEN** a user with the role `WORKSPACE_MEMBER` cancels a running backup of a database of the workspace and deletes a completed one
- **THEN** both requests succeed

#### Scenario: Admin renames the workspace

- **WHEN** an admin of a workspace renames it
- **THEN** the workspace has the new name

#### Scenario: Admin deletes the workspace

- **WHEN** an admin of a workspace who is not its owner deletes it
- **THEN** the response is `403` with `code` `permission_denied`, and the workspace remains
