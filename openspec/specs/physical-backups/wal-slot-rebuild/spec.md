# WAL Slot Rebuild Specification

## Purpose

Defines when Databasus asks to drop and recreate the replication slot of a WAL streaming database. A rebuild always leaves a gap in the WAL chain and forces a new full backup, so Databasus asks for one only when keeping the slot would harm the source or cannot work.

## Requirements

### Requirement: The lag threshold asks for a rebuild only when the running receiver does not reduce the lag

When a lag threshold is configured, Databasus SHALL judge the slot lag only over time the receiver spends running. It SHALL compare consecutive samples taken within the same receiver run, and add up their duration and their lag change. Time while the receiver is stopped, and the lag that builds up during it, SHALL count for nothing. A receiver restart SHALL NOT discard what was added up before it.

Databasus SHALL ask for a rebuild with the lag threshold as the reason when, over at least 5 minutes of added-up running time:

- every sample had a lag above the threshold;
- the lag shrank in total by less than 5% of the threshold.

When the lag shrank by 5% of the threshold or more over 5 minutes of running time, the receiver is draining. The added-up time and change then SHALL start again from zero.

They SHALL also start again from zero when a sample has a lag at or below the threshold, or when the slot's status is `lost`, `extended` or `unreserved`. A lag threshold of zero SHALL never ask for a rebuild.

A rebuild that Databasus asks for SHALL be subject to the existing limit on rebuild attempts per hour.

#### Scenario: Resume after a back-pressure pause

- **WHEN** the receiver was paused until the local queue drained, the slot lag reached twice the threshold meanwhile, and the receiver now runs and the lag shrinks
- **THEN** no rebuild is asked for

#### Scenario: Start after Databasus was down

- **WHEN** Databasus starts after 10 minutes of downtime in front of a busy source, and the receiver begins with a lag above the threshold that shrinks
- **THEN** no rebuild is asked for

#### Scenario: Slow catch-up longer than 5 minutes

- **WHEN** the receiver has run for 20 minutes with the lag above the threshold, and the lag shrank by more than 5% of the threshold in every 5 minutes
- **THEN** no rebuild is asked for

#### Scenario: Short runs between pauses

- **WHEN** back pressure pauses the receiver every 3 minutes, the lag grows during each pause, and it shrinks during each run
- **THEN** no rebuild is asked for

#### Scenario: Receiver cannot keep up

- **WHEN** the receiver runs and the lag stays above the threshold and grows
- **THEN** a rebuild with the lag threshold as the reason is asked for on the first sample after 5 minutes of running time, and not before

#### Scenario: Lag fluctuates above the threshold

- **WHEN** the receiver runs and the lag moves up and down by less than 5% of the threshold around a level above the threshold
- **THEN** a rebuild is asked for on the first sample after 5 minutes of running time

#### Scenario: Stalled receiver restarted repeatedly

- **WHEN** the receiver is restarted every 70 seconds because it stopped confirming WAL, and the lag stays above the threshold and grows
- **THEN** a rebuild is asked for once the sampled intervals inside those runs add up to 5 minutes
- **AND** with 30-second samples, that is roughly 7 to 12 minutes of wall time

#### Scenario: Receiver stopped

- **WHEN** the lag is above the threshold and the receiver is not running
- **THEN** no rebuild is asked for, and the time does not count towards the 5 minutes

#### Scenario: Lag drops below the threshold

- **WHEN** the lag stays above the threshold for 4 minutes of running time, drops below it for one sample, and rises above it again
- **THEN** no rebuild is asked for until 5 more minutes of running time pass with the lag above the threshold

#### Scenario: Extended episode ends above the threshold

- **WHEN** the slot's status was `extended` and returns to `reserved` with the lag still above the threshold
- **THEN** no rebuild is asked for until 5 minutes of running time pass after the return, with the lag above the threshold and not shrinking by 5% of the threshold

#### Scenario: Hourly limit reached

- **WHEN** the lag rule asks for a rebuild after three rebuild attempts in the last hour
- **THEN** the slot is not dropped

#### Scenario: No threshold

- **WHEN** the lag threshold is zero
- **THEN** no rebuild is ever asked for because of lag

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
