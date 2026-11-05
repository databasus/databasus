# Backup Configs Specification

## Purpose

Defines the backup config of a database: what it holds for each backup method, its defaults, how it is read, saved, listed, copied and deleted, and how a database moves to another workspace together with its storage and notifiers.

## Requirements

### Requirement: A database has one backup config of its method

Every database SHALL have exactly one backup config, of the backup method its type implies: a logical config for a PostgreSQL, MySQL, MariaDB or MongoDB database, and a physical config for a physical PostgreSQL database. Both methods SHALL hold whether backups run, the storage, the encryption and the notification triggers, the backup outcomes that send a notification. A logical config SHALL also hold one schedule, a retention policy (a time period, a number of backups, or grandfather-father-son) and whether a failed backup is retried and how many times. A physical config SHALL also hold the schedule of full backups, the schedule of incremental backups, the chain and full-backup retention, and the WAL lag threshold. Saving a config of the other method SHALL get `400` with `code` `invalid_request`.

#### Scenario: A logical database is saved with a physical config

- **WHEN** a member saves a physical config for a MySQL database
- **THEN** the response is `400` with `code` `invalid_request`, and the database's config is unchanged

### Requirement: A config that was never saved reads as the default

Reading the config of a database that has never had one saved SHALL return the default config of its method and SHALL store nothing. The logical default SHALL have backups turned off, run daily at 04:00, keep backups for three months, retry a failed backup up to three times, notify on a failed and on a successful backup, use no encryption and name no storage. The physical default SHALL have backups turned off, run a full backup daily at 04:00, keep the last seven full backups, notify on every notification trigger of the method, use no encryption and name no storage, adjusted to the database's physical backup type as a type change adjusts it. The list of configs SHALL NOT include a database whose config was never saved.

#### Scenario: Reading a new database's config

- **WHEN** a member creates a PostgreSQL database and reads its config
- **THEN** the response is `200` with the logical default, and listing the workspace's configs does not include the database

#### Scenario: The first save stores the config

- **WHEN** a member saves the config of that database with backups turned on
- **THEN** reading the config returns the saved values, and the workspace's list includes it

### Requirement: An invalid config is refused without a change

Saving a config SHALL check the schedule, the retention and every rule of the method, and SHALL get `400` with `code` `invalid_request` and the offending fields when any of them fails. A physical config SHALL fit the database's physical backup type: a type without incremental backups takes no incremental schedule, no WAL lag threshold and only full-backup retention, and a type with incremental backups needs an incremental schedule that runs more often than the full one. A refused save SHALL change nothing.

#### Scenario: A retention count of zero

- **WHEN** a member saves a logical config with the retention policy `COUNT` and a count of `0`
- **THEN** the response is `400` with `code` `invalid_request` that names the retention count, and the stored config is unchanged

#### Scenario: Incremental schedule slower than the full one

- **WHEN** a member saves a physical config of a database with incremental backups whose incremental schedule is weekly and whose full schedule is daily
- **THEN** the response is `400` with `code` `invalid_request`

### Requirement: A config follows its database's physical backup type

When the physical backup type of a database changes, its saved physical config SHALL be adjusted in the same step to fit the new type, keeping every value the new type allows. When the adjusted config still fails validation, such as a stored full schedule with a cron expression that has no date to run on, the change SHALL get `400` with `code` `invalid_request` and SHALL change neither the database nor its config. When WAL streaming is no longer expected, the WAL streamer of the database SHALL be stopped and removed after the change is saved.

An earlier release could leave a saved physical config that does not fit its database's type, because its type switch did not adjust the config. Reading such a config, alone or in a list, SHALL return it adjusted to the type and SHALL store nothing; the next save stores the adjusted values.

#### Scenario: Switching to full backups only

- **WHEN** a physical database with incremental backups and a weekly incremental schedule is switched to full backups only
- **THEN** its config has no incremental schedule and keeps its full schedule, storage and notifications

#### Scenario: Switching off WAL streaming

- **WHEN** a physical database whose WAL is streamed is switched to full and incremental backups without WAL streaming
- **THEN** after the switch is saved, the database has no WAL streamer

