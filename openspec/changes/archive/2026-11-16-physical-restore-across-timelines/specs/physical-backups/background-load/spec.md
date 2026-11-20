## MODIFIED Requirements

### Requirement: Retention runs once a minute and reads each closed chain's WAL once

The retention cleaner SHALL run once a minute. To decide which closed chains are eligible for deletion and in what order, including the grace check, it SHALL read each closed chain's archived WAL at most once per pass, as a summary rather than as individual segment records, and SHALL NOT read WAL outside that chain's span along its timeline line. A summary over a line that crosses fork points MAY take one query per timeline on the line. Deleting an eligible chain and sweeping WAL that belongs to no chain are outside this rule and keep reading individual segments.

#### Scenario: Idle instance with retained chains

- **WHEN** a retention pass runs and no chain is eligible for deletion
- **THEN** no individual archived WAL segment record is loaded to decide that

#### Scenario: A chain becomes eligible for deletion

- **WHEN** a closed chain falls outside the configured retention and outside the grace period
- **THEN** its deletion begins within about one minute

### Requirement: A chain's end time is unchanged

A chain's end time, which drives retention ordering and the grace period, SHALL be the latest of three moments: the completion of its full backup, the completion of any of its incremental backups, and the receipt of any archived WAL segment inside its span along its timeline line.

#### Scenario: WAL arrives after the last incremental backup

- **WHEN** a chain's newest archived WAL segment was received after its last incremental backup completed
- **THEN** the chain's end time is the receipt time of that segment

#### Scenario: Chain without archived WAL

- **WHEN** a chain has no archived WAL segment in its span
- **THEN** the chain's end time is the latest completion time among its full and incremental backups

#### Scenario: WAL on the promoted timeline sets the end time

- **WHEN** a chain's FULL is on timeline 1, its line continues on timeline 2, and the newest segment of its span is a timeline 2 segment received after every backup of the chain completed
- **THEN** the chain's end time is the receipt time of that timeline 2 segment

### Requirement: A chain's WAL lookup returns exactly the chain's segments

Looking up the archived WAL of a chain SHALL return every segment that overlaps the chain's span along its timeline line, ordered by position, and SHALL NOT read segments that end at or before the span begins. A segment that starts before the span but carries data from the span's start SHALL be included. The segment holding a fork point's LSN is the one whose range contains that LSN, so when the LSN falls on a segment boundary it is the segment that starts there. On a timeline the line leaves, the lookup SHALL return and read only segments that start before the fork point's LSN. On the following timeline it SHALL start at the segment holding that LSN.

#### Scenario: Older chain among many

- **WHEN** the WAL of a chain in the middle of a database's history is looked up
- **THEN** the result holds that chain's segments only
- **AND** no segment of an earlier chain is read

#### Scenario: Full backup starts in the middle of a segment

- **WHEN** a chain's full backup starts partway into an archived segment
- **THEN** that segment is included in the chain's WAL

#### Scenario: Newest chain

- **WHEN** the WAL of the newest chain is looked up
- **THEN** the result runs from the segment holding the chain's start to the newest archived segment along its line

#### Scenario: Chain that crosses a promotion

- **WHEN** a chain's FULL is on timeline 1, timeline 2 leaves timeline 1 at a fork point inside a segment, and the WAL of the chain is looked up
- **THEN** the result holds timeline 1 segments from the chain's start through the segment holding the fork point's LSN, followed by timeline 2 segments from the segment holding that LSN
- **AND** no timeline 1 segment that starts after the fork point's LSN is returned or read

#### Scenario: Fork point on a segment boundary

- **WHEN** timeline 2 leaves timeline 1 exactly at the start of a segment
- **THEN** the timeline 1 part of the result ends with the segment before that boundary, and the timeline 2 part starts with the segment that starts there
