# Database Version Support Specification

## Purpose

Defines which PostgreSQL, MySQL, MariaDB and MongoDB server versions Databasus accepts for backup, how the version it detects governs the tooling and the safety checks applied to that database, and how it behaves when it meets a server it ships no client for.

## Requirements

### Requirement: Supported MySQL server versions are accepted

The system SHALL accept a MySQL server reporting major version 5, 8, 9 or 26 when it ships a client for that server's release line on the running architecture, and SHALL record a version identity that names the line the server belongs to rather than its exact version.

#### Scenario: Add a MySQL server on a previously supported line

- **WHEN** an authorized user tests the connection to a reachable MySQL server reporting version 5.7, 8.0, 8.4 or 9.x with sufficient backup privileges
- **THEN** the connection test succeeds and the database records the matching version identity

#### Scenario: Add a MySQL server on the calendar scheme

- **WHEN** an authorized user tests the connection to a reachable MySQL server reporting version 26.7 with sufficient backup privileges
- **THEN** the connection test succeeds and the database records a version identity of 26 rather than the identity of an earlier line

#### Scenario: A later release inside the calendar line

- **WHEN** an authorized user tests the connection to a MySQL server on a release later than the newest 26.x client the product ships, while still inside the 26 line
- **THEN** the connection test succeeds, the database records the identity 26, and the shipped 26 client is used

#### Scenario: Managed MySQL reporting a vendor suffix

- **WHEN** the MySQL server reports its version with a vendor suffix, such as `26.7.0-cloud` on a managed instance
- **THEN** the system reads the version from the leading numbers and accepts the server

### Requirement: Supported MariaDB server versions are accepted

The system SHALL accept a MariaDB server reporting major version 5, 10, 11, 12 or 13, and SHALL record a version identity that names the line the server belongs to rather than its exact version.

#### Scenario: Add a MariaDB 13 server

- **WHEN** an authorized user tests the connection to a reachable MariaDB server reporting version 13.0 with sufficient backup privileges
- **THEN** the connection test succeeds and the database records a version identity of 13.0

#### Scenario: A later minor inside the MariaDB 13 line

- **WHEN** an authorized user tests the connection to a MariaDB server reporting a 13.x version later than the client the product ships for that line, such as 13.2 against a 13.0 client
- **THEN** the connection test succeeds and the server is served by the shipped 13 client, because the client and the server belong to the same release line

#### Scenario: Add a MariaDB server on a previously supported line

- **WHEN** an authorized user tests the connection to a reachable MariaDB server reporting version 5.5, 10.x, 11.x or 12.x with sufficient backup privileges
- **THEN** the connection test succeeds and the database records the matching version identity

### Requirement: A server on an unserved release line is refused, never dumped by a client from an older line

The system SHALL refuse a MySQL or MariaDB server whose release line no shipped client serves, and SHALL never dump such a server with a client from an earlier line, so that no backup is produced by a program from a generation that does not know the server's features. Within one release line the client may be older than the server: that gap is closed by refreshing the bundle, not by refusing the database.

#### Scenario: Server on a release line the product ships no client for

- **WHEN** an authorized user tests the connection to a reachable MySQL or MariaDB server whose release line is newer than every line the product ships a client for, such as a MariaDB 14.1 server while the newest MariaDB client serves the 13 line
- **THEN** the connection test fails, names the versions the product supports, and no database is added

#### Scenario: Server ahead of its client inside a served line

- **WHEN** an authorized user tests the connection to a server whose release line is served but whose own version is newer than the client shipped for that line, such as a MariaDB 13.2 server against a 13.0 client
- **THEN** the connection test succeeds and the database is served by that line's client

#### Scenario: Release line whose client exists on one architecture only

- **WHEN** an authorized user tests the connection to a server whose client is absent for the architecture the product runs on, such as MySQL 5.7 on an arm64 deployment
- **THEN** the connection test fails and reports that this version is unsupported on this architecture

#### Scenario: A database recorded before its client became unavailable

- **WHEN** a backup starts for a database whose recorded version has no client on the architecture the product now runs on
- **THEN** the backup fails with a message naming the version and the architecture, rather than with a missing-file error

#### Scenario: A newer client serves an older line by design

- **WHEN** a server's release line has no client of its own but a newer shipped client is designated to serve that line, as the modern MariaDB client serves every line from 10.2 up
- **THEN** the server is accepted and that client is used

### Requirement: An unusable server is rejected with a reason

The system SHALL refuse a server it cannot serve and SHALL name what it found, so that the user can correct the choice rather than discover the problem when a backup fails.

#### Scenario: MariaDB server added under the MySQL type

- **WHEN** an authorized user tests a connection to a MariaDB server while the database type is MySQL
- **THEN** the connection test fails, reports the version it found and directs the user to the MariaDB type

#### Scenario: Version string cannot be read

