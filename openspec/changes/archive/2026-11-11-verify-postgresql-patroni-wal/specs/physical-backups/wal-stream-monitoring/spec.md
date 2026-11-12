## MODIFIED Requirements

### Requirement: The statistics state one health verdict with reasons

The statistics SHALL state one verdict:

- **not streaming**: WAL streaming is not expected to run, because the database is not in WAL streaming mode, its backups are disabled or it has no storage;
- **collecting data**: the streamer is not failing, no sample is newer than the current session's start, and that session started less than 2 minutes ago, or no streamer has started yet;
- **broken**: at least one reason applies at the broken level;
- **needs attention**: at least one reason applies at the needs-attention level and none at the broken level;
- **healthy**: no reason applies.

The first matching verdict in this order wins.

Each reason SHALL carry its code and its level. It SHALL also carry the time the condition began when that time is known, and the measured amount and the threshold it was compared with when the rule compares an amount. A needs-attention reason that is read from samples SHALL appear only when its condition holds in the two newest samples, so a single unusual sample does not change the verdict. Reasons whose rule already waits for a duration are exempt.

| Reason | Needs attention | Broken |
|---|---|---|
| Streaming stopped | - | the streamer has been failing for 1 minute or longer, or the newest sample is older than 2 minutes, and the newest sample did not find the source unreachable |
| Source unreachable | the newest sample found the source unreachable, and the source has been unreachable for 1 to 5 minutes since | the same for 5 minutes or longer, also when streaming stopped after that sample |
| WAL chain broken | - | the newest completed full backup's chain has a WAL gap below the archive progress point that the latest restore cannot cross, because no completed backup of that chain starts after the gap; it clears when a completed incremental backup of that chain or a newer full backup starts after the gap |
| Slot missing | - | the newest sample found the source without the slot |
| Slot lost | - | the newest sample reports the slot's status as `lost` |
| Archive not advancing | the archive progress point has stood still for 10 minutes or longer while the source keeps writing | the same for the archive staleness threshold of 30 minutes or longer, which is when the "archiving fell behind" notification fires |
| Archive slower than source | the unarchived WAL grew by at least the backlog threshold over the last 10 minutes | - |
| Slot keeps extra WAL | the slot's status has been `unreserved` for longer than 5 minutes, or `extended` for longer than 5 minutes while the slot lag grows | - |
| Slot limit approaching | the source sets a limit, and the remaining WAL allowance would run out in under 1 hour if the slot lag keeps growing at its rate of the last 10 minutes | - |
| Near slot rebuild | the slot lag is at least 80% of the configured lag at which Databasus rebuilds the slot, and the slot's status is neither `extended` nor `unreserved` | - |
| Local queue filling | the local queue is at least half the size at which the receiver is paused | - |
| Frequent restarts | receiver restarts and streaming session starts in the last hour add up to 3 or more | - |
| Forced WAL switch refused | the newest sample says the source refused the forced WAL switch | - |

The backlog threshold is 64 MiB or four WAL segments, whichever is larger. Growth of the unarchived WAL SHALL be measured between the lowest value in the first 2.5 minutes of the window and the lowest value in its last 2.5 minutes, so normal upload batches do not count as growth.

When the newest sample is more than 2 minutes old, reasons read from that sample SHALL be dropped, except the source unreachable reason described in the table. The WAL chain broken reason does not depend on samples and stays.

Reasons read from the slot or the archive progress point SHALL be dropped while the newest sample has no source-side values.

The statistics SHALL also list notices, which never change the verdict:

- **no slot WAL limit**: the newest sample was taken with the source reachable, the slot's status is not `lost`, and the source sets no limit on WAL kept for the slot;
- **slot keeps extra WAL steadily**: the slot's status has been `extended` for longer than 5 minutes while the slot lag does not grow;
- **no full backup yet**: no full backup of the database has completed, so archived WAL cannot be restored yet.

#### Scenario: Everything in order

- **WHEN** the source is reachable, archiving keeps pace and no threshold above is crossed
- **THEN** the verdict is healthy and no reason is listed

#### Scenario: Single unusual sample

- **WHEN** only the newest sample shows the local queue above half its pause size
- **THEN** the local queue filling reason is not listed

#### Scenario: Short network blip

- **WHEN** the source stopped answering 30 seconds ago
- **THEN** the source unreachable reason is not listed

