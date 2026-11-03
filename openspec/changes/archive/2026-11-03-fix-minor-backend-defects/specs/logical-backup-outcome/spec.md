## ADDED Requirements

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
