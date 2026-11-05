## MODIFIED Requirements

### Requirement: Deleting a workspace removes its notifiers, storages and members

Deleting a workspace without databases SHALL remove the workspace, its notifiers, its storages and its members, unless one of its storages still holds backups of a database in another workspace, as the requirement "A workspace whose storage holds another workspace's backups cannot be deleted" describes. The users who were its members SHALL keep their accounts and stay members of their other workspaces.

#### Scenario: Workspace with a notifier and a storage

- **WHEN** the owner of a workspace with no databases, one notifier and one storage deletes the workspace
- **THEN** the workspace is no longer listed, and its notifier and storage no longer exist

#### Scenario: Workspace whose storage has pending cleanup

- **WHEN** the owner of a workspace with no databases deletes it while its storage holds a file with a pending deletion
- **THEN** the deletion succeeds, the file is removed from the provider, and the storage no longer exists

#### Scenario: Members of a deleted workspace

- **WHEN** a workspace whose members also belong to another workspace is deleted
- **THEN** each of those users can still sign in and still sees the other workspace

## ADDED Requirements

### Requirement: A workspace whose storage holds another workspace's backups cannot be deleted

Deleting a workspace without databases SHALL get `409` with `code` `in_use` and SHALL delete nothing when one of its storages still holds backups of a database in another workspace. A workspace that still has databases SHALL get `workspace_not_empty` as before, whatever its storages hold. Such backups remain, for example, after a storage moves to another workspace without the database that wrote them, or after a backup config switches to another storage and leaves its logical backups on the old one. The workspace, its members, notifiers and storages SHALL remain, and so SHALL the backups. The response SHALL NOT reveal the internal cause of the refusal.

#### Scenario: Storage with backups of a database in another workspace

- **WHEN** the owner of a workspace without databases deletes it while one of its storages holds a backup of a database in another workspace
- **THEN** the response is `409` with `code` `in_use`, and the workspace, its members, its notifiers and its storages are all still listed
- **AND** a member of the other workspace still sees that backup in the backup list of its database
