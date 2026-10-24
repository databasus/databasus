# Backup Streaming Specification

## Purpose

Defines how backup and restore data moves between a database and a storage, so that the memory and disk Databasus needs do not grow with the size of the database or with the speed difference between the two ends.

## Requirements

### Requirement: Memory per operation does not depend on the backup size

Backup and restore data SHALL move as a stream of bounded chunks, and the producer SHALL NOT run ahead of the consumer by more than a small fixed window. The memory one backup or restore uses SHALL be the same for a 500 MB and a 500 GB database. A slow, throttled or stalled storage SHALL slow the backup down instead of exhausting the host's memory.

#### Scenario: Storage slower than the dump

- **WHEN** a database dumps faster than the storage accepts data
- **THEN** the dump is slowed to the storage's pace
- **AND** the memory used by the backup stays within the fixed window

#### Scenario: Several backups at once

- **WHEN** ten backups run at the same time
- **THEN** their memory together is about ten windows, regardless of the databases' sizes

### Requirement: Backup data does not land on the Databasus host's disk

A backup SHALL stream from the database to the storage, and a restore from the storage to the database, without writing the backup data to the Databasus host's disk. A database larger than the host's free disk SHALL be backed up, and restored by every restore except the parallel one below.

Two cases are exempt. First, WAL segments received through a WAL stream SHALL wait in a local queue until they are uploaded. Second, a PostgreSQL restore with parallel jobs needs the archive as a file. Before such a restore, Databasus SHALL check that the disk has room for the whole archive, and SHALL refuse the restore with a clear message when it does not.

#### Scenario: Database larger than the free disk

- **WHEN** a 2 TB database is backed up from a host with 100 GB of free disk
- **THEN** the backup completes

#### Scenario: Parallel restore without disk space

- **WHEN** a user starts a parallel restore of a backup larger than the free disk
- **THEN** the restore is refused before it starts
- **AND** the message says how much space the restore needs
