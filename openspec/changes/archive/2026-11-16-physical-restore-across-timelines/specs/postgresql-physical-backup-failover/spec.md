## RENAMED Requirements

- FROM: `### Requirement: Incremental backups use their root FULL timeline`
- TO: `### Requirement: Incremental backups continue along their chain's timeline line`

- FROM: `### Requirement: Same-timeline PITR remains available during a newer FULL`
- TO: `### Requirement: PITR follows the timeline line across promotions`

## MODIFIED Requirements

### Requirement: Incremental backups continue along their chain's timeline line
The system SHALL start an incremental backup on the live PostgreSQL timeline when that timeline equals the timeline of the chain's newest included backup, or descends from it through promotions. A descendant timeline SHALL extend the chain only when all of these hold: the verified history of the live timeline lists the newest included backup's timeline as an ancestor, that backup's stop LSN is not later than the fork point's LSN at which the line leaves its timeline, every history on the line matches the published history for the same database and timeline, and PostgreSQL accepts the incremental backup against the chain's latest manifest. An equal system identifier alone SHALL NOT authorize the extension.

When the live timeline is newer and the chain cannot be extended, the system SHALL refuse the incremental backup as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`. This includes a promoted node that lacks the WAL summaries the incremental backup needs.

#### Scenario: Timeline matches the root FULL
- **WHEN** an incremental backup starts and the live timeline matches the timeline of the chain's newest included backup, which is the root FULL's timeline until the chain crosses a promotion
- **THEN** the system proceeds with the incremental backup

#### Scenario: Promotion happened before the incremental started
- **WHEN** the live timeline is newer than the timeline of the chain's newest included backup, its verified history lists that timeline as an ancestor with a fork point's LSN at or after that backup's stop LSN, `summarize_wal` was on for the promoted node before its promotion, and the promoted node has WAL summaries covering the chain's latest backup through the new start position
- **THEN** the system completes the incremental backup as part of the same chain, with no replacement FULL

#### Scenario: The newest backup lies past the fork point
- **WHEN** the chain's newest included backup stopped later than the fork point's LSN at which the live timeline's history leaves that backup's timeline
- **THEN** the system refuses the incremental backup as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`

#### Scenario: The live timeline does not descend from the chain
- **WHEN** the verified history of the live timeline does not list the timeline of the chain's newest included backup
- **THEN** the system refuses the incremental backup as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`

#### Scenario: The promoted node lacks WAL summaries
- **WHEN** the live timeline descends from the chain but the promoted node has no WAL summaries for part of the range from the chain's latest backup to the new start position
- **THEN** the system refuses the incremental backup as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`

#### Scenario: Source timeline regressed
- **WHEN** the live timeline is older than the timeline of the chain's newest included backup
- **THEN** the system refuses the incremental backup with reason `TIMELINE_REGRESSION`

#### Scenario: Cataloged history is newer than the root FULL
- **WHEN** a history file for the live timeline is already cataloged and the root FULL belongs to an older timeline
- **THEN** the system extends the chain only if the live timeline's verified history matches the cataloged entry and the other conditions for continuing along the line hold
- **AND** otherwise it refuses the incremental backup as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`

#### Scenario: Two promotions before the next incremental
- **WHEN** the source has advanced from the chain's timeline through two promotions, each fork point's LSN lies at or after the chain's newest backup's stop LSN, `summarize_wal` was on for every promoted node, and the live node has the required WAL summaries
- **THEN** the system completes the incremental backup as part of the same chain

### Requirement: Backup failures are rechecked for promotion
The system SHALL check PostgreSQL cluster identity again when a physical backup stream fails unexpectedly while the backup is still active.

#### Scenario: Promotion interrupts an incremental stream
- **WHEN** the live timeline advances past the timeline observed at incremental preflight and the stream fails
- **THEN** the system records the incremental as `ERROR` with reason `FAILOVER_DURING_BACKUP`
- **AND** the chain stays extendable, so the next due incremental backup applies the rules for continuing along the timeline line

#### Scenario: Stream failure on a chain that already crossed a promotion
- **WHEN** a chain's root FULL is on timeline 1, its newest incremental is on timeline 2, the live timeline stays 2 and an incremental stream fails
- **THEN** the system keeps the original stream failure and does not classify it as a promotion

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
The system SHALL create one immediate replacement FULL after a FULL ends as `ERROR` with reason `FAILOVER_DURING_BACKUP` or an incremental ends as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`. An incremental backup that continues its chain across a promotion is not a failure and SHALL NOT trigger a replacement FULL. An incremental that ends as `ERROR` with reason `FAILOVER_DURING_BACKUP` SHALL NOT trigger one either: its chain stays extendable, and the next incremental runs at its configured time.

