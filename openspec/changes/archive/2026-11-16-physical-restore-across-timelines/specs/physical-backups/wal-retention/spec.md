## MODIFIED Requirements

### Requirement: A WAL segment overlapping a retained backup's recovery window is never deleted as unreferenced

The retention sweep SHALL treat a WAL segment as unreferenced only when no retained, successfully completed physical full backup has the segment on its timeline line and begins before the end of that segment, and the segment is not protected as pending history.

A full backup's line starts on its own timeline at its start LSN. It follows the line of the newest timeline, as the PostgreSQL physical backup failover specification defines it, including the conditions that stop the line at a fork point: history not published, marked conflicting, published without a fork point, or outside the backup's storage. On a timeline the line leaves, the line holds the segments that start before the fork point's LSN. A full backup whose start LSN is at or after the fork point's LSN of every descendant of its timeline is not on the newest line; its own line is its timeline from its start, without an end.

A segment on a timeline above 1 is protected as pending history while that timeline has no published history entry and no completed full backup. This covers the time between the arrival of a promoted timeline's first WAL and the publication of its history.

Recovery positions inside a full backup do not fall on segment boundaries: a backup begins a short distance into the segment that holds it. The segment holding that position therefore begins earlier than the backup does, and it still carries every write recorded after the backup finished. That segment SHALL be retained.

#### Scenario: The full backup starts and ends inside one segment

- **WHEN** a full backup's start and end positions both fall inside a single WAL segment, and that segment has been archived
- **THEN** the retention sweep leaves the segment in place
- **AND** a restore to any point in time after the backup's end position and covered by the archived WAL succeeds

#### Scenario: A segment entirely older than every retained backup

- **WHEN** an archived WAL segment ends at or before the start of the earliest retained full backup whose line contains its timeline
- **THEN** the retention sweep deletes it as unreferenced

#### Scenario: The anchoring backup is pruned

- **WHEN** retention removes the oldest full backup, and a later full backup remains whose start position falls inside a WAL segment that the removed backup used to anchor
- **THEN** that segment is still retained, because the remaining backup starts before the segment ends
- **AND** point-in-time restore from the remaining backup stays available

#### Scenario: WAL of a promoted timeline before its first backup

- **WHEN** the source was promoted to timeline 2, the history of timeline 2 is published in the storage of a retained completed full backup on timeline 1, and no backup on timeline 2 has completed
- **THEN** the retention sweep keeps the timeline 2 WAL from the segment holding the fork point's LSN onward, also while no full backup is running
- **AND** a restore to a target on timeline 2 from the timeline 1 full backup succeeds

#### Scenario: Promoted timeline WAL arrives before its history

- **WHEN** timeline 2 WAL is archived, the history of timeline 2 is not published yet, no full backup on timeline 2 has completed, and the retention sweep runs with no full backup running
- **THEN** the sweep keeps the timeline 2 WAL
- **AND** once the history is published, a restore to a target on timeline 2 from the timeline 1 full backup succeeds

#### Scenario: Promoted timeline behind a stopped fork point

- **WHEN** the history of timeline 2 is marked conflicting or was published without a fork point, and no full backup on timeline 2 has completed
- **THEN** the retention sweep deletes the timeline 2 WAL as unreferenced, as it did before this change

#### Scenario: The former primary's WAL beyond the fork point

- **WHEN** archived timeline 1 WAL starts at or after the fork point's LSN at which timeline 2 leaves timeline 1, and no retained full backup on timeline 1 started at or after that LSN
- **THEN** that WAL is on the line of no full backup, and the retention sweep deletes it as unreferenced
- **AND** no restorable target depended on it, because restores follow the line of the newest timeline

#### Scenario: A full backup taken on the former timeline after the fork point

- **WHEN** a retained completed full backup on timeline 1 started after the fork point's LSN at which timeline 2 leaves timeline 1, for example on the former primary before it was fenced
- **THEN** its own timeline 1 WAL from its start is kept for as long as it is retained
- **AND** it stays restorable by its backup identifier

### Requirement: A completed full backup leaves an unbroken recovery chain

Once a physical full backup reports success and WAL streaming stays healthy, the system SHALL keep the archived WAL contiguous along that backup's timeline line from its end position onward, for as long as the backup is retained. No background maintenance SHALL introduce a gap immediately after a successful backup or at a fork point on its line.

#### Scenario: Retention runs seconds after a full backup completes

- **WHEN** a full backup completes, its boundary WAL segment is archived, and the retention sweep runs before any further backup exists
- **THEN** the archived WAL is unbroken from the backup's end position to the newest archived segment
- **AND** the reported latest restorable point is later than the backup's end position rather than equal to it

#### Scenario: Restoring to a point shortly after the backup

- **WHEN** a user requests a restore to a point in time a few seconds after a successful full backup, and WAL covering that point has been archived
- **THEN** the restore proceeds instead of being refused for a WAL gap

#### Scenario: Restoring while a newer full backup is running

- **WHEN** a newer full backup has started but has not completed, and the archived WAL remains contiguous from the preceding completed full backup
- **THEN** a point-in-time restore during that interval uses the preceding completed full backup
- **AND** the restore set contains the full available contiguous WAL run, including WAL beyond the newer backup's start and stop positions

#### Scenario: Retention runs after a promotion

- **WHEN** the source was promoted after a full backup completed, the promoted timeline's history is published in that backup's storage, and the retention sweep runs before any backup on the promoted timeline completes
- **THEN** the archived WAL stays unbroken from the backup's end position through the segment holding the fork point's LSN and on the promoted timeline to its newest archived segment

### Requirement: Deleting a full backup preserves WAL required by a surviving predecessor

Deleting a completed full backup SHALL retain its WAL when an older completed full backup remains on the same database and has that WAL on its timeline line. The older backup becomes the restore base after deletion and SHALL remain capable of replaying the shared WAL.

This rule applies when the FULL itself is removed. A `FULL_BACKUPS` retention policy may deliberately keep a FULL as a standalone restore point while removing its incrementals and owned WAL.

#### Scenario: A newer full backup is deleted

- **WHEN** a completed full backup is deleted while an older completed full backup remains on the same timeline
- **THEN** no WAL is deleted with the newer full backup
- **AND** point-in-time restore during the deleted backup's former interval succeeds from the older backup

#### Scenario: A full backup on a promoted timeline is deleted

- **WHEN** a completed full backup on timeline 2 is deleted while a completed full backup on timeline 1 remains and its line continues on timeline 2
- **THEN** no timeline 2 WAL or timeline 2 history is deleted with the timeline 2 backup
- **AND** point-in-time restore during the deleted backup's former interval succeeds from the timeline 1 backup across the fork point

#### Scenario: The earliest surviving full backup is deleted

- **WHEN** a completed full backup has no older completed predecessor whose line contains its timeline
- **THEN** deletion removes only WAL segments fully contained in its ownership span along its line
- **AND** a segment crossing the next full backup's start position remains available
- **AND** a timeline history is removed only when no surviving completed full backup has that timeline on its line
