## ADDED Requirements

### Requirement: Every WAL session belongs to the configured cluster

Before accepting WAL from a new source connection, Databasus SHALL compare the system identifier reported on that connection with the configured database's saved system identifier. The connection that supplies WAL SHALL be the connection whose identity was verified. This applies to initial reception, reception after a Databasus restart, and every reconnect, including any reconnect performed inside a receiver. Missing saved identity SHALL prevent reception rather than cause Databasus to learn an identity from the current address.

An identity mismatch SHALL prevent foreign WAL from entering the configured database's local queue, backup storage or catalog. Matching credentials, slot names, timeline numbers, WAL positions or WAL filenames SHALL NOT substitute for identity verification. A failed identity check SHALL NOT update the saved identifier.

#### Scenario: First reception reaches an unrelated cluster

- **WHEN** the configured address accepts a replication connection to another cluster with working credentials, a matching timeline, overlapping WAL positions and the configured slot
- **THEN** Databasus rejects reception because the system identifier differs
- **AND** no WAL from that cluster enters the configured database's queue, storage or catalog

#### Scenario: Running reception reconnects to another cluster

- **WHEN** reception from the expected cluster loses its connection and a subsequent connection reaches an unrelated cluster with otherwise compatible replication settings
- **THEN** the new connection is verified and refused before its WAL is accepted

#### Scenario: Preliminary checks and WAL reception reach different clusters

- **WHEN** preliminary SQL checks reach the expected cluster but a new WAL connection reaches an unrelated cluster
- **THEN** the actual WAL connection is refused because of its system identifier
- **AND** this applies both to initial reception and to a reconnect of an already running receiver

#### Scenario: Saved identity is missing

- **WHEN** a database configured for WAL reception has no saved system identifier
- **THEN** Databasus refuses reception until the configuration has a valid saved identity
- **AND** it does not accept the identity supplied by the current endpoint as a replacement

### Requirement: WAL receiver setup has a bounded lifetime

Each receiver attempt SHALL bound setup until the verified upstream session first enters replication. This bound SHALL include native receiver commands after local connection startup. A setup timeout SHALL stop the child, close the attempt's connections and release its private socket before retrying through normal supervision. The timeout SHALL NOT limit ongoing streaming after setup succeeds.

#### Scenario: Native setup stalls after successful identity verification

- **WHEN** the actual WAL session passes identity verification but the source stops answering a subsequent native setup command before entering replication
- **THEN** the setup deadline ends the attempt with a retryable outcome without reporting streaming
- **AND** the attempt releases its child, connections and private socket before another attempt starts
- **AND** a later attempt can stream when the source responds again

### Requirement: Identity refusal preserves the original recoverable archive

Rejecting a foreign source SHALL leave the expected cluster's existing backups and legitimate completed queued WAL available. Queued segments SHALL either remain available for the existing upload mechanism or be archived normally; refusal SHALL NOT discard them to reset reception. An ordinary Databasus restart with completed queued WAL SHALL obey the same rule. Normal retention and explicit user deletion remain governed by their existing requirements.

#### Scenario: Restart with original WAL queued and a foreign source

- **WHEN** Databasus stops normally with completed WAL of cluster A still queued and starts with its saved address now reaching cluster B
- **THEN** reception from B is refused
- **AND** completed queued WAL of A remains available locally or through its normal archived representation
- **AND** existing backups of A remain downloadable

#### Scenario: Original copy remains recoverable after refusal

- **WHEN** a foreign source has been refused and the expected source returns
- **THEN** an existing valid recovery point from before the refusal can be downloaded and restored with the original cluster's data
- **AND** retaining catalog rows alone does not establish successful recovery

### Requirement: WAL reception resumes only against the saved identity

While WAL streaming remains enabled, Databasus SHALL retry a refused source through its normal receiver supervision without changing the saved identifier. Once the expected source returns and a verified WAL session begins streaming, reception SHALL resume with the retained queue and existing resume rules. An identity refusal alone SHALL NOT request a slot rebuild or a replacement FULL. Existing backup-scheduler rules for an unresolved backup identity mismatch remain unchanged.

#### Scenario: Wrong source remains behind the address

- **WHEN** later receiver attempts still reach an unrelated cluster
- **THEN** each attempt is refused without receiving foreign WAL or adopting its identity

#### Scenario: Expected source returns

- **WHEN** the address again reaches the configured cluster after an identity refusal
- **THEN** a later supervised attempt verifies that cluster and resumes WAL reception without requiring the operator to change the saved identifier or erase the queue

#### Scenario: Backups were independently paused by their identity check

- **WHEN** reception has resumed against the expected cluster but automatic backups were paused by an unresolved backup identity mismatch
- **THEN** WAL resumption does not clear that backup failure
- **AND** automatic backups resume only after the existing successful operator-requested FULL recovery condition is met
