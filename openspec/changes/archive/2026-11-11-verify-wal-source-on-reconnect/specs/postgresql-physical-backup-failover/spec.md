## ADDED Requirements

### Requirement: WAL identity enforcement permits promotion of the original cluster

When an unchanged source address moves from a primary to its real physical replica after promotion, Databasus SHALL accept the retained system identifier for WAL reception and apply its existing timeline rules. An equal system identifier SHALL NOT authorize an incremental backup to extend a root FULL from another timeline. The existing timeline-related failure and immediate replacement FULL policy SHALL remain in effect. Recovery after the switch SHALL use the new FULL and the existing recovery policy, without promising PITR across timelines.

#### Scenario: Address switches to a promoted replica

- **WHEN** a primary and its real replica have equal system identifiers, the replica is promoted to a newer timeline, and the unchanged address starts routing new connections to that replica
- **THEN** WAL reception accepts the replica's identity and can archive WAL from its timeline
- **AND** an incremental against the original timeline is refused as `CHAIN_BROKEN` with reason `TIMELINE_SWITCH_DETECTED`
- **AND** the existing policy produces a replacement FULL on the promoted source

#### Scenario: Replacement FULL is recovered

- **WHEN** a replacement FULL completes after the same-cluster endpoint switch
- **THEN** its recovery bundle can be downloaded and restored with control rows from the promoted source
- **AND** successful recovery does not depend on replay across the promotion boundary
