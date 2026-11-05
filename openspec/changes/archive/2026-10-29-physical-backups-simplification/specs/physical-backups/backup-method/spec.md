## MODIFIED Requirements

### Requirement: Physical backup traffic is compressed on the source

A full or incremental backup SHALL be compressed by the source server before it crosses the network, so a large database takes hours rather than days over a slow link. Every backup SHALL first ask the source for zstd compression. When the source cannot compress with zstd, Databasus SHALL fall back to gzip and then to no compression within the same backup. Databasus SHALL NOT remember the compression that worked, so a source rebuilt with zstd support gets zstd on its next backup.

#### Scenario: Source built without zstd

- **WHEN** the source rejects zstd compression
- **THEN** the backup is taken with gzip compression instead of failing
- **AND** the next backup of that database tries zstd first again, so a source rebuilt with zstd support gets zstd
