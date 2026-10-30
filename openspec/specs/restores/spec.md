# restores Specification

## Purpose
Defines how a logical backup is restored into a target database the user names: what is checked before the restore starts, how its target's credentials are handled, how it is followed and cancelled, and how its outcome is recorded.

## Requirements

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

Cancelling a running restore SHALL answer `202`, stop its restore tool together with every process the tool started, including while the backup is still being read from storage before the tool starts, and record the restore as `CANCELED`, unless the archive had already failed before cleanup as the TimescaleDB cleanup requirement states. A cancellation received during cleanup SHALL NOT replace that established failure. Cancelling a restore that is no longer running SHALL get `409` with `code` `not_running` and SHALL change nothing.

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

### Requirement: TimescaleDB leaves restoring mode after an interrupted restore

After successfully entering TimescaleDB restoring mode, a restore SHALL attempt to leave that mode on success, failure, ordinary cancellation or timeout. Ordinary cancellation SHALL NOT cancel cleanup. During normal application operation cleanup SHALL have its own 30-second maximum and SHALL finish before that restore execution is considered drained.

Graceful shutdown SHALL preserve the existing eight-second total exit limit, consisting of at most six seconds to drain and two seconds to flush logs. Cleanup SHALL be bounded by the remaining drain time minus a one-second reserve for recording the restore outcome; this smaller limit SHALL also apply when shutdown starts during an already-running cleanup. If no cleanup time remains, no new cleanup work SHALL start. The application SHALL NOT wait beyond its shutdown limit even when cleanup does not cooperate with cancellation.

A successful archive restore whose cleanup fails SHALL be reported as failed. Cleanup failure SHALL NOT replace an archive failure or an ordinary cancellation. An archive failure established before cleanup SHALL remain the recorded failure when user cancellation, the restore timeout or shutdown occurs during cleanup. When shutdown causes the interruption, the existing shutdown-failure outcome SHALL apply. When user cancellation is the first cause of the restore stopping before shutdown, the restore SHALL remain canceled while cleanup obeys the shortened shutdown budget. A cleanup failure or skipped cleanup SHALL report that the target may remain in restoring mode when logging can finish within the budget. If cleanup does not return before forced exit, the shutdown drain-timeout report SHALL identify the unfinished work; any restore without a recorded final outcome SHALL be failed by the existing next-start recovery. Neither case SHALL claim that the target left restoring mode.

#### Scenario: Cancellation after pre-restore

- **WHEN** a member cancels a restore after TimescaleDB entered restoring mode and the target remains reachable during ordinary application operation
- **THEN** the restore is canceled and the target leaves restoring mode before its execution is drained

#### Scenario: Restore failure

- **WHEN** restoring the archive fails after TimescaleDB entered restoring mode and cleanup can reach the target
- **THEN** the target leaves restoring mode and the restore retains the archive failure

#### Scenario: Cleanup fails after an otherwise successful restore

- **WHEN** archive restoration succeeds but leaving restoring mode fails
- **THEN** the restore is failed and the cleanup error is recorded

#### Scenario: Target is unreachable during cleanup

- **WHEN** the target becomes unreachable after entering restoring mode and the restore is canceled during ordinary application operation
- **THEN** cleanup stops within its 30-second maximum, the restore remains canceled, and the log states that the target may still be in restoring mode

#### Scenario: Shutdown cleanup succeeds within the budget

- **WHEN** shutdown starts after pre-restore and cleanup can finish within the remaining cleanup time
- **THEN** the target leaves restoring mode, the restore records the existing shutdown failure and the application exits within eight seconds

#### Scenario: Shutdown interrupts slow cleanup

- **WHEN** cleanup after pre-restore exceeds the remaining shutdown cleanup budget
- **THEN** cleanup is interrupted at its cutoff and, when it returns cooperatively, the remaining drain time is used to record the restore's shutdown failure and warning
- **AND** the application exits within eight seconds and the target may remain in restoring mode

#### Scenario: Shutdown begins during an ordinary cleanup attempt

- **WHEN** an ordinary cleanup attempt with a 30-second maximum is still running when shutdown starts
- **THEN** its limit is shortened to the shared shutdown cleanup cutoff rather than retaining or restarting a 30-second window
- **AND** the application exits within the unchanged eight-second total limit

#### Scenario: Shutdown budget is already exhausted

- **WHEN** cleanup becomes ready only after its shutdown cutoff
- **THEN** the application starts no new cleanup SQL, reports possible remaining restoring mode when budget permits, and preserves the existing exit limit

#### Scenario: Cleanup does not return before forced exit

- **WHEN** cleanup ignores cancellation and does not return before the drain deadline
- **THEN** the application reports unfinished draining and exits within eight seconds without claiming target cleanup succeeded
- **AND** the target may remain in restoring mode and a restore without a final recorded outcome is failed on the next start


#### Scenario: Shutdown follows user cancellation

- **WHEN** a member cancels an archive restoration that has not already failed and shutdown begins while its cleanup is still running
- **THEN** cleanup obeys the shutdown cutoff and the restore remains canceled when its outcome can be recorded

#### Scenario: User cancellation follows an archive failure

- **WHEN** archive restoration fails and a member cancels the restore while cleanup is still running
- **THEN** the restore is failed with the archive failure rather than canceled

#### Scenario: Shutdown follows an archive failure

- **WHEN** archive restoration fails and shutdown begins while cleanup is still running
- **THEN** cleanup observes the shutdown budget and the recorded restore failure retains the archive error rather than the shutdown interruption message
