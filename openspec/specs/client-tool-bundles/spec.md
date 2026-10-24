# Client Tool Bundles Specification

## Purpose

Defines the database client programs that ship inside the product, what the system guarantees about the server versions they can serve, and how a missing or unusable client becomes visible to an operator instead of surfacing as a failed backup.

## Requirements

### Requirement: Every accepted version identity has a designated client on every supported architecture

The system SHALL ship, for each processor architecture it runs on, a client designated to dump and restore every MySQL, MariaDB and MongoDB version identity it accepts.

For MySQL and MariaDB, the designated client is either the one built for that identity's own release line, or a client from another line documented as serving it — a newer one, as the modern MariaDB client serves every line from 10.2 up, or an older one, as the legacy MariaDB client serves the lines the modern client cannot dump. A MySQL or MariaDB client MAY trail the server inside a single release line, and SHALL never come from a line older than the server's own.

MongoDB clients are numbered independently of server release lines, so the release-line clauses above do not apply to them. MongoDB is served by a single client for every accepted identity, and that client SHALL be a release whose vendor compatibility statement lists every MongoDB major line the system accepts.

#### Scenario: Start on a supported architecture

- **WHEN** the product starts on either supported architecture
- **THEN** it reports each expected client set as verified, except for a client set documented as unavailable on that architecture

#### Scenario: A version identity served by a newer client

- **WHEN** a version identity has no client built specifically for it and a newer shipped client is designated to serve it
- **THEN** the system serves it with that client, and the designation is recorded in the client documentation

#### Scenario: A version identity served by an older client on purpose

- **WHEN** a version identity exists for servers a newer client cannot dump, as MariaDB 5.5 and 10.1 need a client older than 10.2
- **THEN** the product ships a client from a line that can dump them on every architecture, and that client is used for those identities

#### Scenario: An architecture with no client for a version identity

- **WHEN** an architecture ships no client able to serve a version identity, as arm64 ships none for MySQL 5.7
- **THEN** that identity is unsupported on that architecture and a server reporting it is refused rather than attempted

#### Scenario: One MongoDB client serves every accepted MongoDB identity

- **WHEN** a backup or restore runs for a MongoDB database of any accepted identity, from 4 to 9, on either supported architecture
- **THEN** the shipped MongoDB tools perform it, and the vendor's compatibility statement for that tools release lists the server's major line

### Requirement: Shipped clients stay on a maintained release line

The system SHALL ship clients from a release line that the vendor still patches, or, when the line has reached its end of life, from the final patch that line received.

#### Scenario: A line the vendor still patches

- **WHEN** a client is shipped for a line the vendor still supports
- **THEN** it is the newest patch released for that line at the time the change ships

#### Scenario: A line that has ended

- **WHEN** a client is shipped for a line whose upstream support has ended, because servers on that line are still in use
- **THEN** it is the last patch that line received

### Requirement: A client that cannot run is reported at startup

The startup check SHALL confirm that each shipped client can actually execute in the runtime environment, not only that its file is present and executable, so that a missing shared library is reported before a backup needs it. The health check SHALL report that result without running the clients again, so that repeated probing does not depend on starting every client binary.

#### Scenario: A client whose shared library is absent

- **WHEN** a shipped client cannot start because a shared library it needs is absent from the runtime environment
- **THEN** the check reports that client as failed, naming the client and the reason

#### Scenario: A client that runs

- **WHEN** a shipped client starts and reports its own version
- **THEN** the check reports that client as verified

#### Scenario: Repeated health probes

- **WHEN** the health check runs repeatedly while the product is up
- **THEN** it reports the startup verdict for each client together with a fresh confirmation that the client files are still present

### Requirement: A client set holds the version it is named for

The startup check SHALL compare the version each client reports with the release line that client set is registered under, and SHALL report a mismatch as a failure, so that a client set cannot silently hold a different version than the one the system selects it for.

#### Scenario: A client set holding a different version

- **WHEN** a client set registered for one release line contains a client that reports a version from another line
- **THEN** the check reports that client set as failed, naming the line expected and the version found

#### Scenario: A client set holding a later patch of its own line

- **WHEN** a client set registered for a release line contains a client reporting a later patch or minor of that same line
- **THEN** the check reports that client set as verified

### Requirement: A missing client disables its engine without stopping the product

The system SHALL treat a missing or unusable MySQL, MariaDB or MongoDB client as a warning that disables backups for the affected version, and SHALL treat a missing or unusable PostgreSQL client as fatal.

#### Scenario: MySQL client set is missing

- **WHEN** a MySQL client set is missing at startup
- **THEN** the product starts, logs a warning naming the client set, and keeps serving databases of other engines

#### Scenario: PostgreSQL client set is missing

- **WHEN** a PostgreSQL client set is missing at startup
- **THEN** the product reports the failure and exits

### Requirement: The MongoDB client tools are the vendor's unmodified build from a recorded source

The system SHALL ship MongoDB client tools that are byte-identical to the programs inside an archive the vendor publishes, and the client documentation SHALL record, for each architecture, the tools release, the archive's download address and the archive's published checksum. Both architectures SHALL come from the same tools release and the same vendor build family, so that one documented release means one build on every architecture.

#### Scenario: An operator verifies the shipped MongoDB tools

- **WHEN** an operator downloads the archive the client documentation names for an architecture and checks it against the recorded checksum
- **THEN** the checksum matches, and the shipped dump and restore programs for that architecture are byte-identical to the ones in the archive

#### Scenario: Both architectures carry the same tools release

- **WHEN** the version embedded in the shipped MongoDB dump and restore programs is read for each supported architecture
- **THEN** every program carries the tools release the client documentation records

### Requirement: Building the image downloads no client tools

The client tools SHALL be committed to the repository and copied into the image. Building the image SHALL NOT download client tools from engine vendors' package repositories, so a vendor mirror that is down or has changed its keys cannot break or alter a build.

#### Scenario: Vendor repository is down

- **WHEN** a vendor's package repository is unreachable during an image build
- **THEN** the build completes with the same client tools as before
