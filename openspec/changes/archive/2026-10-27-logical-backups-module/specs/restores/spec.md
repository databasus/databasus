## Purpose

Defines how a logical backup is restored into a target database the user names: what is checked before the restore starts, how its target's credentials are handled, how it is followed and cancelled, and how its outcome is recorded.

## ADDED Requirements

### Requirement: A restore is checked before it starts and answers with itself

Starting a restore of a completed logical backup SHALL name a target database of the backup's engine with its connection settings and the restore options of that engine. Before it answers, Databasus SHALL connect to the target, through its SSH tunnel when one is given, and detect its server version. The request SHALL then answer `202` with the new restore in the status `IN_PROGRESS`.

The request SHALL be refused, and SHALL start nothing, when:
- the backup is not completed, the target is of another engine, or the target's version cannot take the backup: `400` with `code` `invalid_request`;
- the target cannot be reached or refuses the credentials: `400` with `code` `database_connection_failed`;
- a restore into the same database of the backup is already running: `409` with `code` `already_running`;
- a PostgreSQL restore that needs the archive as a file, because it runs with parallel jobs, excludes extensions or skips user mappings, lacks disk space: `409` with `code` `insufficient_disk_space`, whose detail names the bytes needed and the bytes free, as the `backup-streaming` spec states.

#### Scenario: Starting a restore

- **WHEN** a member restores a completed PostgreSQL backup into a reachable PostgreSQL server of the same major version
- **THEN** the response is `202` with a restore in `IN_PROGRESS`, and reading it later shows `COMPLETED`

#### Scenario: The target is unreachable

- **WHEN** a member restores a backup into a host that refuses the connection
- **THEN** the response is `400` with `code` `database_connection_failed`, and the backup has no new restore

#### Scenario: A second restore of the same database

- **WHEN** a member starts a restore of a backup while another restore of a backup of the same database is running
- **THEN** the response is `409` with `code` `already_running`

#### Scenario: The backup is not completed

- **WHEN** a member restores a backup that failed
- **THEN** the response is `400` with `code` `invalid_request`, and the backup has no new restore

#### Scenario: A target of another engine

- **WHEN** a member restores a PostgreSQL backup into a MySQL target
- **THEN** the response is `400` with `code` `invalid_request`, and the backup has no new restore

#### Scenario: A target version that cannot take the backup

- **WHEN** a member restores a backup of PostgreSQL 17 into a PostgreSQL 15 server
- **THEN** the response is `400` with `code` `invalid_request`, and the backup has no new restore

#### Scenario: Not enough disk for a parallel restore

- **WHEN** a member starts a parallel PostgreSQL restore of a backup larger than the free disk
- **THEN** the response is `409` with `code` `insufficient_disk_space`, its detail names the bytes needed and the bytes free, and the backup has no new restore

### Requirement: The target's credentials are never stored or returned

The password, TLS key and SSH secrets of a restore target SHALL be used only by the running restore and SHALL NOT be written to the internal database, to a cache or to a log. Reading or listing restores SHALL NOT return any part of the target's settings. A restore that cannot continue after Databasus restarted SHALL NOT be resumed from stored credentials. This is a security rule: a restore target is often a production server of the user.

#### Scenario: Reading a restore

- **WHEN** a member reads a restore that was started with a target password
- **THEN** the response holds the restore's status, times and fail message and no field of the target

#### Scenario: Nothing persists the password

- **WHEN** a restore with the target password `restore-secret-5d1e` completes
- **THEN** no table of the internal database and no log record holds `restore-secret-5d1e`

### Requirement: Every accepted restore ends with an outcome

A restore SHALL end as `COMPLETED`, `FAILED` with a fail message, or `CANCELED`. Any failure after the restore was accepted SHALL record `FAILED` with its reason, including a failure before the restore tool starts, such as a storage that cannot be read. A failed restore SHALL report the restore tool's error output, as the `database-tool-failure-output` spec states. A restore SHALL read the backup from the storage it was written to, as the `backup-storage-assignment` spec states.

#### Scenario: The backup's storage cannot be opened

- **WHEN** a restore is accepted and the storage that holds the backup refuses the read
- **THEN** the restore ends as `FAILED` with a fail message that names the storage error, and a new restore of the database can start

### Requirement: A running restore can be cancelled

Cancelling a running restore SHALL answer `202`, stop its restore tool together with every process the tool started, including while the backup is still being read from storage before the tool starts, and record the restore as `CANCELED`. Cancelling a restore that is no longer running SHALL get `409` with `code` `not_running` and SHALL change nothing.

#### Scenario: Cancelling a running restore

- **WHEN** a member cancels a running restore
- **THEN** the response is `202`, reading the restore later shows `CANCELED`, and no process of its restore tool is left running

#### Scenario: Cancelling a restore while its archive downloads

- **WHEN** a member cancels a parallel PostgreSQL restore while its archive is still being downloaded into a file
- **THEN** the response is `202`, and reading the restore later shows `CANCELED`
- **AND** no restore tool process was started, and the temporary archive file is gone

#### Scenario: Cancelling a finished restore

- **WHEN** a member cancels a restore that has completed
- **THEN** the response is `409` with `code` `not_running`, and the restore is still `COMPLETED`

### Requirement: A stop of Databasus fails running restores

A restore that runs while Databasus shuts down SHALL end as the `application-shutdown` spec states. A restore still marked as in progress when Databasus starts, because the previous run stopped without recording its end, SHALL be recorded as `FAILED` with a fail message that contains `Restore failed due to application restart`, before any new restore can start.

#### Scenario: Shutdown during a restore

- **WHEN** Databasus is stopped while a restore runs
- **THEN** after the next start, the restore shows `FAILED` with a fail message that contains `restore interrupted by application shutdown`

#### Scenario: A restore left in progress

- **WHEN** a restore row is `IN_PROGRESS` when Databasus starts
- **THEN** after the start, the restore shows `FAILED` with a fail message that contains `Restore failed due to application restart`
- **AND** a new restore of the same database is accepted

### Requirement: A backup's restores are listed newest first

The restores of a backup SHALL be listed newest first, in pages with a continuation cursor, each with its status, its start time, its duration and its fail message.

#### Scenario: Two restores of one backup

- **WHEN** a backup was restored twice, and the first restore failed
- **THEN** listing its restores returns the second restore first and the failed one after it
