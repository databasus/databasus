## MODIFIED Requirements

### Requirement: A manual start asks the scheduler for a backup

Starting a physical backup SHALL answer `202` with no body. The request SHALL name the kind of backup: a full backup, an incremental backup, or the kind the scheduler would choose. That kind is an incremental backup when the database's backup type includes incremental backups and its newest chain can be extended, and a full backup otherwise. The backup SHALL appear in the list when the scheduler starts it. When an incremental backup is requested and the database has no chain to extend, the request SHALL get `409` with `code` `no_extendable_chain` and SHALL request nothing. A database that is not a physical PostgreSQL database SHALL get `400` with `code` `invalid_request`. A database whose backups are turned off, a database whose backup config was never saved included, or whose backup config names no storage SHALL get `400` with `code` `invalid_request` and SHALL request nothing, whichever kind the request names.

#### Scenario: Starting a full backup

- **WHEN** a member starts a full backup of a physical database with backups turned on
- **THEN** the response is `202`, and a full backup of the database appears in the list in `IN_PROGRESS` or `COMPLETED` after the scheduler's next pass

#### Scenario: The scheduler's choice for a full-only backup type

- **WHEN** a member starts a backup of the scheduler's choice for a physical database whose backup type takes only full backups and whose newest full backup is completed
- **THEN** the response is `202`, and the backup that appears in the list after the scheduler's next pass is a full backup

#### Scenario: An incremental backup without a chain

- **WHEN** a member starts an incremental backup of a physical database that has no completed full backup
- **THEN** the response is `409` with `code` `no_extendable_chain`, and no backup appears in the list after the scheduler's next pass

#### Scenario: A logical database

- **WHEN** a member starts a physical backup of a MySQL database
- **THEN** the response is `400` with `code` `invalid_request`

#### Scenario: Backups turned off

- **WHEN** a member starts a full backup of a physical database whose backups are turned off and that has no earlier backup request pending
- **THEN** the response is `400` with `code` `invalid_request`, and the WAL stream statistics do not report a full backup as pending

#### Scenario: An incremental backup with backups turned off and no chain

- **WHEN** a member starts an incremental backup of a physical database whose backups are turned off and that has no completed full backup
- **THEN** the response is `400` with `code` `invalid_request`, not `409`

#### Scenario: An incremental backup with backups turned off and a chain

- **WHEN** a member starts an incremental backup of a physical database whose backups are turned off and whose newest chain can be extended, and then turns its backups on while no backup is due by its schedules
- **THEN** the start answers `400` with `code` `invalid_request`, and no incremental backup appears in the list after the scheduler's next pass

#### Scenario: A backup config that was never saved

- **WHEN** a member starts a backup of a physical database whose backup config was never saved
- **THEN** the response is `400` with `code` `invalid_request`

#### Scenario: No storage

- **WHEN** a member starts a backup of a physical database whose backups are turned on, whose backup config names no storage and that has no earlier backup request pending
- **THEN** the response is `400` with `code` `invalid_request`, and the WAL stream statistics do not report a full backup as pending

### Requirement: Cancelling stops a running physical backup

Cancelling a running full or incremental backup SHALL answer `200` with the backup in its final status, in the same form the list gives it. By the time the response is sent, the backup SHALL already be recorded as `CANCELED`, and its process SHALL stop afterwards. A backup that is no longer running SHALL get `409` with `code` `not_running` and SHALL stay as it is. A backup ID that names no full or incremental backup SHALL get `404` with `code` `not_found`.

#### Scenario: Cancelling a running backup

- **WHEN** a member cancels a running full backup
- **THEN** the response is `200` with the backup as `CANCELED`, the list read right after it shows the same backup with the same values, and its `pg_basebackup` process ends

#### Scenario: Cancelling a running incremental backup

- **WHEN** a member cancels a running incremental backup whose parent is the root full backup
- **THEN** the response is `200` with the backup as `CANCELED`, and it equals the row the list returns right after, its `parentIncrementalBackupId` included

#### Scenario: Cancelling a backup whose process is gone

- **WHEN** a member cancels a backup that is recorded as running but has no live process in Databasus
- **THEN** the response is `200` with the backup as `CANCELED`, and the backup is listed as `CANCELED`

#### Scenario: A finished backup

- **WHEN** a member cancels a completed full backup
- **THEN** the response is `409` with `code` `not_running`, and the backup is still `COMPLETED`
