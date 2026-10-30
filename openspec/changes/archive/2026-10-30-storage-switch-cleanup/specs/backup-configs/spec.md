## MODIFIED Requirements

### Requirement: A database moves to another workspace in one step

A member SHALL be able to move a database to another workspace. The request SHALL name the target workspace and choose one of two storage options: move the database's storage with it, or switch the database to a storage of the target workspace. It MAY also ask to move the database's notifiers with it, and MAY name notifiers of the target workspace that replace the database's notifiers. The workspace move, config change, notifier changes, removal of old backups when switching, and recording their deletion obligations SHALL commit completely or not at all. Failure before commit SHALL leave the database, config, storages, notifiers and backups unchanged.

- Moving the storage with the database SHALL be refused with 409 and code in_use when another database backs up into it, and with 400 and code invalid_request when the database has no storage. A successful move with the same storage SHALL answer 204 and preserve its backups.
- A target storage or replacing notifier outside the target workspace SHALL be refused with 400 and code invalid_request.
- Moving the notifiers SHALL move each notifier that no other database uses. A notifier another database uses SHALL stay in its workspace.
- Switching to a target storage SHALL reset the backup history as specified by backup-storage-assignment and answer 202 with a cleanup operation. Remote deletion SHALL NOT delay or undo the workspace move.
- The same reset and cleanup SHALL apply when the source config names no storage but the database retains backups from an earlier assignment; their actual storages SHALL determine where files are deleted.
- A database whose config was never saved SHALL get the default config of its method with the target storage saved.

#### Scenario: Moving with the storage

- **WHEN** a member of both workspaces moves a database with its storage, which no other database uses
- **THEN** the response is 204 and the database and storage are listed in the target workspace
- **AND** the config still names that storage and its backups are unchanged

#### Scenario: A later step fails

- **WHEN** a member moves a database with its storage and names a replacing notifier that does not exist
- **THEN** the response is 400 with code invalid_request and the database and its storage remain in the source workspace

#### Scenario: The storage is shared

- **WHEN** a member moves a database with its storage and another database backs up into that storage
- **THEN** the response is 409 with code in_use and nothing moves

#### Scenario: Switching an unsaved config to a target storage

- **WHEN** a member moves a database whose config was never saved and switches it to storage T of the target workspace
- **THEN** the response is 202 with an operation and the database appears in the target workspace with its default config naming T
- **AND** the target workspace's config list includes it

#### Scenario: Switching removes the old backup history

- **WHEN** a member transfers a database from a workspace using A to another workspace using B
- **THEN** the response is 202, the database and config belong to the target workspace and its old backups are absent from the list
- **AND** the old files are queued for deletion from A, even when A is unavailable

#### Scenario: Transfer assigns storage after the previous storage was cleared

- **WHEN** a database whose config names no storage still has backups in A and is transferred with target storage B
- **THEN** the workspace move, assignment of B, removal of the old history and deletion requests against A commit together, and the response is 202 with a cleanup operation
- **AND** enabled physical backups start a new FULL in B and expected WAL streaming restarts for B

#### Scenario: Notifier replacement fails during a storage switch

- **WHEN** a transfer that switches storage names a replacing notifier that does not exist
- **THEN** the response is 400 with code invalid_request
- **AND** the original workspace, config, backup catalog and deletion obligations are unchanged
