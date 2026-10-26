## MODIFIED Requirements

### Requirement: Deleting a storage reports the files it leaves behind

The system SHALL reject deleting a storage configuration while a backup configuration uses it or backup records still reference it, because those records are the only place its file names exist. The database itself SHALL refuse the removal, so that a missing check in the application cannot delete a backup configuration or a backup record together with the storage. A rejected deletion SHALL change nothing and SHALL remove no file from the provider. When nothing references the storage, deletion SHALL attempt the storage's outstanding cleanup and SHALL then succeed. The cleanup SHALL skip a file whose write is still running in the application process. Deletion SHALL record how many files it could not remove or skipped, and their names up to a fixed limit. That record SHALL be somewhere the user can read. Outstanding cleanup SHALL NOT make a storage or a workspace permanently undeletable.

#### Scenario: Storage is deleted while backups still reference it

- **WHEN** a user deletes a storage that still has backup records
- **THEN** deletion is refused with a message that names the backups as the reason and tells the user to remove them first
- **AND** the backups stay readable and removable through their own deletion path
- **AND** no file is removed from the provider, the files of the storage's pending cleanup included

#### Scenario: Storage is deleted while a backup configuration uses it

- **WHEN** a user deletes a storage that the backup configuration of a database names
- **THEN** deletion is refused with a message that names the backup configuration as the reason and tells the user to remove it first
- **AND** the database keeps its backup configuration with that storage, and no file is removed from the provider

#### Scenario: Storage row deleted directly in the database

- **WHEN** the row of a storage is deleted directly in the database while a backup configuration, a logical backup record or a physical backup record references it
- **THEN** the database refuses the statement
- **AND** the configuration and the records remain

#### Scenario: Storage with reachable pending cleanup is deleted

- **WHEN** a storage with no backup records, no backup configuration using it and reachable pending cleanup is deleted
- **THEN** the pending files are removed from the provider before the deletion commits, so a reader still sees the storage until its files are gone
- **AND** deletion succeeds

#### Scenario: Storage with an unreachable provider is deleted

- **WHEN** a storage is deleted while its provider cannot be reached
- **THEN** deletion still succeeds
- **AND** the audit log names the storage, how many files were left behind and their names up to a fixed limit

#### Scenario: Storage is deleted while a write to it is still running

- **WHEN** a storage that nothing references is deleted while a write of a file to it is still running in the application
- **THEN** deletion succeeds without removing that file
- **AND** the audit log counts that file among the files left behind and names it

#### Scenario: Workspace containing such a storage is deleted

- **WHEN** a workspace is deleted whose storages still hold pending cleanup
- **THEN** deletion follows the same path as deleting each storage
- **AND** the caller never sees a raw database constraint error
