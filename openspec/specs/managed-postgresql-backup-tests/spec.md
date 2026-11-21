# Managed PostgreSQL Backup Tests Specification

## Purpose

Checks Databasus backups and restores against a real managed PostgreSQL service, Neon, through the HTTP API, so that behavior that containers cannot reproduce stays verified.

## Requirements

### Requirement: A Neon backup and restore cycle runs through the API

With a Neon connection string available, an end-to-end test SHALL run the application in the backend test harness and drive it only through its HTTP API, without the interface. For a source database it creates in the Neon project, holding several tables with rows, a foreign key, a function and a view that calls the function, the test SHALL:

- pass a connection test of the source's settings;
- complete a logical backup as the role of the connection string;
- create a limited user through the API, then find that the API's limited-user check for that user's settings reports no privileges beyond what a backup needs, and complete a logical backup as that user;
- restore each backup into a new database in the same project and over the source database, through the API;
- after each restore, find the same rows, view results and sequence positions as in the source, and fail naming each value that differs.

The same cycle SHALL also run against a PostgreSQL server in a container of the test harness, so that the cycle's own setup, comparison and cleanup are verified on every run without Neon. Both tests SHALL belong to the logical backup end-to-end tests, so that `mise run test-full` runs them. `mise run test` SHALL NOT run them.

#### Scenario: The cycle passes

- **WHEN** the test runs with a reachable Neon project
- **THEN** every backup completes, every restore completes and every restored database holds the source's data

#### Scenario: A restored database differs from the source

- **WHEN** the comparison finds a differing row, view result or sequence position after a restore
- **THEN** the test fails with a message naming each differing value

#### Scenario: The cycle runs without Neon

- **WHEN** `mise run test-full` runs without a Neon connection string
- **THEN** the cycle still runs against the container PostgreSQL server and passes

### Requirement: Concurrent runs share one Neon project without interference

Several local worktrees and both CI architectures SHALL be able to run the test at once against one Neon project. Each run SHALL use only databases and roles it created. A database SHALL carry the test's prefix, a run-unique part and its creation time in its name. A role SHALL carry the same marker in its name or, when Databasus chose its name, in its comment. When its test ends, whether it passed or failed, a run SHALL remove the databases and roles it created, and SHALL fail naming each object it could not remove. A run killed before its cleanup leaves its objects to later runs: before its own work, a run SHALL remove databases and roles carrying the test's marker whose creation time is older than one hour, reading the time from the name or the comment, and SHALL leave younger ones alone. The test SHALL NOT change or remove any database or role without the test's marker.

#### Scenario: Two runs overlap

- **WHEN** two runs of the test execute at the same time against one Neon project
- **THEN** both pass and neither reads, restores into or removes the other's databases or roles

#### Scenario: A killed run left objects behind

- **WHEN** a run starts while the project holds a marked database and a marked role created two hours earlier, and another of each created five minutes earlier
- **THEN** the run removes the older database and role and keeps the younger ones

#### Scenario: Objects without the marker

- **WHEN** a run starts while the project holds a database and a role created long ago without the test's marker
- **THEN** the run leaves both unchanged

#### Scenario: A run ends

- **WHEN** a run's test finishes, whether it passed or failed
- **THEN** the project holds no database or role that this run created

#### Scenario: Cleanup cannot remove an object

- **WHEN** a run cannot drop one of its databases or roles at the end
- **THEN** the test fails with a message naming that database or role

### Requirement: A missing Neon connection string skips locally and fails where it is required

The Neon test SHALL read the Neon connection string from the environment variable `TEST_NEON_DATABASE_URL`; `mise run test-full` SHALL provide it from `.env` like the other test settings. When the variable is empty or missing, the Neon test SHALL skip with a message that names the variable. When `TEST_IS_NEON_REQUIRED` is `true`, a missing or empty connection string SHALL fail the test instead. An unreachable project or refused credentials SHALL fail the test whether or not the string is required; they SHALL NOT be reported as a skip.

#### Scenario: A local run without the variable

- **WHEN** a developer runs `mise run test-full` without `TEST_NEON_DATABASE_URL` in `.env`
- **THEN** the Neon test is reported as skipped with a message naming `TEST_NEON_DATABASE_URL`
- **AND** the run's other suites are unaffected

#### Scenario: A required run without the variable

- **WHEN** the test runs with `TEST_IS_NEON_REQUIRED=true` and no connection string
- **THEN** the test fails with a message naming `TEST_NEON_DATABASE_URL`

#### Scenario: Wrong credentials

- **WHEN** the connection string holds a password the server refuses
- **THEN** the test fails rather than skips

### Requirement: The Neon password never reaches test output

Neither the full connection string nor its password SHALL appear in test output, failure messages, test logs or files the run keeps. The host, the user and the database name MAY appear, because a reader needs them to tell which project failed. The documentation of the test SHALL say that the project must be a dedicated throwaway project holding no other data.

#### Scenario: A connection is refused

- **WHEN** the test connects with a password the server refuses
- **THEN** its failure message contains neither that password nor the full connection string
