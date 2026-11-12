## ADDED Requirements

### Requirement: Timeline history is verified against its source cluster

Before publishing new timeline history, Databasus SHALL verify that it was obtained from the configured PostgreSQL cluster and belongs to the requested timeline. A matching file name alone SHALL NOT establish cluster identity. Missing cluster identity, an identity mismatch, a mismatched response name, empty or malformed history, and a timeline that is invalid for the source SHALL prevent publication.

#### Scenario: Another cluster uses the same timeline number

- **WHEN** the source address reaches another cluster with a history file bearing the requested name
- **THEN** Databasus rejects the history
- **AND** no artifact, metadata or catalog entry is published for it

#### Scenario: Invalid history response

- **WHEN** the returned history has the wrong name, empty content or invalid ancestor timeline or WAL positions
- **THEN** Databasus rejects the response before writing it to backup storage

#### Scenario: Earlier history is still pending

- **WHEN** history for an earlier timeline remains pending and the configured cluster has since advanced to a later timeline
- **THEN** Databasus can publish the earlier history after verifying its cluster and requested timeline
- **AND** it does not require the requested timeline to equal the cluster's current timeline

### Requirement: Timeline history publication is atomic and repeatable

Databasus SHALL publish a timeline history artifact, its metadata and its catalog entry as one recoverable result. Repeated or concurrent processing of one database's timeline SHALL retain one published history entry and one matching artifact/metadata pair. Unpublished attempts SHALL remain eligible for cleanup. History publication SHALL obey the current backup storage selection and its generation, including when reusing an existing entry.

#### Scenario: The receiver and a FULL process the same history

- **WHEN** WAL reception and FULL completion concurrently process the same timeline
- **THEN** both resolve to the same published history entry
- **AND** any losing attempt's files are scheduled for cleanup without deleting the winning pair

#### Scenario: Storage changes during a history upload

- **WHEN** the selected backup storage changes before an in-flight history upload publishes
- **THEN** the old upload cannot publish into the replaced storage
- **AND** its artifact and metadata remain covered by the existing cleanup rules
- **AND** the current generation can acquire and publish the required history again

#### Scenario: Acquisition or storage write fails

- **WHEN** acquiring history or writing its artifact or metadata fails
- **THEN** no successful history entry is published
- **AND** a locally pending history notification remains available for a later attempt
- **AND** cancellation stops the acquisition without an unbounded wait
