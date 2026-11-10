## ADDED Requirements

### Requirement: WAL identity refusal is a persistent broken-stream reason

While WAL streaming is enabled, a known mismatch between the actual WAL source and the saved system identifier SHALL produce health level `BROKEN` with reason `SYSTEM_IDENTIFIER_MISMATCH` through existing WAL statistics. The reason SHALL remain observable after a streamer exits, during later startup attempts, and across an ordinary Databasus restart until a verified session against the expected cluster begins streaming. A known mismatch SHALL take priority over collecting-data grace periods and SHALL NOT require two periodic samples before becoming visible.

The live receiver SHALL NOT be reported as running merely because its process was started; its source session must first be verified and have entered WAL streaming. An unrelated SQL connection reaching the expected source SHALL NOT clear a mismatch observed on the WAL connection. Uploading legitimate old queued WAL or recording a periodic sample SHALL NOT clear it either.

#### Scenario: Mismatch occurs before the first sample

- **WHEN** a new WAL session is refused because its source identifier differs before any periodic sample is recorded
- **THEN** the existing statistics report `BROKEN` with reason `SYSTEM_IDENTIFIER_MISMATCH`
- **AND** live receiver state is not running

#### Scenario: Reclaim and restart preserve the reason

- **WHEN** a refused streamer is retried or Databasus restarts before a verified WAL session resumes
- **THEN** the mismatch reason remains visible throughout the new attempt

#### Scenario: SQL reaches the original cluster while WAL reaches another

- **WHEN** a SQL probe succeeds against the expected cluster while the actual WAL session is refused against another cluster
- **THEN** statistics retain `SYSTEM_IDENTIFIER_MISMATCH` and do not report healthy reception

#### Scenario: Old queue continues uploading

- **WHEN** legitimate queued WAL is archived after an identity refusal
- **THEN** that successful upload does not clear the mismatch reason

#### Scenario: Verified WAL reception resumes

- **WHEN** a session against the saved identity successfully enters WAL streaming
- **THEN** the mismatch reason is cleared
- **AND** normal monitoring determines stream health from then on

#### Scenario: WAL streaming is disabled

- **WHEN** the operator disables WAL streaming while a mismatch remains recorded
- **THEN** monitoring follows the existing `NOT_STREAMING` behavior
- **AND** disabling does not establish that the source identity has been verified

#### Scenario: Disabling overlaps terminal refusal persistence

- **WHEN** the receiver has detected an identity mismatch and streaming is disabled before that attempt finishes recording its terminal cause
- **THEN** disabling does not discard the unresolved refusal
- **AND** monitoring reports `NOT_STREAMING` while disabled and retains the refusal across restart and re-enable until verified reception resumes

### Requirement: Identity diagnostics identify the refusal and the recovery action

Existing WAL diagnostics and enabled chain-failure notifications SHALL identify a source identity refusal consistently as `SYSTEM_IDENTIFIER_MISMATCH`. Operator guidance SHALL explain that the configured address must again reach the original cluster and that the next verified attempt can resume reception. Diagnostics SHALL NOT recommend accepting a new identifier, discarding the original archive or rebuilding a slot on the unrelated cluster. Connection secrets SHALL NOT appear in diagnostics. Existing notification configuration and throttling SHALL continue to apply.

#### Scenario: Operator inspects a refused stream

- **WHEN** the operator opens existing WAL monitoring after an identity mismatch
- **THEN** the interface names the source identity problem and explains how reception resumes after the correct source returns

#### Scenario: Chain-failure notification is enabled

- **WHEN** an identity refusal stops reception and the database enables chain-failure notifications
- **THEN** its existing notification channel receives the identity refusal and recovery guidance under existing throttling rules
