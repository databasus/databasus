# internal-database-migrations Specification

## Purpose

Describes how the application brings the schema of its internal database to the current version: by itself, at every start, without a separate tool, and what happens when a migration fails or when two processes apply migrations at once.

## Requirements

### Requirement: The application applies the migrations of its internal database at startup

Except when the unsupported-storage upgrade refusal below applies, at every server start the application SHALL apply to the internal database every migration of its version that is not applied yet, and only then start accepting HTTP requests and running background work. This SHALL NOT require any program but the Databasus binary: the image contains no separate migration tool. A migration that is already applied SHALL NOT run again.

#### Scenario: Fresh install

- **WHEN** the operator starts the image with an empty data volume
- **THEN** the application creates the whole schema of the internal database, starts answering requests and shows the sign-up screen

#### Scenario: Upgrade to a new version

- **WHEN** the operator starts a newer image on a volume whose internal database was created by an earlier version and containing no Google Drive or rclone storage
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

### Requirement: An upgrade preserves provider-defined storage classes

An upgrade SHALL start successfully when the previous release accepted an S3-compatible storage class outside the application's suggested classes. It SHALL preserve that class exactly and SHALL NOT silently replace it with a default. Invalid application-owned settings SHALL still be validated or repaired according to their domain requirements.

#### Scenario: A class supplied by an S3-compatible service

- **WHEN** the previous release accepted an S3 storage configured with class COLD and the instance is upgraded
- **THEN** the instance starts successfully and reading the storage returns COLD

#### Scenario: An AWS class outside the suggestions

- **WHEN** the previous release accepted an S3 storage configured with class DEEP_ARCHIVE and the instance is upgraded
- **THEN** the instance starts successfully and reading the storage returns DEEP_ARCHIVE

### Requirement: Google Drive storages stop an upgrade before any pending migration

When the internal database contains any Google Drive storage, the server and every console command that initializes the application SHALL exit with a non-zero code before applying any pending migration, accepting HTTP requests or running background work. The refusal SHALL apply across all workspaces, including unused storages and a Google Drive storage whose provider settings are missing. The check SHALL NOT require credentials, contact Google, modify application records or advance migration history. It SHALL work on older database layouts as well as the layout immediately before removal, and SHALL allow a fresh database with no storage records to initialize.

When Google Drive is the only blocking type, the diagnostic SHALL state that Google Drive support ended after Databasus 3.67.0, which still supports it. It SHALL instruct the operator to roll back to 3.67.0, remove all Google Drive storages and retry the upgrade, or continue using 3.67.0. For a database created by 3.67.0, this refusal SHALL leave it usable by that same version on the same volume. When rclone storages also remain, the combined-diagnostic requirement below SHALL apply and identify 3.67.0 for removing both types. This SHALL NOT promise a downgrade after a successful schema upgrade.

#### Scenario: An unused Google Drive storage exists

- **WHEN** the new server starts on an older installation with an unused Google Drive storage in any workspace
- **THEN** startup fails with the retirement and recovery diagnostic before any pending migration runs
- **AND** application records, stored secrets and migration history remain unchanged

#### Scenario: A storage has incomplete provider settings

- **WHEN** a Google Drive storage record exists without its provider settings
- **THEN** startup refuses before migrations rather than silently repairing or deleting that record

#### Scenario: A console command initializes the application

- **WHEN** an operator runs a console command on an older database that contains a Google Drive storage
- **THEN** it exits non-zero with the same recovery guidance and applies no migration or requested mutation

#### Scenario: Returning to the supported release

- **WHEN** an operator starts 3.67.0 on its original volume after the new version refused the upgrade
- **THEN** 3.67.0 starts and the existing storage and other application records remain available

#### Scenario: The operator removes the blocking storages

- **WHEN** all Google Drive storages are removed through 3.67.0, no rclone storages remain and the operator retries the upgrade
- **THEN** the Google Drive check permits migration and supported application records remain available

#### Scenario: Fresh database

- **WHEN** an operator starts with an empty internal database
- **THEN** the retirement check permits normal initialization

### Requirement: Direct removal migration refuses existing Google Drive storages

Applying the Google Drive removal migration directly SHALL fail if any Google Drive storage remains. That migration SHALL run in one transaction and SHALL NOT delete storages, backup records, credentials or remote files to make the upgrade proceed. Its failure SHALL preserve its input schema and records and SHALL leave that migration unapplied. Its diagnostic SHALL give the Google Drive-specific retirement and 3.67.0 recovery guidance used by startup. It need not aggregate rclone blockers checked by application startup. Earlier migrations explicitly applied through a separate migration tool are outside the startup rollback guarantee.

When no Google Drive storage remains, the migration SHALL remove its obsolete settings schema while preserving every supported provider's settings and encrypted secrets. A fresh installation SHALL finish without the obsolete schema. Historical migration support for older installation layouts SHALL remain available.

#### Scenario: Direct migration with a remaining storage

- **WHEN** an operator directly applies the removal migration to a database with a Google Drive storage
- **THEN** the migration fails with recovery guidance and its input schema, records, secrets and migration-history entry are unchanged

#### Scenario: Removal with supported storages only

- **WHEN** the removal migration runs with no Google Drive storage and with supported storages present
- **THEN** the obsolete Google Drive settings schema is absent and the supported storages and their secrets are preserved

### Requirement: Retirement diagnostics do not disclose credentials

The Google Drive and rclone upgrade checks, combined diagnostic and removal migrations SHALL NOT decrypt credentials or include stored client secrets, tokens, rclone configuration contents or serialized provider settings in errors, standard output, standard error or logs.

