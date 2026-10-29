# Backup Verification Specification

## Purpose

Defines the relationship between on-demand backup verification and the configuration that controls automatic verification runs.

## Requirements

### Requirement: Manual verification is independent of automatic scheduling

The system SHALL allow an authorized user to enqueue and run a manual backup verification while automatic verification is disabled.

#### Scenario: Enqueue manual verification while automatic verification is disabled

- **WHEN** an authorized user requests manual verification of an eligible backup whose automatic verification is disabled and no other manual verification for that database is pending or running
- **THEN** the system accepts the request and keeps the manual verification pending for an agent

#### Scenario: Disable automatic verification while a manual verification is pending

- **WHEN** automatic verification is disabled for a database with a pending manual verification
- **THEN** the manual verification remains pending

#### Scenario: Disable automatic verification while a manual verification is running

- **WHEN** automatic verification is disabled for a database with a running manual verification
- **THEN** the manual verification remains running and can reach its normal terminal status

### Requirement: Disabling automatic verification cancels only automatic runs

The system SHALL cancel non-terminal automatic verification runs when automatic verification is disabled without canceling manual verification runs for the same database.

#### Scenario: Disable automatic verification with a time-based run

- **WHEN** automatic verification is disabled for a database with a pending or running time-based verification
- **THEN** the system cancels that automatic verification

#### Scenario: Disable automatic verification with an after-backup run

- **WHEN** automatic verification is disabled for a database with a pending or running after-backup verification
- **THEN** the system cancels that automatic verification

#### Scenario: Manual and automatic runs coexist when scheduling is disabled

- **WHEN** automatic verification is disabled for a database with both manual and automatic non-terminal verifications
- **THEN** the system cancels the automatic verifications and leaves the manual verifications unchanged

### Requirement: A verification accepts only ignorable errors that say nothing about the backup

An ignorable error is an error that the PostgreSQL restore tool reported and skipped, and that still lets the restore count as successful. When the restore tool of a verification exits with an error, the verification agent SHALL count the restore as successful only if two conditions hold. First, the tool reports how many errors it skipped, and it listed exactly that many errors. Second, every listed error is an ignorable error of a verification: the `public` schema that already exists in the restore target, or an extension that the verification environment cannot create, together with the comment on that extension. The agent SHALL judge by the tool's whole error output, however long it is.

When the restore tool fails because the restore ran out of the job's disk budget, the agent SHALL report the failure without the tool's exit code, so the backend retries the verification instead of rejecting the backup, as before this change. Any other failure that is not made only of ignorable errors, a count that does not match the listed errors, or a missing count SHALL fail the verification with the tool's exit code.

A missing extension is an ignorable error of a verification because the verification environment does not carry the extensions of the source. A restore into the user's database does not treat it as ignorable, as the `database-tool-failure-output` spec states.

#### Scenario: Only extensions the environment lacks

- **WHEN** the restore tool of a verification exits with an error, reports 4 skipped errors and lists two extensions that are not available, each with the failed comment on it
- **THEN** the verification completes and collects the statistics of the restored database

#### Scenario: Only the existing public schema

- **WHEN** the restore tool of a verification exits with an error, reports 1 skipped error and lists only that the `public` schema already exists
- **THEN** the verification completes

#### Scenario: A missing extension that the data depends on

- **WHEN** the restore tool reports that an extension is not available, and then that a type from it and a table using that type do not exist
- **THEN** the verification fails with the tool's exit code

#### Scenario: The count does not match the listed errors

- **WHEN** the restore tool reports 3 skipped errors but lists only 2, both about an extension that is not available
- **THEN** the verification fails with the tool's exit code

#### Scenario: Ignorable errors that make the output long

- **WHEN** the restore tool lists more than 8 KiB of errors about extensions that are not available and reports their exact count
- **THEN** the verification completes

#### Scenario: A non-zero exit without a count

- **WHEN** the restore tool exits with an error and does not report how many errors it skipped
- **THEN** the verification fails with the tool's exit code

#### Scenario: The restore runs out of the job's disk budget

