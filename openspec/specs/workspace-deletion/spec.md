# Workspace Deletion Specification

## Purpose

Defines what happens when a workspace is deleted: what goes away with it, when the deletion is refused, and that it never leaves a workspace half deleted.

## Requirements

### Requirement: A workspace with databases cannot be deleted

Deleting a workspace that still has databases SHALL get `409` with `code` `workspace_not_empty` and SHALL delete nothing: the workspace, its members, notifiers and storages SHALL remain.

#### Scenario: Workspace with a database

- **WHEN** the owner of a workspace with one database, one notifier and one storage deletes the workspace
- **THEN** the response is `409` with `code` `workspace_not_empty`, and the workspace, its members, the database, the notifier and the storage are all still listed

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

### Requirement: A workspace whose storage holds another workspace's backups cannot be deleted

Deleting a workspace without databases SHALL get `409` with `code` `in_use` and SHALL delete nothing when one of its storages still holds backups of a database in another workspace. A workspace that still has databases SHALL get `workspace_not_empty` as before, whatever its storages hold. Such backups remain, for example, after a storage moves to another workspace without the database that wrote them, or after a backup config switches to another storage and leaves its logical backups on the old one. The workspace, its members, notifiers and storages SHALL remain, and so SHALL the backups. The response SHALL NOT reveal the internal cause of the refusal.

#### Scenario: Storage with backups of a database in another workspace

- **WHEN** the owner of a workspace without databases deletes it while one of its storages holds a backup of a database in another workspace
- **THEN** the response is `409` with `code` `in_use`, and the workspace, its members, its notifiers and its storages are all still listed
- **AND** a member of the other workspace still sees that backup in the backup list of its database

### Requirement: A workspace is deleted completely or not at all

When any part of a workspace deletion fails, the deletion SHALL be undone as a whole: the workspace, its members, notifiers and storages SHALL remain as they were before the request. The response SHALL NOT reveal the internal cause of the failure.

#### Scenario: Deletion fails after the notifiers and storages were removed

- **WHEN** a workspace deletion removes the workspace's notifiers and storages and then fails in a later step with an internal error
- **THEN** the response is `500` with `code` `internal` and a `detail` that does not contain the text of the internal error, and the workspace, its members, its notifiers and its storages are all still listed

#### Scenario: Content added while the workspace is deleted

- **WHEN** a database is added to a workspace after its deletion checked for databases and before the workspace itself is removed
- **THEN** the deletion answers `409` with `code` `workspace_not_empty`, and the workspace, its members, notifiers, storages and the new database are all still listed
