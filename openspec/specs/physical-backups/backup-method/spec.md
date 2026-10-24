# Physical Backup Method Specification

## Purpose

Defines which PostgreSQL sources can be backed up physically, how Databasus reaches them, and which configurations it refuses instead of delivering a weaker backup than the user chose.

## Requirements

### Requirement: Physical backups require PostgreSQL 17 or later

Databasus SHALL accept a physical backup configuration only for a source running PostgreSQL 17 or later. Incremental backups, and combining them on restore, exist only from that release. For an older source, the connection setup SHALL refuse the configuration with a message that points to logical backups, which cover the same data.

#### Scenario: PostgreSQL 16 source

- **WHEN** a user sets up a physical backup of a PostgreSQL 16 source
- **THEN** the setup is refused
- **AND** the message says physical backups need PostgreSQL 17 or later and suggests a logical backup

#### Scenario: PostgreSQL 17 source

- **WHEN** a user sets up a physical backup of a PostgreSQL 17 source that meets the other preconditions
- **THEN** the setup is accepted

### Requirement: Incremental backups require WAL summarization on the source

When a user chooses a backup type with incremental backups, Databasus SHALL refuse to save the configuration until the source reports `summarize_wal = on`. The refusal SHALL show the command that enables summarization on a self-managed server, and the parameter to change on a managed one. When summarization is off, Databasus SHALL NOT replace incrementals with full backups plus WAL replay. That substitute would silently make every restore replay all WAL since the last full backup. A configuration with full backups only SHALL NOT be held to this requirement.

#### Scenario: Incrementals chosen while summarization is off

- **WHEN** a user chooses full and incremental backups for a source with `summarize_wal = off`
- **THEN** the configuration is not saved
- **AND** the user sees how to enable WAL summarization and can try again

#### Scenario: Full backups only

- **WHEN** a user chooses full backups only for a source with `summarize_wal = off`
- **THEN** the configuration is saved

### Requirement: Physical backups run remotely from Databasus

Databasus SHALL take physical backups and stream WAL from its own container over the PostgreSQL replication protocol, either directly or through an SSH tunnel to a jump host. It SHALL NOT require installing anything on the database host. Setting up a source SHALL need only a role with the replication privilege and a matching access rule on the source.

#### Scenario: Managed PostgreSQL

- **WHEN** a user sets up a physical backup of a managed PostgreSQL that forbids installing software on its host
- **THEN** the backup works with a replication role alone

#### Scenario: Source behind a jump host

- **WHEN** the source is reachable only from a jump host inside the user's network
- **THEN** the user configures an SSH tunnel to that host, and backups and the WAL stream run through it

### Requirement: Sources with custom tablespaces are refused

A physical backup streams one archive of the cluster to the storage without staging it on disk. PostgreSQL produces one more archive for each extra tablespace. Databasus SHALL therefore refuse a physical backup of a source that has any tablespace besides `pg_default` and `pg_global`, before any backup is recorded. The refusal SHALL name the tablespaces and suggest a logical backup, which handles tablespaces. While the source keeps such a tablespace, Databasus SHALL notify about the refusal at most once per database per 24 hours.

#### Scenario: Source with a custom tablespace

- **WHEN** a scheduled physical backup finds a tablespace named `fast_disk` on the source
- **THEN** no backup is recorded
- **AND** the user is notified that `fast_disk` prevents physical backups and that a logical backup can be used

#### Scenario: Repeated refusal

- **WHEN** the same source still has the tablespace on the next scheduled attempts within 24 hours
- **THEN** no further notification is sent in that period

### Requirement: Only completed WAL segments reach the storage

Databasus SHALL upload a WAL segment only after the source has completed it. A segment that is still being written SHALL stay on the Databasus host and SHALL NOT be uploaded, so every archived segment is final and never rewritten. The physical backup settings SHALL tell the user that, while the database is idle, `archive_timeout` on the source bounds how much recent WAL a source failure can lose.

#### Scenario: Idle database

- **WHEN** a database with a WAL stream writes little WAL and its current segment is not complete
- **THEN** that segment is not uploaded until the source completes it

#### Scenario: Archived segment is final

- **WHEN** a segment has been uploaded
- **THEN** Databasus never uploads another version of the same segment over it

### Requirement: Physical backup traffic is compressed on the source

A full or incremental backup SHALL be compressed by the source server before it crosses the network, so a large database takes hours rather than days over a slow link. When the source cannot compress with zstd, Databasus SHALL fall back to gzip and then to no compression, and SHALL remember for the database which compression worked.

#### Scenario: Source built without zstd

- **WHEN** the source rejects zstd compression
- **THEN** the backup is taken with gzip compression instead of failing
- **AND** the next backup of that database uses gzip without trying zstd first

### Requirement: Every physical backup has a valid backup manifest

Every full and incremental backup SHALL be stored together with a backup manifest that describes exactly the files of that backup. The next incremental backup and `pg_combinebackup` need this manifest. The manifest SHALL pass `pg_verifybackup` against the stored archive.

#### Scenario: Manifest of a full backup

- **WHEN** a full backup completes
- **THEN** its stored manifest lists every file of the archive with its size and checksum
- **AND** `pg_verifybackup` accepts the archive with that manifest

#### Scenario: Incremental backup uses the parent's manifest

- **WHEN** an incremental backup starts
- **THEN** it is taken against the manifest stored with its parent
