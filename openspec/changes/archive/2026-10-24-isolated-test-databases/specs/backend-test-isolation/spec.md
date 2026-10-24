## Purpose

Defines how backend tests get their internal databases and how test runs from several worktrees share one machine without breaking each other.

## ADDED Requirements

### Requirement: A test gets its own migrated internal database

A backend test that asks for an internal database SHALL get a new database that no other test uses, with every migration of its checkout already applied. Two tests running at the same time, in one run or in runs from different worktrees, SHALL NOT see each other's rows. When the migrations of a checkout change, the next test SHALL get a database with the changed schema without any separate preparation step. Two worktrees with different migrations SHALL each get databases with their own schema, even when their tests run at the same time.

#### Scenario: Two tests write the same row

- **WHEN** two tests running at the same time each create an account with the same email in their internal database
- **THEN** both creations succeed and neither test sees the other's account

#### Scenario: A new migration appears

- **WHEN** a developer adds a migration and runs a single backend test package
- **THEN** the test's database already has the table the new migration creates

#### Scenario: Two worktrees with different migrations

- **WHEN** a worktree with an extra migration and a worktree without it run backend tests at the same time
- **THEN** the databases of the first have the table that migration creates and the databases of the second do not

### Requirement: A failed test keeps its database for inspection

When a test that got its own internal database passes, its database SHALL be removed when the test ends. When the test fails, its database SHALL be kept, and the test output SHALL show the address to connect to it: host, port, user and database name, without the password.

#### Scenario: Test passes

- **WHEN** a test that got its own database passes
- **THEN** the database no longer exists after the test ends

#### Scenario: Test fails

- **WHEN** a test that got its own database fails
- **THEN** the database still exists after the run
- **AND** the test output shows the address of that database without a password

### Requirement: Abandoned test databases are removed

Test databases are the per-test and per-process copies. A test database that has no connections and was created more than an hour ago SHALL be removed by the next test process that asks for a test database, once per process. The migrated templates the copies are made from SHALL NOT be removed. A test database that has an open connection SHALL NOT be removed, whatever its age. Databases on the test server that are not test databases SHALL NOT be removed.

#### Scenario: A killed run left databases behind

- **WHEN** a test run was killed two hours ago and left its test databases, and a new test package starts
- **THEN** those databases are removed

#### Scenario: A long test is still connected

- **WHEN** a test database was created more than an hour ago and a test still holds a connection to it
- **THEN** another test process does not remove it

### Requirement: Tests of code outside the target layout get a database per test process

A test of backend code outside the target layout, which reaches the internal database through the process-wide database access, SHALL get one migrated internal database for its whole test process. Two such test processes running at the same time SHALL use different databases. The number of test packages that run at once SHALL NOT be limited by a number of prepared databases.

#### Scenario: Eight packages run at once

- **WHEN** eight backend test packages of old code run at the same time
- **THEN** each package's process uses a database no other process uses

#### Scenario: More packages than before

- **WHEN** a developer runs backend tests with twelve packages at once
- **THEN** every package gets its database and none waits for another to finish

### Requirement: Resetting the shared database servers does not break running tests

Test runs share three long-lived test PostgreSQL servers (one for logical backups, two for physical backups), and a full run first resets them: it drops the databases, roles and replication slots that tests left there. The reset SHALL start only after every test process on the machine that uses those servers has finished. A test process that starts while the reset runs SHALL wait for the reset to finish before it uses those servers. When the reset cannot start within 600 seconds because test processes keep using the servers, the full run SHALL stop with a non-zero status and a message that says other test runs are using the shared servers, and SHALL NOT reset them. Servers of other engines are not shared between runs and are not reset.

#### Scenario: An agent's package run is in progress

- **WHEN** a full run starts while a test package of another worktree is using the shared servers
- **THEN** the reset starts only after that package's process has ended
- **AND** that package's tests are not affected by the reset

#### Scenario: Another run keeps the servers longer than the limit

- **WHEN** a full run starts while another worktree's test process keeps using the shared servers for longer than 600 seconds
- **THEN** the full run exits with a non-zero status, prints that other test runs are using the shared servers, and does not reset them

#### Scenario: A package starts during the reset

- **WHEN** a test package starts while the reset runs
- **THEN** its tests use the shared servers only after the reset has finished
