# Replication Credentials Specification

## Purpose

Covers what a Databasus-provisioned replication user can do on a PostgreSQL source, and which source capabilities Databasus requires before it accepts a physical-backup configuration that promises recovery to a point in time.

## Requirements

### Requirement: A provisioned replication user can force WAL rotation where the platform allows it

When Databasus provisions a replication user on a source cluster, it SHALL also give that user the ability to force a WAL segment switch, so the recovery point of a rarely-written database is bounded by the rotation interval rather than by how long the current segment takes to fill.

Where the connected administrator cannot confer that ability, provisioning SHALL still succeed and deliver a working replication user, and SHALL report that forced WAL rotation is unavailable with those credentials.

#### Scenario: Self-managed source with an administrator that can confer the privilege

- **WHEN** a user asks Databasus to create a replication user on a self-managed PostgreSQL cluster, connected as an administrator able to confer it
- **THEN** the created user can force a WAL segment switch
- **AND** the credentials are accepted for a configuration that streams WAL

#### Scenario: Managed platform that withholds the privilege

- **WHEN** the same request is made against a managed platform whose administrator cannot confer the privilege
- **THEN** the replication user is still created and usable for physical backups
- **AND** the result reports that forced WAL rotation is unavailable

### Requirement: Provisioning widens the role's privileges by exactly one ability

Provisioning SHALL confer the ability to force a WAL segment switch and nothing else. The provisioned role SHALL remain limited to replication: no data access, no ability to create roles or databases, and no membership in an administrative role.

The ability SHALL be conferred on the provisioned role alone, never on a group that would extend it to other roles on the cluster. Where the source refuses, the cluster's privileges SHALL be left exactly as they were, with no partial or substitute grant.

#### Scenario: The provisioned role's reach after a successful grant

- **WHEN** provisioning succeeds including the ability to force a segment switch
- **THEN** the role can stream WAL and force a segment switch
- **AND** it still cannot read table data, create roles or databases, or act as an administrator

#### Scenario: No other role gains the ability

- **WHEN** provisioning confers the ability to force a segment switch
- **THEN** no role other than the one Databasus just created gains it

#### Scenario: The source refuses

- **WHEN** the connected administrator cannot confer the ability
- **THEN** the source's privileges are unchanged apart from the replication user's own creation
- **AND** no weaker or broader substitute privilege is granted in its place

### Requirement: A configuration that promises recovery to a point in time is refused unless the source can bound it

Recovery to an arbitrary point in time replays WAL segments that reached storage, and a segment reaches storage only once the source closes it. When a user configures continuous WAL streaming, Databasus SHALL refuse the configuration unless the supplied credentials can force a WAL segment switch.

The refusal SHALL name both ways out: confer the ability on the source, or choose a backup type that does not replay archived WAL. It SHALL NOT direct the user to enable archiving on the source, which would make the cluster retain every segment until an archiver confirms it. The refusal SHALL be reported the same way as the other unmet preconditions of a physical source, so a user meets it at the same moment they meet the rest.

This SHALL apply to credentials a user enters by hand as well as to credentials Databasus provisioned, and to a database being created as well as one being changed.

#### Scenario: Streaming configuration with credentials that cannot force a switch

- **WHEN** a user configures continuous WAL streaming with credentials that can stream WAL but cannot force a segment switch
- **THEN** the configuration is refused
- **AND** the refusal names conferring the ability and choosing a backup type without WAL replay

#### Scenario: A managed platform that will not confer the ability

- **WHEN** the source is a managed platform whose administrator cannot confer the ability on any role
- **THEN** the streaming configuration is refused there as well
- **AND** the refusal says continuous WAL streaming is unavailable on that source, rather than offering a source setting that would make the cluster retain WAL indefinitely

#### Scenario: Streaming configuration with credentials that can force a switch

- **WHEN** the credentials can force a segment switch
- **THEN** the configuration is accepted

#### Scenario: An existing streaming database that no longer qualifies

- **WHEN** a user changes a saved streaming database whose credentials cannot force a segment switch
- **THEN** the change is refused with the same explanation
- **AND** the backups already running for that database are not stopped by the refusal

### Requirement: Backup types that never replay archived WAL are not held to this requirement

A backup type that restores only to the point a backup itself finished does not read an archived segment, so the ability to force a segment switch SHALL NOT be required of it. Databasus SHALL accept such a configuration with credentials that can only stream.

#### Scenario: Full backups with credentials that cannot force a switch

- **WHEN** a user configures full backups, with or without incrementals, using credentials that cannot force a segment switch
- **THEN** the configuration is accepted
- **AND** backups and restores to those backups work with no privilege error

#### Scenario: Raising the backup type later

- **WHEN** a user changes such a database to stream WAL continuously
- **THEN** the requirement above applies to the change, and it is refused if the source cannot bound the recovery point

### Requirement: Timeline history requires no additional source privileges

Databasus SHALL obtain timeline history with the replication credentials it provisions, without granting administrative membership, server-file read access, broad table access or execution of SQL server-file reading functions. Obtaining history SHALL NOT change any source role's privileges.

#### Scenario: History after promotion with provisioned credentials

- **WHEN** credentials created by Databasus are used for physical backups and the source standby is promoted
- **THEN** Databasus obtains the new timeline's history using those credentials
- **AND** the role still cannot read server files through SQL, read application tables, create roles or databases, or act as an administrator
- **AND** no other role gains privileges

#### Scenario: History request fails

- **WHEN** obtaining timeline history fails
- **THEN** Databasus leaves the source privileges unchanged
- **AND** it does not propose server-file read privileges as a required workaround