- **WHEN** the restore tool exits with an error that reports no space left on the device of the restore container
- **THEN** the agent reports the failure without the tool's exit code, and the backend retries the verification instead of rejecting the backup

### Requirement: Removing a backup removes its verifications

Whenever a logical backup is removed, whether a member deletes it, retention drops it, its database's backup config moves to another storage or its database is deleted, its verifications SHALL be removed in the same step, whatever their status, a running one included. When that removal is refused, the backup and all its verifications SHALL stay. Deleting a database SHALL also remove its verification config.

#### Scenario: A member deletes a verified backup

- **WHEN** a member deletes a logical backup that has a completed verification, a running one and a pending one
- **THEN** the response is `204`, and none of the three verifications is listed for the database any more

#### Scenario: Retention drops a verified backup

- **WHEN** a database's retention no longer keeps a backup that has a verification
- **THEN** after the next retention check the backup is not listed, and its verification is no longer listed for the database

#### Scenario: The storage of a verified backup changes

- **WHEN** a member saves a database's backup config with another storage while one of its backups has a verification
- **THEN** the save succeeds, the backup is not listed, and its verification is no longer listed for the database

#### Scenario: A database with verifications is deleted

- **WHEN** a member deletes a database that has a verification config and verified backups
- **THEN** the response is `204`, and no verification or verification config of the database remains

#### Scenario: A refused deletion keeps the verifications

- **WHEN** a member deletes a database while one of its logical backups is in progress, and another backup of it has a verification
- **THEN** the response is `409` with `code` `in_use`, the backup is still listed, the verification is still listed for the database, and the database's verification config is still returned

### Requirement: Only an instance admin manages verification agents

Creating, listing, rotating the token of and deleting verification agents SHALL be allowed only to an instance admin. Any other signed-in user SHALL get `403` with `code` `permission_denied`, a request without a session SHALL get `401` with `code` `unauthenticated`, and neither SHALL change anything. Creating an agent and rotating its token SHALL return the agent's token once, in that response only. No other response SHALL carry a token or its hash. After a rotation the previous token SHALL stop authenticating the agent. A deleted agent SHALL no longer be listed, its token SHALL stop authenticating it, and a verification it was running SHALL be handed back for another agent while attempts remain, as when an agent goes silent. Agents SHALL be listed newest first, one page at a time, and a cursor the list did not issue SHALL get `400` with `code` `invalid_request`.

#### Scenario: An admin creates an agent

- **WHEN** an instance admin creates an agent with a name
- **THEN** the response is `201` with the agent and its token, and the agent is listed without a token

#### Scenario: Paging the agents

- **WHEN** an admin lists three agents two per page
- **THEN** the first page holds the two newest and a cursor, and the second page holds the oldest and no cursor

#### Scenario: A request without a session

- **WHEN** a request without a session creates or lists agents
- **THEN** each response is `401` with `code` `unauthenticated`, and no agent is added

#### Scenario: A workspace owner manages agents

- **WHEN** a workspace owner who is not an instance admin creates, lists, rotates the token of or deletes an agent
- **THEN** each response is `403` with `code` `permission_denied`, the list an admin reads is unchanged, and the agent's heartbeat with its current token still succeeds

#### Scenario: A rotated token

- **WHEN** an admin rotates an agent's token and the agent then sends a heartbeat with the previous token and with the new one
- **THEN** the heartbeat with the previous token is refused with `401`, and the one with the new token succeeds

#### Scenario: A deleted agent

- **WHEN** an admin deletes an agent that is running a verification
- **THEN** the agent is no longer listed, its next heartbeat is refused with `401`, and after the scheduler's next pass the verification is pending again with one more attempt

#### Scenario: An unknown agent

- **WHEN** an admin rotates the token of or deletes an agent that does not exist or was deleted
- **THEN** the response is `404` with `code` `not_found`

### Requirement: Any signed-in user sees whether agents exist

Any signed-in user SHALL be able to read how many verification agents exist, so the interface can say whether a verification can run. The answer SHALL carry only the number, and nothing that identifies an agent.