- **WHEN** the server answers the version query with a value the system cannot parse
- **THEN** the connection test fails and reports the value it received

#### Scenario: MySQL major below the supported range

- **WHEN** a MySQL server reports a major version below 5
- **THEN** the connection test fails and names the versions the system supports

#### Scenario: MariaDB major below the supported range

- **WHEN** a MariaDB server reports a major version below 5
- **THEN** the connection test fails and names the versions the system supports

### Requirement: The downgrade guard covers every accepted version

The system SHALL refuse to restore a backup taken from a newer server onto an older server for every MySQL, MariaDB and MongoDB version it accepts, including versions added after the guard was written.

#### Scenario: Restore a calendar-scheme MySQL backup onto an earlier line

- **WHEN** an authorized user requests a restore of a backup taken from a MySQL 26.x server onto a MySQL 9.x or 8.x server
- **THEN** the system refuses the restore and explains that the backup comes from a newer server

#### Scenario: Restore a MariaDB 13 backup onto MariaDB 12

- **WHEN** an authorized user requests a restore of a backup taken from a MariaDB 13.x server onto a MariaDB 12.x server
- **THEN** the system refuses the restore and explains that the backup comes from a newer server

#### Scenario: Restore a MongoDB 9 backup onto MongoDB 8

- **WHEN** an authorized user requests a restore of a backup taken from a MongoDB 9.x server onto a MongoDB 8.x server
- **THEN** the system refuses the restore and explains that the backup comes from a newer server

#### Scenario: Restore onto an equal or newer server

- **WHEN** an authorized user requests a restore of a backup onto a server of the same or a newer version identity
- **THEN** the system allows the restore

### Requirement: Backups of new versions use the modern transfer settings

The system SHALL apply the same network compression negotiation to newly supported MySQL versions as it applies to MySQL 8.0 and later, rather than falling back to the legacy setting reserved for MySQL 5.7.

#### Scenario: Back up a calendar-scheme MySQL server

- **WHEN** a backup of a MySQL 26.x database starts
- **THEN** the system negotiates compression from the modern algorithm set and falls back through it to an uncompressed transfer if the server or its proxy refuses

### Requirement: The displayed version is never blank

The system SHALL display a version for every MySQL and MariaDB database it accepted, for every version identity it can record.

#### Scenario: View a database on a newly supported version

- **WHEN** an authorized user views a MySQL 26.x or MariaDB 13.x database
- **THEN** the version field shows that version rather than an empty value

### Requirement: Supported MongoDB server versions are accepted

The system SHALL accept a MongoDB server reporting version 4.2 or later up to and including the 9 line, and SHALL record a version identity that is the server's major version alone, so that every release inside one major line shares one identity.

#### Scenario: Add a MongoDB 9 server

- **WHEN** an authorized user tests the connection to a reachable MongoDB server reporting version 9.0 with sufficient backup privileges
- **THEN** the connection test succeeds and the database records a version identity of 9

#### Scenario: A later minor inside the MongoDB 9 line

- **WHEN** an authorized user tests the connection to a MongoDB server reporting a 9.x version later than 9.0, such as 9.2
- **THEN** the connection test succeeds and the database records a version identity of 9

#### Scenario: Add a MongoDB server on a previously supported line

- **WHEN** an authorized user tests the connection to a reachable MongoDB server reporting version 4.2, 4.4, 5.x, 6.x, 7.x or 8.x with sufficient backup privileges
- **THEN** the connection test succeeds and the database records the server's major version as its identity

#### Scenario: A database recorded before this change

- **WHEN** the product starts with MongoDB databases recorded under identities 4 to 8, and a backup and a restore run for one of them
- **THEN** the database keeps its recorded identity, and the backup and the restore succeed with the shipped MongoDB client

### Requirement: A MongoDB server outside the supported range is refused with a reason

The system SHALL refuse a MongoDB server older than 4.2 or newer than the 9 line, and SHALL name the version it found and the range it supports, so that no backup is attempted against a server whose major line the shipped client's vendor compatibility statement does not list. Inside an accepted major line the server may be newer than the releases that statement names, such as 9.2 against a client listed for 9.0: that gap is closed by refreshing the client, not by refusing the database.

#### Scenario: MongoDB server below 4.2

- **WHEN** an authorized user tests the connection to a MongoDB server reporting version 4.0 or 3.6
- **THEN** the connection test fails, reports the version it found, names 4.2 as the minimum supported version, and no database is added

#### Scenario: MongoDB server above the newest supported line

- **WHEN** an authorized user tests the connection to a MongoDB server reporting version 10.0
- **THEN** the connection test fails, reports the version it found and the supported range, and no database is added

#### Scenario: MongoDB version string cannot be read

- **WHEN** the MongoDB server answers the version query with a value the system cannot parse
- **THEN** the connection test fails and reports the value it received

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
