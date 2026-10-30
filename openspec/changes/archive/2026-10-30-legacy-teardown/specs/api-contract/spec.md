## MODIFIED Requirements

### Requirement: Backend API tests run against the real application

A backend API test SHALL call an application built by the same code that builds the server, with the same routes and middleware, its own internal database and no background loops running on their own. Data one test writes SHALL NOT be visible to another test. Multiple test applications SHALL be able to remain alive in the same process. Each SHALL own its internal database, filesystem directories, encryption key, clocks, mutable service state and lifecycle. Independent API tests SHALL run concurrently within a package as well as across packages. A test that intentionally changes process-wide environment settings or uses a shared external fixture MAY remain sequential.

A test MAY ask its test application to run the physical backup scheduler and the reconciliation of WAL streams live, as background jobs of the backups module, for as long as the application lives. Such an application SHALL run on the system clock, because WAL stream samples and their health are measured in real time, and it SHALL expose no clock the test can move. Without that request, the application runs on a clock the test moves, no physical backup starts on its own and no WAL stream runs. The physical backup cleaner SHALL NOT run on its own even then. A test SHALL be able to run one pass of the physical backup scheduler, of the physical backup cleaner and of the reconciliation of WAL streams by their job names, `backups_physical_scheduler` and `backups_physical_cleaner`, with or without the live jobs. One pass of the reconciliation of WAL streams, `backups_physical_wal_stream_manager`, SHALL run only with the live jobs; without them it SHALL fail the test with a message saying so, because a WAL stream measured on a clock the test moves would report a false health. A pass of the scheduler SHALL return once it has started the backups it decided on, without waiting for them to end. When the test ends, the live jobs SHALL stop, and every physical backup and WAL stream of the application SHALL have ended, before the application closes its internal database. A physical backup or a WAL stream that is still running at that point SHALL fail the test.

When a test restarts the modules of its application, the modules SHALL stop as they stop at a shutdown of the server and start as they start with the server. Physical backups still running are recorded as failed with the shutdown message, WAL streams stop and keep their replication slots and local WAL queue, physical backups left in progress are recorded as failed, WAL streamers whose heartbeat went stale are recorded as failed, and backup replication slots no backup holds are dropped on every physical source that answers. With the live jobs, every WAL stream begins a new session after the restart.

A test MAY shorten, for its own application only, the interval of forced WAL rotation, the time after which a stalled WAL archive counts as stale, and the size of the local WAL queue at which the receiver pauses. The server SHALL always use the production values: a forced rotation every 5 minutes, an archive counted stale after 30 minutes, and a pause size derived from the WAL segment size.

#### Scenario: A test reaches the server's router

- **WHEN** a test builds a test application and calls `GET /api/v1/system/version` through it
- **THEN** the response passes through the same middleware as on the server, so it carries an `X-Request-Id` header

#### Scenario: Two tests do not share data

- **WHEN** one test creates a row in its test application's internal database and a later test of the same package builds its own test application
- **THEN** the later test's database does not hold that row

#### Scenario: Two test applications at once

- **WHEN** a test builds a test application while another one is alive in the same process
- **THEN** both applications serve requests successfully and each reads only its own accounts and workspace data

#### Scenario: Physical backups do not start without the request

- **WHEN** a test builds a test application without asking for the live physical jobs and enables the backups of a physical database
- **THEN** no physical backup starts and no WAL stream runs for that database

#### Scenario: Live physical loops back up a database

- **WHEN** a test builds a test application with the live physical jobs and enables the backups of a physical database
- **THEN** the first full backup starts without any further request and completes

#### Scenario: Live physical loops stop with the test

- **WHEN** a test that asked for the live physical jobs ends while its database streams WAL
- **THEN** the WAL stream and the jobs stop before the test application closes its internal database, and a later test of the same package starts its own jobs without failing

#### Scenario: One scheduler pass starts a due backup

- **WHEN** a test without the live jobs enables the backups of a physical database that has no full backup and runs one pass of the physical backup scheduler
- **THEN** right after the pass returns, the list holds a full backup of that database that is in progress or completed
- **AND** that backup reaches completed without another pass

#### Scenario: A restart fails a backup left in progress

- **WHEN** a test records a physical backup as in progress with SQL and restarts the modules of its application
- **THEN** the backup is listed as failed with a message saying the application restarted

#### Scenario: Shortened rotation applies only to the test

- **WHEN** a test builds its application with a forced WAL rotation interval of a few seconds and its database streams WAL from a source that writes rarely
- **THEN** a WAL segment is archived within that interval

#### Scenario: Live jobs run on the system clock

- **WHEN** a test builds a test application with the live physical jobs and its database streams WAL for longer than two sample intervals
- **THEN** the stream statistics report samples taken at distinct times and a streamer that is running

#### Scenario: The cleaner does not run with the live jobs

- **WHEN** a test builds a test application with the live physical jobs, records with SQL a committed WAL segment of a physical database that lies outside every chain, and lets the live jobs run for longer than the cleaner's interval
- **THEN** the segment is still listed, and it is gone after one pass of `backups_physical_cleaner`

#### Scenario: One cleaner pass removes an orphan WAL segment

- **WHEN** a test records with SQL a committed WAL segment of a physical database that lies outside every chain and runs one pass of `backups_physical_cleaner`
- **THEN** the segment is no longer listed

#### Scenario: A restart sweeps an orphan backup slot and a stale streamer

- **WHEN** a test creates a backup replication slot of its physical database on the source, records with SQL a WAL streamer as running with a heartbeat older than 90 seconds, and restarts the modules of its application
- **THEN** the slot is gone from the source and the streamer is no longer recorded as running

#### Scenario: A restart with the live loops keeps the slot and starts a new session

- **WHEN** a test with the live physical jobs restarts the modules of its application while its database streams WAL
- **THEN** the replication slot of the stream is still on the source after the restart
- **AND** the stream statistics count one more streaming session start than before the restart

#### Scenario: The server uses the production timings

- **WHEN** the server builds its application from its configuration
- **THEN** its WAL stream timings are a forced rotation every 5 minutes, an archive counted stale after 30 minutes and a pause size derived from the WAL segment size

#### Scenario: A restart fails a running backup with the shutdown message

- **WHEN** a test restarts the modules of its application while a physical full backup runs
- **THEN** the backup is listed as failed with the fail message `backup interrupted by application shutdown`

#### Scenario: A manager pass stops the stream of a database whose backups were disabled

- **WHEN** a test with the live physical jobs disables the backups of a database that streams WAL and runs one pass of `backups_physical_wal_stream_manager`
- **THEN** the stream statistics report that no streamer has started for the database
- **AND** the replication slot of the stream is still on the source

#### Scenario: A manager pass needs the live jobs

- **WHEN** a test without the live physical jobs runs one pass of `backups_physical_wal_stream_manager`
- **THEN** the test fails with a message that the pass runs only with the live jobs

#### Scenario: Live jobs expose no clock

- **WHEN** a test builds a test application with the live physical jobs
- **THEN** the application offers no clock the test can move

#### Scenario: Concurrent applications write the same local filename

- **WHEN** two live test applications write different content under the same local storage filename
- **THEN** each application reads its own content and their staging, backup and WAL queue directories are distinct
- **AND** each application uses its own encryption key file

#### Scenario: One application ends before its neighbour

- **WHEN** one of two live test applications shuts down while the other continues to handle requests and write stored files
- **THEN** the remaining application keeps serving its own data and files
- **AND** shutting down the first application does not cancel the other's work or remove its files
