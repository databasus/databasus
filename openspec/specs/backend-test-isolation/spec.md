# backend-test-isolation Specification

## Purpose
Defines how backend tests get their internal databases and how test runs from several worktrees share one machine without breaking each other.

## Requirements

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

Test databases are the copies allocated to individual tests. A test database that has no connections and was created more than an hour ago SHALL be removed by the next test process that asks for a test database, once per process. The migrated templates the copies are made from SHALL NOT be removed. A test database that has an open connection SHALL NOT be removed, whatever its age. Databases on the test server that are not test databases SHALL NOT be removed.

#### Scenario: A killed run left databases behind

- **WHEN** a test run was killed two hours ago and left its test databases, and a new test package starts
- **THEN** those databases are removed

#### Scenario: A long test is still connected

- **WHEN** a test database was created more than an hour ago and a test still holds a connection to it
- **THEN** another test process does not remove it

### Requirement: Test runs preserve unrelated database sources

Local and CI test runs SHALL NOT reset shared source servers or remove databases, roles, permissions or replication slots belonging to another test, worktree or development session. Source fixtures that a test changes SHALL belong to that test or use uniquely named objects with cleanup limited to those objects. The existing cleanup of abandoned internal test databases remains governed by its separate requirement.

#### Scenario: A local run starts beside a development session
- **WHEN** a local backend run starts while a development source holds a database, a role and a replication slot
- **THEN** preparing the run leaves those objects unchanged and issues no shared-source reset

#### Scenario: CI prepares a backend selection
- **WHEN** CI prepares a backend test selection
- **THEN** it does not run a shared-source reset or wait for fixed source ports that selection does not use

#### Scenario: Seed tests overlap
- **WHEN** two development-seed tests run concurrently using their own backend instances and source fixtures
- **THEN** both can create their configured backups without changing the other test's source or internal data

### Requirement: The shared test server gives up durability

The shared PostgreSQL server that holds internal test databases SHALL run without flushing its writes to durable storage, as the engine test containers do. A crash of that server or of the host MAY destroy its contents. Recreating the server with an empty data directory SHALL be enough to recover: the next test SHALL get a migrated database without any separate preparation step. The development database SHALL keep its durability.

#### Scenario: The test server runs without durability

- **WHEN** the shared test server is started from the compose file
- **THEN** `SHOW fsync`, `SHOW synchronous_commit` and `SHOW full_page_writes` each return `off`

#### Scenario: New settings keep the existing databases

- **WHEN** the running server holds a migrated template and `docker compose up -d backend-db-test` recreates it with a changed compose command
- **THEN** the template still exists and `SHOW fsync` returns `off`

#### Scenario: The test server is recreated

- **WHEN** the shared test server is recreated with an empty data directory and a developer runs a backend test package
- **THEN** the test gets a database with every migration of its checkout applied

#### Scenario: The development database is unaffected

- **WHEN** a developer starts the local environment
- **THEN** `SHOW fsync` on the development database returns `on`
