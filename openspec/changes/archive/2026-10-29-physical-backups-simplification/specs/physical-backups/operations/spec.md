## MODIFIED Requirements

### Requirement: Cancelling stops a running physical backup

Cancelling a running full or incremental backup SHALL answer `202` with no body. By the time the response is sent, the backup SHALL already be recorded as `CANCELED`, and its process SHALL stop afterwards. A backup that is no longer running SHALL get `409` with `code` `not_running` and SHALL stay as it is. A backup ID that names no full or incremental backup SHALL get `404` with `code` `not_found`.

#### Scenario: Cancelling a running backup

- **WHEN** a member cancels a running full backup
- **THEN** the response is `202`, the list read right after it shows the backup as `CANCELED`, and its `pg_basebackup` process ends

#### Scenario: Cancelling a backup whose process is gone

- **WHEN** a member cancels a backup that is recorded as running but has no live process in Databasus
- **THEN** the response is `202`, and the backup is listed as `CANCELED`

#### Scenario: A finished backup

- **WHEN** a member cancels a completed full backup
- **THEN** the response is `409` with `code` `not_running`, and the backup is still `COMPLETED`

### Requirement: Deleting removes a backup and the backups built on it

Deleting a full backup SHALL remove it and every incremental backup built on it. Deleting an incremental backup SHALL remove it and every later incremental backup of its chain. A backup that is running SHALL be removed before the response, and its process SHALL stop afterwards. Both SHALL answer `204`. An ID that names no full or incremental backup, the ID of a WAL segment included, SHALL get `404` with `code` `not_found` and SHALL delete nothing. The `physical-backup-wal-ranges` spec holds the scenario for a WAL segment.

#### Scenario: Deleting a full backup with incrementals

- **WHEN** a member deletes a completed full backup that has two incremental backups
- **THEN** the response is `204`, and neither the full backup nor its incremental backups are listed

#### Scenario: Deleting a running backup

- **WHEN** a member deletes a full backup that is running
- **THEN** the response is `204`, the list read right after it no longer shows the backup, and its `pg_basebackup` process ends

## ADDED Requirements

### Requirement: A database runs one physical backup at a time

A database SHALL have at most one full or incremental backup recorded as running. Databasus SHALL also start no backup of a database while the process of an earlier backup of that database is still running, including a backup that was cancelled, deleted or stopped because its backups were turned off and whose process has not ended yet. A requested or scheduled backup SHALL start on the scheduler's first pass after that process has ended. Turning a database's backups off SHALL record its running backup as `CANCELED` before the save is answered. A backup still recorded as running when Databasus starts is failed as the `api-contract` spec describes, and the database is then free for its next backup.

#### Scenario: Starting right after a cancel

- **WHEN** a member cancels a running full backup and immediately requests a new full backup of the same database
- **THEN** the new backup starts only after the cancelled backup's `pg_basebackup` has ended, and it completes

#### Scenario: Starting right after a delete

- **WHEN** a member deletes a running full backup and the full backup cadence of the database is due
- **THEN** the next full backup starts only after the deleted backup's `pg_basebackup` has ended, and then completes

#### Scenario: Turning backups off during a backup

- **WHEN** a member turns off the backups of a database while its full backup is running, and turns them back on with the full backup cadence due
- **THEN** the list read right after the first save shows the backup as `CANCELED`
- **AND** the next full backup starts only after the cancelled backup's `pg_basebackup` has ended, and then completes

#### Scenario: A running backup at startup

- **WHEN** Databasus starts and finds a backup recorded as running
- **THEN** the scheduler can start the next backup of that database
