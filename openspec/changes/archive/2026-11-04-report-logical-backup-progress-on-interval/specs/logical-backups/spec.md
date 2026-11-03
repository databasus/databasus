## ADDED Requirements

### Requirement: A backup refreshes its size and duration every 5 seconds while its dump streams

While the dump of a logical backup streams to the storage, Databasus SHALL refresh the size and the duration that reading the backup returns every 5 seconds, and SHALL NOT refresh them more often, apart from one refresh when the stream ends successfully. A refresh that falls due while the previous one is still being saved SHALL be skipped. The duration SHALL advance at every refresh, also when the dump produced no output since the previous one. Before the stream starts, and between the last refresh and the backup's outcome, the size and the duration SHALL NOT refresh. Until the first refresh, reading the backup SHALL return size 0 and duration 0. Copying the dump to the storage SHALL NOT wait for these refreshes to be saved. A periodic refresh that fails during the stream, because its save returns an error or because it panics, SHALL be logged and SHALL NOT fail the backup. A refresh SHALL NOT change the status or the size of a backup that is no longer running.

#### Scenario: A fast dump

- **WHEN** a dump produces many megabytes after a refresh and within the same 5-second interval
- **THEN** reading the backup during that interval returns the size and the duration of that refresh

#### Scenario: Before the first refresh

- **WHEN** a backup has been running for less than 5 seconds of streaming
- **THEN** reading the backup returns `IN_PROGRESS` with size 0 and duration 0

#### Scenario: A backup that finishes within one interval

- **WHEN** a backup completes before its first periodic refresh
- **THEN** reading the backup shows no intermediate value, only size 0 before the stream ends and the size the dump reached afterwards

#### Scenario: A dump that produces no output

- **WHEN** a streaming dump produces no output for longer than 5 seconds
- **THEN** the backup keeps its size, and its duration advances at every refresh

#### Scenario: A backup that completes between refreshes

- **WHEN** a backup completes after its last periodic refresh
- **THEN** reading the backup shows `COMPLETED` with the size the dump reached when it completed, not the size of the last periodic refresh

#### Scenario: A slow internal database

- **WHEN** saving a refresh takes longer than the rest of the dump
- **THEN** the dump still reaches the storage while the refresh is being saved

#### Scenario: A periodic refresh that fails

- **WHEN** a periodic refresh panics or its save returns an error during the dump
- **THEN** the failure is logged, and the stream of the backup ends without an error because of it

#### Scenario: A backup cancelled during its dump

- **WHEN** a user cancels a backup while its dump streams
- **THEN** the backup stays `CANCELED` with size 0, and no refresh changes either
