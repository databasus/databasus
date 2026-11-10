## Purpose

Lets the instance owner preserve Databasus state in existing workspace storages and see whether each destination has a usable encrypted copy.

## ADDED Requirements

### Requirement: Databasus backups preserve an internally consistent instance snapshot

Databasus SHALL back up its complete internal database, including users, workspace membership, encrypted credentials, settings, schedules and the database-backup catalog. The archive SHALL exclude the master key, database-backup payloads, local WAL queues, logs and other host files. Each archive SHALL identify its source release, snapshot time and format, contain everything needed to interpret its encryption, and be recoverable using that file and the separately saved key without the original instance.

The Databasus backup file SHALL contain an authenticated terminal record identifying the completed dump's byte length and digest. Recovery SHALL require that record, matching dump bytes and end of file with no additional records or bytes. A missing terminal record, including a file truncated exactly between encrypted chunks, SHALL be refused before import. This requirement SHALL apply to the new Databasus backup container without changing existing user-database backup formats.

#### Scenario: Server is lost after a completed backup

- **WHEN** the original server is unavailable and the owner retains a completed archive and its key
- **THEN** those inputs suffice to recover the instance snapshot on the exact source release
- **AND** database-backup files remain in their original storages rather than inside the archive

#### Scenario: Databasus backup file is cut between encrypted chunks

- **WHEN** a Databasus backup file is truncated on an encrypted-chunk boundary before its authenticated terminal record
- **THEN** recovery refuses it as incomplete before importing data or replacing the target key

#### Scenario: Data follows the authenticated terminal record

- **WHEN** bytes or additional records follow the terminal record of a Databasus backup file
- **THEN** recovery refuses the file before importing it

### Requirement: Archives and their access are protected

Every archive SHALL be encrypted with the instance's existing master key before it reaches persistent temporary storage or a destination. Archive configuration, history, start and download operations SHALL require the recorded root administrator; another instance administrator SHALL NOT inherit this access. Responses, diagnostics and messages SHALL exclude key material and credentials. A downloaded archive SHALL have the same encryption and recovery information as its stored copy.

#### Scenario: Another administrator requests an archive

- **WHEN** an authenticated instance administrator who is not root admin requests a Databasus backup archive or its history
- **THEN** the request is refused with 403 and no archive or instance-backup details are returned

#### Scenario: Archive is downloaded for recovery

- **WHEN** root admin downloads a successfully stored copy
- **THEN** the downloaded file remains encrypted and contains the information required for recovery without a separate metadata file

### Requirement: Destinations are live references to existing workspace storages

Root admin SHALL select zero or more distinct existing storages from any workspace. Selection SHALL store a reference to the existing storage, not a copy of its provider settings or credentials. Normal storage edit and transfer permissions SHALL remain unchanged. A later backup SHALL use the storage's current settings. The interface SHALL explain that editing shared storage affects these backups and that an owner can restrict workspace membership using existing controls. Unknown or duplicate destination identifiers SHALL be refused without a partial configuration change.

#### Scenario: Workspace editor changes a selected destination

- **WHEN** an authorized workspace editor changes the connection of a selected storage
- **THEN** the change succeeds under the ordinary rules and the next Databasus backup uses the changed connection

#### Scenario: Selected storage moves between workspaces

- **WHEN** an otherwise permitted transfer moves a selected storage to another workspace
- **THEN** it remains selected by the same identity and the root-admin view reports its current workspace

### Requirement: Scheduling and delivery have independent outcomes per destination

The initial shared schedule SHALL be hourly and retention SHALL default to 30 days. Root admin SHALL be able to change both. Adding the first destination after having none SHALL request an immediate backup. A manual request SHALL return an in-progress operation without waiting for completion. Concurrent starts SHALL NOT create overlapping dumps; a duplicate manual start SHALL get 409 with code `already_running`. With no destinations, a manual start SHALL get 400 with code `invalid_request`.

One completed encrypted dump SHALL be delivered independently to all destinations selected for that run. A failure at one destination SHALL NOT invalidate a copy already completed elsewhere. Transient delivery failures SHALL be retried with bounded backoff. Removing a destination SHALL stop new attempts for that selection and SHALL NOT delete previously saved archives. A removed and re-added destination SHALL NOT accept late publication from its previous selection.

#### Scenario: One of two destinations is unavailable

- **WHEN** the dump succeeds and delivery succeeds to one selected storage but fails to the other
- **THEN** history reports one successful copy out of two, keeps the successful copy available and retries the unsuccessful delivery independently

#### Scenario: No destination was selected and one is added

- **WHEN** root admin saves the first destination
- **THEN** an immediate backup is requested and the state remains initializing until a copy succeeds

#### Scenario: A destination is removed during delivery

- **WHEN** root admin deselects a storage while a delivery to it is in progress
- **THEN** a late completion cannot restore the selection or report that destination as currently protected

### Requirement: Retention and restart preserve usable copies and truthful history

History SHALL report source version, snapshot time, overall result and each destination's last successful copy and sanitized failure. Interrupted operations SHALL cease to appear running after restart. Retention SHALL delete only expired Databasus backup objects owned by this instance, independently per destination, and SHALL keep its newest successful copy until a newer successful copy exists there. It SHALL NOT delete users' database backups. Removing a selection SHALL leave existing archives untouched; explicitly deleting a subsequently unused storage SHALL leave historical entries visibly unavailable rather than claiming their copies remain accessible.

#### Scenario: A failed run follows an older successful copy

- **WHEN** a destination's newest successful copy is older than the retention period and newer deliveries fail
- **THEN** retention keeps that last successful copy and the interface continues to show the failure and age

#### Scenario: Databasus stops during an upload

- **WHEN** the application restarts after an incomplete upload
- **THEN** that upload is not shown as successful and its unfinished objects remain eligible for managed cleanup
