## ADDED Requirements

### Requirement: Neon sources are refused for physical backups

Neon accepts no physical replication connections, so a physical backup of a Neon database cannot run, and a missing `pg_hba.conf` entry is not a cause the user can fix. A physical PostgreSQL database whose host is `neon.tech` or a name under it SHALL be refused before Databasus opens any connection for it, to the source or to an SSH bastion:

- testing unsaved physical settings SHALL answer that the test failed with the reason `PHYSICAL_REPLICATION_UNSUPPORTED`;
- creating a physical database or updating one to such a host SHALL be refused with `400` and the code `database_connection_failed`, whose detail says that Neon supports only logical backups, and SHALL save nothing.

This SHALL hold with or without an SSH tunnel, judged by the configured source host. The form SHALL explain the reason in the interface language: Neon supports only logical backups, and the database should be added with the logical backup type. The explanation SHALL NOT suggest editing `pg_hba.conf` or enabling replication in the provider console. Hosts outside `neon.tech` SHALL be handled as before.

#### Scenario: A Neon host in the physical form

- **WHEN** a user tests physical settings whose host is `ep-young-dew-123456.c-8.us-east-2.aws.neon.tech`
- **THEN** the test fails with the reason `PHYSICAL_REPLICATION_UNSUPPORTED`
- **AND** Databasus opens no connection to that host
- **AND** the form says that Neon supports only logical backups and suggests the logical backup type

#### Scenario: A Neon host behind an SSH tunnel

- **WHEN** a user tests physical settings whose host is under `neon.tech` and whose SSH tunnel points at a bastion
- **THEN** the test fails with the reason `PHYSICAL_REPLICATION_UNSUPPORTED`
- **AND** Databasus opens no connection to the bastion

#### Scenario: Saving a physical database on Neon

- **WHEN** a member creates a physical database whose host is under `neon.tech`, or updates a saved physical database to such a host
- **THEN** the answer is `400` with the code `database_connection_failed` and a detail saying that Neon supports only logical backups
- **AND** no database is created, and the saved database keeps its previous settings

#### Scenario: A host that only resembles Neon

- **WHEN** a user tests physical settings whose host is `neon.tech.example.com`
- **THEN** the test connects to the source as for any other host
