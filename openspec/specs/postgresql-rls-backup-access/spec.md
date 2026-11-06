# postgresql-rls-backup-access Specification

## Purpose

Ensures PostgreSQL backup credentials can preserve every row requested by a logical backup, detects RLS restrictions during connection validation, and defines safe backup outcomes when RLS changes or physical replication is used.

## Requirements

### Requirement: Logical connection validation rejects effective RLS restrictions

When a PostgreSQL logical database is created, updated, or tested with saved or unsaved settings, Databasus SHALL reject credentials subject to row-level security on a table whose rows the requested backup includes. The check SHALL evaluate the connecting role's effective access, including ownership and forced row-level security, rather than require every role to have a particular role attribute. A policy that permits some or even all rows SHALL NOT make a role subject to RLS acceptable for a complete logical backup. The refusal SHALL identify the connecting role and affected tables, explain that RLS prevents a complete backup, and identify using an appropriately authorized backup role as the remedy.

Saved and unsaved connection tests SHALL use the existing successful HTTP response containing a failed connection outcome and diagnostic. Creation and updates SHALL use the existing database connection failure response. A failed update SHALL keep the stored configuration, and a saved connection test SHALL leave settings, health and backup outcome fields unchanged.

#### Scenario: SELECT does not grant complete access through RLS

- **WHEN** a role has schema access and SELECT on every included table and sequence, but RLS limits its access to an included table's rows
- **THEN** testing the connection reports failure and identifies the role, table and RLS restriction

#### Scenario: A policy permits all rows

- **WHEN** the connecting role is subject to RLS on an included table and its current SELECT policy permits every row
- **THEN** connection validation still refuses the credentials because a complete logical dump cannot run with RLS applied

#### Scenario: Creating a database with restricted credentials

- **WHEN** a member creates a logical PostgreSQL database with credentials restricted by RLS on included data
- **THEN** creation returns the existing database connection failure response and no database is created

#### Scenario: Updating a database to restricted credentials

- **WHEN** a member updates a logical PostgreSQL database to credentials restricted by RLS on included data
- **THEN** the update returns the existing database connection failure response and the stored configuration is unchanged

#### Scenario: Testing saved credentials after RLS was enabled

- **WHEN** RLS becomes effective for a saved logical connection and a member tests that connection
- **THEN** the test reports failure with the RLS diagnostic without modifying settings, health or backup outcome fields

#### Scenario: Testing unsaved restricted credentials

- **WHEN** a member tests unsaved logical settings whose credentials are restricted by RLS on included data
- **THEN** the test reports a failed connection outcome using the existing test response format

#### Scenario: An authorized bypass role

- **WHEN** the role has BYPASSRLS and all other required access to the requested dump
- **THEN** connection validation succeeds even when included tables use forced RLS

#### Scenario: A superuser

- **WHEN** the connecting role is a superuser and the requested dump includes tables with forced RLS
- **THEN** connection validation succeeds

#### Scenario: A table owner without forced RLS

- **WHEN** the connecting role owns the RLS table, forced RLS is disabled, and all other dump permissions are sufficient
- **THEN** connection validation succeeds without requiring BYPASSRLS

#### Scenario: A table owner with forced RLS

- **WHEN** the connecting role owns an included table with forced RLS and has neither superuser nor BYPASSRLS privileges
- **THEN** connection validation fails with the RLS diagnostic

### Requirement: RLS validation follows the requested logical data scope

RLS restrictions SHALL only block validation for table data the requested logical backup actually includes. The check SHALL respect included schemas, whole-table exclusions, table-data exclusions, and the existing table-pattern matching rules. Excluding data SHALL NOT remove the existing schema and SELECT requirements for a table whose structure is included. Partition data SHALL be evaluated for the relations from which the backup reads rows; an RLS restriction only on a partitioned parent SHALL NOT be treated as a restriction on a directly dumped partition.

#### Scenario: RLS table outside included schemas

- **WHEN** every table whose rows restrict the backup role through RLS is outside the included schemas and all other dump permissions are sufficient
- **THEN** connection validation succeeds

#### Scenario: Whole-table exclusion

- **WHEN** every RLS-restricted table is matched by a whole-table exclusion and all other dump permissions are sufficient
- **THEN** connection validation succeeds

#### Scenario: Data exclusion with sufficient SELECT privileges

- **WHEN** a role has schema access and SELECT on an RLS-restricted table, its data is excluded, and all other dump permissions are sufficient
- **THEN** connection validation succeeds without RLS bypass privileges for that table

#### Scenario: Data exclusion without SELECT privileges

- **WHEN** a role lacks SELECT on a table whose data is excluded
- **THEN** connection validation still fails for missing read access

#### Scenario: An unqualified exclusion does not reach another schema

- **WHEN** an unqualified exclusion matches a table on the role's default search path but an RLS-restricted table with the same name lies outside that path and remains included
- **THEN** connection validation fails and names the included RLS-restricted table

#### Scenario: RLS applies to a dumped partition

- **WHEN** a directly dumped partition restricts the backup role through RLS and its data is included
- **THEN** connection validation fails even if the partitioned parent has no RLS restriction

#### Scenario: RLS applies only to a partitioned parent

- **WHEN** only a partitioned parent restricts the role through RLS, its partitions are dumped directly without RLS restrictions, and all other dump permissions are sufficient
- **THEN** connection validation succeeds

### Requirement: Permission drift cannot publish an incomplete logical backup

If RLS becomes effective for saved logical backup credentials after an earlier successful validation and backup, the next backup that attempts to read restricted data SHALL fail through the database dump operation. Databasus SHALL record the backup as FAILED with the dump operation's RLS diagnostic, SHALL NOT publish it as completed, and SHALL remove files written by the failed attempt through storage cleanup. Previously completed backups SHALL remain available. Connection validation SHALL NOT weaken the runtime protection against exporting only visible rows.

#### Scenario: RLS enabled after a successful backup

- **WHEN** a role successfully validates and backs up a table, RLS is then enabled so the same role can see only some of its rows, and another backup is started without changing the saved connection
- **THEN** the new backup is FAILED with a diagnostic identifying RLS and the affected table
- **AND** after storage cleanup neither its partial backup file nor its metadata file remains, while the earlier completed backup and its files remain available

### Requirement: Physical backups preserve RLS-protected rows without row access

RLS on user tables SHALL NOT cause physical connection validation or physical backups to require SELECT, BYPASSRLS, or superuser privileges. With the existing required replication and backup-type privileges, a physical full backup and a chain containing incremental backups SHALL preserve all rows, including rows unavailable to the replication role through ordinary table queries. Restoring either backup form SHALL recover those rows and the table's RLS configuration.

#### Scenario: A full physical backup with existing RLS

- **WHEN** a role with the required physical backup privileges but without SELECT, BYPASSRLS or superuser privileges validates a physical connection and takes a full backup of a cluster containing an RLS table
- **THEN** validation and backup succeed, and restoring the full backup recovers every table row and its RLS configuration

#### Scenario: RLS introduced between physical backups

- **WHEN** a full physical backup succeeds with such a replication role, RLS is then enabled and more rows are added before an incremental backup
- **THEN** the incremental backup succeeds and restoring the chain recovers the rows from both stages and the RLS configuration
