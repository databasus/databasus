## MODIFIED Requirements

### Requirement: Backup data does not land on the Databasus host's disk

A user-database backup SHALL stream from the database to the storage, and a restore from the storage to the database, without writing the backup data to the Databasus host's disk. A user database larger than the host's free disk SHALL be backed up, and restored by every restore except the file-based ones below.

Three cases are exempt. First, WAL segments received through a WAL stream SHALL wait in a local queue until they are uploaded. Second, a PostgreSQL restore needs the archive as a file when it runs with parallel jobs, excludes extensions or skips user mappings, because the tool reads such an archive more than once. Before such a restore, Databasus SHALL check that the disk has room for the whole archive, and SHALL refuse the restore with a clear message when it does not. Third, a backup of Databasus's own internal state SHALL be allowed to stage an encrypted archive for independent multi-destination delivery and retries. This third exception SHALL never stage plaintext, SHALL retain bounded memory use and SHALL report insufficient disk space without publishing a partial backup; temporary archives SHALL be cleaned after delivery or their bounded retry lifetime.

#### Scenario: Database larger than the free disk

- **WHEN** a 2 TB user database is backed up from a host with 100 GB of free disk
- **THEN** the backup completes

#### Scenario: Parallel restore without disk space

- **WHEN** a user starts a parallel restore of a backup larger than the free disk
- **THEN** the restore is refused before it starts
- **AND** the message says how much space the restore needs

#### Scenario: A restore that skips user mappings without disk space

- **WHEN** a user starts a single-job restore of a PostgreSQL backup whose database skips user mappings, and the backup is larger than the free disk
- **THEN** the restore is refused before it starts
- **AND** the message says how much space the restore needs

#### Scenario: Databasus backup delivery is retried

- **WHEN** a completed encrypted Databasus backup file needs another delivery attempt
- **THEN** the system can reuse its encrypted temporary file without rerunning the dump or staging plaintext

#### Scenario: Databasus backup staging runs out of disk

- **WHEN** the host cannot finish the temporary Databasus backup file
- **THEN** the run reports failure, publishes no partial archive and releases its temporary file
