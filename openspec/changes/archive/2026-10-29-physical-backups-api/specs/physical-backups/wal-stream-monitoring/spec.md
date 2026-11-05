## MODIFIED Requirements

### Requirement: Only users who can view the database can read its WAL stream statistics

The WAL stream statistics of a database and the current values of its WAL stream SHALL be returned only to an authenticated user who can view that database. A user who is neither a member of the database's workspace nor an instance admin SHALL get `404` with `code` `not_found`, the same answer as for a database that does not exist. A request without a valid session SHALL get `401` with `code` `unauthenticated`. The statistics and the current values SHALL contain no connection details, credentials, error messages from the source or WAL contents.

#### Scenario: Member of the database's workspace

- **WHEN** a member of the workspace that owns the database requests its statistics and its current values
- **THEN** both are returned

#### Scenario: User from another workspace

- **WHEN** a user without access to the database's workspace requests its statistics or its current values
- **THEN** the response is `404` with `code` `not_found`
- **AND** no statistics or values are returned

#### Scenario: Unauthenticated request

- **WHEN** a request without valid authentication asks for statistics or current values
- **THEN** the response is `401` with `code` `unauthenticated`

#### Scenario: Nonexistent database

- **WHEN** an authenticated user requests statistics or current values for a database ID that does not exist
- **THEN** the response is `404` with `code` `not_found`
- **AND** no statistics or values are returned

### Requirement: The API serves the current values every second

A second authenticated endpoint SHALL return the current values of a database's running WAL stream, read from the running streamer at the time of the request and not stored: the time of the reading, the local queue and its pause size, whether receiving runs, the archive progress point, the time the newest WAL segment was archived, and how much is not saved to storage yet. The values from the source database, its current position, the slot's status, the slot lag and the remaining allowance, SHALL come from the newest probe of the source that the streamer already makes, at least every 10 seconds, together with the time of that probe; the endpoint SHALL not open a connection to the source itself. Only users who can view the database SHALL read it, with the answers of the requirement on reading WAL stream statistics. While no streamer runs for the database, the response SHALL say so and carry no values.

#### Scenario: Buffer drains between two samples

- **WHEN** the local queue drains from 300 MB to 0 B within five seconds
- **THEN** a request one second later reports 0 B in the local queue

#### Scenario: Source values follow the probe

- **WHEN** the newest source probe ran 7 seconds ago
- **THEN** the source-side values are those of that probe, with its time

#### Scenario: Saving moves past the probed position

- **WHEN** segments reached storage after the newest source probe, past the source position it found
- **THEN** the response says that nothing is waiting to be saved, not that the amount is unknown

#### Scenario: Streamer not running

- **WHEN** WAL streaming is off for the database
- **THEN** the response says that no streamer runs and carries no values

#### Scenario: Member without access

- **WHEN** a user who is not a member of the database's workspace and not an instance admin requests its current values
- **THEN** the response is `404` with `code` `not_found`, as the requirement on reading WAL stream statistics states
