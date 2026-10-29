## ADDED Requirements

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
