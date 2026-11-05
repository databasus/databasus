## MODIFIED Requirements

### Requirement: Backend API tests run against the real application

A backend API test SHALL call an application built by the same code that builds the server, with the same routes and middleware, its own internal database and no background loops running on their own. Data one test writes SHALL NOT be visible to another test. While code outside the target layout is wired through process-wide state, the API tests of one package SHALL run one at a time: a second test application started in a process while one is alive SHALL fail that test with a message saying so. Test packages SHALL still run in parallel with each other.

Until physical backups become background jobs of the backups module, a test MAY ask its test application to run the physical backup scheduler and the WAL stream supervisor live, for as long as the application lives. Without that request, no physical backup starts and no WAL stream runs in a test application. The physical backup cleaner SHALL NOT run on its own even then; a test runs one pass of it explicitly. When the test ends, the scheduler and the supervisor SHALL stop, and every physical backup and WAL stream they started SHALL have ended, before the application closes its internal database. A physical backup or a WAL stream that is still running at that point SHALL fail the test.

#### Scenario: A test reaches the server's router

- **WHEN** a test builds a test application and calls `GET /api/v1/system/version` through it
- **THEN** the response passes through the same middleware as on the server, so it carries an `X-Request-Id` header

#### Scenario: Two tests do not share data

- **WHEN** one test creates a row in its test application's internal database and a later test of the same package builds its own test application
- **THEN** the later test's database does not hold that row

#### Scenario: Two test applications at once

- **WHEN** a test builds a test application while another one is alive in the same process
- **THEN** that test fails with a message that API tests of one package run one at a time

#### Scenario: Physical backups do not start without the request

- **WHEN** a test builds a test application without asking for the live physical loops and enables the backups of a physical database
- **THEN** no physical backup starts and no WAL stream runs for that database

#### Scenario: Live physical loops back up a database

- **WHEN** a test builds a test application with the live physical loops and enables the backups of a physical database
- **THEN** the first full backup starts without any further request and completes

#### Scenario: Live physical loops stop with the test

- **WHEN** a test that asked for the live physical loops ends while its database streams WAL
- **THEN** the WAL stream and the loops stop before the test application closes its internal database, and a later test of the same package starts its own loops without failing
