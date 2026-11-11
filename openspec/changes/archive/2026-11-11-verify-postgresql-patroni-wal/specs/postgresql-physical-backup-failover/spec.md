## MODIFIED Requirements

### Requirement: Same-timeline PITR remains available during a newer FULL

The system SHALL allow a point-in-time restore from an older FULL when the target comes before a newer FULL completes on the same timeline. It SHALL NOT use a FULL completed after the target to make that target reachable. A target after promotion and before the first FULL on the promoted timeline completes SHALL remain unreachable when reaching it from an eligible earlier FULL would require replay across timelines, including after the replacement FULL has completed.

#### Scenario: Target falls during the newer FULL

- **WHEN** the requested target is later than the older FULL completion and earlier than the newer FULL completion on the same timeline
- **THEN** the restore uses the older FULL and contiguous WAL through the target, even when that WAL crosses the newer FULL start position

#### Scenario: Target crosses a promotion

- **WHEN** reaching the target would require WAL from more than one timeline
- **THEN** the system reports that the target is not restorable

#### Scenario: A replacement FULL does not fill the earlier recovery gap

- **WHEN** a FULL on the promoted timeline has completed and a user requests a target between promotion and that FULL's completion that cannot be reached on one timeline
- **THEN** the request returns `409` with code `target_time_unreachable`
- **AND** no restore token is issued for that target

## ADDED Requirements

### Requirement: Patroni sources recover through an unchanged primary address

On PostgreSQL 17 and 18, Databasus SHALL support a source whose primary is selected by Patroni and exposed through a stable HAProxy address. When that address changes from the primary to its promoted physical replica, Databasus SHALL reconnect without an operator changing its address, restarting Databasus or restarting its backup modules. The same-cluster identity and existing incremental-chain and replacement-FULL policies SHALL apply after every switch.

#### Scenario: Initial recovery through the primary address

- **WHEN** a Patroni source has a completed FULL and contiguous archived WAL after it, received through the stable primary address
- **THEN** the downloaded recovery set starts an independent writable PostgreSQL cluster containing rows and schema changes represented only by that subsequent WAL

#### Scenario: Two consecutive planned switches

- **WHEN** Patroni performs two consecutive switchovers between the two source nodes and each switch follows a control transaction replayed by the future primary
- **THEN** the unchanged address reaches a different PostgreSQL node with a newer timeline after each switch, while the system identifier stays unchanged
- **AND** WAL reception resumes against that node without restarting Databasus or its backup modules
- **AND** an INCREMENTAL requested against the preceding timeline ends as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`, followed by a FULL under the existing immediate-replacement policy
- **AND** a restore from each replacement FULL and subsequent archived WAL includes control rows written before the switch, after the switch and after that FULL

### Requirement: Patroni crash recovery preserves transactions replayed by the elected replica

For a durability-enabled source on PostgreSQL 17 and 18, recovery after a Patroni failover SHALL preserve control transactions committed on the former primary and replayed by the replica before that replica is elected. This guarantee SHALL NOT include acknowledged transactions that never reached the elected replica.

#### Scenario: Abrupt loss after confirmed replica replay

- **WHEN** a control transaction is confirmed on the future primary, the old primary node stops abruptly, and Patroni elects that replica
- **THEN** Databasus reconnects through the same address and follows its existing timeline and replacement-FULL policy
- **AND** an independent restore from the promoted source's replacement FULL and subsequent WAL preserves the control transaction, committed schema changes and usable generated IDs
- **AND** the former primary remains unavailable during recovery

### Requirement: Patroni switches during transfer preserve physical backup failure semantics

On PostgreSQL 17 and 18, a Patroni switch while a physical backup is actively transmitting SHALL NOT publish a partial copy as a completed backup. The interrupted operation SHALL reach a terminal result and follow the existing identity-recheck, retry and replacement-FULL policies. Detecting an active backup record alone SHALL NOT establish that the switch interrupted data transfer.

#### Scenario: A switchover interrupts a FULL transfer

- **WHEN** a FULL is transferring data, Patroni switches the primary before that transfer completes, and the failure recheck observes the newer timeline
- **THEN** the old FULL ends as `ERROR` with reason `FAILOVER_DURING_BACKUP`
- **AND** the existing policy schedules one immediate replacement FULL whose downloaded recovery set starts a writable cluster with the control rows and schema
- **AND** the partial old copy cannot be selected as a completed backup

#### Scenario: A switchover interrupts an INCREMENTAL transfer

- **WHEN** an INCREMENTAL is transferring data, Patroni switches the primary before that transfer completes, and the failure recheck observes the newer timeline
- **THEN** the old INCREMENTAL ends as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`
- **AND** the existing policy schedules one immediate replacement FULL whose downloaded recovery set starts a writable cluster with the control rows and schema
- **AND** the partial old copy cannot be selected as a completed backup

