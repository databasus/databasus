## ADDED Requirements

### Requirement: CloudNativePG sources recover through an unchanged Kubernetes Service address

On PostgreSQL 17 and 18, Databasus SHALL support a source whose primary is selected by CloudNativePG and reached through a Kubernetes Service that follows CloudNativePG's primary selection. When CloudNativePG moves the primary role to the other PostgreSQL node and the Kubernetes Service starts routing to it, Databasus SHALL reconnect without an administrator changing its address, restarting Databasus or restarting its backup modules. The same-cluster identity and the existing incremental-chain and replacement-FULL policies SHALL apply after every switch.

#### Scenario: Initial recovery through the Kubernetes Service address

- **WHEN** a CloudNativePG source has a completed FULL, a completed INCREMENTAL and contiguous archived WAL after them, all received through the Kubernetes Service address
- **THEN** the downloaded recovery set starts an independent writable PostgreSQL cluster containing rows and schema changes represented only by that subsequent WAL
- **AND** new rows inserted into tables with generated IDs get values that do not conflict with restored rows

#### Scenario: Two consecutive planned switches

- **WHEN** CloudNativePG performs two consecutive switchovers between the two PostgreSQL nodes and each switch follows a control transaction replayed by the future primary
- **THEN** after each switch the Kubernetes Service routes to the other PostgreSQL node, the timeline is newer and the system identifier is unchanged
- **AND** WAL reception resumes against that node without restarting Databasus or its backup modules
- **AND** an INCREMENTAL requested against the preceding timeline ends as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`, followed by a FULL under the existing immediate-replacement policy
- **AND** a restore from each replacement FULL and subsequent archived WAL includes control rows written before the switch, after the switch and after that FULL

#### Scenario: Recovery points around a switch

- **WHEN** a target before a switch and a target after its replacement FULL are both archived and protected by retention
- **THEN** both targets restore to running clusters with their recorded rows, schema and usable generated IDs, also after the physical cleaner and the storage deletion worker have run
- **AND** a target between the promotion and the completion of the replacement FULL gets `409 target_time_unreachable` and no restore token, before and after that FULL completes

### Requirement: CloudNativePG crash recovery preserves transactions replayed by the elected node

For a durability-enabled CloudNativePG source on PostgreSQL 17 and 18, recovery after a failover SHALL preserve control transactions committed on the former primary and replayed by the other PostgreSQL node before CloudNativePG promoted it. This guarantee SHALL NOT include acknowledged transactions that never reached the promoted node. A restart of the same node in the primary role SHALL NOT count as a failover.

#### Scenario: Abrupt loss after confirmed replay

- **WHEN** a control transaction is confirmed as replayed on the future primary, the primary node is removed without a clean shutdown, and CloudNativePG promotes the other PostgreSQL node
- **THEN** the Kubernetes Service routes to a different node on a newer timeline with the same system identifier
- **AND** Databasus reconnects through the same address and follows its existing timeline and replacement-FULL policy
- **AND** an independent restore from the replacement FULL and subsequent WAL preserves the control transaction, committed schema changes and usable generated IDs without contacting the source cluster

### Requirement: CloudNativePG switches during transfer preserve physical backup failure semantics

On PostgreSQL 17 and 18, a CloudNativePG switch while a physical backup is actively transmitting SHALL NOT publish a partial copy as a completed backup. The interrupted operation SHALL reach a terminal result and follow the existing identity-recheck, retry and replacement-FULL policies. Detecting an active backup record alone SHALL NOT establish that the switch interrupted data transfer.

#### Scenario: A switchover interrupts a FULL transfer

- **WHEN** a FULL is transferring data, CloudNativePG switches the primary before that transfer completes, and the failure recheck observes the newer timeline
- **THEN** the old FULL ends as `ERROR` with reason `FAILOVER_DURING_BACKUP`
- **AND** the existing policy schedules one immediate replacement FULL whose downloaded recovery set starts a writable cluster with the control rows and schema
- **AND** the partial old copy cannot be selected as a completed backup

#### Scenario: A switchover interrupts an INCREMENTAL transfer

- **WHEN** an INCREMENTAL is transferring data, CloudNativePG switches the primary before that transfer completes, and the failure recheck observes the newer timeline
- **THEN** the old INCREMENTAL ends as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`
- **AND** the existing policy schedules one immediate replacement FULL whose downloaded recovery set starts a writable cluster with the control rows and schema
- **AND** the partial old copy cannot be selected as a completed backup

