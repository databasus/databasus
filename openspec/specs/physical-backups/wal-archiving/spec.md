# WAL Archiving Specification

## Purpose

Defines how finalized WAL segments of a streaming PostgreSQL database reach storage, how uploads overlap and respond to cancellation, and when Databasus reports archiving as broken or fallen behind.

## Requirements

### Requirement: Up to four segments of one database upload at the same time

For a database in WAL streaming mode, Databasus SHALL upload up to four finalized WAL segments at the same time. It SHALL NOT upload more than four segments of one database at once. Segments waiting in the local queue SHALL start uploading in ascending order of timeline first and WAL position second. Segments recovered after a restart and segments set aside from the receiver's resume path SHALL go through the same limit.

#### Scenario: Slow storage with a queued backlog

- **WHEN** eight finalized segments wait in the local queue and the first four uploads are held in progress
- **THEN** four segments are uploading at once
- **AND** the remaining four wait until an upload slot becomes available

#### Scenario: Limit is never exceeded

- **WHEN** fifty finalized segments wait in the local queue
- **THEN** at no moment are more than four of that database's segments uploading

#### Scenario: Start order follows WAL position

- **WHEN** segments 10, 11, 12, 13 and 15 wait in the local queue, segment 14 waits among the segments set aside from the receiver's resume path, and one upload slot frees up after 10 to 13 have started
- **THEN** segment 14 starts next
- **AND** segment 15 starts only after segment 14

### Requirement: A segment is uploaded by at most one upload at a time

Databasus SHALL NOT start a second upload of a segment while an upload of that segment is still running. A segment whose upload failed SHALL stay in the local queue and SHALL be retried on a later pass.

#### Scenario: Queue is scanned while a slow upload runs

- **WHEN** a segment's upload is still running and the next scan of the local queue finds that segment's file
- **THEN** no second upload of that segment starts
- **AND** no "held by another in-flight claim" warning is logged for it

#### Scenario: Failed upload is retried

- **WHEN** a segment's storage write fails
- **THEN** the segment's file stays in the local queue
- **AND** a later pass uploads it again under a new object name

### Requirement: Storage accepts overlapping segment writes

For a storage of type S3-compatible, Azure Blob, SFTP, FTP, SMB or local, four writers SHALL be able to consume independent upload streams while all four writes remain unfinished. Each writer SHALL store a 16 MiB segment-sized file followed by its 1 KiB metadata file without mixing, truncating or overwriting another writer's content. Completion SHALL NOT depend on meeting an elapsed-time or transfer-rate threshold.

#### Scenario: Four uploads overlap

- **WHEN** four writers start independent segment uploads and each upload source is held before providing its content
- **THEN** all four writes request content before any source is released
- **AND** after the sources are released, every segment and metadata file can be read back with its original content

### Requirement: Consecutive writes reuse storage connections

For a storage of type S3-compatible, Azure Blob, SFTP, FTP or SMB, consecutive successful writes using unchanged settings SHALL reuse healthy connections. FTP data connections are separate from its reusable control connection.

#### Scenario: Metadata files follow a successful write

- **WHEN** a write establishes a healthy connection and ten metadata files are then written sequentially with unchanged settings
- **THEN** no additional storage session or HTTP connection is opened for those writes

### Requirement: SFTP streams keep blocks in flight

An SFTP upload from a source whose length is unknown SHALL continue consuming source blocks while an earlier write acknowledgement is pending. The resulting file SHALL retain the complete original content.

#### Scenario: Acknowledgement is withheld

- **WHEN** an SFTP upload has opened its remote file and the response to its first data block is withheld
- **THEN** the upload consumes another source block before that response is released
- **AND** after responses resume, the uploaded file can be read back with its original content

### Requirement: Storage connections opened with old settings are never used again

A write SHALL use only a connection opened with the storage's current settings, and it SHALL authenticate with the storage's current credentials. After a storage's connection settings or credentials change, the next write SHALL use the new ones. A connection that has been idle for five minutes SHALL be closed. A deleted storage SHALL receive no further writes.

#### Scenario: Credentials are changed to wrong ones

- **WHEN** a write to a storage succeeds, the storage's password is then changed to a wrong one, and another write follows
- **THEN** the second write fails with an authentication error

#### Scenario: Credentials are corrected

- **WHEN** a write fails because of wrong credentials and the credentials are then corrected
- **THEN** the next write succeeds

#### Scenario: Idle connection

- **WHEN** no write reaches a storage for five minutes
- **THEN** every connection Databasus kept open to that storage is closed

#### Scenario: Storage is deleted

- **WHEN** a write to a storage succeeds and the storage is then deleted
- **THEN** a further write for that storage fails
- **AND** no connection to that storage stays open five minutes after the last write

### Requirement: A storage that stops answering does not hold an upload

For a storage of type S3-compatible, Azure Blob, SFTP, FTP or SMB, cancellation of an upload whose storage has stopped answering SHALL make the write fail without waiting for the storage response. The connection it used SHALL be closed. A later write SHALL NOT reuse that connection. Cancellation checks SHALL observe operation completion rather than enforce a machine-dependent elapsed-time threshold.

