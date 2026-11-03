# Logical Backup Outcome Specification

## Purpose

Decides how a logical backup that did not complete is reported, so that a backup the user stopped on purpose stays quiet while every other unfinished backup is reported as a failure that reaches the user through notifications.

## Requirements

### Requirement: Only a cancellation request makes a backup cancelled

A logical backup of a PostgreSQL, MySQL, MariaDB or MongoDB database SHALL be recorded as cancelled only when a user requested cancellation of that backup before it ended. Every other backup that does not complete SHALL be recorded as failed. This SHALL hold whatever the underlying error says, including an error that describes a timeout or a cancellation inside a storage provider.

A cancelled backup SHALL NOT send a backup failure notification and SHALL NOT mark its database as failing. A failed backup SHALL get a fail message, SHALL mark its database as failing, and SHALL send the backup failure notification to the notifiers configured for it.

#### Scenario: Storage times out after the dump has finished

- **WHEN** the database dump finishes, and the upload of the backup to its storage then fails with a network timeout
- **THEN** the backup is recorded as failed, its database is marked as failing, and the backup failure notification is sent

#### Scenario: Storage times out while the dump is still running

- **WHEN** the upload of the backup fails with a network timeout while the database dump is still producing data
- **THEN** the backup is recorded as failed and the backup failure notification is sent

#### Scenario: Storage error that wraps a cancellation

- **WHEN** the storage provider fails the upload with an error that reports its own request as cancelled, and nobody requested cancellation of the backup
- **THEN** the backup is recorded as failed and the backup failure notification is sent

#### Scenario: User cancels a running backup

- **WHEN** a user cancels a logical backup while it is running
- **THEN** the backup is recorded as cancelled, no backup failure notification is sent, and the database is not marked as failing

#### Scenario: User cancels while the storage is failing

- **WHEN** a user requests cancellation of a running logical backup, and the backup then ends with a storage error instead of a cancellation error
- **THEN** the backup is recorded as cancelled, no backup failure notification is sent, and the database is not marked as failing

#### Scenario: Backup exceeds the time limit

- **WHEN** a logical backup runs longer than the backup time limit of 23 hours and is stopped for that reason
- **THEN** the backup is recorded as failed with the fail message `backup exceeded the time limit of 23 hours`, its database is marked as failing, and the backup failure notification is sent

#### Scenario: Application shuts down during a backup

- **WHEN** the application shuts down while a logical backup is running
- **THEN** the backup is recorded as failed with the fail message `backup interrupted by application shutdown`, not as cancelled

#### Scenario: Backup left in progress by a previous run

- **WHEN** the application starts and finds a logical backup still marked as in progress from its previous run
- **THEN** the backup is recorded as failed with the fail message `Backup failed due to application restart`, not as cancelled, even though cancellation is requested for it

### Requirement: The fail message names the storage error

When a logical backup fails because its storage upload failed, the fail message SHALL contain the error the storage returned, so the user can see which storage operation failed and why. It SHALL NOT be reduced to a generic message such as "backup cancelled". This SHALL hold when the storage fails the upload at once, before it has read any backup data, and the fail message SHALL NOT then report that starting the dump tool was cancelled.

#### Scenario: Fail message after a storage timeout

- **WHEN** a logical backup fails because the upload to its S3 storage timed out while connecting to the endpoint
- **THEN** the fail message shows that saving to storage failed and includes the storage's own description of the timeout, including the endpoint it tried to reach

#### Scenario: Fail message is the same for every engine

- **WHEN** the same storage timeout interrupts a PostgreSQL, a MySQL, a MariaDB and a MongoDB backup
- **THEN** each backup is recorded as failed with a fail message that includes the storage error

#### Scenario: Storage fails before reading any data

- **WHEN** the storage fails the upload of a logical backup at once, before it has read any backup data
- **THEN** the backup is recorded as failed, its fail message shows that saving to storage failed and includes the storage error, and it does not say that starting the dump tool was cancelled

### Requirement: The fail message includes the dump tool's error output

When the dump tool of a logical PostgreSQL, MySQL, MariaDB or MongoDB backup exits with an error and the storage upload did not fail, the backup SHALL be recorded as failed, and its fail message SHALL contain the error output the tool wrote before it exited. This SHALL hold however quickly the tool exits after writing that output, including when it exits before it produced any backup data.

#### Scenario: Dump tool fails before producing data

- **WHEN** the dump tool writes an error such as `permission denied for sequence orders_id_seq` and exits with a non-zero status before producing any backup data
- **THEN** the backup is recorded as failed, and its fail message contains `permission denied for sequence orders_id_seq`

### Requirement: The outcome of a stopped backup does not depend on the stage it reached

A logical backup stopped while it is still preparing, before its dump tool starts, SHALL get the same outcome as one stopped during the dump, whatever error the interrupted step returned. A stop SHALL end the preparation at once, including while an SSH bastion does not answer, so the outcome is recorded within the shutdown limit. A user's cancellation SHALL record it as cancelled, without a failure notification and without marking its database as failing. A shutdown SHALL record it as failed with the fail message `backup interrupted by application shutdown`, and the time limit SHALL record it as failed with the fail message `backup exceeded the time limit of 23 hours`. Both failures SHALL send the backup failure notification, as a failure during the dump does. Reading the backup's config SHALL NOT be cut short by the stop, so the notification settings are known at that point.

#### Scenario: User cancels while the SSH bastion does not answer

- **WHEN** a user cancels a logical backup while it is still opening a tunnel through an SSH bastion that accepts the connection and never answers
- **THEN** the backup is recorded as cancelled, no backup failure notification is sent, and the database is not marked as failing

#### Scenario: Application shuts down while the SSH bastion does not answer

- **WHEN** the application shuts down while a logical backup is still opening a tunnel through an SSH bastion that accepts the connection and never answers
- **THEN** before the application exits the backup is recorded as failed with the fail message `backup interrupted by application shutdown`, not as cancelled, and the backup failure notification is sent

#### Scenario: The time limit ends the preparation

- **WHEN** the time limit of a logical backup ends while it is still opening its tunnel
- **THEN** the backup fails with the error `backup exceeded the time limit of 23 hours`

#### Scenario: The SSH bastion refuses the connection and nobody stopped the backup

- **WHEN** a logical backup cannot open its tunnel because the SSH bastion refuses the connection, and nobody stopped it
- **THEN** the backup is recorded as failed and its fail message contains the connection error
