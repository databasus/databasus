# physical-backups/background-load Specification

## Purpose
Bounds the catalog work that physical backup scheduling and retention do in the background, so an idle instance stays idle no matter how many chains and how much archived WAL it retains, while every scheduling, retention and restore decision stays the same.

## Requirements

### Requirement: A scheduling pass with nothing due does not read chain history

When the scheduler evaluates a physical database and neither a full backup, an incremental backup nor an out-of-cadence request is due, it SHALL read only the newest full backup record, the newest incremental backup record and the system identifier mismatch check of that database, and SHALL NOT read archived WAL segments or build any chain. The work of such a pass SHALL NOT grow with the number of retained chains or archived WAL segments.

#### Scenario: Idle database with weeks of retained WAL

- **WHEN** a physical database retains many chains and thousands of archived WAL segments, and no backup is due
- **THEN** the scheduler's pass for that database reads no archived WAL segment and builds no chain
- **AND** its reads are limited to the newest full backup, the newest incremental backup and the system identifier mismatch check

#### Scenario: Incremental cadence becomes due

- **WHEN** the incremental cadence of a database with an extendable chain becomes due
- **THEN** the scheduler starts an incremental backup on the newest chain, as it does today
- **AND** it decides this without reading the chain's archived WAL segments

### Requirement: Only the newest completed full backup can head an extendable chain

A chain SHALL be extendable only when its full backup is the newest successfully completed full backup of the database and none of the chain's incremental backups ended with a broken chain. Every older chain SHALL be closed by its newer full backup. WAL gaps inside a chain SHALL NOT change its state.

#### Scenario: Newest chain is healthy

- **WHEN** the newest completed full backup has no incremental backup with a broken chain
- **THEN** that chain is the extendable chain
- **AND** every older chain is reported as closed

#### Scenario: Newest chain is broken by an incremental backup

- **WHEN** an incremental backup of the newest chain ended with a broken chain
- **THEN** the database has no extendable chain
- **AND** the newest chain is listed with the closed chains that retention and restore consider

#### Scenario: Newest chain has a WAL gap

- **WHEN** the newest chain is missing a WAL segment in the middle of its span
- **THEN** the chain stays extendable

### Requirement: Retention runs once a minute and reads each closed chain's WAL once

The retention cleaner SHALL run once a minute. To decide which closed chains are eligible for deletion and in what order, including the grace check, it SHALL read each closed chain's archived WAL at most once per pass, as a summary rather than as individual segment records, and SHALL NOT read WAL outside that chain's span. Deleting an eligible chain and sweeping WAL that belongs to no chain are outside this rule and keep reading individual segments.

#### Scenario: Idle instance with retained chains

- **WHEN** a retention pass runs and no chain is eligible for deletion
- **THEN** no individual archived WAL segment record is loaded to decide that

#### Scenario: A chain becomes eligible for deletion

- **WHEN** a closed chain falls outside the configured retention and outside the grace period
- **THEN** its deletion begins within about one minute

### Requirement: A chain's end time is unchanged

A chain's end time, which drives retention ordering and the grace period, SHALL be the latest of three moments: the completion of its full backup, the completion of any of its incremental backups, and the receipt of any archived WAL segment inside its span.

#### Scenario: WAL arrives after the last incremental backup

- **WHEN** a chain's newest archived WAL segment was received after its last incremental backup completed
- **THEN** the chain's end time is the receipt time of that segment

#### Scenario: Chain without archived WAL

- **WHEN** a chain has no archived WAL segment in its span
- **THEN** the chain's end time is the latest completion time among its full and incremental backups

### Requirement: A chain's WAL lookup returns exactly the chain's segments

Looking up the archived WAL of a chain SHALL return every segment that overlaps the chain's span, ordered by position, and SHALL NOT read segments that end at or before the span begins. A segment that starts before the span but carries data from the span's start SHALL be included.

#### Scenario: Older chain among many

- **WHEN** the WAL of a chain in the middle of a database's history is looked up
- **THEN** the result holds that chain's segments only
- **AND** no segment of an earlier chain is read

#### Scenario: Full backup starts in the middle of a segment

- **WHEN** a chain's full backup starts partway into an archived segment
- **THEN** that segment is included in the chain's WAL

#### Scenario: Newest chain

- **WHEN** the WAL of the newest chain is looked up
- **THEN** the result runs from the segment holding the chain's start to the newest archived segment

### Requirement: Scheduling queries do not grow with the number of databases when nothing is due

A full scheduler pass SHALL start 15 seconds after the previous one ended. When no database has a backup due and no backup request is pending, the number of queries that pass sends to the internal database SHALL be the same for any number of databases. A pass of the backup request job SHALL start one second after the previous one ended, and it SHALL send one query when no backup request is pending, whatever the number of databases. Within one pass, the decision for a database SHALL depend only on that database's own config and history, and the full, incremental or no-backup outcome SHALL follow the existing decision rules. Instance health SHALL report both intervals.

#### Scenario: More databases with nothing due

- **WHEN** a full scheduler pass runs with two physical databases and later with five, and none of them has a backup due or a request pending
- **THEN** both passes send the same number of queries to the internal database
- **AND** neither pass starts a backup

#### Scenario: More databases with no pending request

- **WHEN** a pass of the backup request job runs with two physical databases and later with five, and none of them has a request pending
- **THEN** each pass sends exactly one query to the internal database
- **AND** neither pass starts a backup

#### Scenario: One pass decides for databases with different histories

- **WHEN** one full scheduler pass evaluates a database whose full cadence is due, a database whose full cadence is due and which has an unresolved system identifier mismatch, and a database with nothing due and no pending request
- **THEN** a full backup starts for the first database, and no backup starts for the other two

#### Scenario: A resolved mismatch does not block another database

- **WHEN** one full scheduler pass evaluates a database whose mismatch was followed by a completed full backup and a database whose mismatch was not, and the full cadence of both is due
- **THEN** a full backup starts for the first database only

#### Scenario: Instance health reports both intervals

- **WHEN** a client reads the instance health with `GET /api/v1/system/health`
- **THEN** the job `backups_physical_scheduler` reports an interval of 15 seconds and the job `backups_physical_request_scheduler` reports an interval of 1 second
