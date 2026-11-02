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

### Requirement: A backup waits for the storage after the dump tool exits

After the tool that produces a backup exits, the backup SHALL keep delivering the tool's remaining output to the storage until the storage has accepted it. A limit of 10 seconds SHALL apply only to waiting for more output from the tool, and the time the storage spends accepting data SHALL NOT count against it. A storage that takes longer than those 10 seconds SHALL NOT by itself fail the backup.

The other limits on a backup stay as they are. A storage error and the 23-hour limit of a logical backup SHALL still fail it. A full or incremental physical backup SHALL still fail as a stalled stream, whether or not its tool has exited, when the storage accepts no data for 10 minutes from the start of the backup or, once it has accepted data, for 60 seconds. These limits cover the delivery of all the backup's data, including what an encrypted backup still buffers when its tool exits. Once all the data is delivered, waiting for the storage to finish the upload has no such limit and ends with the storage's own result or a cancellation.

A process the tool left behind can keep the output open with no data arriving. The backup SHALL fail 10 seconds after the output goes quiet in that case, rather than wait until it is cancelled.

#### Scenario: Storage finishes long after a successful dump

- **WHEN** a logical dump tool exits successfully and the storage needs longer than 10 seconds to accept the remaining data
- **THEN** the backup waits for the storage
- **AND** the backup completes with all of the tool's output stored

#### Scenario: Physical backup on a slow storage

- **WHEN** the tool of a full or incremental physical backup exits successfully, the storage has already accepted data, and it needs between 10 and 60 seconds to accept the rest
- **THEN** the backup waits for the storage and completes

#### Scenario: Physical backup on a storage that accepts nothing

- **WHEN** the tool of a full or incremental physical backup has exited, the storage has already accepted data, and it then accepts no data for 60 seconds
- **THEN** the backup fails as a stalled stream

#### Scenario: Dump tool fails while the storage is slow

- **WHEN** a logical dump tool exits with a failure and the storage is still accepting its output
- **THEN** the backup fails with the tool's failure, not with a timeout

#### Scenario: A process left behind holds the output open

- **WHEN** the dump tool exits successfully and a process it started keeps the output open without writing to it
- **THEN** the backup fails about 10 seconds after the output goes quiet

#### Scenario: Backup cancelled while waiting for the storage

- **WHEN** a backup is cancelled after its dump tool exited and the storage is not accepting data
- **THEN** the backup ends as cancelled within 10 seconds, without waiting for the storage
