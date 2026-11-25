## MODIFIED Requirements

### Requirement: A Neon backup and restore cycle runs through the API

With a Neon connection string available, an end-to-end test SHALL run the application in the backend test harness and drive it only through its HTTP API, without the interface. For a source database it creates in the Neon project, holding several tables with rows, a foreign key, a function and a view that calls the function, the test SHALL:

- pass a connection test of the source's settings;
- complete a logical backup as the role of the connection string;
- create a limited user through the API, then find that the API's limited-user check for that user's settings reports no privileges beyond what a backup needs, and complete a logical backup as that user;
- restore each backup into a new database in the same project and over the source database, through the API;
- after each restore, find the same rows, view results and sequence positions as in the source, and fail naming each value that differs.

The same cycle SHALL also run against a PostgreSQL server in a container of the test harness, so that the cycle's own setup, comparison and cleanup are verified on every run without Neon. Both tests SHALL belong to the logical backup end-to-end tests, so that `mise run test` runs them, and a test plan that selects those tests runs them too.

#### Scenario: The cycle passes

- **WHEN** the test runs with a reachable Neon project
- **THEN** every backup completes, every restore completes and every restored database holds the source's data

#### Scenario: A restored database differs from the source

- **WHEN** the comparison finds a differing row, view result or sequence position after a restore
- **THEN** the test fails with a message naming each differing value

#### Scenario: The cycle runs without Neon

- **WHEN** `mise run test` runs without a Neon connection string
- **THEN** the cycle still runs against the container PostgreSQL server and passes

#### Scenario: A test plan selects the logical PostgreSQL backup tests

- **WHEN** a change's test plan selects `backend ./internal/modules/backups/e2e/postgresql/...` and `mise run test-change` runs it without `TEST_NEON_DATABASE_URL`
- **THEN** the container cycle runs and passes, and the Neon test is reported as skipped

### Requirement: A missing Neon connection string skips locally and fails where it is required

The Neon test SHALL read the Neon connection string from the environment variable `TEST_NEON_DATABASE_URL`; `mise run test` and `mise run test-change` SHALL provide it from `.env` like the other test settings. When the variable is empty or missing, the Neon test SHALL skip with a message that names the variable. When `TEST_IS_NEON_REQUIRED` is `true`, a missing or empty connection string SHALL fail the test instead. An unreachable project or refused credentials SHALL fail the test whether or not the string is required; they SHALL NOT be reported as a skip.

#### Scenario: A local run without the variable

- **WHEN** a developer runs `mise run test` without `TEST_NEON_DATABASE_URL` in `.env`
- **THEN** the Neon test is reported as skipped with a message naming `TEST_NEON_DATABASE_URL`
- **AND** the run's other suites are unaffected

#### Scenario: A required run without the variable

- **WHEN** the test runs with `TEST_IS_NEON_REQUIRED=true` and no connection string
- **THEN** the test fails with a message naming `TEST_NEON_DATABASE_URL`

#### Scenario: Wrong credentials

- **WHEN** the connection string holds a password the server refuses
- **THEN** the test fails rather than skips
