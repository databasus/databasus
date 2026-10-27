## MODIFIED Requirements

### Requirement: Backup data does not land on the Databasus host's disk

A backup SHALL stream from the database to the storage, and a restore from the storage to the database, without writing the backup data to the Databasus host's disk. A database larger than the host's free disk SHALL be backed up, and restored by every restore except the file-based ones below.

Two cases are exempt. First, WAL segments received through a WAL stream SHALL wait in a local queue until they are uploaded. Second, a PostgreSQL restore needs the archive as a file when it runs with parallel jobs, excludes extensions or skips user mappings, because the tool reads such an archive more than once. Before such a restore, Databasus SHALL check that the disk has room for the whole archive, and SHALL refuse the restore with a clear message when it does not.

#### Scenario: Database larger than the free disk

- **WHEN** a 2 TB database is backed up from a host with 100 GB of free disk
- **THEN** the backup completes

#### Scenario: Parallel restore without disk space

- **WHEN** a user starts a parallel restore of a backup larger than the free disk
- **THEN** the restore is refused before it starts
- **AND** the message says how much space the restore needs

#### Scenario: A restore that skips user mappings without disk space

- **WHEN** a user starts a single-job restore of a PostgreSQL backup whose database skips user mappings, and the backup is larger than the free disk
- **THEN** the restore is refused before it starts
- **AND** the message says how much space the restore needs
