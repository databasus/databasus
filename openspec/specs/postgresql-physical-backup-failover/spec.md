# PostgreSQL Physical Backup Failover Specification

## Purpose

Keep PostgreSQL physical backups recoverable when the source moves to a new timeline. Databasus rejects incremental backups that would extend an obsolete chain, creates a replacement FULL, and preserves the WAL needed during the transition.

## Requirements

### Requirement: Incremental backups use their root FULL timeline
The system SHALL start an incremental backup only when the live PostgreSQL timeline matches the timeline of the chain's root FULL.

#### Scenario: Timeline matches the root FULL
- **WHEN** an incremental backup starts and the live timeline matches its root FULL timeline
- **THEN** the system proceeds with the incremental backup

#### Scenario: Promotion happened before the incremental started
- **WHEN** the live timeline is newer than the root FULL timeline
- **THEN** the system refuses the incremental backup as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`

#### Scenario: Source timeline regressed
- **WHEN** the live timeline is older than the root FULL timeline
- **THEN** the system refuses the incremental backup with reason `TIMELINE_REGRESSION`

#### Scenario: Cataloged history is newer than the root FULL
- **WHEN** a history file for the live timeline is already cataloged but the root FULL belongs to an older timeline
- **THEN** the system still refuses the incremental backup as `TIMELINE_SWITCH_DETECTED`

### Requirement: Backup failures are rechecked for promotion
The system SHALL check PostgreSQL cluster identity again when a physical backup stream fails unexpectedly while the backup is still active.

#### Scenario: Promotion interrupts an incremental stream
- **WHEN** the timeline advances after incremental preflight and the stream fails
- **THEN** the system records the incremental as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`

#### Scenario: Promotion interrupts a FULL stream
- **WHEN** the timeline advances after FULL preflight and the stream fails
- **THEN** the system records the FULL as `ERROR` with reason `FAILOVER_DURING_BACKUP`

#### Scenario: Backup was canceled
- **WHEN** an operator or configuration change cancels the backup
- **THEN** the system preserves the cancellation result and does not classify it as a promotion

#### Scenario: Recheck cannot connect
- **WHEN** the second identity check fails
- **THEN** the system preserves the original backup result

#### Scenario: Cluster identity changed
- **WHEN** the second identity check reaches a cluster with a different system identifier
- **THEN** the system rejects that cluster and pauses automatic backup scheduling until an operator requests a FULL that completes against the expected cluster

#### Scenario: Operator requests a FULL after cluster identity changed
- **WHEN** automatic scheduling is paused by a system identifier mismatch and an operator requests a FULL
- **THEN** the system runs preflight again and refuses to write a backup unless the live system identifier matches the configured database

#### Scenario: Forced FULL still reaches a different cluster
- **WHEN** the operator-requested FULL is refused with `SYSTEM_IDENTIFIER_MISMATCH`
- **THEN** automatic backup scheduling remains paused

#### Scenario: Forced FULL fails without proving cluster identity
- **WHEN** an operator-requested FULL fails for another reason or is canceled
- **THEN** automatic backup scheduling remains paused because no later FULL has completed against the expected cluster

#### Scenario: Forced FULL reaches the expected cluster
- **WHEN** an operator-requested FULL completes against the configured system identifier
- **THEN** that newer completed FULL supersedes earlier mismatch results and normal automatic scheduling resumes

### Requirement: Timeline failures trigger one immediate replacement FULL
The system SHALL create one immediate replacement FULL after a backup ends with a timeline-related failure.

