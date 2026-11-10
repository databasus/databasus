## MODIFIED Requirements

### Requirement: The statistics state what the backups can restore

The statistics SHALL state the latest moment to which the database's backups can restore it at the time of the request. That moment SHALL be the one a restore to the latest available point reaches. Archived WAL on a timeline above 1 SHALL extend recovery only after that timeline's verified history is published under the current backup storage selection. Timeline 1 needs no history file. Without required history, the latest restorable moment SHALL be the newest completed backup's completion time, and each completed backup opens only its own completion point rather than a stretch through subsequent WAL. These values SHALL use the same prerequisite as restore token issuance and restore downloads.

That moment starts from the newest completed backup of the newest chain, full or incremental, and, when the history prerequisite is satisfied, extends through the WAL that follows that backup without a gap. With eligible WAL after that backup, it is the time Databasus received the last segment before the first gap after that backup, or the newest segment when there is no such gap. Without eligible WAL after that backup, it is the backup's completion time. When no full backup has completed, the statistics SHALL state that no moment can be restored yet.

The statistics SHALL also state the stretches of time the newest chain can restore to, oldest first. When the history prerequisite is satisfied, each completed backup of that chain, full or incremental, opens a stretch from its completion time to the moment the WAL after it reaches without a gap, by the same rule as above. Stretches that overlap or touch SHALL be merged. A WAL segment that is still being uploaded SHALL end a stretch like a gap. The last stretch SHALL end at the latest restorable moment. Without a completed full backup, the list SHALL be empty.

#### Scenario: Streaming without gaps

- **WHEN** the newest full backup completed yesterday, required timeline history is published or the timeline is 1, and every WAL segment since then is archived without a gap
- **THEN** the latest restorable moment is the time Databasus received the newest archived segment

#### Scenario: Gap in the newest chain

- **WHEN** the history prerequisite is satisfied, the slot was rebuilt at 07:51, which left a WAL gap after the segment received at 07:50, no backup completed after the gap, and segments after the gap keep arriving
- **THEN** the latest restorable moment is 07:50
- **AND** a restore to the latest available point reaches the same moment

#### Scenario: Incremental backup after the gap

- **WHEN** the history prerequisite is satisfied, the newest chain has a WAL gap at 07:51, an incremental backup of that chain completed at 09:00, and every segment after 09:00 is archived without a gap
- **THEN** the latest restorable moment is the time Databasus received the newest archived segment
- **AND** the restorable stretches are two: from the full backup's completion to 07:50, and from 09:00 to the newest archived segment

#### Scenario: Incremental backup before the gap

- **WHEN** the history prerequisite is satisfied, an incremental backup of the newest chain completed at 06:00, and the chain has a WAL gap at 07:51 with segments after it
- **THEN** the latest restorable moment is the time Databasus received the last segment before the gap
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
