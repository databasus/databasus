## ADDED Requirements

### Requirement: A broken PostgreSQL client stops the server before the schema changes

The server SHALL run the startup check of its client sets before it touches the internal database. When a PostgreSQL client is missing or unusable, or the client tools cannot be found, the server SHALL report the failure and exit without opening the internal database, so the schema stays at the version the previous image can run. A broken client of another engine SHALL still only warn, and the server SHALL go on to migrate. This check SHALL run before the refusal of a schema newer than the binary's, so a server that fails both reports the client failure. A console command runs no client and SHALL NOT be stopped by this check.

#### Scenario: An image with a broken PostgreSQL client replaces a working one

- **WHEN** the server starts from a new image whose PostgreSQL client cannot run, on an internal database with an older schema
- **THEN** the server reports the failure and exits
- **AND** no migration is applied, so the previous image can start on that database again

#### Scenario: The client tools cannot be found

- **WHEN** the server starts where the client tools cannot be found
- **THEN** the server reports the failure and exits, and no migration is applied

#### Scenario: A MySQL client is broken

- **WHEN** the server starts with a working PostgreSQL client and a MySQL client that cannot run
- **THEN** it logs a warning naming the client set, applies its migrations and starts

#### Scenario: A console command on a host without client tools

- **WHEN** an operator runs a console command where the client tools cannot be found
- **THEN** the command runs as usual
