# backup-storage-assignment Specification

## Purpose
Defines which storage a restore reads a backup from, what happens to existing backups when a backup configuration moves to another storage, and that saving a backup configuration never changes the storages themselves.

## Requirements

### Requirement: A restore reads the backup from the storage it was written to

A restore SHALL read the backup from the storage recorded with that backup when it was written. The storage that the database's backup configuration names at the time of the restore SHALL NOT decide where the restore reads from.

#### Scenario: The configuration names another storage

- **WHEN** a user restores a logical backup written to storage A, and the database's backup configuration now names storage B
- **THEN** the restore reads the backup from storage A and completes

#### Scenario: The configuration names no storage

- **WHEN** a user restores a logical backup, and the database's backup configuration names no storage
- **THEN** the restore reads the backup from the storage recorded with it and completes

### Requirement: Moving a physical backup configuration to another storage deletes its physical backups

When a user saves a physical backup configuration with a storage different from the one it had, the database's full and incremental backups that are kept in another storage SHALL be removed from the backup list, together with the WAL segments and timeline history files that only those backups need. Their files SHALL be requested for deletion from those storages. WAL that a full backup kept in the new storage needs SHALL be kept, wherever it is stored, so the WAL retention rules hold. A WAL segment that no remaining backup needs SHALL be removed no later than the next run of the orphan WAL cleanup. A physical backup that is running on the old storage SHALL be stopped. WAL streaming, when the configuration expects it, SHALL continue into the new storage. The next full backup SHALL be written to the new storage.

When the removal fails, the save SHALL report the error. The configuration SHALL keep the new storage, and the next save of the configuration SHALL retry the removal of the backups that are still kept in another storage.

#### Scenario: Chain on the old storage

- **WHEN** a database has a full backup, incremental backups and WAL segments in storage A, and a user saves its physical backup configuration with storage B
- **THEN** the backup list of the database shows none of these backups
- **AND** deletion of their files from storage A is requested

#### Scenario: Retry after a failed removal

- **WHEN** the removal of the old backups fails during the save, and the user saves the configuration again without changing it
- **THEN** the backups still kept in the old storage are removed

#### Scenario: Full backup running during the change

- **WHEN** a full backup of the database is running on storage A, and a user saves the configuration with storage B
- **THEN** the running backup stops and does not appear in the backup list

#### Scenario: WAL streaming continues on the new storage

- **WHEN** the configuration expects WAL streaming, and a user saves it with storage B
- **THEN** WAL segments written after the change are kept in storage B

#### Scenario: The storage does not change

- **WHEN** a user saves a physical backup configuration with the storage it already had
- **THEN** no backup of the database is removed

### Requirement: Saving a backup configuration never creates or changes a storage

Saving a logical or a physical backup configuration SHALL NOT create a storage and SHALL NOT change any field of an existing storage. A configuration SHALL refer to a storage only by the identifier of an existing storage in the database's workspace, or name no storage. Any storage fields the request carries beside that identifier SHALL be ignored. A storage SHALL change only through the storage endpoints and their access checks. This is a security measure: the storage endpoints are the only place that checks who may create or edit a storage.

A storage identifier that no storage has, or that names a storage of another workspace, SHALL get `400` with `code` `invalid_request`, and the configuration SHALL stay unchanged. The check SHALL hold against a transfer of the storage that runs at the same time: either the configuration is saved and the transfer is refused with `409` and `code` `in_use`, or the transfer completes and the save gets `400` with `code` `invalid_request`, leaving the configuration with its previous storage.

#### Scenario: The request names an existing storage

- **WHEN** a user saves a backup configuration with the identifier of a storage of the database's workspace
- **THEN** the save succeeds and the configuration names that storage
- **AND** the stored storage keeps its name and credentials

#### Scenario: The request carries changed storage fields

- **WHEN** a user saves a backup configuration whose body carries, beside the identifier of an existing storage, that storage's name and credentials with different values
- **THEN** the response is `200`, and the configuration names that storage
- **AND** the stored storage keeps its name and credentials

#### Scenario: The request carries a storage without an identifier

- **WHEN** a user saves a backup configuration with a storage object in place of a storage identifier
- **THEN** the configuration names no storage
- **AND** no new storage exists after the request

#### Scenario: The request carries an unknown storage

- **WHEN** a user saves a backup configuration with a storage identifier that no storage has
- **THEN** the response is `400` with `code` `invalid_request`
- **AND** no storage with that identifier exists after the request

#### Scenario: The request names a storage of another workspace

- **WHEN** a user saves a backup configuration with the identifier of a storage of another workspace
- **THEN** the response is `400` with `code` `invalid_request`, and the configuration keeps its storage

#### Scenario: A save races a transfer of its storage

- **WHEN** a user saves a backup configuration naming storage S while another request transfers S to another workspace
- **THEN** exactly one request succeeds: either the save answers `200`, the transfer answers `409` with `code` `in_use` and S stays in the database's workspace, or the transfer answers `204`, the save answers `400` with `code` `invalid_request` and the configuration keeps its previous storage

### Requirement: Moving a logical backup configuration to another storage deletes its logical backups

When a user saves a logical backup configuration with a storage different from the one it had, the database's logical backups SHALL be removed from the backup list and their files SHALL be requested for deletion, in the same step that saves the configuration. When a logical backup of the database is running, the save SHALL get `409` with `code` `in_use`, and neither the configuration nor the backups SHALL change. When the removal fails for another reason, the save SHALL get `500` with `code` `internal`, the configuration SHALL keep its previous storage and the backups SHALL stay listed.

#### Scenario: Backups on the old storage

- **WHEN** a database has logical backups in storage A, and a user saves its logical backup configuration with storage B
- **THEN** the backup list of the database shows none of these backups
- **AND** deletion of their files from storage A is requested

#### Scenario: A backup is running

- **WHEN** a logical backup of the database is running, and a user saves its logical backup configuration with another storage
- **THEN** the response is `409` with `code` `in_use`, the configuration names the old storage, and the backups are still listed

#### Scenario: The storage does not change

- **WHEN** a user saves a logical backup configuration with the storage it already had
- **THEN** no backup of the database is removed

#### Scenario: The removal fails

- **WHEN** a user saves a logical backup configuration with another storage, and the removal of the old backups fails
- **THEN** the response is `500` with `code` `internal`, the configuration names the old storage, and the backups are still listed
