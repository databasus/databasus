# internal-database-migrations Specification

## Purpose

Describes how the application brings the schema of its internal database to the current version: by itself, at every start, without a separate tool, and what happens when a migration fails or when two processes apply migrations at once.

## Requirements

### Requirement: The application applies the migrations of its internal database at startup

At every server start the application SHALL apply to the internal database every migration of its version that is not applied yet, and only then start accepting HTTP requests and running background work. This SHALL NOT require any program but the Databasus binary: the image contains no separate migration tool. A migration that is already applied SHALL NOT run again.

#### Scenario: Fresh install

- **WHEN** the operator starts the image with an empty data volume
- **THEN** the application creates the whole schema of the internal database, starts answering requests and shows the sign-up screen

#### Scenario: Upgrade to a new version

- **WHEN** the operator starts a newer image on a volume whose internal database was created by an earlier version
- **THEN** the application applies the migrations the database does not have yet, and every workspace, database, storage, notifier and backup stays in place

#### Scenario: Restart without new migrations

- **WHEN** the application restarts on a database where every migration of its version is applied
- **THEN** the application starts and does not change the schema

### Requirement: A missed migration is applied

A migration whose number is lower than the number of the latest applied migration, but which is not applied itself, SHALL be applied at startup. The order in which migrations entered the release SHALL NOT affect whether a migration is applied.

#### Scenario: Migration merged after a migration with a higher number

- **WHEN** the internal database has a migration with a higher number applied, and a migration with a lower number from the same version is not applied yet
- **THEN** at startup the application applies the migration with the lower number and starts

### Requirement: Two processes do not apply migrations at the same time

When two Databasus processes on one internal database start applying migrations at the same time, for example the server of a new image and a console command started in the same container, the second process SHALL wait until the first finishes. No migration SHALL run twice, and neither process SHALL fail because of the other. After that both processes SHALL work with the schema of the current version.

#### Scenario: Console command while a new version server starts

- **WHEN** a new version server is applying migrations and the operator runs a password reset in the container at the same time
- **THEN** the command waits until the server finishes the migrations, applies none of them again, resets the password and exits with code 0
- **AND** the server starts and answers requests

### Requirement: A failed migration stops the start

When a migration fails, the application SHALL exit with a non-zero code. Before that it SHALL log which migration failed and why. The application SHALL NOT start accepting HTTP requests or running background work on a database whose schema has not reached the current version. The migrations applied before the error SHALL stay applied, and the failed one SHALL NOT leave partial changes in the database. This holds because every migration SHALL run whole in one transaction: a migration that opts out of its transaction SHALL NOT reach a release, and the architecture check rejects it before the merge.

#### Scenario: Migration fails during an upgrade

- **WHEN** one of the new migrations fails at startup
- **THEN** the process exits with a non-zero code, the log names that migration and the error, and the HTTP port does not open
- **AND** the migrations that ran before it are recorded as applied in the internal database
- **AND** the database holds none of the failed migration's changes, including the tables and columns it creates before the failing statement
- **AND** the next start after the fix applies that migration again
