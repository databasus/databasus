## ADDED Requirements

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
