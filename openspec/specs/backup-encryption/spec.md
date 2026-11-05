# Backup Encryption Specification

## Purpose

Defines how Databasus protects the contents of a backup file in a storage, so that a storage operator cannot read a backup, a damaged file is never restored as if it were whole, and files written by earlier releases stay readable.

## Requirements

### Requirement: An encrypted backup is unreadable without the instance's master key

When a backup config has encryption turned on, Databasus SHALL encrypt every backup file before the first byte leaves for the storage. The key of each file SHALL be derived from the instance's master key, the file's own backup ID and a random salt stored with the backup, so no two files share a key. The stored file SHALL NOT contain any part of the dumped data in clear text. Only the instance that holds the master key SHALL be able to decrypt the file.

#### Scenario: The stored file hides the data

- **WHEN** the dump of a database contains the text `encryption-marker-7f3a`, and it is backed up with encryption turned on
- **THEN** the backup file in the storage does not contain `encryption-marker-7f3a`
- **AND** a restore of that backup feeds the restore tool the dump unchanged, marker included

#### Scenario: Two backups of the same data

- **WHEN** the same unchanged dump is backed up twice with encryption turned on
- **THEN** the two stored files differ, and so do the salts in their headers

### Requirement: A damaged encrypted backup fails to restore

Databasus SHALL authenticate every chunk of an encrypted backup file as it decrypts it, and SHALL NOT pass a chunk that fails the check to the restore tool. A restore of a stored file that was changed or cut inside a chunk SHALL be recorded as failed, with a failure message that says the backup stream from storage failed. A restore of a stored file that was replaced by the file of another backup SHALL be recorded as failed before the restore tool receives any data, because the file's header does not match the backup.

The decryption alone cannot notice a file cut exactly at a chunk boundary, because the format marks no last chunk, so such a file decrypts as a shorter stream. Whether that restore fails then depends on the stored format: a MySQL or MariaDB backup ends inside its zstd frame, and `pg_restore` and `mongorestore` read their archive to its end. This change keeps the format, so it adds no requirement for that case.

#### Scenario: One byte of the stored file is changed

- **WHEN** one byte in the middle of an encrypted backup file is changed in the storage, and the backup is restored
- **THEN** the restore is recorded as failed, and its failure message says that the backup stream from storage failed

#### Scenario: The file is cut inside a chunk

- **WHEN** an encrypted backup file in the storage loses its last half chunk, and the backup is restored
- **THEN** the restore is recorded as failed, and its failure message says that the backup stream from storage failed

#### Scenario: The file of another backup

- **WHEN** an encrypted backup file in the storage is replaced by the encrypted file of another backup of the same database, and the first backup is restored
- **THEN** the restore is recorded as failed, and the restore tool receives no data

### Requirement: Backups written by earlier releases keep decrypting

An upgrade SHALL NOT change the format of encrypted backup files or how their keys are derived. A file that an earlier release encrypted SHALL decrypt with the current release to the bytes that release dumped, as long as the instance keeps its master key. Restores and downloads decrypt through the same code, so both keep working for such a file.

#### Scenario: A file of the previous format decrypts

- **WHEN** the current release decrypts a stored fixture file that the code before this change encrypted with a known master key, backup ID, salt and nonce
- **THEN** the result equals the original bytes of the fixture
