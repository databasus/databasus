## MODIFIED Requirements

### Requirement: The application applies the migrations of its internal database at startup

Except when the unsupported-storage upgrade refusal below applies or startup recovery is requested, at every server start the application SHALL apply to the internal database every migration of its version that is not applied yet, and only then start accepting HTTP requests and running background work. This SHALL NOT require any program but the Databasus binary: the image contains no separate migration tool. A migration that is already applied SHALL NOT run again.

Startup recovery SHALL validate the complete Databasus backup file, exact source release and schema identity before importing or applying migrations. Schema identity SHALL compare the ordered migration-bundle digest and effective applied versions under `databasus-recovery`: historical applied versions absent from that same release's bundle SHALL remain permitted when they are no newer than its newest migration. A mismatch SHALL change neither the target database nor its key. Successful same-version recovery SHALL import into an unused target before ordinary application initialization, retain the snapshot's migration history and start paused as `databasus-recovery` specifies. An upgrade SHALL be a subsequent ordinary start, not part of recovery.

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

#### Scenario: Recovering an upgraded installation of the same release

- **WHEN** the Databasus backup file has the exact target release and migration bundle, all carried migrations applied, and additional older historical applied migrations absent from the current bundle
- **THEN** recovery preserves those historical migration records and starts paused without applying or deleting migrations
