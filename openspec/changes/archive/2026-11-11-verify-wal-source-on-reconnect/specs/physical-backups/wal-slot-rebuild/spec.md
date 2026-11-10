## ADDED Requirements

### Requirement: WAL slot mutations require the expected cluster

Before creating a WAL reception slot or changing a slot as part of a rebuild, Databasus SHALL verify the saved cluster identity on the same connection that performs the mutation. Verification on another connection SHALL NOT authorize the mutation. Missing saved identity or an identity mismatch SHALL prevent creation, deletion, recreation and termination of a slot consumer as part of that operation. Refusal SHALL NOT request a replacement FULL merely to continue the original archive against the foreign cluster.

#### Scenario: First streamer startup reaches another cluster

- **WHEN** the streamer starts against an unrelated cluster where its configured slot is absent
- **THEN** the identity check refuses the operation before a slot is created

#### Scenario: Rebuild reaches another cluster

- **WHEN** a rebuild trigger was observed on the expected source but the connection used to mutate the slot reaches an unrelated cluster
- **THEN** no slot is dropped, created or recreated on that cluster
- **AND** no consumer there is terminated as part of the refused rebuild

#### Scenario: Rebuild uses separate connections

- **WHEN** slot deletion and slot creation use different connections and only one reaches the expected cluster
- **THEN** each mutation is independently authorized by identity verification on its own connection
- **AND** the mutation whose connection reaches a different cluster is refused

#### Scenario: Consumer termination reaches another cluster after release probes

- **WHEN** slot-release probes reach the expected cluster but the connection used to inspect and terminate a consumer reaches an unrelated cluster
- **THEN** the operation is refused because of source identity
- **AND** the unrelated cluster's same-named slot and active consumer remain unchanged

## MODIFIED Requirements

### Requirement: A lost slot or a slot held by another consumer is handled on the first sample

On the first sample from the verified expected cluster that reports the slot's status as `lost`, Databasus SHALL ask for a rebuild with the lost slot as the reason, whatever the receiver's state. On the first sample from the verified expected cluster that shows the slot held by a backend that is not Databasus's own receiver, it SHALL ask for a rebuild with the stolen slot as the reason. It SHALL NOT drop a slot while a consumer that is not its own holds it. Both are subject to the hourly limit on rebuild attempts. Source identity refusal SHALL take priority over these triggers and SHALL NOT be classified as a lost or stolen slot.

A slot whose status is `extended` or `unreserved` SHALL raise the existing slot retention alert after 5 minutes. Databasus SHALL NOT ask for a rebuild because of that status or because of lag while the status lasts.

#### Scenario: Lost slot

- **WHEN** a sample from the verified expected cluster reports the slot's status as `lost`
- **THEN** a rebuild with the lost slot as the reason is asked for in that tick, whatever the receiver's state

#### Scenario: Foreign consumer

- **WHEN** a sample from the verified expected cluster shows the slot held by a backend that is not Databasus's receiver
- **THEN** a rebuild with the stolen slot as the reason is asked for in that tick
- **AND** the slot is not dropped while that backend holds it

#### Scenario: Extended slot above the lag threshold

- **WHEN** the slot's status is `extended`, the receiver has run for 10 minutes and the lag is above the threshold and growing
- **THEN** no rebuild is asked for
- **AND** the slot retention alert is raised once the status has lasted 5 minutes

#### Scenario: Unrelated cluster reports a lost slot

- **WHEN** a slot inspection reaches an unrelated cluster whose same-named slot reports `lost`
- **THEN** Databasus reports the identity refusal rather than requesting a rebuild based on that slot
