# Spec Delta

## MODIFIED Requirements

### Requirement: Promotion recovery works with the provisioned backup role

On PostgreSQL 17, 18 and 19, a source standby promoted while Databasus uses its provisioned backup role SHALL have its new timeline history archived and cataloged. The chain that existed before promotion SHALL restore committed rows from before promotion and after promotion, before any backup on the promoted timeline completes. After an incremental backup on the promoted timeline extends that chain, its restore SHALL include rows written after that incremental backup.

#### Scenario: Recovery after promotion on each supported version

- **WHEN** Databasus provisions the backup role on a primary, that role is replicated to its standby, `summarize_wal` is on for the standby, physical backup and WAL reception use the role, and the standby is promoted
- **THEN** Databasus archives the new timeline's history
- **AND** PostgreSQL started from a restore set for a target after promotion and before any backup on the promoted timeline contains committed rows written before and after promotion
- **AND** an incremental backup on the promoted timeline extends the existing chain, and a later restore contains rows written after it
- **AND** no administrative or server-file read privileges are added to the backup role

### Requirement: Patroni sources recover through an unchanged primary address

On PostgreSQL 17, 18 and 19, Databasus SHALL support a source whose primary is selected by Patroni and exposed through a stable HAProxy address. When that address changes from the primary to its promoted physical replica, Databasus SHALL reconnect without an operator changing its address, restarting Databasus or restarting its backup modules. The same-cluster identity, the rules for continuing a chain along the timeline line and the replacement-FULL fallback SHALL apply after every switch.

#### Scenario: Initial recovery through the primary address

- **WHEN** a Patroni source has a completed FULL and contiguous archived WAL after it, received through the stable primary address
- **THEN** the downloaded recovery set starts an independent writable PostgreSQL cluster containing rows and schema changes represented only by that subsequent WAL

#### Scenario: Two consecutive planned switches

- **WHEN** `summarize_wal` is on for both source nodes, Patroni performs two consecutive switchovers between them and each switch follows a control transaction replayed by the future primary
- **THEN** the unchanged address reaches a different PostgreSQL node with a newer timeline after each switch, while the system identifier stays unchanged
- **AND** WAL reception resumes against that node without restarting Databasus or its backup modules
- **AND** an INCREMENTAL requested after each switch completes as part of the chain that existed before it, with no replacement FULL
- **AND** a restore to a target one minute after each switch, before that INCREMENTAL completes, includes control rows written before and after the switch
- **AND** a restore after each INCREMENTAL includes control rows written before the switch, after the switch and after that INCREMENTAL

### Requirement: Patroni crash recovery preserves transactions replayed by the elected replica

For a durability-enabled source on PostgreSQL 17, 18 and 19, recovery after a Patroni failover SHALL preserve control transactions committed on the former primary and replayed by the replica before that replica is elected. This guarantee SHALL NOT include acknowledged transactions that never reached the elected replica.

#### Scenario: Abrupt loss after confirmed replica replay

- **WHEN** a control transaction is confirmed on the future primary, the old primary node stops abruptly, and Patroni elects that replica
- **THEN** Databasus reconnects through the same address and follows its timeline-line policy
- **AND** an independent restore to a target after the failover preserves the control transaction, committed schema changes and usable generated IDs
- **AND** the former primary remains unavailable during recovery

### Requirement: Patroni switches during transfer preserve physical backup failure semantics

On PostgreSQL 17, 18 and 19, a Patroni switch while a physical backup is actively transmitting SHALL NOT publish a partial copy as a completed backup. The interrupted operation SHALL reach a terminal result and follow the existing identity-recheck and retry policies, the rules for continuing a chain along the timeline line and the replacement-FULL fallback. Detecting an active backup record alone SHALL NOT establish that the switch interrupted data transfer.

#### Scenario: A switchover interrupts a FULL transfer

- **WHEN** a FULL is transferring data, Patroni switches the primary before that transfer completes, and the failure recheck observes the newer timeline
- **THEN** the old FULL ends as `ERROR` with reason `FAILOVER_DURING_BACKUP`
- **AND** the existing policy schedules one immediate replacement FULL whose downloaded recovery set starts a writable cluster with the control rows and schema
- **AND** the partial old copy cannot be selected as a completed backup

#### Scenario: A switchover interrupts an INCREMENTAL transfer

- **WHEN** `summarize_wal` is on for both source nodes, an INCREMENTAL is transferring data, Patroni switches the primary before that transfer completes, and the failure recheck observes the newer timeline
- **THEN** the old INCREMENTAL ends as `ERROR` with reason `FAILOVER_DURING_BACKUP`
- **AND** the next INCREMENTAL extends the same chain on the promoted timeline, and its downloaded recovery set starts a writable cluster with the control rows and schema
- **AND** the partial old copy cannot be selected as a completed backup

#### Scenario: The failure recheck runs during the routing outage

- **WHEN** a Patroni switch interrupts a transfer and its failure recheck cannot connect to a source
- **THEN** the operation preserves its original failure rather than claiming a verified timeline change
- **AND** once the failed worker has released its reservation and the new primary is reachable, no additional immediate FULL is requested for that generic failure
- **AND** with no competing manual or slot-rebuild request, the next FULL is dispatched by the recorded schedule after its due time and its recovery set starts successfully
- **AND** an unprefixed cron expression uses UTC regardless of the stored offset of the previous backup timestamp

### Requirement: External physical slots respect the selected Patroni ownership mode

On PostgreSQL 17, 18 and 19, a source configured to ignore the exact name and physical type of Databasus's external slot SHALL preserve that slot through Patroni management cycles, including cycles while the consumer is disconnected. Ignoring a slot SHALL NOT establish that its state has been transferred to another node. If a slot is absent or its required WAL has been lost, Databasus SHALL either resume from WAL that remains available under the existing slot recovery policy or expose the broken stream through its existing health diagnostics. Creating a new slot SHALL NOT establish continuity for a missing WAL interval.

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

After each Patroni switch on PostgreSQL 17, 18 and 19, Databasus SHALL validate and publish the promoted timeline's history with its catalog entry in the storage used by the chain that continues across the switch. Archive-based recovery that crosses the switch SHALL include and use that history.

#### Scenario: History after the second switchover

- **WHEN** two Patroni switchovers have advanced the timeline twice and WAL is archived after each
- **THEN** each promoted timeline has valid history matching its source history and catalog metadata in the chain's storage
- **AND** a downloaded recovery set for a target after the second switch includes both histories and restores rows written on all three timelines

### Requirement: WAL reception continues after a failover that lost the former primary's tail

On PostgreSQL 17, 18 and 19, when Databasus archived WAL of the former timeline beyond the fork point's LSN at which the promoted node's history leaves it, WAL reception SHALL resume on the promoted timeline from the fork point without an operator action. The former timeline's WAL beyond the fork point SHALL NOT be treated as the continuation of the line. Archiving SHALL NOT report a gap between the fork point and the promoted timeline's WAL when that WAL is archived.

#### Scenario: The standby missed the former primary's last writes

- **WHEN** the standby stops receiving WAL, the primary commits further transactions that Databasus archives, the primary is lost and the standby is promoted
- **THEN** WAL reception resumes against the promoted node and archives its timeline from the segment holding the fork point's LSN, without an operator action
- **AND** no WAL gap notification is sent for the fork point, and no slot rebuild or replacement FULL is triggered by the resumption
- **AND** a restore to a target after the promotion contains the rows the promoted node had and none of the transactions only the former primary committed
- **AND** restore coverage does not extend on the former timeline past the fork point, and a token request for a target that only the former primary's WAL covers returns `409` with code `target_time_unreachable`
