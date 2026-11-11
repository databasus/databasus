## MODIFIED Requirements

### Requirement: The application applies the migrations of its internal database at startup

Except when the unsupported-storage upgrade refusal below applies or startup recovery is requested, at every server start the application SHALL apply to the internal database every migration of its version that is not applied yet, and only then start accepting HTTP requests and running background work. This SHALL NOT require any program but the Databasus binary: the image contains no separate migration tool. A migration that is already applied SHALL NOT run again.

Startup recovery and UI file import SHALL validate the complete Databasus backup file, exact source release and schema identity before importing or applying migrations to that snapshot. Schema identity SHALL compare the ordered migration-bundle digest and effective applied versions under `databasus-recovery`: historical applied versions absent from that same release's bundle SHALL remain permitted when they are no newer than its newest migration. A mismatch SHALL change neither the target database nor its key. Successful same-version startup recovery SHALL import into an unused target before ordinary application initialization. UI import SHALL replace an initialized target only after current-root confirmation and live-work quiescence. Both paths SHALL retain the snapshot's migration history, apply no migration to make the archive fit and start paused as `databasus-recovery` specifies. An upgrade SHALL be a subsequent ordinary start, not part of recovery. Successfully consumed startup archives SHALL be skipped before compatibility checks on subsequent starts, so leaving their paths configured SHALL NOT prevent an ordinary upgrade.

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