#### Scenario: Incremental chain broke after promotion
- **WHEN** an incremental ends as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED` and no later FULL attempt exists
- **THEN** the system schedules a FULL without waiting for the configured incremental or FULL interval

#### Scenario: FULL was interrupted by promotion
- **WHEN** the latest FULL ends as `ERROR` with reason `FAILOVER_DURING_BACKUP`
- **THEN** the system schedules one replacement FULL without waiting for the configured interval

#### Scenario: Replacement FULL fails normally
- **WHEN** the immediate replacement FULL ends with a non-timeline error or is canceled
- **THEN** the system resumes its existing schedule and failed-FULL retry limits instead of scheduling a FULL on every scheduler tick

#### Scenario: Another promotion interrupts the replacement
- **WHEN** the timeline advances again during the replacement FULL
- **THEN** the system schedules one new immediate replacement FULL for the later promotion

#### Scenario: Two schedulers hold the same stale replacement decision
- **WHEN** two scheduler instances decide that the same terminal backup needs an immediate replacement
- **THEN** only the instance that confirms the failure is still current creates a FULL row

### Requirement: Active FULL backups retain archived WAL
The system SHALL keep orphan cleanup from deleting WAL for a database while a FULL backup for that database is active.

#### Scenario: Cleaner runs during the first FULL
- **WHEN** WAL is archived while the database's first FULL is still running and orphan cleanup starts
- **THEN** the WAL remains cataloged and stored for use after the FULL completes

#### Scenario: Cleaner finishes before a FULL starts
- **WHEN** orphan cleanup starts before the scheduler starts a FULL for the same database
- **THEN** the FULL starts only after cleanup finishes

### Requirement: Same-timeline PITR remains available during a newer FULL
The system SHALL allow a point-in-time restore from an older FULL when the target comes before a newer FULL completes on the same timeline.

#### Scenario: Target falls during the newer FULL
- **WHEN** the requested target is later than the older FULL completion and earlier than the newer FULL completion
- **THEN** the restore uses the older FULL and contiguous WAL through the target, even when that WAL crosses the newer FULL start position

#### Scenario: Target crosses a promotion
- **WHEN** reaching the target would require WAL from more than one timeline
- **THEN** the system reports that the target is not restorable until a FULL on the promoted timeline is available

### Requirement: An immediate replacement FULL starts on the next pass that evaluates its database

While no system identifier mismatch is unresolved, an immediate replacement FULL of a database with no pending backup request SHALL start on the first full scheduler pass that finds the earlier backup's process ended, without waiting for a configured interval. A full scheduler pass starts 15 seconds after the previous one ended. A pass of the backup request job SHALL NOT evaluate such a database, because a replacement FULL writes no backup request. While no system identifier mismatch is unresolved, a database with a pending incremental request SHALL get the replacement FULL on the first pass of the backup request job that finds that process ended, because that pass evaluates the database with the same decision as a full scheduler pass, and that decision prefers the replacement FULL to the incremental request. A database with a pending full request SHALL get exactly one FULL, on such a pass, and the request SHALL be cleared. While a system identifier mismatch is unresolved, no replacement FULL SHALL start.

#### Scenario: A replacement FULL with no pending request

- **WHEN** an incremental ends as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`, no later FULL attempt exists, the database has no pending backup request and a pass of the backup request job runs
- **THEN** no backup starts
- **AND** the next full scheduler pass starts the replacement FULL without waiting for the configured incremental or FULL interval

#### Scenario: A replacement FULL after an interrupted FULL with no pending request

- **WHEN** the latest FULL ends as `ERROR` with reason `FAILOVER_DURING_BACKUP`, the database has no pending backup request and a pass of the backup request job runs
- **THEN** no backup starts
- **AND** the next full scheduler pass starts the replacement FULL

#### Scenario: A replacement FULL with a pending incremental request

- **WHEN** an incremental ends as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`, no later FULL attempt exists, the database has a pending incremental request and a pass of the backup request job runs
- **THEN** the replacement FULL starts on that pass

#### Scenario: A timeline failure with a pending full request

- **WHEN** an incremental ends as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`, no later FULL attempt exists, the database has a pending full request and a pass of the backup request job runs
- **THEN** exactly one new FULL starts on that pass, and the full request is cleared

#### Scenario: A failover during a FULL with an unresolved mismatch

- **WHEN** the latest FULL ends as `ERROR` with reason `FAILOVER_DURING_BACKUP`, a system identifier mismatch is unresolved, the database has a pending incremental request, and a pass of the backup request job and a full scheduler pass run
- **THEN** no backup starts
