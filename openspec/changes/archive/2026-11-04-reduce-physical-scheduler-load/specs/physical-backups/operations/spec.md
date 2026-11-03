## ADDED Requirements

### Requirement: A pending backup request is evaluated without waiting for a full scheduler pass

The backup request job SHALL evaluate each pending backup request with the same decision as a full scheduler pass, whether the request came from a manual start, a WAL slot rebuild or a storage switch. A request the decision accepts SHALL start its backup on the first pass of the backup request job that finds the database free. While the process of an earlier backup of that database is running, no backup SHALL start for the request, and the first pass after that process has ended SHALL apply the decision to it. An incremental request SHALL be cleared when incremental backups are not enabled or the database has no extendable chain, and it SHALL stay pending while a system identifier mismatch is unresolved. The job SHALL act only on databases with a pending request, whose backups are turned on and whose backup config names a storage. A backup that is due only by its schedule SHALL start on the next full scheduler pass, which starts 15 seconds after the previous one ended.

#### Scenario: A requested full backup between full scheduler passes

- **WHEN** a member requests a full backup of a physical database and only the backup request job runs afterwards
- **THEN** the full backup starts, and it appears in the list

#### Scenario: A requested incremental backup between full scheduler passes

- **WHEN** a member requests an incremental backup of a physical database with an extendable chain and only the backup request job runs afterwards
- **THEN** an incremental backup on that chain starts, and it appears in the list

#### Scenario: A full request recorded by a WAL slot rebuild

- **WHEN** a WAL slot rebuild has recorded a full request for a database and only the backup request job runs afterwards
- **THEN** the full backup starts, and it appears in the list

#### Scenario: A full request recorded by a storage switch

- **WHEN** a member switches the storage of a database with backups turned on and only the backup request job runs afterwards
- **THEN** a full backup in the new storage starts, and it appears in the list

#### Scenario: A full request filed during a running backup

- **WHEN** a member requests a full backup while the process of an earlier backup of the same database is running, and a pass of the backup request job runs
- **THEN** no second backup starts, and the request stays pending
- **AND** after that process has ended, the next pass starts the requested backup

#### Scenario: A schedule that is due without a request

- **WHEN** the full cadence of a database becomes due, the database has no pending request, and only the backup request job runs
- **THEN** no backup starts until the next full scheduler pass

#### Scenario: An incremental request that the decision clears

- **WHEN** a database whose backup config does not enable incremental backups has a pending incremental request and no backup due, and a pass of the backup request job runs
- **THEN** no backup starts, and the request is gone

#### Scenario: A requested incremental backup with an unresolved mismatch

- **WHEN** a database with an extendable chain has an unresolved system identifier mismatch and a pending incremental request, and a pass of the backup request job runs
- **THEN** no backup starts, and the request stays pending
- **AND** a requested full backup of that database starts on the next pass

#### Scenario: A request of a database with backups turned off

- **WHEN** a database has a pending full request and its backups are turned off, and a pass of the backup request job runs
- **THEN** no backup starts, and the WAL stream statistics still report a full backup as pending
- **AND** after the backups are turned on, the next pass starts the requested backup

#### Scenario: A request of a database with no storage

- **WHEN** a database has a pending full request and its backup config names no storage, and a pass of the backup request job runs
- **THEN** no backup starts, and the WAL stream statistics still report a full backup as pending