#### Scenario: The failure recheck runs during a routing outage

- **WHEN** a CloudNativePG switch interrupts a transfer and its failure recheck cannot connect to a source
- **THEN** the operation preserves its original failure rather than claiming a verified timeline change
- **AND** once the failed worker has released its reservation and the new primary is reachable, no additional immediate FULL is requested for that generic failure
- **AND** with no competing manual or slot-rebuild request, the next FULL is dispatched by the recorded schedule after its due time and its recovery set starts successfully
- **AND** an unprefixed cron expression uses UTC regardless of the stored offset of the previous backup timestamp

### Requirement: Databasus's physical slot follows CloudNativePG slot synchronization

On PostgreSQL 17 and 18, when CloudNativePG synchronizes user-defined physical replication slots to its standby, Databasus's slot SHALL be usable after promotion only if the standby's copy retains the WAL that Databasus still needs. The existence of a slot with the same name SHALL NOT establish that its retained position is sufficient. If the slot is absent on the promoted node or its required WAL has been lost, Databasus SHALL either resume from WAL that remains available under the existing slot recovery policy or expose the broken stream through its existing health diagnostics. Creating a new slot SHALL NOT establish continuity for a missing WAL interval.

#### Scenario: The synchronized slot survives a switch

- **WHEN** slot synchronization is enabled, the standby's copy of Databasus's slot is observed with a retained position no later than the WAL Databasus has not yet archived, and CloudNativePG then switches the primary to that standby
- **THEN** WAL reception resumes on the promoted node without losing WAL the slot was required to retain
- **AND** a restore from the replacement FULL and subsequent WAL contains the control rows written around the switch

#### Scenario: The slot is excluded from synchronization

- **WHEN** Databasus's slot name matches the source's synchronization exclusion patterns, its absence on the standby is observed, and CloudNativePG promotes that standby
- **THEN** Databasus handles the absence under its existing recovery policy or exposes the missing slot as `BROKEN` with reason `SLOT_MISSING`
- **AND** no recovery target is advertised as reachable solely because a new slot exists

#### Scenario: Slot synchronization is turned off

- **WHEN** slot synchronization is turned off after Databasus's slot was copied to the standby, CloudNativePG's removal of that copy is observed, and CloudNativePG promotes that standby
- **THEN** Databasus handles the absence under the same policy and does not report uninterrupted archiving without sufficient archived WAL

#### Scenario: Slot loss leaves a required WAL gap

- **WHEN** loss of Databasus's slot allows an identified required source segment to be recycled before Databasus archives it
- **THEN** a target beyond that gap cannot obtain a restore token from the affected chain, even after a slot or a FULL is created again
- **AND** later recovery requires a valid eligible backup and contiguous WAL for the chosen target

### Requirement: CloudNativePG timeline history is published for the recovery storage

After each CloudNativePG switch on PostgreSQL 17 and 18, Databasus SHALL validate and publish the promoted timeline's history with its catalog entry in the storage used by the corresponding FULL. Archive-based recovery after that FULL SHALL include and use the required published history while remaining on that FULL's timeline. The existence of a history file SHALL NOT imply support for replay across timelines.

#### Scenario: History after the second switchover

- **WHEN** two CloudNativePG switchovers have advanced the timeline twice and a replacement FULL and subsequent WAL are available after each
- **THEN** each promoted timeline has valid history matching its source history and catalog metadata in the corresponding FULL's storage
- **AND** a downloaded recovery set uses its required history to restore rows written after the selected FULL without replaying across the promotion boundary

