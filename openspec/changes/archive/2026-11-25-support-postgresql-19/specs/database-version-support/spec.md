# Spec Delta

## ADDED Requirements

### Requirement: Supported PostgreSQL server versions are accepted

The system SHALL accept a PostgreSQL server for logical backups when it reports major version 12 or later and the product ships a client for that major. It SHALL record the major version alone as the version identity, so that every minor release of one major shares one identity and is served by that major's client.

#### Scenario: Add a PostgreSQL 19 server for logical backups

- **WHEN** an authorized user tests the connection to a reachable PostgreSQL 19 server with sufficient backup privileges, as a logical backup source
- **THEN** the connection test succeeds and the database records the version identity 19

#### Scenario: A later minor inside a served major

- **WHEN** an authorized user tests the connection to a PostgreSQL server reporting 19.3 while the product ships a 19.0 client
- **THEN** the connection test succeeds, the database records the identity 19, and the shipped 19 client is used

#### Scenario: View a PostgreSQL 19 database

- **WHEN** an authorized user views a PostgreSQL 19 logical backup database
- **THEN** the version field shows 19 rather than an empty value

### Requirement: A PostgreSQL server on a major without a shipped client is refused

The system SHALL refuse a PostgreSQL logical backup source whose major version has no shipped client, and SHALL never back it up with a client of an older major. The refusal SHALL name the major version it found and the versions the product supports, so the user learns that the product does not support that release yet rather than that the server is broken.

#### Scenario: A server on a major newer than every shipped client

- **WHEN** an authorized user tests the connection to a reachable PostgreSQL 20 server while the newest shipped client serves PostgreSQL 19
- **THEN** the connection test fails, reports that it found version 20 and names the supported versions, and no database is added

#### Scenario: A server below the supported range

- **WHEN** an authorized user tests the connection to a PostgreSQL 11 server as a logical backup source
- **THEN** the connection test fails, reports that it found version 11 and names the supported versions, and no database is added
