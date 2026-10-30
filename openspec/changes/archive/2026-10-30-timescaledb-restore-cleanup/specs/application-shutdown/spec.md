## MODIFIED Requirements

### Requirement: A stop signal ends running logical backups and restores with a recorded outcome

When the application receives `SIGTERM` or `SIGINT`, every logical backup and every restore that is running SHALL stop. Before the application exits, each of them SHALL be recorded as failed, except for a restore stopped first by user cancellation before the stop signal. A logical backup interrupted by shutdown SHALL get the fail message `backup interrupted by application shutdown`. A restore interrupted by shutdown SHALL get a fail message that contains `restore interrupted by application shutdown`. Neither operation interrupted by shutdown SHALL be recorded as cancelled.

A restore whose archive had already failed before cleanup SHALL retain that failure if a stop signal arrives during cleanup.

A restore stopped first by user cancellation before shutdown SHALL retain its canceled outcome, including when its terminal status has not yet been written because cleanup is still running. Shutdown SHALL shorten that cleanup's time limit without replacing its interruption cause. The fixed shutdown limit and next-start recovery SHALL still apply when an outcome cannot be recorded before exit.

#### Scenario: Logical backup running at the stop

- **WHEN** the application receives a stop signal while a logical backup is running
- **THEN** the backup is recorded as failed with the fail message `backup interrupted by application shutdown` before the application exits
- **AND** the backup is not recorded as cancelled

#### Scenario: Restore running at the stop

- **WHEN** the application receives a stop signal while a restore is running and its user has not canceled it
- **THEN** the restore is recorded as failed with a fail message that contains `restore interrupted by application shutdown` before the application exits
- **AND** the restore is not recorded as cancelled

#### Scenario: A user cancels a backup while the application runs

- **WHEN** a user cancels a running logical backup and no stop signal has arrived
- **THEN** the backup is recorded as cancelled

#### Scenario: A user cancels a restore before shutdown

- **WHEN** a user cancels an archive restoration that has not already failed and a stop signal arrives while its cleanup is still running
- **THEN** cleanup obeys the remaining shutdown budget and the restore is recorded as canceled when persistence can finish before exit
- **AND** the stop signal does not replace that cancellation with a shutdown failure

#### Scenario: Archive failure precedes shutdown

- **WHEN** archive restoration fails before a stop signal arrives during cleanup
- **THEN** cleanup obeys the remaining shutdown budget and the restore retains its archive failure rather than the shutdown interruption message