#### Scenario: A config left unfit by an earlier release

- **WHEN** an earlier release left a saved physical config with an incremental schedule while its database takes full backups only, and a member reads the config and lists the workspace's configs
- **THEN** both responses hold the config without an incremental schedule
- **AND** the stored config keeps its incremental schedule until the next save, which stores it without one

#### Scenario: Changing the backup type with a stored full expression that has no date

- **WHEN** an earlier release stored the cron expression `0 0 31 2 *` as the full schedule of a physical config and a member changes the database's physical backup type
- **THEN** the response is `400` with `code` `invalid_request`, and the database and its stored config are unchanged

### Requirement: Configs are listed per workspace or per storage

Listing configs SHALL take exactly one filter, a workspace or a storage, and SHALL return the saved configs of that workspace's databases or of the databases that back up into that storage, ordered by database ID, in pages with a continuation cursor. A storage filter SHALL be answered as the storage's workspace is: `404` with `code` `not_found` for a user who is not a member of it. A request with both filters or neither SHALL get `400` with `code` `invalid_request`.

#### Scenario: The database list reads its configs once

- **WHEN** a member lists the configs of a workspace with three databases whose configs are saved
- **THEN** one response holds the three configs, each with its database ID and storage ID

#### Scenario: Is a storage in use

- **WHEN** a member lists the configs that use a storage, with a limit of one, and one database backs up into it
- **THEN** the response holds that database's config

### Requirement: A copy of a database gets the copy of its logical config

Copying a logical database whose config was saved SHALL save a copy of that config for the copy, in the copy's step, with the same values and storage. A copy of a physical database SHALL start without a saved config and read as the default.

#### Scenario: Copying a database with enabled backups

- **WHEN** a member copies a MySQL database whose saved config runs backups hourly into storage A
- **THEN** the copy's config runs backups hourly into storage A

### Requirement: Deleting a database deletes its config

Deleting a database SHALL delete its saved config in the same step. When the deletion is refused or fails, the config SHALL stay.

#### Scenario: A refused deletion keeps the config

- **WHEN** a member deletes a database whose config was saved while a logical backup of it is running, and the deletion is refused
- **THEN** listing the workspace's configs still includes the config

#### Scenario: A deleted database leaves no config

- **WHEN** a member deletes a database whose config was saved
- **THEN** listing the workspace's configs no longer includes it

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

### Requirement: Only a member of both workspaces moves a database

Moving a database SHALL require the right to edit content in the database's workspace and in the target workspace. A user who is not a member of the database's workspace or of the target workspace SHALL get `404` with `code` `not_found`, the same answer as for a workspace or database that does not exist. A member whose role in either workspace does not allow editing content SHALL get `403` with `code` `permission_denied`. Neither refusal SHALL move anything. This is a security rule: a move gives the target workspace's members the database's stored secrets.

#### Scenario: Viewer of the target workspace

- **WHEN** a member of the database's workspace who is a viewer of the target workspace moves the database there
- **THEN** the response is `403` with `code` `permission_denied`, and the database stays in its workspace

#### Scenario: Not a member of the target workspace

- **WHEN** a member of the database's workspace moves the database to a workspace they are not a member of, and then to a workspace ID that does not exist
- **THEN** both responses are `404` with `code` `not_found` and the same body apart from the ID

### Requirement: An upgrade keeps every backup config

An upgrade SHALL keep every saved config with its values, its storage and its notification triggers. A database that has a saved config of each method SHALL keep the one of its own method, and the other SHALL be deleted. A stored application-owned value that the current application cannot use SHALL be replaced by the default of its method during the upgrade, so the instance starts with a usable config. Provider-defined storage classes SHALL be preserved as the storage requirements state.

#### Scenario: A database with both configs

- **WHEN** an instance with a MySQL database that has both a logical and a physical config row is upgraded
- **THEN** the database's config is its logical config with its saved values

#### Scenario: A value the new constraints reject

- **WHEN** an instance with a logical config whose retention period is an unknown value is upgraded
- **THEN** the config reads with the default retention period of the logical method and keeps every other saved value
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
