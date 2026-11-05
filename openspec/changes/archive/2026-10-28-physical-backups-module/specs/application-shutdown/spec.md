## ADDED Requirements

### Requirement: A stop signal ends running physical backups with a recorded outcome and stops WAL streamers

When the application receives `SIGTERM` or `SIGINT`, every physical backup that is running, full or incremental, SHALL stop, and its client process SHALL be interrupted. Before the application exits, each such backup SHALL be recorded as failed with the fail message `backup interrupted by application shutdown`, its files SHALL be handed to the deletion of stored files, and the failure notification SHALL go to the notifiers of a database that asked for failure notifications, as for any failed backup. None of these backups SHALL be recorded as cancelled. A backup whose outcome is not recorded within the shutdown limit SHALL be recorded as failed at the next start, as before this change.

Every WAL stream SHALL stop as well. Its replication slot and the WAL segments already received into the local queue SHALL be kept, so the next start resumes streaming from the slot. Each stopped stream SHALL be recorded as stopped before the application exits, so the next start takes it over at once instead of waiting 90 seconds for its heartbeat to go stale.

A user's cancel of a running physical backup SHALL keep its outcome from before this change: a full or incremental backup cancelled while `pg_basebackup` streams it is recorded as cancelled with `context canceled` as its fail message.

#### Scenario: Full backup running at the stop

- **WHEN** the application receives a stop signal while a physical full backup is running
- **THEN** after the next start the backup is listed as failed with the fail message `backup interrupted by application shutdown`
- **AND** the backup is not listed as cancelled
- **AND** its stored files are gone once the deletion of stored files has run
- **AND** a notifier of the database subscribed to failed backups has received one failure notification for it, and a notifier not subscribed to failed backups has received none

#### Scenario: Incremental backup running at the stop

- **WHEN** the application receives a stop signal while a physical incremental backup is running
- **THEN** after the next start the backup is listed as failed with the fail message `backup interrupted by application shutdown`
- **AND** its stored files are gone once the deletion of stored files has run
- **AND** a notifier of the database subscribed to failed backups has received one failure notification for it

#### Scenario: WAL stream at the stop

- **WHEN** the application receives a stop signal while a database streams WAL, and the application starts again
- **THEN** the replication slot the stream used is still on the source
- **AND** within 30 seconds of the start the stream statistics report the streamer as running, well before a stale heartbeat would let the start take it over
- **AND** on the source, the replication slot with the name from before the stop is active, and no other slot exists for that database

#### Scenario: A user cancels a physical backup while the application runs

- **WHEN** a user cancels a running physical full backup, or a running physical incremental backup, while `pg_basebackup` streams it and no stop signal has arrived
- **THEN** the backup is listed as cancelled with `context canceled` as its fail message, as before this change
- **AND** its fail message is not `backup interrupted by application shutdown`

## MODIFIED Requirements

### Requirement: The application exits within a fixed limit after a stop signal

After a stop signal, the application SHALL stop accepting new connections, finish the HTTP requests in progress, wait for its running backups, logical and physical, and its restores to record their outcome, wait for its WAL streams to record that they stopped, and flush its log. All of this SHALL take no longer than 8 seconds in total, after which the application SHALL exit even when a request, a task, a WAL stream or the log flush has not finished. The limit leaves the embedded PostgreSQL server time for a clean shutdown within Docker's default stop timeout of 10 seconds. A task that has not recorded its outcome by then SHALL be recorded as failed on the next start, as before this change. A WAL stream that has not recorded its stop by then SHALL be taken over at the next start once its heartbeat is stale, as before this change.

#### Scenario: Nothing is running

- **WHEN** the application receives a stop signal while no request, backup, restore or WAL stream is in progress
- **THEN** the application exits within 3 seconds

#### Scenario: A task does not stop in time

- **WHEN** the application receives a stop signal while a task runs that does not end within the limit
- **THEN** the application exits no later than 8 seconds after the signal
- **AND** the next start records that task as failed

#### Scenario: A second signal arrives during shutdown

- **WHEN** the application receives a second stop signal while it is already shutting down
- **THEN** the second signal neither kills the process nor starts the shutdown over
- **AND** the shutdown finishes once, within the same limit

#### Scenario: A physical client process does not stop in time

- **WHEN** the application receives a stop signal while a physical backup or a WAL stream runs whose client process ignores the interrupt
- **THEN** the application exits no later than 8 seconds after the signal
- **AND** the next start records that backup as failed, and resumes that WAL stream from its replication slot
