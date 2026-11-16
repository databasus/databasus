## MODIFIED Requirements

### Requirement: A restore token opens one restore set

Issuing a restore token SHALL answer `201` with the token and the time it expires, 15 minutes after it is issued. A token SHALL name either a database with an optional target time, which restores to that time or to the latest restorable moment, or one full or incremental backup, which restores that backup and the backups it is built on without replaying WAL. The token SHALL be refused at issue, and no token issued, when:

- the database has no completed full backup, or the chosen backup is not completed or misses a backup it is built on: `409` with `code` `no_restorable_backup`;
- the target time is earlier than the earliest full backup, a WAL gap lies between the full backup and the target, reaching the target requires crossing a fork point that stops the timeline line, or only former-timeline WAL past a fork point covers the target: `409` with `code` `target_time_unreachable`. The PostgreSQL physical backup failover specification lists when a fork point stops the line, including a timeline above 1 whose verified history is not published;
- another download or restore stream of the same user is running: `409` with `code` `already_running`;
- the chosen backup does not exist: `404` with `code` `not_found`;
- the database is not a physical PostgreSQL database: `400` with `code` `invalid_request`.

#### Scenario: Restoring to the latest moment

- **WHEN** a viewer asks for a restore token of a database with a completed full backup and archived WAL after it, without a target time and with the required timeline history published
- **THEN** the response is `201` with a token and its expiry time

#### Scenario: A target time behind a WAL gap

- **WHEN** a viewer asks for a restore token to a time that lies after a gap in the archived WAL
- **THEN** the response is `409` with `code` `target_time_unreachable`, and its detail names the latest restorable position

#### Scenario: No full backup yet

- **WHEN** a viewer asks for a restore token of a database that has no completed full backup
- **THEN** the response is `409` with `code` `no_restorable_backup`

#### Scenario: A running download

- **WHEN** a user asks for a restore token while a backup download of the same user runs
- **THEN** the response is `409` with `code` `already_running`

Archive-based restores SHALL follow the timeline line defined by the PostgreSQL physical backup failover specification. A restore set SHALL include archived WAL of a timeline above 1 only when that timeline's verified history is published under the current backup storage selection, and SHALL then include the history of every timeline on the line after the selected FULL's timeline. On a timeline the line leaves, the set SHALL include only the segments that start before the fork point's LSN; when that LSN falls inside a segment and the former timeline's copy of that segment is archived, they end with that copy. On the following timeline the set SHALL start with the segment holding that LSN, the segment whose range contains it. When the set contains WAL, the recovery script SHALL set `recovery_target_timeline` to the newest timeline in the set; a set without WAL SHALL set no target timeline.

When the line stops at a fork point, a latest restore SHALL end at the end of the contiguous WAL on the last timeline the line reaches, or at the chain's newest completed backup when no WAL follows it, and SHALL include no WAL of later timelines and no history files beyond that timeline. An explicit target beyond that point SHALL get `409` with `code` `target_time_unreachable`. Restoring an explicitly selected completed backup SHALL remain independent of separately archived WAL and history. Timeline 1 SHALL require no history file.

#### Scenario: Latest restore while promoted timeline history is pending

- **WHEN** a completed backup on timeline 2 has subsequent archived WAL but its verified timeline history has not been published
- **THEN** a latest restore token is issued for recovery through that backup only
- **AND** the downloaded set excludes the subsequent WAL

#### Scenario: Target requires unpublished history

- **WHEN** an explicit restore target requires WAL after the newest included backup on timeline 2 and that timeline's verified history is not published
- **THEN** token issuance returns `409` with `code` `target_time_unreachable`
- **AND** no token is issued

#### Scenario: Explicit backup remains restorable

- **WHEN** a completed backup is selected by its identifier and separately archived history is unavailable
- **THEN** token issuance succeeds if that backup and its required backup ancestors remain available
- **AND** the restore set contains no additional archived WAL or history

#### Scenario: History becomes available

- **WHEN** the verified history for timeline 2 is published while its completed backups and contiguous archived WAL remain available
- **THEN** latest and targeted restore selection can include that WAL without creating another FULL

#### Scenario: First timeline needs no history

- **WHEN** a restore on timeline 1 has a completed backup and sufficient contiguous archived WAL
- **THEN** absence of a timeline history file does not restrict that restore

#### Scenario: Latest restore after a promotion

- **WHEN** a chain's newest completed backup is on timeline 1, the source was promoted to timeline 2, the history of timeline 2 is published and WAL is archived without a gap through the fork point and on timeline 2
- **THEN** a latest restore token is issued
- **AND** the downloaded set contains the chain's backups, timeline 1 WAL that ends before the fork point's LSN, the history of timeline 2 and timeline 2 WAL starting with the segment that holds that LSN
- **AND** the recovery script started from the set writes `recovery_target_timeline` set to 2, and the recovered cluster contains the newest archived rows of timeline 2

#### Scenario: Timeline 1 chain while the promoted timeline's history is pending

- **WHEN** a chain's newest completed backup is on timeline 1, the source was promoted to timeline 2, timeline 2 WAL is archived and the history of timeline 2 is not published yet
- **THEN** a latest restore token is issued
- **AND** the downloaded set contains the chain's backups and the contiguous timeline 1 WAL after them, and no timeline 2 WAL or history
- **AND** an explicit target that only timeline 2 WAL covers gets `409` with `code` `target_time_unreachable`

#### Scenario: A set without WAL

- **WHEN** a completed backup is restored by its identifier
- **THEN** the recovery script writes no `recovery_target_timeline`

#### Scenario: Incremental backup on a promoted timeline

