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

### Requirement: The interface never fills a restore target from the backed-up database

The restore target form in the interface SHALL open with the default settings of a new database of the backup's engine. It SHALL NOT carry any connection setting of the backed-up database, including its host, port, user, database name, TLS certificates and SSH bastion, and it SHALL NOT present any secret as already stored. The target SHALL require the same fields as adding a database of that engine, including the password. No request the interface makes for a restore SHALL carry the backed-up database's ID as the source of omitted settings, and the flow SHALL send no database settings test, so no stored secret of that database is used to reach the target. While the dialog of a backup stays open, opening the form again after a start SHALL show the values and restore options entered for the previous target, with the advanced settings open when any of them holds a non-default value, so a failed restore can be retried. Closing that dialog SHALL discard them. The form SHALL show only settings a restore uses, so it SHALL NOT offer exclusions, schema selections or other settings that only affect a backup. A start answer SHALL change nothing on the screen if the dialog has closed or the session has changed. This is a security rule: a restore target is usually another server, and a secret of the source must not reach it unless the user types it.

#### Scenario: The source database uses an SSH tunnel and a client certificate

- **WHEN** a member opens the restore target form for a backup of a PostgreSQL database that connects through an SSH bastion with a client certificate on a non-default port
- **THEN** the form shows the engine's default port, an empty host, user, password and database name, a disabled SSH tunnel and no stored certificate or SSH credentials

#### Scenario: The password is left empty

- **WHEN** a member fills the host, user and database name of the target and leaves its password empty
- **THEN** the start button is disabled and no restore start request is sent

#### Scenario: The restore request holds only entered values

- **WHEN** a member enters a target and starts the restore
- **THEN** the start request holds only the values entered in the form and the engine defaults, and no request of the flow carries the backed-up database's ID

#### Scenario: A retry after a failed restore

- **WHEN** a restore of a PostgreSQL backup started with the extension exclusion option fails and the member opens the target form again without closing the dialog
- **THEN** the form shows the entered target with the advanced settings open and the extension exclusion option on, and any typed certificate appears as typed rather than as stored

#### Scenario: Closing the dialog discards the target

- **WHEN** a member enters a target, closes the backup's restore dialog and opens it again
- **THEN** the target form shows the engine's defaults with an empty password

#### Scenario: The dialog closes during a start

- **WHEN** a member closes the backup's restore dialog while a start is pending and the start then answers
- **THEN** no feedback appears and the dialog stays closed

#### Scenario: Backup-only settings of MySQL, MariaDB and MongoDB targets

- **WHEN** a member opens the advanced settings of the restore target form for a MySQL, MariaDB or MongoDB backup
- **THEN** the form offers no table, event or collection exclusions, and a MariaDB target still offers its Galera option

### Requirement: The interface checks a restore target by starting the restore

The restore target form SHALL start the restore without a separate connection test, relying on the restore start to connect to the target and check its version before it answers. While a start is pending, the form SHALL prevent another submission. A refused start SHALL show the refusal with the reason the server gave and SHALL keep the form open with every entered value, so the user can correct a field and start again. Among submissions, only an accepted start SHALL close the form. A workspace viewer SHALL be able to start a restore from the interface, as the `workspace-access` spec allows.

#### Scenario: A wrong password

- **WHEN** a member starts a restore into a target that refuses the entered password
- **THEN** the form stays open with every entered value and shows the connection refusal with the server's reason, and the backup lists no new restore

#### Scenario: Starting again after a correction

- **WHEN** after a refused start the member corrects the password and starts the restore again
- **THEN** the restore starts with the corrected values and the form closes

#### Scenario: A repeated click during a pending start

- **WHEN** a member clicks the start button twice while the first start is still pending
- **THEN** exactly one restore start request is sent

#### Scenario: A viewer restores a backup

- **WHEN** a workspace viewer enters a reachable target and starts the restore
- **THEN** no database settings test request is sent and the restore start answers `202`