#### Scenario: Source unreachable for a few minutes

- **WHEN** the source has been unreachable for 3 minutes and the streamer keeps sampling
- **THEN** the verdict is needs attention with the source unreachable reason and the time it began

#### Scenario: Source down for a while

- **WHEN** the newest sample found the source unreachable, the streamer then stopped, and the source has been unreachable for 6 minutes
- **THEN** the verdict is broken with the source unreachable reason, not the streaming stopped reason

#### Scenario: Archive stuck

- **WHEN** the source keeps writing and the archive progress point has not moved for 12 minutes
- **THEN** the verdict is needs attention with the archive not advancing reason
- **AND** after 30 minutes without movement the same reason is listed at the broken level

#### Scenario: Streamer stopped

- **WHEN** the streamer failed and the newest sample, taken with the source reachable, is ten minutes old
- **THEN** the verdict is broken with the streaming stopped reason
- **AND** the response carries the time of that newest sample
- **AND** no other reason read from that sample is listed

#### Scenario: Restart after a deploy

- **WHEN** Databasus restarted 20 seconds ago after 10 minutes of downtime, and the new session has not taken a sample yet
- **THEN** the verdict is collecting data, not broken

#### Scenario: Crash loop on a new database

- **WHEN** WAL streaming was enabled five minutes ago and every session has failed before its first sample
- **THEN** the verdict is broken with the streaming stopped reason

#### Scenario: Slot rebuilt

- **WHEN** Databasus rebuilt the slot after it was lost and the requested full backup has not completed yet
- **THEN** the verdict is broken with the WAL chain broken reason, even though the new slot is healthy
- **AND** once a newer full backup completes, the reason is no longer listed

#### Scenario: Gap bridged by an incremental backup

- **WHEN** the newest chain has a WAL gap below the archive progress point, an incremental backup of that chain completed after the gap, and every segment after that backup is archived without a gap
- **THEN** the WAL chain broken reason is not listed
- **AND** the latest restorable moment is the latest confirmed inclusive PITR bound from source transaction timestamps in contiguous archived WAL

#### Scenario: Gap after the newest incremental backup

- **WHEN** an incremental backup of the newest chain completed, and a WAL gap below the archive progress point follows it
- **THEN** the verdict is broken with the WAL chain broken reason

#### Scenario: Failed incremental backup after the gap

- **WHEN** the only incremental backup started after the gap failed or is still running
- **THEN** the verdict is broken with the WAL chain broken reason

#### Scenario: Slot removed on the source

- **WHEN** the newest sample found the source without the slot
- **THEN** the verdict is broken with the slot missing reason, not the source unreachable reason

#### Scenario: Just enabled

- **WHEN** WAL streaming was just enabled and no sample exists yet
- **THEN** the verdict is collecting data

#### Scenario: Backups disabled

- **WHEN** a database in WAL streaming mode has its backups disabled
- **THEN** the verdict is not streaming
- **AND** the series still holds the samples recorded before

#### Scenario: Several problems at once

- **WHEN** the local queue has been half full for two samples and the slot is missing
- **THEN** the verdict is broken
- **AND** both reasons are listed, each with its own level

#### Scenario: Busy cluster with steady extra WAL

- **WHEN** the slot's status has been `extended` for an hour while the slot lag stays flat
- **THEN** the slot keeps extra WAL reason is not listed
- **AND** the slot keeps extra WAL steadily notice is listed

#### Scenario: Paused receiver above the rebuild lag

- **WHEN** the receiver is paused for a full local queue and the slot lag is at 90% of the rebuild lag with status `reserved`
- **THEN** the near slot rebuild reason is listed

#### Scenario: No limit on the source

- **WHEN** the source's `max_slot_wal_keep_size` is `-1` and nothing else is wrong
- **THEN** the verdict is healthy
- **AND** the no slot WAL limit notice is listed

#### Scenario: Slot already lost

- **WHEN** the newest sample reports the slot's status as `lost`
- **THEN** the no slot WAL limit notice is not listed

### Requirement: The statistics state what the backups can restore

