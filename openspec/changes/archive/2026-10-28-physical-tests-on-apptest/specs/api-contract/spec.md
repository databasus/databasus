## MODIFIED Requirements

### Requirement: Backend API tests run against the real application

A backend API test SHALL call an application built by the same code that builds the server, with the same routes and middleware, its own internal database and no background loops running on their own. Data one test writes SHALL NOT be visible to another test. While code outside the target layout is wired through process-wide state, the API tests of one package SHALL run one at a time: a second test application started in a process while one is alive SHALL fail that test with a message saying so. Test packages SHALL still run in parallel with each other.

Until physical backups become background jobs of the backups module, a test MAY ask its test application to run the physical backup scheduler and the WAL stream supervisor live, for as long as the application lives. Without that request, no physical backup starts on its own and no WAL stream runs in a test application. The physical backup cleaner SHALL NOT run on its own even then. A test SHALL be able to run one pass of the physical backup scheduler and one pass of the physical backup cleaner by the job names `backups_physical_scheduler` and `backups_physical_cleaner`, which they keep when they become jobs of the backups module, with or without the live loops. A pass of the scheduler SHALL return once it has started the backups it decided on, without waiting for them to end. When the test ends, the scheduler and the supervisor SHALL stop, and every physical backup and WAL stream they or a scheduler pass started SHALL have ended, before the application closes its internal database. A physical backup or a WAL stream that is still running at that point SHALL fail the test.

When a test restarts the modules of its application, the physical startup work SHALL run as it runs at the start of the server: physical backups left in progress are recorded as failed, WAL streamers whose heartbeat went stale are recorded as failed, and backup replication slots no backup holds are dropped on every physical source that answers. With the live loops, the restart SHALL stop the scheduler and the supervisor the way a stop of the server does, keeping the replication slots and the local WAL queue, and start them again, so every WAL stream begins a new session. A restart SHALL wait until every physical backup that is running has recorded its outcome, because until physical backups become jobs of the backups module a restart does not interrupt them.

A test MAY shorten, for its own application only, the interval of forced WAL rotation, the time after which a stalled WAL archive counts as stale, and the size of the local WAL queue at which the receiver pauses. The server SHALL always use the production values: a forced rotation every 5 minutes, an archive counted stale after 30 minutes, and a pause size derived from the WAL segment size.

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

#### Scenario: One scheduler pass starts a due backup

- **WHEN** a test without the live loops enables the backups of a physical database that has no full backup and runs one pass of the physical backup scheduler
- **THEN** right after the pass returns, the list holds a full backup of that database that is in progress or completed
- **AND** that backup reaches completed without another pass

#### Scenario: A restart fails a backup left in progress

- **WHEN** a test records a physical backup as in progress with SQL and restarts the modules of its application
- **THEN** the backup is listed as failed with a message saying the application restarted

#### Scenario: Shortened rotation applies only to the test

- **WHEN** a test builds its application with a forced WAL rotation interval of a few seconds and its database streams WAL from a source that writes rarely
- **THEN** after the initial full backup completes, a later WAL segment is archived within a bounded window tied to the shortened interval and far shorter than the production interval

#### Scenario: One cleaner pass removes an orphan WAL segment

- **WHEN** a test records with SQL a WAL segment of a physical database that lies outside every chain and runs one pass of `backups_physical_cleaner`
- **THEN** the segment is no longer listed

#### Scenario: A restart sweeps an orphan backup slot and a stale streamer

- **WHEN** a test creates a backup replication slot of its physical database on the source, records with SQL a WAL streamer as running with a heartbeat older than 90 seconds, and restarts the modules of its application
- **THEN** the slot is gone from the source and the streamer is no longer recorded as running

#### Scenario: A restart with the live loops keeps the slot and starts a new session

- **WHEN** a test with the live loops restarts the modules of its application while its database streams WAL
- **THEN** the replication slot of the stream is still on the source after the restart
- **AND** the stream statistics count one more streaming session start than before the restart

#### Scenario: The server uses the production timings

- **WHEN** the server builds its application from its configuration
- **THEN** its WAL stream timings are a forced rotation every 5 minutes, an archive counted stale after 30 minutes and a pause size derived from the WAL segment size