### Requirement: The backup's restore dialog stays open when its backup leaves the list

The backup's restore dialog SHALL hold the backup the user opened it for. When that backup leaves the backup list, for example because retention removed it, the dialog SHALL stay open with its restores and with any entered target. A start for a backup that no longer exists SHALL show the translated refusal and keep the form open with every entered value, as any refused start does.

#### Scenario: Retention removes the backup while the dialog is open

- **WHEN** the backup's restore dialog is open and a list refresh no longer contains that backup
- **THEN** the dialog stays open and shows the same restores

#### Scenario: A target is being entered when the backup goes away

- **WHEN** a member is entering a restore target and the backup leaves the list
- **THEN** the entered values stay in the form

#### Scenario: A start after the backup was removed

- **WHEN** a member submits the target form after the backup was removed and the server refuses the start as not found
- **THEN** the translated reason is shown, the form stays open and every entered value is kept

### Requirement: Restore list failures are recoverable and never repeat

The first failed load of a backup's restores in the dialog SHALL show the translated reason with a retry and SHALL not appear as an empty list. Until a load succeeds, the dialog SHALL offer no way to start a restore and periodic refreshes SHALL wait. After that, a failed refresh SHALL keep the shown restores and show no blocking alert or message.

#### Scenario: The first load fails

- **WHEN** a member opens the backup's restore dialog and its first request for restores fails
- **THEN** the dialog shows the translated reason with a retry instead of the empty list text, offers no target selection and sends no refresh request

#### Scenario: A retry succeeds

- **WHEN** the first load failed and the member retries successfully
- **THEN** the restores and the target selection appear and periodic refreshes begin

#### Scenario: The server becomes unreachable

- **WHEN** the dialog shows restores and its periodic refreshes keep failing
- **THEN** the restores stay on the screen and no blocking alert or message appears

### Requirement: A failed restore cancel is reported once

When a confirmed cancel of a running restore fails, the dialog SHALL show the translated reason once and SHALL keep the restore in the list.

#### Scenario: A running restore cannot be cancelled

- **WHEN** a member confirms cancelling a running restore and the request fails
- **THEN** the translated reason is shown once and the restore stays in the list

### Requirement: A failed archive download does not report a partially restored target

When a PostgreSQL restore that needs the archive as a file fails while it reads the backup from storage, the restore tool has not started and has written no backup data to the target. The restore's fail message SHALL say that reading the backup failed before the restore tool started, SHALL contain the storage error, and SHALL NOT say that the target was partially restored. A restore that streams the backup directly into the tool and loses the stream SHALL keep reporting that the target is left partially restored.

#### Scenario: Storage fails during a parallel restore's download

- **WHEN** a PostgreSQL restore with parallel jobs loses its storage connection while downloading the archive into a file
- **THEN** the restore is recorded as failed, its fail message contains the storage error and says the restore tool did not start
- **AND** its fail message does not say the target is partially restored

#### Scenario: Storage fails during a streamed restore

- **WHEN** a PostgreSQL restore that streams the backup into the restore tool loses its storage connection
- **THEN** the restore is recorded as failed and its fail message says the target is left partially restored

### Requirement: A failed version comparison is not reported as a too-old target

When a MySQL, MariaDB or MongoDB restore cannot compare the backup's server version with the target's, starting it SHALL answer `500` with `code` `internal`, close the connection to the target and create no restore. The answer SHALL NOT say that the target's version is lower than the backup's.

#### Scenario: The recorded version of the backup's database cannot be read

- **WHEN** a member starts a restore of a MySQL, MariaDB or MongoDB backup whose database has an unreadable recorded server version, into a reachable target of that engine
- **THEN** the response is `500` with `code` `internal`, no restore is created, and the response does not say the target is too old

#### Scenario: The target is older than the backup

- **WHEN** a member starts a restore of a MySQL, MariaDB or MongoDB backup into an older target of that engine
- **THEN** the response is still `400` with `code` `invalid_request` and names both versions