#### Scenario: A member reads the number of agents

- **WHEN** a member who is not an instance admin reads the number of agents while two agents exist and a third was deleted
- **THEN** the answer is `200` with the number 2 and no agent's ID, name or token

### Requirement: A database has one verification config

Reading the verification config of a database SHALL return the saved config, or the default when none was saved: automatic verification off, run after each backup, a weekly interval schedule on Sunday at 04:00, and only the failure notification. Saving SHALL replace every editable field: whether automatic verification is on, whether it runs after each backup or on an interval schedule, the schedule, and which outcomes send a notification. A save with automatic verification on and the interval type SHALL need a valid schedule, and a save with an invalid schedule, schedule type or notification SHALL get `400` with `code` `invalid_request` and the field in `errors`, and SHALL change nothing.

#### Scenario: A database without a saved config

- **WHEN** a member reads the verification config of a database whose config was never saved
- **THEN** the answer is the default config

#### Scenario: Saving a config

- **WHEN** a member saves a config with automatic verification on, a weekly interval schedule and both notifications, and reads it again
- **THEN** the save answers `200` with the saved config, and the read returns the same values

#### Scenario: An invalid schedule

- **WHEN** a member saves a config with automatic verification on, the interval type and a cron schedule whose expression does not parse
- **THEN** the response is `400` with `code` `invalid_request` and the schedule in `errors`, and the config is unchanged

### Requirement: A manual verification starts only for a completed backup

Starting a manual verification of a logical backup SHALL answer `202` with the verification pending, without waiting for an agent. A backup that is not completed SHALL get `409` with `code` `backup_not_completed`. A second manual verification of a database while one is pending or running SHALL get `409` with `code` `already_running`. Neither refusal SHALL create a verification.

#### Scenario: Starting a verification

- **WHEN** a member starts a verification of a completed backup
- **THEN** the response is `202` with a pending manual verification of that backup, and reading it returns the same

#### Scenario: A backup that is not completed

- **WHEN** a member starts a verification of a failed backup
- **THEN** the response is `409` with `code` `backup_not_completed`, and the database's list of verifications is unchanged

#### Scenario: A manual verification is already pending

- **WHEN** a member starts a manual verification of a database that already has a pending manual one
- **THEN** the response is `409` with `code` `already_running`, and only the first verification is listed

### Requirement: A pending or running verification can be cancelled

Cancelling a pending or running verification SHALL answer `200` with the verification cancelled. When an agent runs it, the agent's next heartbeat that reports it as current SHALL list it among the verifications to abort. Cancelling a verification that is completed, failed or cancelled SHALL get `409` with `code` `not_running` and SHALL change nothing.

#### Scenario: Cancelling a running verification

- **WHEN** a member cancels a verification that an agent is running
- **THEN** the response is `200` with the verification cancelled, and the agent's next heartbeat that reports the verification as current lists it among the verifications to abort

#### Scenario: Cancelling a finished verification

- **WHEN** a member cancels a completed verification
- **THEN** the response is `409` with `code` `not_running`, and the verification is still completed

### Requirement: The verifications of a database are listed newest first

The verifications of a database SHALL be listed newest first, one page at a time, with a cursor that leads to the next page and is absent on the last one. A list entry SHALL carry the verification's status, trigger, attempts, times, durations and results, without the row counts of its tables. Reading one verification SHALL also return the row count of each table the agent counted. A cursor the list did not issue SHALL get `400` with `code` `invalid_request`.

#### Scenario: Paging the verifications

- **WHEN** a member lists the verifications of a database that has three, two per page
- **THEN** the first page holds the two newest and a cursor, and the second page holds the oldest and no cursor

#### Scenario: Table counts of one verification

- **WHEN** a member reads a completed verification after its agent reported two tables
- **THEN** the answer holds both tables with their row counts, and the list entry of the same verification holds none

#### Scenario: A foreign cursor

- **WHEN** a member lists the verifications of a database with a cursor the list did not issue
- **THEN** the response is `400` with `code` `invalid_request`