- **WHEN** a chain has a FULL on timeline 1 and an incremental backup on timeline 2 that extends it, and a target lies after that incremental backup
- **THEN** the restore set contains the FULL, the incremental backups up to the target, the history of timeline 2 and timeline 2 WAL after the newest included backup
- **AND** it contains no timeline 1 WAL

### Requirement: Delayed WAL publication does not extend a source timeline's PITR time

An explicit PITR target SHALL require source-time evidence from complete transaction records in the contiguous published WAL along the selected timeline line, or a completed-backup consistency point. Records in a former-timeline segment that holds or follows the fork point's LSN at which the line leaves that timeline SHALL NOT be evidence. Receipt, claim and upload timestamps SHALL NOT establish source-time reachability. With inclusive recovery, a confirming WAL transaction SHALL be strictly later than the target. Old WAL without source-time evidence SHALL NOT use receipt time as a substitute. Latest recovery and explicit completed-backup recovery SHALL remain available without claiming an unconfirmed explicit target.

User-controlled descriptive payload, including multiline prepared-transaction identifiers, SHALL NOT be interpreted as COMMIT/ABORT records or extend source-time evidence. Genuine transaction completion after PREPARE SHALL remain usable as evidence.

Confirmed source time SHALL retain the maximum timestamp among trusted complete transaction records. A later record with an older timestamp, including an independently verified completion after PREPARE, SHALL NOT reduce already confirmed coverage.

When an explicit target equals the selected completed backup's consistency point and contiguous WAL contains no later confirming transaction, recovery SHALL use the completed backup and its required ancestors without additional archived WAL. Supplying that advertised target to the recovery script SHALL start a writable cluster without requiring a later transaction. Latest recovery SHALL retain eligible WAL, and targets beyond the confirmed bound SHALL remain refused.

#### Scenario: A completed backup is the only confirmed time

- **WHEN** a completed FULL or INCREMENTAL has subsequent archived WAL but no transaction later than its completion, and the owner requests the completion point advertised by statistics
- **THEN** the downloaded set contains the selected backup and its required ancestors without additional archived WAL
- **AND** starting recovery with that explicit target succeeds and permits new writes
- **AND** latest recovery retains eligible WAL without advertising an unconfirmed later PITR target

#### Scenario: Transaction timestamps decrease in WAL order

- **WHEN** a later complete transaction record has a timestamp older than an already confirmed record
- **THEN** statistics and target validation retain the greatest confirmed source timestamp as their time evidence
- **AND** an independently verified completion after PREPARE cannot reduce an earlier trusted bound

#### Scenario: A prepared identifier imitates transaction output

- **WHEN** a prepared-transaction identifier contains a multiline COMMIT description or a fabricated WAL record header with a future timestamp
- **THEN** decoding retains only time confirmed by actual completed WAL records, and the fabricated time cannot extend recovery coverage or authorize a target
- **AND** a subsequent genuine COMMIT or ABORT remains observable

#### Scenario: Old timeline WAL is published after promotion

- **WHEN** former-timeline WAL is published after promotion, its receipt timestamp is later than a requested target after the promotion, and the promoted timeline's WAL that would confirm that target is not archived
- **THEN** token issuance and download validation refuse that target with `409 target_time_unreachable`
- **AND** a pre-promotion target confirmed by the former timeline's archived transactions up to the fork point remains recoverable

#### Scenario: A final transaction is not a later confirmation

- **WHEN** a requested target equals the last decoded transaction timestamp and no later transaction is published
- **THEN** token issuance refuses that explicit target
- **AND** a target one microsecond earlier remains eligible when its WAL is contiguous

#### Scenario: Confirming WAL disappears after token issuance

- **WHEN** a valid target token was issued while confirming WAL was available, that WAL is removed, and remaining late-received WAL has insufficient source-time evidence
- **THEN** downloading with that token returns `409 target_time_unreachable` before archive streaming begins

## ADDED Requirements

### Requirement: Cleanup preserves recovery points around repeated Patroni switches

On PostgreSQL 17 and 18, physical backup cleanup and storage file deletion SHALL preserve the files, manifests, WAL and timeline histories required by recovery points still protected by retention after repeated Patroni switches. This SHALL include a reachable target before a switch, a target between a switch and the next completed backup, and a target after that backup.

#### Scenario: Retained targets around switches survive both cleanup stages

- **WHEN** two Patroni switchovers have completed, retention still protects a reachable target before the first switch, a target one minute after the newest switch that lies before the next completed backup on the promoted timeline, and a target after that backup, and physical cleanup and storage deletion have processed their work
- **THEN** newly downloaded sets for all three targets start independent PostgreSQL clusters with their respective rows, schema definitions and sequence definitions
- **AND** new generated-ID writes succeed in every recovered cluster
- **AND** each archive-based set includes the published history of every timeline after its FULL's timeline on its line, in its FULL's storage

#### Scenario: Cleanup keeps the WAL that crosses a switch

- **WHEN** a completed FULL on the former timeline is retained, the promoted timeline's history is published in that FULL's storage, no backup has completed on the promoted timeline yet, a transaction later than a target after the switch is archived, and both cleanup stages have run
- **THEN** the WAL of the promoted timeline and its history remain archived
- **AND** that target receives a restore token, and the recovered cluster contains the rows committed up to it

## REMOVED Requirements

### Requirement: Cleanup preserves retained recovery points after repeated Patroni switches

**Reason**: It required a target between a promotion and its replacement FULL to stay unreachable. Restores now cross the promotion, so cleanup has to keep that window restorable instead.

**Migration**: Replaced by "Cleanup preserves recovery points around repeated Patroni switches", which protects targets before a switch, inside the switch window and after the next backup.