#### Scenario: Refusal on a storage with saved secrets

- **WHEN** startup or direct removal refuses a Google Drive or rclone storage that has stored credentials
- **THEN** the diagnostic gives recovery instructions without containing its client secret, tokens, configuration contents or serialized settings

### Requirement: Rclone storages stop an upgrade before any pending migration

When the internal database contains any rclone storage, the server and every console command that initializes the application SHALL exit with a non-zero code before applying any pending migration, accepting HTTP requests or running background work. The refusal SHALL apply across all workspaces, including unused storages and a storage whose provider settings are missing. It SHALL work on older database layouts as well as the layout immediately before removal. It SHALL NOT require credentials, contact a storage destination, read local files from a stored configuration, modify application records or advance migration history. A fresh database with no storages SHALL initialize normally.

When rclone is the only blocking type, the diagnostic SHALL state that Databasus 3.67.1 supports it and instruct the operator to roll back to 3.67.1, manually remove all rclone storages and retry the upgrade, or continue using 3.67.1. A refusal on a database created by 3.67.1 SHALL leave that database usable by the same version on the same volume. This SHALL NOT promise a downgrade after a successful schema upgrade.

#### Scenario: An unused rclone storage exists

- **WHEN** the new server starts on an older installation with an unused rclone storage in any workspace and no Google Drive storage
- **THEN** it exits non-zero with the 3.67.1 recovery guidance before any pending migration runs
- **AND** stored records, encrypted settings and migration history remain unchanged

#### Scenario: Provider settings are missing

- **WHEN** a parent rclone storage record remains without its settings
- **THEN** startup refuses before any migration rather than repairing or deleting the record

#### Scenario: A console command initializes the application

- **WHEN** an operator runs an application-initializing console command against a database containing rclone
- **THEN** it exits non-zero with the same recovery guidance and applies no migration or requested mutation

#### Scenario: Returning to 3.67.1

- **WHEN** an operator restarts 3.67.1 on its original volume after the new version refused the upgrade
- **THEN** 3.67.1 starts and existing records remain available

#### Scenario: The operator removes the blocker

- **WHEN** the operator removes all rclone storages through 3.67.1, no Google Drive storage remains and the operator retries the upgrade
- **THEN** startup applies the pending migrations and supported records remain available

### Requirement: One pre-upgrade diagnostic names every blocking storage type

The pre-upgrade check SHALL inspect both Google Drive and rclone across all workspaces before selecting its diagnostic. If either remains, one diagnostic SHALL name every blocking type and one recovery release that supports all named types. Google Drive alone SHALL select 3.67.0; rclone alone SHALL select 3.67.1; both SHALL select 3.67.0 and instruct the operator to remove both types there. It SHALL offer remaining on that release as an alternative. Selecting the first blocker and hiding the other until a later retry SHALL NOT satisfy this requirement.

After removing all blocking storages through the indicated release, the operator SHALL be able to retry the target version directly without installing another intermediate release. The new binary SHALL NOT automatically launch or install an intermediate version. Existing storage-in-use and pending-cleanup rules SHALL remain in force in the recovery release; no data SHALL be automatically deleted to permit the upgrade. If inspection fails, initialization SHALL fail before migrations rather than assume that the database has no blockers.

#### Scenario: Both retired types exist

- **WHEN** a database contains Google Drive and rclone storages, including when they belong to different workspaces
- **THEN** startup refuses once with both types named and guidance to use 3.67.0 to remove both
- **AND** no migration or application mutation occurs

#### Scenario: Both types have been removed on the common release

- **WHEN** the operator removes both types through 3.67.0 and starts the target version directly
- **THEN** the target applies pending migrations and starts without requiring an intermediate 3.67.1 installation

#### Scenario: The database has only supported providers

- **WHEN** the database contains local, S3-compatible, NAS, Azure Blob, FTP or SFTP storages without either blocking type
- **THEN** the pre-upgrade check permits migrations

#### Scenario: Inspection cannot determine the stored types

- **WHEN** initialization cannot inspect the stored types
- **THEN** it exits non-zero before migrations and reports the inspection failure

### Requirement: Direct removal migration refuses existing rclone storages

Applying the rclone removal migration directly SHALL fail if any rclone storage remains, including a parent record without provider settings. The migration SHALL run in one transaction, preserve its input schema, records and encrypted settings on failure, and remain unapplied. Its diagnostic SHALL identify 3.67.1 and instruct manual removal through that release before retrying, or remaining on that release. The migration SHALL NOT delete storages, backups, credentials or remote files to make the upgrade proceed. Earlier migrations applied directly through a separate tool are outside the startup rollback guarantee.

When no rclone storage remains, the migration SHALL remove its obsolete settings schema while preserving supported storage settings and encrypted secrets. A fresh installation SHALL complete without either obsolete rclone settings table. Historical migrations SHALL remain usable for older installation layouts.

#### Scenario: A direct migration finds a remaining storage

- **WHEN** the rclone removal migration is applied directly with a remaining rclone storage
- **THEN** it fails with 3.67.1 recovery guidance and leaves its input schema, records and migration-history state unchanged

#### Scenario: A direct migration finds an incomplete storage

- **WHEN** the rclone removal migration is applied directly with a parent rclone record lacking settings
- **THEN** it refuses without deleting or repairing that record

#### Scenario: Only supported storages remain

- **WHEN** the migration runs without rclone storages
- **THEN** the obsolete settings schema is absent and supported records and encrypted secrets remain unchanged

#### Scenario: Fresh installation

- **WHEN** all migrations complete on an empty database
- **THEN** neither historical nor module-owned rclone settings tables remain
