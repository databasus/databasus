## MODIFIED Requirements

### Requirement: A stop signal ends running physical backups with a recorded outcome and stops WAL streamers

When the application receives `SIGTERM` or `SIGINT`, every physical backup that is running, full or incremental, SHALL stop, and its client process SHALL be interrupted if it is still running. Before the application exits, each backup whose execution was interrupted SHALL be recorded as failed with error reason `APPLICATION_RESTART` and fail message `backup interrupted by application shutdown`, its files SHALL be handed to the deletion of stored files, and the failure notification SHALL go to the notifiers of a database that asked for failure notifications, as for any failed backup. None of these interrupted backups SHALL be recorded as cancelled. A backup whose outcome is not recorded within the shutdown limit SHALL be recorded as failed at the next start, as before this change.

A physical full or incremental backup whose execution succeeds, including its archive and all required accompanying files, SHALL retain its successful outcome when shutdown reaches it before that outcome is recorded. This includes required accompanying writes that finish successfully during shutdown. Provided the backup remains eligible for publication and its outcome is recorded within the existing shutdown limit, it SHALL be listed as `COMPLETED`, its files SHALL be retained after the deletion of stored files has run, and shutdown SHALL NOT add an error reason or fail message. Its notification SHALL follow the database's success-notification settings, and it SHALL NOT send a failure notification solely because of shutdown. A copied archive alone SHALL NOT count as successful execution when a required accompanying write fails.

Every WAL stream SHALL stop as well. Its replication slot and the WAL segments already received into the local queue SHALL be kept, so the next start resumes streaming from the slot. Each stopped stream SHALL be recorded as stopped before the application exits, so the next start takes it over at once instead of waiting 90 seconds for its heartbeat to go stale.

A user's cancel of a running physical backup SHALL keep its outcome from before this change: a full or incremental backup cancelled while `pg_basebackup` streams it is recorded as cancelled with `context canceled` as its fail message.

#### Scenario: Full backup running at the stop

- **WHEN** the application receives a stop signal while a physical full backup is still copying its archive
- **THEN** after the next start the backup is listed as failed with error reason `APPLICATION_RESTART` and fail message `backup interrupted by application shutdown`
- **AND** the backup is not listed as cancelled
- **AND** its stored files are gone once the deletion of stored files has run
- **AND** a notifier of the database subscribed to failed backups has received one failure notification for it, and a notifier not subscribed to failed backups has received none

#### Scenario: Incremental backup running at the stop

- **WHEN** the application receives a stop signal while a physical incremental backup is still copying its archive
- **THEN** after the next start the backup is listed as failed with error reason `APPLICATION_RESTART` and fail message `backup interrupted by application shutdown`
- **AND** its stored files are gone once the deletion of stored files has run
- **AND** a notifier of the database subscribed to failed backups has received one failure notification for it

#### Scenario: Successful full backup awaiting its recorded outcome during shutdown

- **WHEN** shutdown reaches a physical full backup after its execution has succeeded but before its successful outcome is recorded
- **AND** it remains eligible for publication and the outcome is recorded within the shutdown limit
- **THEN** after the next start the backup is listed as `COMPLETED` without an error reason or fail message
- **AND** its archive and required accompanying files remain after the deletion of stored files has run
- **AND** a notifier subscribed to successful backups receives one success notification and no failure notification for it

#### Scenario: Successful incremental backup awaiting its recorded outcome during shutdown

- **WHEN** shutdown reaches a physical incremental backup after its execution has succeeded but before its successful outcome is recorded
- **AND** it remains eligible for publication and the outcome is recorded within the shutdown limit
- **THEN** after the next start the backup is listed as `COMPLETED` without an error reason or fail message
- **AND** its archive and required accompanying files remain after the deletion of stored files has run
- **AND** a notifier subscribed to successful backups receives one success notification and no failure notification for it

#### Scenario: Required accompanying writes finish during shutdown

- **WHEN** a stop signal arrives after a physical full or incremental backup has copied its archive and while a required accompanying write is still running
- **AND** all remaining execution steps succeed, the backup remains eligible for publication, and its outcome is recorded within the shutdown limit
- **THEN** the backup is listed as `COMPLETED` after the next start
- **AND** its archive and required accompanying files remain after the deletion of stored files has run

#### Scenario: WAL stream at the stop

- **WHEN** the application receives a stop signal while a database streams WAL, and the application starts again
- **THEN** the replication slot the stream used is still on the source
- **AND** within 30 seconds of the start the stream statistics report the streamer as running, well before a stale heartbeat would let the start take it over
- **AND** on the source, the replication slot with the name from before the stop is active, and no other slot exists for that database

#### Scenario: A user cancels a physical backup while the application runs

- **WHEN** a user cancels a running physical full backup, or a running physical incremental backup, while `pg_basebackup` streams it and no stop signal has arrived
- **THEN** the backup is listed as cancelled with `context canceled` as its fail message, as before this change
- **AND** its fail message is not `backup interrupted by application shutdown`
