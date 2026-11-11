## MODIFIED Requirements

### Requirement: The application applies the migrations of its internal database at startup

Except when the unsupported-storage upgrade refusal below applies or startup recovery is requested, at every server start the application SHALL apply to the internal database every migration of its version that is not applied yet, and only then start accepting HTTP requests and running background work. This SHALL NOT require any program but the Databasus binary: the image contains no separate migration tool. A migration that is already applied SHALL NOT run again.

Startup recovery and UI file import SHALL validate the complete Databasus backup file, exact source release and schema identity before importing or applying migrations to that snapshot. Schema identity SHALL compare the ordered migration-bundle digest and effective applied versions under `databasus-recovery`: historical applied versions absent from that same release's bundle SHALL remain permitted when they are no newer than its newest migration. A mismatch SHALL change neither the target database nor its key. Successful same-version startup recovery SHALL import into a fresh or automatically initialized target before ordinary application initialization. Existing migrations and application tables SHALL NOT prevent this path when no user accounts, user data or user-changed settings exist under `databasus-recovery`; recovery SHALL NOT apply migrations to classify the target or bypass the newer-schema refusal. UI import SHALL replace an initialized target only after current-root confirmation and live-work quiescence. Both paths SHALL retain the snapshot's migration history, apply no migration to make the archive fit and start paused as `databasus-recovery` specifies. An upgrade SHALL be a subsequent ordinary start, not part of recovery. Successfully consumed startup archives SHALL be skipped before archive compatibility checks on subsequent starts, so leaving their paths configured SHALL NOT prevent an ordinary upgrade.

#### Scenario: Fresh install

- **WHEN** the operator starts the image with an empty data volume and no recovery inputs
- **THEN** the application creates the whole schema of the internal database, starts answering requests and shows the sign-up screen

#### Scenario: Upgrade to a new version

- **WHEN** the operator starts a newer image on a volume whose internal database was created by an earlier version and containing no Google Drive or rclone storage
- **THEN** the application applies the migrations the database does not have yet, and every workspace, database, storage, notifier and backup stays in place

#### Scenario: Restart without new migrations

- **WHEN** the application restarts on a database where every migration of its version is applied
- **THEN** the application starts and does not change the schema

#### Scenario: Recovery uses a different release

- **WHEN** startup recovery receives an archive from another release
- **THEN** the process refuses before importing or applying migrations and explains recovery on the source release followed by a separate upgrade

#### Scenario: Recovery files are supplied after automatic initialization

- **WHEN** a normal start has already applied migrations and created only automatic state, and the owner later restarts the same release with valid mounted recovery files and both paths configured
- **THEN** existing tables and migration records do not prevent recovery
- **AND** recovery preserves the imported snapshot's migration history without applying migrations to the target to establish eligibility

#### Scenario: Recovering an upgraded installation of the same release

- **WHEN** the Databasus backup file has the exact target release and migration bundle, all carried migrations applied, and additional older historical applied migrations absent from the current bundle
- **THEN** recovery preserves those historical migration records and starts paused without applying or deleting migrations

#### Scenario: UI imports a same-release snapshot over an initialized target

- **WHEN** root admin confirms a compatible uploaded archive and the running application stops its work and connections for publication
- **THEN** recovery retains the imported migration history without applying or deleting migrations and returns the application paused

#### Scenario: UI imports a snapshot from another release

- **WHEN** root admin uploads an archive from a different release to an initialized target
- **THEN** preparation refuses it before import without changing the active database or its key
- **AND** it does not migrate the archive to the target release

#### Scenario: Upgrade retains consumed startup archive settings

- **WHEN** an instance starts a newer release while the archive already consumed by successful startup recovery and both file paths remain configured
- **THEN** it skips that archive and applies only the ordinary pending upgrade migrations to its current database

### Requirement: The application refuses to start on a schema newer than its own

Before applying migrations, the application SHALL compare the newest migration recorded in the internal database with the newest migration it carries. When the database records a newer migration, the server and every console command that opens the internal database SHALL exit with a non-zero code without applying any migration, opening the HTTP port or changing the database. The message SHALL state the newest migration version (its timestamp) recorded in the database and the newest migration version the running binary carries, and SHALL say that returning to an older version after an upgrade is not supported and that the newer image has to be started instead to retain the installation.

The same diagnostic SHALL also explain that an owner who deliberately wants the older release can delete the current installation's data and recreate the deployment with that image as a fresh installation. Before cleanup commands, it SHALL warn that accounts, settings, the master key and locally stored backup files will be lost, and instruct the owner to retain needed archives, the master key and local files outside the storage being removed. It SHALL distinguish this destructive reset from a downgrade or recovery of a newer-release archive. It SHALL provide separate, labeled command examples for removing the container and the documented bind-mounted data directory, removing the container and its named Docker data volume, and removing the container together with its anonymous data volume when no explicit mount was supplied. Example container names, directories and volume names SHALL be identified as values to adapt to the deployment. The application SHALL NOT execute cleanup or bypass the schema check. The examples SHALL be scoped to the supplied embedded-database deployment and SHALL NOT claim that deleting a volume resets an externally configured database.

A migration recorded in the database that the running version does not carry, but that is older than the running version's newest migration, SHALL NOT stop the start. Releases have shipped migrations that later releases deleted or renamed, and instances that applied them SHALL keep upgrading.

#### Scenario: Operator rolls the image back after an upgrade

- **WHEN** an instance was upgraded to a version with a new migration and the operator starts the previous image on the same volume
- **THEN** the process exits with a non-zero code before opening the HTTP port
- **AND** the message states both migration versions, says that going back after an upgrade is not supported and recommends the newer image to preserve the installation
- **AND** it describes the destructive fresh-install alternative with separate cleanup commands for bind-mounted, named-volume and anonymous-volume data and an instruction to recreate the selected older image
- **AND** the internal database, master key and local backup files are unchanged

#### Scenario: Console command on a newer schema

- **WHEN** the operator runs a password reset of an older image on a database migrated by a newer version
- **THEN** the command exits with a non-zero code with the same refusal and fresh-install guidance and changes nothing

#### Scenario: Reset warning precedes every storage-specific example

- **WHEN** the newer-schema refusal displays cleanup commands
- **THEN** it first explains their data loss and the need to retain recovery material outside the deleted storage
- **AND** it does not claim that container removal alone deletes a bind mount or a named Docker volume

#### Scenario: A migration that a later release removed

- **WHEN** the internal database records a migration that was shipped in an earlier release and later deleted, and every migration it records is older than the running version's newest one
- **THEN** the application applies its missing migrations and starts