#### Scenario: The failure recheck runs during the routing outage

- **WHEN** a Patroni switch interrupts a transfer and its failure recheck cannot connect to a source
- **THEN** the operation preserves its original failure rather than claiming a verified timeline change
- **AND** once the failed worker has released its reservation and the new primary is reachable, no additional immediate FULL is requested for that generic failure
- **AND** with no competing manual or slot-rebuild request, the next FULL is dispatched by the recorded schedule after its due time and its recovery set starts successfully
- **AND** an unprefixed cron expression uses UTC regardless of the stored offset of the previous backup timestamp

### Requirement: External physical slots respect the selected Patroni ownership mode

On PostgreSQL 17 and 18, a source configured to ignore the exact name and physical type of Databasus's external slot SHALL preserve that slot through Patroni management cycles, including cycles while the consumer is disconnected. Ignoring a slot SHALL NOT establish that its state has been transferred to another node. If a slot is absent or its required WAL has been lost, Databasus SHALL either resume from WAL that remains available under the existing slot recovery policy or expose the broken stream through its existing health diagnostics. Creating a new slot SHALL NOT establish continuity for a missing WAL interval.

#### Scenario: Patroni ignores an inactive external slot

- **WHEN** the exact external slot is configured in `ignore_slots` and its consumer is disconnected across several completed Patroni management cycles
- **THEN** the slot remains present with a valid retained position
- **AND** reconnection resumes reception without losing WAL required by that slot

#### Scenario: The promoted primary has no external slot

- **WHEN** Patroni promotes a replica on which Databasus's external slot is absent
- **THEN** Databasus handles the absence under its existing recovery policy or exposes the missing slot as `BROKEN` with reason `SLOT_MISSING`
- **AND** no recovery target is advertised as reachable solely because a new slot exists

#### Scenario: Patroni removes an unprotected external slot

- **WHEN** Databasus's inactive external slot is not protected by `ignore_slots` or configured as a permanent slot and Patroni removes it during management
- **THEN** Databasus handles the absence under the same policy and does not report uninterrupted archiving without sufficient archived WAL

#### Scenario: Slot loss leaves a required WAL gap

- **WHEN** loss of the external slot allows required source WAL to be recycled before Databasus archives it
- **THEN** a target beyond that gap cannot obtain a restore token from the affected chain
- **AND** later recovery requires a valid eligible backup and contiguous WAL for the chosen target, rather than an assertion that the old archive is continuous

### Requirement: Patroni timeline history is published for the recovery storage

After each Patroni switch on PostgreSQL 17 and 18, Databasus SHALL validate and publish the promoted timeline's history with its catalog entry in the storage used by the corresponding FULL. Archive-based recovery after that FULL SHALL include and use the required published history while remaining on that FULL's timeline. The existence of a history file SHALL NOT imply support for replay across timelines.

#### Scenario: History after the second switchover

- **WHEN** two Patroni switchovers have advanced the timeline twice and a replacement FULL and subsequent WAL are available after each
- **THEN** each promoted timeline has valid history matching its source history and catalog metadata in the corresponding FULL's storage
- **AND** a downloaded recovery set uses its required history to restore rows written after the selected FULL without replaying across the promotion boundary

### Requirement: Patroni backup access preserves limited-user security

Databasus SHALL acquire FULL backups, INCREMENTAL backups, WAL and required timeline history from a Patroni source using its provisioned limited user, without granting superuser, arbitrary server-file reading or unrestricted table reading. The actual replication session supplying WAL or timeline history SHALL be verified against the configured system identifier; a check performed only on a separate SQL connection SHALL NOT authorize it.

#### Scenario: Restricted access survives promotion

- **WHEN** Patroni switches the primary while Databasus uses its provisioned limited user
- **THEN** backup and WAL reception and history publication succeed against the original cluster's new primary without additional administrative or server-file privileges
- **AND** attempts by that user to read arbitrary server files or a table outside its permitted access remain denied