#### Scenario: Incremental chain broke after promotion
- **WHEN** an incremental ends as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED` and no later FULL attempt exists
- **THEN** the system schedules a FULL without waiting for the configured incremental or FULL interval

#### Scenario: Incremental continued across a promotion
- **WHEN** an incremental completes on a promoted timeline as part of its chain
- **THEN** the system schedules no replacement FULL and keeps the configured intervals

#### Scenario: Incremental interrupted by promotion
- **WHEN** the latest incremental ends as `ERROR` with reason `FAILOVER_DURING_BACKUP`
- **THEN** no immediate replacement FULL is scheduled and the next incremental runs at its configured time

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

### Requirement: PITR follows the timeline line across promotions

The system SHALL restore a point in time along the line of the newest timeline: the timelines the newest verified history lists as ancestors, each up to the fork point's LSN at which the line leaves it, followed by the newest timeline. A target on that line SHALL be reachable from the newest completed FULL on the line that completed at or before the target, through that chain's eligible incrementals and contiguous WAL, also when the WAL crosses one or more fork points. Reaching the target SHALL require the published verified history of every timeline after the FULL's timeline on the line, in the FULL's storage. The system SHALL NOT use a FULL completed after the target to make that target reachable.

WAL of a former timeline that starts at or after the fork point's LSN at which the line leaves it SHALL NOT be included in a restore set and SHALL NOT extend the reachable time. A target that only such WAL covers SHALL be refused. A fork point SHALL stop the line there when its timeline's history is not published, is marked conflicting, was published by a Databasus version that did not record fork points, or lies outside the FULL's storage, or when WAL the line needs beyond it lies only in another storage. Targets beyond a stopped fork point SHALL be refused from that chain with `409` and code `target_time_unreachable`. A FULL whose start LSN is at or after the fork point's LSN of every descendant of its timeline does not lie on the line of the newest timeline; it stays restorable by its backup identifier and is not used for targets by time or for the latest restore.

#### Scenario: Target falls during the newer FULL

- **WHEN** the requested target is later than the older FULL completion and earlier than the newer FULL completion on the same timeline
- **THEN** the restore uses the older FULL and contiguous WAL through the target, even when that WAL crosses the newer FULL start position

#### Scenario: Target crosses a promotion

- **WHEN** a target lies one minute after a promotion, before any backup on the promoted timeline has completed, and the WAL from the chain's latest eligible backup through the fork point and on the promoted timeline through the target is archived without a gap
- **THEN** the restore token is issued
- **AND** the restore set contains the chain's FULL and eligible incrementals, the former timeline's WAL up to the fork point, the promoted timeline's history and its WAL through the target
- **AND** the recovered cluster contains rows committed before and after the promotion up to the target

#### Scenario: A replacement FULL does not fill the earlier recovery gap

- **WHEN** a replacement FULL on the promoted timeline has completed and the target lies between the promotion and that FULL's completion
- **THEN** the restore does not use the replacement FULL, because it completed after the target
- **AND** the restore uses the earlier chain across the fork point, and a restore token is issued

#### Scenario: Target covered only by the former primary's lost WAL

- **WHEN** Databasus archived WAL of the former timeline beyond the fork point and a target is covered by that WAL alone
- **THEN** the request returns `409` with code `target_time_unreachable`
- **AND** no restore token is issued for that target

#### Scenario: Promoted timeline history is not published

- **WHEN** reaching the target requires WAL of a promoted timeline whose verified history is not published in the FULL's storage
- **THEN** the request returns `409` with code `target_time_unreachable`

#### Scenario: Promoted timeline history published without a fork point

- **WHEN** the promoted timeline's history was published by a Databasus version that did not record fork points and the target lies after that promotion
- **THEN** the request returns `409` with code `target_time_unreachable`

#### Scenario: Promoted timeline history or WAL in another storage

- **WHEN** the promoted timeline's history, or WAL the target needs after the fork point, is present only in a storage other than the FULL's
- **THEN** the request returns `409` with code `target_time_unreachable`

#### Scenario: Two promotions before the target

- **WHEN** the target lies after a second promotion, both promoted timelines have published history, and WAL along the line is archived without a gap
- **THEN** the restore set crosses both fork points and the recovered cluster contains rows written on all three timelines up to the target

### Requirement: Promotion recovery works with the provisioned backup role

On PostgreSQL 17 and 18, a source standby promoted while Databasus uses its provisioned backup role SHALL have its new timeline history archived and cataloged. The chain that existed before promotion SHALL restore committed rows from before promotion and after promotion, before any backup on the promoted timeline completes. After an incremental backup on the promoted timeline extends that chain, its restore SHALL include rows written after that incremental backup.

#### Scenario: Recovery after promotion on each supported version

- **WHEN** Databasus provisions the backup role on a primary, that role is replicated to its standby, `summarize_wal` is on for the standby, physical backup and WAL reception use the role, and the standby is promoted
- **THEN** Databasus archives the new timeline's history
- **AND** PostgreSQL started from a restore set for a target after promotion and before any backup on the promoted timeline contains committed rows written before and after promotion
- **AND** an incremental backup on the promoted timeline extends the existing chain, and a later restore contains rows written after it
- **AND** no administrative or server-file read privileges are added to the backup role

### Requirement: WAL identity enforcement permits promotion of the original cluster

When an unchanged source address moves from a primary to its real physical replica after promotion, Databasus SHALL accept the retained system identifier for WAL reception and archive WAL of the promoted timeline. An equal system identifier SHALL NOT by itself authorize an incremental backup to extend a chain from another timeline; the rules for continuing along the timeline line decide that. Recovery after the switch SHALL follow the timeline line.

#### Scenario: Address switches to a promoted replica

- **WHEN** a primary and its real replica have equal system identifiers, `summarize_wal` is on for the replica, the replica is promoted to a newer timeline, and the unchanged address starts routing new connections to that replica
- **THEN** WAL reception accepts the replica's identity and archives WAL from its timeline
- **AND** the next incremental backup extends the existing chain

#### Scenario: Recovery after the switch

- **WHEN** WAL of the promoted timeline is archived after the same-cluster endpoint switch
- **THEN** a restore set for a target on the promoted timeline can be downloaded and restored with control rows from both sides of the switch

#### Scenario: Replacement FULL is recovered

- **WHEN** the chain could not continue across the switch and a replacement FULL completes after the same-cluster endpoint switch
- **THEN** its recovery bundle can be downloaded and restored with control rows from the promoted source
- **AND** recovery from it does not depend on replay across the promotion boundary

### Requirement: Patroni sources recover through an unchanged primary address

On PostgreSQL 17 and 18, Databasus SHALL support a source whose primary is selected by Patroni and exposed through a stable HAProxy address. When that address changes from the primary to its promoted physical replica, Databasus SHALL reconnect without an operator changing its address, restarting Databasus or restarting its backup modules. The same-cluster identity, the rules for continuing a chain along the timeline line and the replacement-FULL fallback SHALL apply after every switch.

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

For a durability-enabled source on PostgreSQL 17 and 18, recovery after a Patroni failover SHALL preserve control transactions committed on the former primary and replayed by the replica before that replica is elected. This guarantee SHALL NOT include acknowledged transactions that never reached the elected replica.

#### Scenario: Abrupt loss after confirmed replica replay

- **WHEN** a control transaction is confirmed on the future primary, the old primary node stops abruptly, and Patroni elects that replica
- **THEN** Databasus reconnects through the same address and follows its timeline-line policy
- **AND** an independent restore to a target after the failover preserves the control transaction, committed schema changes and usable generated IDs
- **AND** the former primary remains unavailable during recovery

### Requirement: Patroni switches during transfer preserve physical backup failure semantics

On PostgreSQL 17 and 18, a Patroni switch while a physical backup is actively transmitting SHALL NOT publish a partial copy as a completed backup. The interrupted operation SHALL reach a terminal result and follow the existing identity-recheck and retry policies, the rules for continuing a chain along the timeline line and the replacement-FULL fallback. Detecting an active backup record alone SHALL NOT establish that the switch interrupted data transfer.

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

### Requirement: Patroni timeline history is published for the recovery storage

After each Patroni switch on PostgreSQL 17 and 18, Databasus SHALL validate and publish the promoted timeline's history with its catalog entry in the storage used by the chain that continues across the switch. Archive-based recovery that crosses the switch SHALL include and use that history.

#### Scenario: History after the second switchover

- **WHEN** two Patroni switchovers have advanced the timeline twice and WAL is archived after each
- **THEN** each promoted timeline has valid history matching its source history and catalog metadata in the chain's storage
- **AND** a downloaded recovery set for a target after the second switch includes both histories and restores rows written on all three timelines

### Requirement: CloudNativePG sources recover through an unchanged Kubernetes Service address

On PostgreSQL 17 and 18, Databasus SHALL support a source whose primary is selected by CloudNativePG and reached through a Kubernetes Service that follows CloudNativePG's primary selection. When CloudNativePG moves the primary role to the other PostgreSQL node and the Kubernetes Service starts routing to it, Databasus SHALL reconnect without an administrator changing its address, restarting Databasus or restarting its backup modules. The same-cluster identity, the rules for continuing a chain along the timeline line and the replacement-FULL fallback SHALL apply after every switch.

#### Scenario: Initial recovery through the Kubernetes Service address

- **WHEN** a CloudNativePG source has a completed FULL, a completed INCREMENTAL and contiguous archived WAL after them, all received through the Kubernetes Service address
- **THEN** the downloaded recovery set starts an independent writable PostgreSQL cluster containing rows and schema changes represented only by that subsequent WAL
- **AND** new rows inserted into tables with generated IDs get values that do not conflict with restored rows

#### Scenario: Two consecutive planned switches

- **WHEN** `summarize_wal` is on for both PostgreSQL nodes, CloudNativePG performs two consecutive switchovers between them and each switch follows a control transaction replayed by the future primary
- **THEN** after each switch the Kubernetes Service routes to the other PostgreSQL node, the timeline is newer and the system identifier is unchanged
- **AND** WAL reception resumes against that node without restarting Databasus or its backup modules
- **AND** an INCREMENTAL requested after each switch completes as part of the chain that existed before it, with no replacement FULL
- **AND** a restore after each INCREMENTAL includes control rows written before the switch, after the switch and after that INCREMENTAL

#### Scenario: Recovery points around a switch

- **WHEN** a target before a switch, a target one minute after the switch and before the next INCREMENTAL completes, and a target after that INCREMENTAL are archived and protected by retention
- **THEN** all three targets restore to running clusters with their recorded rows, schema and usable generated IDs, also after the physical cleaner and the storage deletion worker have run

### Requirement: CloudNativePG crash recovery preserves transactions replayed by the elected node

For a durability-enabled CloudNativePG source on PostgreSQL 17 and 18, recovery after a failover SHALL preserve control transactions committed on the former primary and replayed by the other PostgreSQL node before CloudNativePG promoted it. This guarantee SHALL NOT include acknowledged transactions that never reached the promoted node. A restart of the same node in the primary role SHALL NOT count as a failover.

#### Scenario: Abrupt loss after confirmed replay

- **WHEN** a control transaction is confirmed as replayed on the future primary, the primary node is removed without a clean shutdown, and CloudNativePG promotes the other PostgreSQL node
- **THEN** the Kubernetes Service routes to a different node on a newer timeline with the same system identifier
- **AND** Databasus reconnects through the same address and follows its timeline-line policy
- **AND** an independent restore to a target after the failover preserves the control transaction, committed schema changes and usable generated IDs without contacting the source cluster

### Requirement: CloudNativePG switches during transfer preserve physical backup failure semantics

On PostgreSQL 17 and 18, a CloudNativePG switch while a physical backup is actively transmitting SHALL NOT publish a partial copy as a completed backup. The interrupted operation SHALL reach a terminal result and follow the existing identity-recheck and retry policies, the rules for continuing a chain along the timeline line and the replacement-FULL fallback. Detecting an active backup record alone SHALL NOT establish that the switch interrupted data transfer.

#### Scenario: A switchover interrupts a FULL transfer

- **WHEN** a FULL is transferring data, CloudNativePG switches the primary before that transfer completes, and the failure recheck observes the newer timeline
- **THEN** the old FULL ends as `ERROR` with reason `FAILOVER_DURING_BACKUP`
- **AND** the existing policy schedules one immediate replacement FULL whose downloaded recovery set starts a writable cluster with the control rows and schema
- **AND** the partial old copy cannot be selected as a completed backup

#### Scenario: A switchover interrupts an INCREMENTAL transfer

- **WHEN** `summarize_wal` is on for both PostgreSQL nodes, an INCREMENTAL is transferring data, CloudNativePG switches the primary before that transfer completes, and the failure recheck observes the newer timeline
- **THEN** the old INCREMENTAL ends as `ERROR` with reason `FAILOVER_DURING_BACKUP`
- **AND** the next INCREMENTAL extends the same chain on the promoted timeline, and its downloaded recovery set starts a writable cluster with the control rows and schema
- **AND** the partial old copy cannot be selected as a completed backup

#### Scenario: The failure recheck runs during a routing outage

- **WHEN** a CloudNativePG switch interrupts a transfer and its failure recheck cannot connect to a source
- **THEN** the operation preserves its original failure rather than claiming a verified timeline change
- **AND** once the failed worker has released its reservation and the new primary is reachable, no additional immediate FULL is requested for that generic failure
- **AND** with no competing manual or slot-rebuild request, the next FULL is dispatched by the recorded schedule after its due time and its recovery set starts successfully
- **AND** an unprefixed cron expression uses UTC regardless of the stored offset of the previous backup timestamp

### Requirement: CloudNativePG timeline history is published for the recovery storage

After each CloudNativePG switch on PostgreSQL 17 and 18, Databasus SHALL validate and publish the promoted timeline's history with its catalog entry in the storage used by the chain that continues across the switch. Archive-based recovery that crosses the switch SHALL include and use that history.

#### Scenario: History after the second switchover

- **WHEN** two CloudNativePG switchovers have advanced the timeline twice and WAL is archived after each
- **THEN** each promoted timeline has valid history matching its source history and catalog metadata in the chain's storage
- **AND** a downloaded recovery set for a target after the second switch includes both histories and restores rows written on all three timelines

## ADDED Requirements

### Requirement: WAL reception continues after a failover that lost the former primary's tail

On PostgreSQL 17 and 18, when Databasus archived WAL of the former timeline beyond the fork point's LSN at which the promoted node's history leaves it, WAL reception SHALL resume on the promoted timeline from the fork point without an operator action. The former timeline's WAL beyond the fork point SHALL NOT be treated as the continuation of the line. Archiving SHALL NOT report a gap between the fork point and the promoted timeline's WAL when that WAL is archived.

#### Scenario: The standby missed the former primary's last writes

- **WHEN** the standby stops receiving WAL, the primary commits further transactions that Databasus archives, the primary is lost and the standby is promoted
- **THEN** WAL reception resumes against the promoted node and archives its timeline from the segment holding the fork point's LSN, without an operator action
- **AND** no WAL gap notification is sent for the fork point, and no slot rebuild or replacement FULL is triggered by the resumption
- **AND** a restore to a target after the promotion contains the rows the promoted node had and none of the transactions only the former primary committed
- **AND** restore coverage does not extend on the former timeline past the fork point, and a token request for a target that only the former primary's WAL covers returns `409` with code `target_time_unreachable`

### Requirement: Incremental backups across a promotion require WAL summaries on standbys

An incremental backup can extend a chain across a promotion only when the promoted node summarized WAL before it was promoted. Databasus SHALL NOT enable WAL summarization on standbys itself. Without the required summaries, the fallback replacement FULL SHALL keep backups going, and restores to targets after the promotion SHALL still follow the timeline line from the existing chain.

#### Scenario: Summarization was off on the standby

- **WHEN** `summarize_wal` was off on the standby before its promotion and an incremental backup is due on the promoted timeline
- **THEN** the incremental backup ends as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED` and one immediate replacement FULL follows
- **AND** a target between the promotion and that FULL's completion remains restorable from the existing chain across the fork point
