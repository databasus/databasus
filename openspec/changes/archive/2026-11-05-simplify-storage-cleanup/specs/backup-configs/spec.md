## MODIFIED Requirements

### Requirement: A database moves to another workspace in one step

A member SHALL be able to move a database to another workspace. The request SHALL name the target workspace and choose one of two storage options: move the database's storage with it, or switch the database to a storage of the target workspace. It MAY also ask to move the database's notifiers with it, and MAY name notifiers of the target workspace that replace the database's notifiers. The workspace move, config change, notifier changes, removal of old backups when switching, and recording their deletion obligations SHALL commit completely or not at all. Failure before commit SHALL leave the database, config, storages, notifiers and backups unchanged.

- Moving the storage with the database SHALL be refused with 409 and code in_use when another database backs up into it, and with 400 and code invalid_request when the database has no storage. A successful move with the same storage SHALL answer 204 and preserve its backups.
- A target storage or replacing notifier outside the target workspace SHALL be refused with 400 and code invalid_request.
- Moving the notifiers SHALL move each notifier that no other database uses. A notifier another database uses SHALL stay in its workspace.
- Switching to a target storage SHALL reset the backup history as specified by backup-storage-assignment and answer 204 after commit. Remote deletion SHALL NOT delay or undo the workspace move.
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
- **THEN** the response is 204 and the database appears in the target workspace with its default config naming T
- **AND** the target workspace's config list includes it

#### Scenario: Switching removes the old backup history

- **WHEN** a member transfers a database from a workspace using A to another workspace using B
- **THEN** the response is 204, the database and config belong to the target workspace and its old backups are absent from the list
- **AND** the old files are queued for deletion from A, even when A is unavailable

#### Scenario: Transfer assigns storage after the previous storage was cleared

- **WHEN** a database whose config names no storage still has backups in A and is transferred with target storage B
- **THEN** the workspace move, assignment of B, removal of the old history and deletion requests against A commit together, and the response is 204
- **AND** enabled physical backups start a new FULL in B and expected WAL streaming restarts for B

#### Scenario: Notifier replacement fails during a storage switch

- **WHEN** a transfer that switches storage names a replacing notifier that does not exist
- **THEN** the response is 400 with code invalid_request
- **AND** the original workspace, config, backup catalog and deletion obligations are unchanged

## ADDED Requirements

### Requirement: A backup configuration exposes current file cleanup

Every successful configuration read and configuration-save acknowledgement SHALL include read-only `cleanup` with `pendingFileCount` and `failedFileCount`, including defaults for a never-saved config and items returned in configuration lists. The counts SHALL describe current authorized work, with nonnegative values and failed files a subset of pending files. Delayed and claimed obligations SHALL remain pending until confirmed absent or explicitly abandoned. A zero summary SHALL not persist a configuration or create operation history.

Ordinary protected uploads SHALL NOT be presented as cleanup caused by a storage change. Configuration input SHALL NOT let callers persist or override computed cleanup counts. A successful configuration save SHALL return 200 with the acknowledged config even when it queues file deletion. A successful workspace transfer SHALL return 204. Neither operation SHALL return a switch identity or history location.

#### Scenario: Configuration save starts cleanup

- **WHEN** changing storage removes old backup history and queues its files
- **THEN** the response is 200 with the saved config and its current outstanding counts

#### Scenario: Configurations are listed

- **WHEN** an authorized viewer lists configurations
- **THEN** every returned config includes the same current-scope summary available through its individual read

#### Scenario: Never-saved config is read

- **WHEN** a database has no saved configuration and no attributed cleanup
- **THEN** its default config includes zero cleanup counts without creating a saved configuration

#### Scenario: A normal backup upload is protected

- **WHEN** an upload is still eligible to be published and has not been taken for storage-change deletion
- **THEN** that registration alone does not increase the database's storage-change cleanup counts

#### Scenario: A claimed deletion has not completed

- **WHEN** a worker has claimed an obligation but has not confirmed absence
- **THEN** the file remains in the pending count

#### Scenario: Input includes computed counts

- **WHEN** a configuration request attempts to supply cleanup counts
- **THEN** caller-supplied values do not change stored cleanup or the computed summary

#### Scenario: Storage deletion abandons work

- **WHEN** a user explicitly deletes an unused source storage with outstanding work
- **THEN** a subsequent configuration read omits that abandoned work from its current counts without exposing a completed operation
