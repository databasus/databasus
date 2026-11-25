# Spec Delta

## MODIFIED Requirements

### Requirement: Physical backups require PostgreSQL 17 or later

Databasus SHALL accept a physical backup configuration only for a source running PostgreSQL 17 or later, on a major for which it ships the physical backup clients. Incremental backups, and combining them on restore, exist only from PostgreSQL 17. For a source from PostgreSQL 12 to 16, the connection setup SHALL refuse the configuration with a message that points to logical backups, which cover the same data. A source below PostgreSQL 12, which logical backups do not accept either, SHALL be refused as an unsupported version without that suggestion. For a source newer than every shipped client, the connection setup SHALL refuse the configuration with a message that names the version found and the versions physical backups support: PostgreSQL 17 up to the newest shipped client.

#### Scenario: PostgreSQL 16 source

- **WHEN** a user sets up a physical backup of a PostgreSQL 16 source
- **THEN** the setup is refused
- **AND** the message says physical backups need PostgreSQL 17 or later and suggests a logical backup

#### Scenario: PostgreSQL 11 source

- **WHEN** a user sets up a physical backup of a PostgreSQL 11 source
- **THEN** the setup is refused
- **AND** the message reports version 11 and names PostgreSQL 17 to 19 as the supported versions, without suggesting a logical backup

#### Scenario: PostgreSQL 17 source

- **WHEN** a user sets up a physical backup of a PostgreSQL 17 source that meets the other preconditions
- **THEN** the setup is accepted

#### Scenario: PostgreSQL 19 source

- **WHEN** a user sets up a physical backup of a PostgreSQL 19 source that meets the other preconditions
- **THEN** the setup is accepted
- **AND** full backups, incremental backups and the WAL stream run with the PostgreSQL 19 clients

#### Scenario: Source newer than every shipped client

- **WHEN** a user sets up a physical backup of a PostgreSQL 20 source while the newest shipped clients serve PostgreSQL 19
- **THEN** the setup is refused
- **AND** the message reports version 20 and names PostgreSQL 17 to 19 as the supported versions
