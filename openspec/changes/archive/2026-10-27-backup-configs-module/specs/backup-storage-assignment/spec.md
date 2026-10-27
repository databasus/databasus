## MODIFIED Requirements

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

## ADDED Requirements

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
