## ADDED Requirements

### Requirement: Conflicting timeline history blocks its fork point

When Databasus publishes verified history for a database and timeline, it SHALL record the history's parent timeline and its fork point's LSN with the published entry. When the database already has a published entry for that timeline, Databasus SHALL compare the parent timeline and fork point's LSN of the fresh history with those of the entry:

- If they match, the entry SHALL be reused unchanged.
- If they differ, for example because a cluster restored from a backup reused a timeline number with another fork point, the entry's artifact, parent timeline and fork point SHALL be kept, the entry SHALL be marked conflicting, and nothing new SHALL be published. The mark SHALL be permanent: a later report that matches the entry SHALL NOT make the fork point usable again. No restore set SHALL cross a conflicting fork point, and no incremental backup SHALL extend a chain across it. The conflict SHALL be logged with the database and timeline, without the history contents.
- If the entry was published by a Databasus version that did not record fork points, the entry SHALL be reused unchanged and SHALL NOT be marked conflicting. Its fork point stays unknown and therefore unusable.

Matching cluster identity SHALL NOT establish that two histories describe the same branch.

#### Scenario: Same timeline number, different fork point

- **WHEN** history for timeline 2 is published with a fork point at one LSN and the source later reports a valid history for timeline 2 with another fork point
- **THEN** the published artifact, parent timeline and fork point are kept, the entry is marked conflicting, and nothing new is published
- **AND** a warning names the database and timeline and contains no history contents
- **AND** a restore target that would require crossing into timeline 2 from an earlier timeline is refused with `409` and code `target_time_unreachable`
- **AND** an incremental backup that would extend an earlier-timeline chain onto timeline 2 ends as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`

#### Scenario: The same history arrives again

- **WHEN** the source reports history for a timeline whose parent timeline and fork point match a published entry that was never marked conflicting
- **THEN** the published entry is reused and the fork point remains usable

#### Scenario: A conflict mark persists

- **WHEN** an entry was marked conflicting and the source later reports history whose parent timeline and fork point match the entry
- **THEN** the entry stays marked conflicting and its fork point stays unusable

#### Scenario: History arrives for an entry published without a fork point

- **WHEN** an entry for timeline 2 was published by an earlier Databasus version without a fork point and the source reports history for timeline 2
- **THEN** the entry is reused unchanged, no new entry is published and the entry is not marked conflicting
- **AND** a restore target that would require crossing into timeline 2 from an earlier timeline is refused with `409` and code `target_time_unreachable`
