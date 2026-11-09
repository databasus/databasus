## MODIFIED Requirements

### Requirement: The application applies the migrations of its internal database at startup

Except when the Google Drive upgrade refusal below applies, at every server start the application SHALL apply to the internal database every migration of its version that is not applied yet, and only then start accepting HTTP requests and running background work. This SHALL NOT require any program but the Databasus binary: the image contains no separate migration tool. A migration that is already applied SHALL NOT run again.

#### Scenario: Fresh install

- **WHEN** the operator starts the image with an empty data volume
- **THEN** the application creates the whole schema of the internal database, starts answering requests and shows the sign-up screen

#### Scenario: Upgrade to a new version

- **WHEN** the operator starts a newer image on a volume whose internal database was created by an earlier version and containing no Google Drive storage
- **THEN** the application applies the migrations the database does not have yet, and every workspace, database, storage, notifier and backup stays in place

#### Scenario: Restart without new migrations

- **WHEN** the application restarts on a database where every migration of its version is applied
- **THEN** the application starts and does not change the schema

## ADDED Requirements

### Requirement: Google Drive storages stop an upgrade before any pending migration

When the internal database contains any Google Drive storage, the server and every console command that initializes the application SHALL exit with a non-zero code before applying any pending migration, accepting HTTP requests or running background work. The refusal SHALL apply across all workspaces, including unused storages and a Google Drive storage whose provider settings are missing. The check SHALL NOT require credentials, contact Google, modify application records or advance migration history. It SHALL work on older database layouts as well as the layout immediately before removal, and SHALL allow a fresh database with no storage records to initialize.

The diagnostic SHALL state that Google Drive support ended after Databasus 3.67.0, which still supports it. It SHALL instruct the operator to roll back to 3.67.0, remove all Google Drive storages and retry the upgrade, or continue using 3.67.0. For a database created by 3.67.0, this refusal SHALL leave it usable by that same version on the same volume. This SHALL NOT promise a downgrade after a successful schema upgrade.

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

- **WHEN** all Google Drive storages are removed through 3.67.0 and the operator retries the upgrade
- **THEN** the Google Drive check permits migration and supported application records remain available

#### Scenario: Fresh database

- **WHEN** an operator starts with an empty internal database
- **THEN** the retirement check permits normal initialization

### Requirement: Direct removal migration refuses existing Google Drive storages

Applying the Google Drive removal migration directly SHALL fail if any Google Drive storage remains. That migration SHALL run in one transaction and SHALL NOT delete storages, backup records, credentials or remote files to make the upgrade proceed. Its failure SHALL preserve its input schema and records and SHALL leave that migration unapplied. Its diagnostic SHALL give the same retirement and recovery guidance as startup. Earlier migrations explicitly applied through a separate migration tool are outside the startup rollback guarantee.

When no Google Drive storage remains, the migration SHALL remove its obsolete settings schema while preserving every supported provider's settings and encrypted secrets. A fresh installation SHALL finish without the obsolete schema. Historical migration support for older installation layouts SHALL remain available.

#### Scenario: Direct migration with a remaining storage

- **WHEN** an operator directly applies the removal migration to a database with a Google Drive storage
- **THEN** the migration fails with recovery guidance and its input schema, records, secrets and migration-history entry are unchanged

#### Scenario: Removal with supported storages only

- **WHEN** the removal migration runs with no Google Drive storage and with supported storages present
- **THEN** the obsolete Google Drive settings schema is absent and the supported storages and their secrets are preserved

### Requirement: Retirement diagnostics do not disclose credentials

The Google Drive upgrade check and removal migration SHALL NOT decrypt credentials or include stored client secrets, tokens or serialized provider settings in errors, standard output, standard error or logs.

#### Scenario: Refusal on a storage with saved secrets

- **WHEN** startup or direct removal refuses a Google Drive storage that has stored credentials
- **THEN** the diagnostic gives recovery instructions without containing its client secret, tokens or serialized settings
