## ADDED Requirements

### Requirement: An immediate replacement FULL starts on the next pass that evaluates its database

While no system identifier mismatch is unresolved, an immediate replacement FULL of a database with no pending backup request SHALL start on the first full scheduler pass that finds the earlier backup's process ended, without waiting for a configured interval. A full scheduler pass starts 15 seconds after the previous one ended. A pass of the backup request job SHALL NOT evaluate such a database, because a replacement FULL writes no backup request. While no system identifier mismatch is unresolved, a database with a pending incremental request SHALL get the replacement FULL on the first pass of the backup request job that finds that process ended, because that pass evaluates the database with the same decision as a full scheduler pass, and that decision prefers the replacement FULL to the incremental request. A database with a pending full request SHALL get exactly one FULL, on such a pass, and the request SHALL be cleared. While a system identifier mismatch is unresolved, no replacement FULL SHALL start.

#### Scenario: A replacement FULL with no pending request

- **WHEN** an incremental ends as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`, no later FULL attempt exists, the database has no pending backup request and a pass of the backup request job runs
- **THEN** no backup starts
- **AND** the next full scheduler pass starts the replacement FULL without waiting for the configured incremental or FULL interval

#### Scenario: A replacement FULL after an interrupted FULL with no pending request

- **WHEN** the latest FULL ends as `ERROR` with reason `FAILOVER_DURING_BACKUP`, the database has no pending backup request and a pass of the backup request job runs
- **THEN** no backup starts
- **AND** the next full scheduler pass starts the replacement FULL

#### Scenario: A replacement FULL with a pending incremental request

- **WHEN** an incremental ends as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`, no later FULL attempt exists, the database has a pending incremental request and a pass of the backup request job runs
- **THEN** the replacement FULL starts on that pass

#### Scenario: A timeline failure with a pending full request

- **WHEN** an incremental ends as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`, no later FULL attempt exists, the database has a pending full request and a pass of the backup request job runs
- **THEN** exactly one new FULL starts on that pass, and the full request is cleared

#### Scenario: A failover during a FULL with an unresolved mismatch

- **WHEN** the latest FULL ends as `ERROR` with reason `FAILOVER_DURING_BACKUP`, a system identifier mismatch is unresolved, the database has a pending incremental request, and a pass of the backup request job and a full scheduler pass run
- **THEN** no backup starts
