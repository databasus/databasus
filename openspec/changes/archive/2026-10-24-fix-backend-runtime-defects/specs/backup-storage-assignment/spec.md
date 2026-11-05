## Purpose

Defines which storage a restore reads a backup from, what happens to existing backups when a backup configuration moves to another storage, and that saving a backup configuration never changes the storages themselves.

## ADDED Requirements

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

Saving a logical or a physical backup configuration SHALL NOT create a storage and SHALL NOT change any field of an existing storage, whatever storage object the request carries. A configuration SHALL refer to a storage only by the identifier of an existing storage in the database's workspace. A storage SHALL change only through the storage endpoints and their access checks. This is a security measure: the storage endpoints are the only place that checks who may create or edit a storage.

#### Scenario: The request carries changed storage fields

- **WHEN** a user saves a backup configuration whose embedded storage object has the identifier of an existing storage but a different name and different credentials
- **THEN** the save succeeds and the configuration names that storage
- **AND** the stored storage keeps its name and credentials

#### Scenario: The request carries a storage without an identifier

- **WHEN** a user saves a backup configuration whose embedded storage object has no identifier
- **THEN** the save is rejected with an error
- **AND** no new storage exists after the request

#### Scenario: The request carries an unknown storage

- **WHEN** a user saves a backup configuration whose embedded storage object has an identifier that no storage has
- **THEN** the save is rejected with an error
- **AND** no storage with that identifier exists after the request
