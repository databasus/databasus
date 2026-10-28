# Physical Backup WAL Ranges Specification

## Purpose

Defines how the physical backup list shows WAL as range rows instead of one row per segment, where a range ends, and how the newest range updates.

## Requirements

### Requirement: Consecutive WAL segments appear as one range row

The physical backup list SHALL show consecutive committed WAL segments as a single WAL range row. A range row SHALL show the names of its first and last segments, the number of segments, their total size, the time the first segment was received and the time the last segment was received.

#### Scenario: Continuous WAL after an incremental backup

- **WHEN** 240 contiguous WAL segments on one timeline were received after an incremental backup and before any later FULL or INCREMENTAL backup
- **THEN** the list shows one WAL range row for them
- **AND** that row shows the first and the last segment names, 240 segments, the sum of their sizes and the receive times of the first and the last segment

### Requirement: Where a WAL range ends

A WAL range SHALL end, and the next segment SHALL start a new range, when any of these holds:

- the next segment does not start where the previous segment on the same timeline ended (a gap),
- the next segment is on a different timeline,
- a FULL or INCREMENTAL backup shown in the same list sits between the two segments in the list order.

#### Scenario: Gap in the WAL sequence

- **WHEN** one segment in the middle of a contiguous sequence is missing from the catalog
- **THEN** the list shows two WAL range rows, one on each side of the missing segment

#### Scenario: Timeline switch

- **WHEN** segments continue on a new timeline after a promotion
- **THEN** the segments of the new timeline form a range separate from the segments of the old timeline

#### Scenario: Backup between segments

- **WHEN** a FULL backup was created while WAL kept streaming
- **THEN** the list shows the FULL backup between two WAL range rows

#### Scenario: Backups filtered out

- **WHEN** the user filters the list to WAL only and the WAL around a FULL backup has no gap
- **THEN** the segments on both sides of that backup form one range row

### Requirement: A range that follows a gap is marked

A WAL range row SHALL be marked as following a gap when an older committed segment exists on the same timeline and the range does not start where the newest such segment ended. A range with no older segment on its timeline SHALL NOT be marked.

#### Scenario: Range after a missing segment

- **WHEN** a segment is missing between two ranges on the same timeline
- **THEN** the newer range is marked as following a gap
- **AND** the older range is not marked unless a gap also precedes it

#### Scenario: Oldest retained range

- **WHEN** retention has deleted every segment older than a range
- **THEN** that range is not marked as following a gap

#### Scenario: First range on a new timeline

- **WHEN** a range is the first one on its timeline
- **THEN** it is not marked as following a gap

### Requirement: Filters apply to segments before they are grouped

The physical backup list filters SHALL select WAL segments first, and only the selected segments SHALL be grouped into ranges.

#### Scenario: Before-date filter cuts a range

- **WHEN** the user filters by a date that falls inside a range of contiguous segments
- **THEN** the list shows a range made of the segments received before that date only, with their count and size

#### Scenario: Status filter that excludes WAL

- **WHEN** the user filters by a status that WAL segments never have
- **THEN** the list shows no WAL range rows

### Requirement: The newest range grows in place

When a new segment is committed right after the last segment of the newest range on the same timeline, and no FULL or INCREMENTAL backup was created after that range, the list SHALL update that range on the next refresh instead of adding a row. The range keeps its position and identity, and its last segment name, segment count, total size and last receive time change.

#### Scenario: Segment committed while the list is open

- **WHEN** the list shows the newest range with 240 segments and one more contiguous segment is committed
- **THEN** after the next refresh the same row shows 241 segments, the new last segment name, the larger size and the later end time
- **AND** the number of rows in the list does not change

### Requirement: A range counts as one row for paging

The physical backup list SHALL count each WAL range row as one row toward the page size, however many segments it covers.

#### Scenario: Many segments and one backup

- **WHEN** a database has 1500 contiguous WAL segments received after its only FULL backup and the user opens the list with a page size of 50
- **THEN** the first page shows the WAL range row and the FULL backup

### Requirement: WAL range rows offer no actions

A WAL range row SHALL offer no delete, download, cancel or restore action, as WAL segment rows did not. The API SHALL NOT delete a WAL segment either: a deletion by the ID of a WAL segment SHALL get `404` with `code` `not_found` and SHALL remove nothing.

#### Scenario: Actions column for a range

- **WHEN** the list shows a WAL range row to a user who can manage the database
- **THEN** the row's actions cell is empty

#### Scenario: Deleting a WAL segment through the API

- **WHEN** a member of the database's workspace deletes a physical backup by the ID of an archived WAL segment
- **THEN** the response is `404` with `code` `not_found`
- **AND** the WAL range that holds the segment is listed with the same count and size as before
