## ADDED Requirements

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