The statistics SHALL state the latest moment to which the database's backups can restore it at the time of the request. That moment SHALL be the latest confirmed explicit PITR target. A latest restore without a target may also apply the final transaction that confirms this boundary. Archived WAL on a timeline above 1 SHALL extend recovery only after that timeline's verified history is published under the current backup storage selection. Timeline 1 needs no history file. Without required history, the latest restorable moment SHALL be the newest completed backup's completion time, and each completed backup opens only its own completion point rather than a stretch through subsequent WAL. These values SHALL use the same prerequisite as restore token issuance and restore downloads.

That moment starts from the newest completed backup of the newest chain, full or incremental, and, when the history prerequisite is satisfied, extends through the WAL that follows that backup without a gap. With eligible WAL after that backup, complete COMMIT or ABORT records in the selected timeline's contiguous published run SHALL confirm the upper bound. Because inclusive time recovery stops before a transaction strictly later than the target, this bound SHALL be one microsecond before the latest confirming source timestamp, never earlier than the completed-backup anchor. Receipt or upload time SHALL NOT extend it. A segment without timestamp evidence SHALL preserve the previously confirmed bound. Source time SHALL use the maximum trusted transaction timestamp even when timestamps decrease in WAL order. Without eligible WAL after that backup, it is the backup's completion time. When no full backup has completed, the statistics SHALL state that no moment can be restored yet.

The statistics SHALL also state the stretches of time the newest chain can restore to, oldest first. When the history prerequisite is satisfied, each completed backup of that chain, full or incremental, opens a stretch from its completion time to the moment the WAL after it reaches without a gap, by the same rule as above. Stretches that overlap or touch SHALL be merged. A WAL segment that is still being uploaded SHALL end a stretch like a gap. The last stretch SHALL end at the latest restorable moment. Without a completed full backup, the list SHALL be empty.

#### Scenario: Streaming without gaps

- **WHEN** the newest full backup completed yesterday, required timeline history is published or the timeline is 1, and every WAL segment since then is archived without a gap
- **THEN** the latest restorable moment is the latest confirmed inclusive PITR bound from source transaction timestamps in contiguous archived WAL

#### Scenario: Gap in the newest chain

- **WHEN** the history prerequisite is satisfied, the slot was rebuilt at 07:51, which left a WAL gap after a segment confirming an inclusive PITR bound of 07:50, no backup completed after the gap, and segments after the gap keep arriving
- **THEN** the latest restorable moment is 07:50
- **AND** an explicit PITR request for 07:50 remains eligible

#### Scenario: Incremental backup after the gap

- **WHEN** the history prerequisite is satisfied, the newest chain has a WAL gap at 07:51, an incremental backup of that chain completed at 09:00, and every segment after 09:00 is archived without a gap
- **THEN** the latest restorable moment is the latest confirmed inclusive PITR bound from source transaction timestamps in contiguous archived WAL
- **AND** the restorable stretches are two: from the full backup's completion to 07:50, and from 09:00 to the newest confirmed inclusive PITR bound

#### Scenario: Incremental backup before the gap

- **WHEN** the history prerequisite is satisfied, an incremental backup of the newest chain completed at 06:00, and the chain has a WAL gap at 07:51 with segments after it
- **THEN** the latest restorable moment is the latest confirmed inclusive PITR bound from source transaction timestamps before the gap
- **AND** the restorable stretches are one, from the full backup's completion to that moment

#### Scenario: Newer full backup

- **WHEN** an older chain has a gap, and a newer full backup completed after it
- **THEN** the restorable stretches start at the newer full backup's completion

#### Scenario: Full backup without WAL after it

- **WHEN** the only completed full backup finished at 14:02 and no WAL segment followed it
- **THEN** the latest restorable moment is 14:02

#### Scenario: No full backup yet

- **WHEN** no full backup of the database has completed
- **THEN** the statistics state that no moment can be restored yet
- **AND** the restorable stretches are empty

#### Scenario: Promoted timeline history is still pending

- **WHEN** the newest backup completed at 14:02 on timeline 2, subsequent WAL is archived, and verified history for timeline 2 is not published
- **THEN** the latest restorable moment is 14:02
- **AND** restorable stretches do not extend through subsequent WAL
- **AND** a latest restore reaches 14:02 as well

#### Scenario: History publication enables existing WAL coverage

- **WHEN** verified history for timeline 2 is published and contiguous archived WAL after the newest backup is already available
- **THEN** the next statistics response extends coverage through that WAL
- **AND** restore selection reaches the same moment without another completed backup
