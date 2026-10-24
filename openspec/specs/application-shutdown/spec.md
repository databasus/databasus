# application-shutdown Specification

## Purpose
Defines what the Databasus application does with its running logical backups and restores after it receives a stop signal, and how long it may take to exit, so that an interrupted task is reported at the moment of the stop.

## Requirements

### Requirement: A stop signal ends running logical backups and restores with a recorded outcome

When the application receives `SIGTERM` or `SIGINT`, every logical backup and every restore that is running SHALL stop. Before the application exits, each of them SHALL be recorded as failed. A logical backup SHALL get the fail message `backup interrupted by application shutdown`. A restore SHALL get a fail message that contains `restore interrupted by application shutdown`. Neither SHALL be recorded as cancelled.

#### Scenario: Logical backup running at the stop

- **WHEN** the application receives a stop signal while a logical backup is running
- **THEN** the backup is recorded as failed with the fail message `backup interrupted by application shutdown` before the application exits
- **AND** the backup is not recorded as cancelled

#### Scenario: Restore running at the stop

- **WHEN** the application receives a stop signal while a restore is running
- **THEN** the restore is recorded as failed with a fail message that contains `restore interrupted by application shutdown` before the application exits
- **AND** the restore is not recorded as cancelled

#### Scenario: A user cancels a backup while the application runs

- **WHEN** a user cancels a running logical backup and no stop signal has arrived
- **THEN** the backup is recorded as cancelled, as before this change

### Requirement: The application exits within a fixed limit after a stop signal

After a stop signal, the application SHALL stop accepting new connections, finish the HTTP requests in progress, wait for its running backups and restores to record their outcome and flush its log. All of this SHALL take no longer than 8 seconds in total, after which the application SHALL exit even when a request, a task or the log flush has not finished. The limit leaves the embedded PostgreSQL server time for a clean shutdown within Docker's default stop timeout of 10 seconds. A task that has not recorded its outcome by then SHALL be recorded as failed on the next start, as before this change.

#### Scenario: Nothing is running

- **WHEN** the application receives a stop signal while no request, backup or restore is in progress
- **THEN** the application exits within 3 seconds

#### Scenario: A task does not stop in time

- **WHEN** the application receives a stop signal while a task runs that does not end within the limit
- **THEN** the application exits no later than 8 seconds after the signal
- **AND** the next start records that task as failed

#### Scenario: A second signal arrives during shutdown

- **WHEN** the application receives a second stop signal while it is already shutting down
- **THEN** the second signal neither kills the process nor starts the shutdown over
- **AND** the shutdown finishes once, within the same limit
