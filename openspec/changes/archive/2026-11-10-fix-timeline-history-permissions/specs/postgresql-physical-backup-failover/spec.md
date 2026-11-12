## ADDED Requirements

### Requirement: Promotion recovery works with the provisioned backup role

On PostgreSQL 17 and 18, a source standby promoted while Databasus uses its provisioned backup role SHALL have its new timeline history archived and cataloged. The replacement FULL and contiguous WAL after it SHALL restore committed rows from before promotion, after promotion and after that FULL. Recovery SHALL remain on the replacement FULL's timeline.

#### Scenario: Recovery after promotion on each supported version

- **WHEN** Databasus provisions the backup role on a primary, that role is replicated to its standby, physical backup and WAL reception use the role, and the standby is promoted
- **THEN** Databasus archives the new timeline's history and completes a replacement FULL
- **AND** PostgreSQL started from the downloaded restore set contains committed rows written before promotion, after promotion and after the replacement FULL
- **AND** no administrative or server-file read privileges are added to the backup role

### Requirement: A FULL can archive history independently of WAL reception

A successful FULL on a timeline above 1 SHALL attempt to acquire and publish that timeline's history even when no local WAL receiver history notification exists. Failure of that additional history operation SHALL NOT invalidate an otherwise valid backup artifact. The archive-based recovery prerequisite is defined by the physical backup operations specification.

#### Scenario: FULL-only backup after promotion

- **WHEN** a FULL runs on a promoted timeline with provisioned credentials and no continuous WAL receiver
- **THEN** Databasus acquires and publishes the timeline history
- **AND** the FULL can be downloaded and started as a restored PostgreSQL cluster

#### Scenario: A valid FULL completes while history acquisition fails

- **WHEN** the FULL artifact is valid but the separate history acquisition fails
- **THEN** the FULL remains completed and can be selected by its backup identifier
- **AND** that completion does not establish availability of archived WAL recovery without the required history
