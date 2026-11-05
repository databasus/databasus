## ADDED Requirements

### Requirement: The application refuses to start on a schema newer than its own

Before applying migrations, the application SHALL compare the newest migration recorded in the internal database with the newest migration it carries. When the database records a newer migration, the server and every console command that opens the internal database SHALL exit with a non-zero code without applying any migration, opening the HTTP port or changing the database. The message SHALL state the newest migration version (its timestamp) recorded in the database and the newest migration version the running binary carries, and SHALL say that returning to an older version after an upgrade is not supported and that the newer image has to be started instead.

A migration recorded in the database that the running version does not carry, but that is older than the running version's newest migration, SHALL NOT stop the start. Releases have shipped migrations that later releases deleted or renamed, and instances that applied them SHALL keep upgrading.

#### Scenario: Operator rolls the image back after an upgrade

- **WHEN** an instance was upgraded to a version with a new migration and the operator starts the previous image on the same volume
- **THEN** the process exits with a non-zero code before opening the HTTP port
- **AND** the message states both migration versions and says that going back after an upgrade is not supported
- **AND** the internal database is unchanged

#### Scenario: Console command on a newer schema

- **WHEN** the operator runs a password reset of an older image on a database migrated by a newer version
- **THEN** the command exits with a non-zero code with the same message and changes nothing

#### Scenario: A migration that a later release removed

- **WHEN** the internal database records a migration that was shipped in an earlier release and later deleted, and every migration it records is older than the running version's newest one
- **THEN** the application applies its missing migrations and starts
