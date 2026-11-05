## MODIFIED Requirements

### Requirement: Each workspace role allows a fixed set of actions

A member's workspace role SHALL decide what the member may do in the workspace, and the roles SHALL build on each other:

- a viewer MAY see the workspace, its members and its content, MAY send test messages through the workspace's notifiers, MAY test the connection of the workspace's saved storages and saved databases, and MAY start, download and restore the logical backups of the workspace's databases;
- a member MAY also create, change and delete the workspace's databases, storages and notifiers and everything attached to them, MAY cancel and delete logical backups and cancel restores, MAY test storage and database settings that are not saved yet, except where the `storages` spec reserves rclone settings for instance admins, and MAY check and create limited users of the workspace's databases;
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

#### Scenario: Member changes a notifier

- **WHEN** a user with the role `WORKSPACE_MEMBER` creates, changes and deletes a notifier of the workspace
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

### Requirement: A user outside a workspace learns nothing about it

An operation on a workspace, on one of its members, on one of its notifiers, on one of its storages, on one of its databases, on a database's backup config, on its logical backups or on their restores, asked by a user who is not a member and not an instance admin, SHALL get `404` with `code` `not_found`, the same answer as for a workspace, a notifier, a storage, a database, a logical backup or a restore that does not exist. The download of a backup file with a token is outside this rule: it carries no session, and a token that does not fit gets `401` as the `logical-backups` spec states. The response SHALL NOT reveal the workspace's name, its members, its notifiers, its storages, its databases, their backup configs, their backups or their restores.

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

#### Scenario: Reading another team's storage

- **WHEN** a user who is not a member of a workspace reads one of its storages by ID and then a storage ID that does not exist
- **THEN** both responses are `404` with `code` `not_found` and the same body apart from the ID

#### Scenario: Changing and deleting another team's storage

- **WHEN** a user who is not a member of a workspace changes one of its storages and then deletes it
- **THEN** both responses are `404` with `code` `not_found`, and the storage keeps its name and is still listed in its workspace

#### Scenario: Listing another team's storages

- **WHEN** a user who is not a member of a workspace lists the storages of that workspace
- **THEN** the response is `404` with `code` `not_found`

#### Scenario: Transferring a storage into another team's workspace

- **WHEN** a member of a storage's workspace transfers the storage to a workspace they are not a member of, and then to a workspace ID that does not exist
- **THEN** both responses are `404` with `code` `not_found` and the same body apart from the ID, and the storage stays in its workspace

#### Scenario: Reading another team's database

- **WHEN** a user who is not a member of a workspace reads one of its databases by ID and then a database ID that does not exist
- **THEN** both responses are `404` with `code` `not_found` and the same body apart from the ID

#### Scenario: Changing, testing and deleting another team's database

- **WHEN** a user who is not a member of a workspace changes one of its databases, tests its connection and then deletes it
- **THEN** each response is `404` with `code` `not_found`, no connection is attempted, and the database keeps its name and is still listed in its workspace

#### Scenario: Listing another team's databases

- **WHEN** a user who is not a member of a workspace lists the databases of that workspace
- **THEN** the response is `404` with `code` `not_found`

#### Scenario: Reading, saving and listing another team's backup configs

- **WHEN** a user who is not a member of a workspace reads the backup config of one of its databases, saves it and lists the workspace's backup configs
- **THEN** each response is `404` with `code` `not_found`, and the config is unchanged

#### Scenario: Reading another team's backups

- **WHEN** a user who is not a member of a workspace lists the logical backups of one of its databases, reads one backup, asks for its download token and lists its restores
- **THEN** each response is `404` with `code` `not_found`

#### Scenario: Same answer as a missing backup or restore

- **WHEN** a user who is not a member of a workspace reads one of its logical backups and then a backup ID that does not exist, and reads one of its restores and then a restore ID that does not exist
- **THEN** every response is `404` with `code` `not_found`, and each pair has the same body apart from the ID

#### Scenario: Changing another team's backups and restores

- **WHEN** a user who is not a member of a workspace starts a backup of one of its databases, cancels a running backup, deletes a completed backup, starts a restore of it and cancels a running restore
- **THEN** each response is `404` with `code` `not_found`
- **AND** no backup or restore is started, the running backup and restore keep running, and the completed backup is still listed

### Requirement: A role too low for the action is refused without a change

An operation on a workspace, on one of its members, on one of its notifiers, on one of its storages, on one of its databases, on a database's backup config, on its logical backups or on their restores, asked by a member whose role does not allow it, SHALL get `403` with `code` `permission_denied` and SHALL change nothing.

#### Scenario: Viewer adds a member

- **WHEN** a viewer of a workspace adds another user as a member
- **THEN** the response is `403` with `code` `permission_denied`, and the user does not become a member

#### Scenario: Viewer changes a notifier

- **WHEN** a viewer of a workspace renames one of its notifiers
- **THEN** the response is `403` with `code` `permission_denied`, and the notifier keeps its name

#### Scenario: Viewer changes a storage

- **WHEN** a viewer of a workspace renames one of its storages and then deletes it
- **THEN** both responses are `403` with `code` `permission_denied`, and the storage keeps its name and is still listed

#### Scenario: Viewer changes a database

- **WHEN** a viewer of a workspace renames one of its databases, copies it and then deletes it
- **THEN** each response is `403` with `code` `permission_denied`, and the database keeps its name, has no copy and is still listed

#### Scenario: Viewer saves a backup config and moves a database

- **WHEN** a viewer of a workspace saves the backup config of one of its databases and then moves the database to a workspace where the viewer is a member
- **THEN** both responses are `403` with `code` `permission_denied`, the config is unchanged, and the database stays in its workspace

#### Scenario: Viewer cancels and deletes a backup and cancels a restore

- **WHEN** a viewer of a workspace cancels a running backup of one of its databases, deletes a completed backup and cancels a running restore
- **THEN** each response is `403` with `code` `permission_denied`, the running backup is still `IN_PROGRESS`, the completed backup is still listed, and the restore keeps running