### Requirement: Recovery from a CloudNativePG source is independent of the source cluster

On PostgreSQL 17 and 18, a recovery set taken from a CloudNativePG source SHALL start an independent PostgreSQL server outside Kubernetes. The recovery script SHALL prepare the configuration that CloudNativePG keeps inside the data directory for the target location, including its authentication rules, and SHALL report the configuration files the source references outside the data directory, such as its TLS certificates. Recovery SHALL use only the selected backup and downloaded WAL and SHALL NOT depend on the source's Kubernetes Service, the former archive or inherited replication and recovery settings.

#### Scenario: Recovery modes without the source cluster

- **WHEN** a recovery set is restored for an explicit target, for the latest point and for an explicit backup without extra WAL, while the source's Kubernetes Service and the former archive are unreachable
- **THEN** each restored server starts, leaves recovery as its mode requires, contains the rows selected by its target, excludes later rows and accepts new writes with generated IDs that do not conflict with restored rows
- **AND** inherited primary connection settings, restore commands, competing recovery targets and standby or recovery signal files from the source do not change the operation

#### Scenario: Configuration outside the data directory

- **WHEN** the copied configuration references files outside the data directory, such as CloudNativePG's certificate files
- **THEN** the recovery script names the missing files and the settings that need an independent value
- **AND** after the independent target configuration is prepared as instructed, without changing the source's TLS or access rules, the restored server starts with the recovered data

#### Scenario: Authentication rules kept in the data directory

- **WHEN** the copied configuration points to authentication files inside the source data directory
- **THEN** the prepared target uses the copied rules at the target's location rather than unrelated default files

### Requirement: CloudNativePG backup access uses a declared limited role over verified TLS

Databasus SHALL acquire FULL backups, INCREMENTAL backups, WAL and required timeline history from a CloudNativePG source using a limited role declared in the source's cluster definition, without granting superuser, arbitrary server-file reading or unrestricted table reading. Every connection to the source SHALL use TLS with the server certificate verified against the certificate authority configured in Databasus, including after a switch. The actual replication session supplying WAL or timeline history SHALL be verified against the configured system identifier; a check performed only on a separate SQL connection SHALL NOT authorize it.

#### Scenario: The declared role works without administrative setup

- **WHEN** the cluster definition declares a login role with replication rights and permission to force a WAL segment switch, and Databasus is given that role's credentials and the source certificate authority
- **THEN** physical backups, WAL reception, forced WAL rotation and history publication succeed without Databasus creating users or granting privileges
- **AND** the role is not a superuser, cannot create roles or databases and cannot bypass row security

#### Scenario: Restricted access survives promotion

- **WHEN** CloudNativePG switches the primary while Databasus uses the declared role
- **THEN** backup, WAL reception and history publication succeed against the original cluster's new primary without additional administrative or server-file privileges
- **AND** attempts by that role to read arbitrary server files or a table outside its permitted access remain denied

#### Scenario: The source certificate comes from another authority

- **WHEN** the source presents a server certificate that the certificate authority configured in Databasus did not issue
- **THEN** the database connection test reports the database as not connected, with an error that names certificate verification
- **AND** a requested FULL ends as `ERROR` with a message that names certificate verification, and no backup completes
- **AND** no WAL segment is archived, and the WAL stream does not report itself as `HEALTHY`

#### Scenario: The source certificate does not name the address

- **WHEN** the source presents a server certificate from the configured authority that does not name the address Databasus connects to
- **THEN** the database connection test reports the database as not connected, with an error that names the certificate's host name mismatch
- **AND** a requested FULL ends as `ERROR` with a message that names the mismatch, and no backup completes
- **AND** no WAL segment is archived, and the WAL stream does not report itself as `HEALTHY`

#### Scenario: The documented access rules reject unencrypted replication

- **WHEN** the declared role attempts a replication connection without TLS to a source configured as documented
- **THEN** the source rejects the connection