#### Scenario: Storage goes silent mid-write

- **WHEN** a storage response is withheld while a segment is being written and the upload is cancelled
- **THEN** the write fails while the response remains withheld
- **AND** the upload slot is free for the next segment
- **AND** the next write to that storage opens a new connection

### Requirement: A WAL gap is reported only for a segment that will not arrive

When a segment is archived and the archived segment just before it on the same timeline does not end where it starts, Databasus SHALL check the segment that ends where the archived segment starts. It SHALL report a WAL gap only if that segment is neither waiting as a full-size file in the local queue or among the segments set aside from the receiver's resume path, including while it uploads, nor recorded in the catalog as claimed or archived. A file shorter than a full segment does not count as waiting, because the receiver never streams a segment below its resume point again. When the hole spans several segments, only that one segment decides. The reported gap SHALL end at the start of the archived segment and SHALL start at the end of the nearest earlier segment on the same timeline that is archived or still waiting as a full-size file in either place. A gap SHALL be reported at most once per streaming session, even when segments around it finish out of order.

#### Scenario: Later segment finishes first

- **WHEN** segment 11 finishes uploading while segment 10 is still uploading
- **THEN** no WAL gap is reported
- **AND** no gap is reported after segment 10 finishes either

#### Scenario: Earlier segment failed and waits for a retry

- **WHEN** segment 11 is archived while segment 10's last upload attempt failed and its file still waits in the local queue
- **THEN** no WAL gap is reported

#### Scenario: Segment that was never received

- **WHEN** segment 20 is archived, segment 19 was never received, and segment 18 is the newest archived segment before it
- **THEN** one WAL gap notification is sent for the range between the end of segment 18 and the start of segment 20

#### Scenario: Segment before the hole is still uploading

- **WHEN** segment 20 is archived, segment 19 was never received, segment 18 is still uploading, and segment 17 is the newest archived segment before segment 20
- **THEN** one WAL gap notification is sent for the range between the end of segment 18 and the start of segment 20

#### Scenario: Segment before the gap left only a short file

- **WHEN** segment 21 is archived, segment 20 exists in the local queue only as a file shorter than a full segment, and segment 19 is the newest archived segment before it
- **THEN** one WAL gap notification is sent for the range between the end of segment 19 and the start of segment 21

#### Scenario: Hole spanning several segments

- **WHEN** segment 30 is archived, segment 27 is the newest archived segment before it, and neither segment 28 nor segment 29 is queued or in the catalog
- **THEN** one WAL gap notification is sent for the range between the end of segment 27 and the start of segment 30

#### Scenario: Two gaps finish out of order

- **WHEN** a gap before segment 40 is reported first and a gap before segment 30 is found afterwards
- **THEN** the gap before segment 30 is reported as well

### Requirement: Archiving counts as fallen behind when the gap-free archive stops advancing

Databasus SHALL track the archive progress point: the WAL position up to which every received segment is archived. It is the start of the oldest segment still uploading, waiting in the local queue or set aside from the receiver's resume path, or the end of the newest archived segment when no older segment is pending. A segment on an older timeline than the newest archived segment counts as older. A segment file counts as pending even when it is shorter than a full segment, because restore cannot pass that position until the segment is archived. Such a file is reported through this alert, not through a WAL gap notification. The "archiving fell behind" alert SHALL fire when the source keeps writing while the archive progress point does not advance for longer than the staleness threshold, even if newer segments keep being archived.

#### Scenario: One segment keeps failing while others succeed

- **WHEN** segment 50's upload fails on every attempt for longer than the staleness threshold, the source keeps writing, and segments 51 to 90 are archived in that time
- **THEN** the "archiving fell behind" alert fires

#### Scenario: Short segment file stays in the queue

- **WHEN** a segment file shorter than a full segment stays in the local queue for longer than the staleness threshold while the source keeps writing and later segments are archived
- **THEN** the "archiving fell behind" alert fires

#### Scenario: Out-of-order finishing within the threshold

- **WHEN** segments finish out of order but every pending segment is archived within a few seconds
- **THEN** the "archiving fell behind" alert does not fire

#### Scenario: Idle source

- **WHEN** the source writes nothing and no segment is pending
- **THEN** the "archiving fell behind" alert does not fire

### Requirement: The periodic archiving log reports upload timing

For each streaming database, the periodic archiving log entry SHALL report the following for the interval since the previous entry:

- the number of segments archived and their size;
- the average and the longest time of each upload step: claiming the segment, writing the segment, writing its metadata and committing;
- the highest number of uploads that ran at the same time.

It SHALL keep reporting how many bytes the archive is behind the source.

#### Scenario: Busy interval

- **WHEN** segments were archived since the previous entry
- **THEN** the entry states how many were archived and their size
- **AND** it states the average and the longest time of each upload step
- **AND** it states the highest number of uploads that ran at once

#### Scenario: Quiet interval

- **WHEN** no segment was archived since the previous entry
- **THEN** the entry reports no step timings
- **AND** it still states how many bytes the archive is behind the source
